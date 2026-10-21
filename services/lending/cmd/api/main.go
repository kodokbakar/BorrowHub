package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"borrowhub/services/lending/internal/auth"
	"borrowhub/services/lending/internal/client"
	"borrowhub/services/lending/internal/config"
	"borrowhub/services/lending/internal/event"
	"borrowhub/services/lending/internal/httpapi"
	"borrowhub/services/lending/internal/repository"
	"borrowhub/services/lending/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	serviceName    = "lending"
	defaultAddress = ":8083"
)

func main() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, nil),
	).With(
		"service",
		serviceName,
	)

	cfg, err := config.Load(defaultAddress)
	if err != nil {
		logger.Error(
			"load configuration",
			"error",
			err,
		)
		os.Exit(1)
	}

	logger = logger.With(
		"environment",
		cfg.Environment,
	)

	poolCtx, poolCancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := pgxpool.New(poolCtx, cfg.DatabaseURL)
	poolCancel()
	if err != nil {
		logger.Error(
			"connect to database",
			"error",
			err,
		)
		os.Exit(1)
	}
	defer pool.Close()

	if err := repository.MigrateUp(cfg.DatabaseURL); err != nil {
		logger.Error(
			"run migrations",
			"error",
			err,
		)
		os.Exit(1)
	}

	var ready atomic.Bool

	readyCheck := func() bool {
		if !ready.Load() {
			return false
		}
		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return pool.Ping(pingCtx) == nil
	}

	jwtMgr := auth.NewJWTManager(cfg.JWTSecret)
	identityClient := auth.NewHTTPIdentityClient(cfg.IdentityBaseURL, cfg.InternalToken, 1*time.Minute)
	inventoryClient := client.NewHTTPInventoryClient(cfg.InventoryBaseURL, cfg.InternalToken)

	repo := repository.NewRepository(pool)
	lendingSvc := service.NewLendingService(repo, identityClient, inventoryClient)

	// Background outbox publisher
	outboxPublisher := event.NewOutboxPublisher(repo, cfg.RabbitMQURL, logger, 500*time.Millisecond)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	outboxPublisher.Start(ctx)

	router := httpapi.NewRouter(httpapi.RouterDependencies{
		LendingService: lendingSvc,
		Repository:     repo,
		TokenManager:   jwtMgr,
		ReadyCheck:     readyCheck,
		InternalToken:  cfg.InternalToken,
	})

	handler := httpapi.RequestIDs(httpapi.Logging(logger)(router))

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		ready.Store(false)

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error(
				"server shutdown",
				"error",
				err,
			)
		}

		outboxPublisher.Stop()
	}()

	ready.Store(true)

	logger.Info(
		"server started",
		"address",
		cfg.HTTPAddress,
	)

	if err := server.ListenAndServe(); !errors.Is(
		err,
		http.ErrServerClosed,
	) {
		logger.Error(
			"server error",
			"error",
			err,
		)
		os.Exit(1)
	}
}
