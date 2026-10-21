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

	"borrowhub/services/inventory/internal/auth"
	"borrowhub/services/inventory/internal/config"
	"borrowhub/services/inventory/internal/httpapi"
	"borrowhub/services/inventory/internal/repository"
	"borrowhub/services/inventory/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	serviceName    = "inventory"
	defaultAddress = ":8082"
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
	repos := repository.NewRepositories(pool)

	categorySvc := service.NewCategoryService(repos.Categories, identityClient)
	itemSvc := service.NewItemService(repos.Items, repos.Categories, identityClient)
	unitSvc := service.NewUnitService(repos.Units, repos.Items, identityClient)
	internalSvc := service.NewInternalInventoryService(repos.Units, repos.Items)

	router := httpapi.NewRouter(httpapi.RouterDependencies{
		CategoryService: categorySvc,
		ItemService:     itemSvc,
		UnitService:     unitSvc,
		InternalService: internalSvc,
		TokenManager:    jwtMgr,
		ReadyCheck:      readyCheck,
		InternalToken:   cfg.InternalToken,
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

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

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
			"listen and serve",
			"error",
			err,
		)
		os.Exit(1)
	}
}
