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

	"borrowhub/workers/notification/internal/auth"
	"borrowhub/workers/notification/internal/config"
	"borrowhub/workers/notification/internal/consumer"
	"borrowhub/workers/notification/internal/email"
	"borrowhub/workers/notification/internal/httpapi"
	"borrowhub/workers/notification/internal/repository"
	"borrowhub/workers/notification/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	serviceName    = "notification-worker"
	defaultAddress = ":8084"
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

	repo := repository.NewRepository(pool)
	mailer := email.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	notifConsumer := consumer.NewConsumer(cfg.RabbitMQURL, repo, mailer, logger)
	notifSvc := service.NewNotificationService(repo)
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret)

	var ready atomic.Bool

	readyCheck := func() bool {
		if !ready.Load() {
			return false
		}
		if !notifConsumer.IsReady() {
			return false
		}
		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return pool.Ping(pingCtx) == nil
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	if err := notifConsumer.Start(ctx); err != nil {
		logger.Error(
			"start rabbitmq consumer",
			"error",
			err,
		)
		os.Exit(1)
	}

	router := httpapi.NewRouter(httpapi.RouterDependencies{
		Service:       notifSvc,
		TokenManager:  jwtMgr,
		InternalToken: cfg.InternalToken,
		ReadyCheck:    readyCheck,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info(
			"notification worker started",
			"http_address",
			cfg.HTTPAddress,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
	}()

	ready.Store(true)

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrCh:
		logger.Error("server error", "error", err)
	}

	ready.Store(false)

	notifConsumer.Stop()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error(
			"server shutdown",
			"error",
			err,
		)
	}

	logger.Info("notification worker shutdown completed")
}
