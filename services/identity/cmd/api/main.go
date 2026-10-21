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

	"borrowhub/services/identity/internal/auth"
	"borrowhub/services/identity/internal/config"
	"borrowhub/services/identity/internal/httpapi"
	"borrowhub/services/identity/internal/repository"
	"borrowhub/services/identity/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	serviceName    = "identity"
	defaultAddress = ":8081"
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
	hasher := auth.DefaultPasswordHasher()
	repos := repository.NewRepositories(pool)

	authSvc := service.NewAuthService(repos.Users, repos.Organizations, repos.Sessions, hasher, jwtMgr)
	orgSvc := service.NewOrgService(repos.Organizations, repos.Memberships)
	memberSvc := service.NewMemberService(repos.Users, repos.Organizations, repos.Memberships, repos.Invitations)

	router := httpapi.NewRouter(httpapi.RouterDependencies{
		AuthService:   authSvc,
		OrgService:    orgSvc,
		MemberService: memberSvc,
		TokenManager:  jwtMgr,
		ReadyCheck:    readyCheck,
		IsSecure:      cfg.Environment != "local",
		InternalToken: cfg.InternalToken,
		Logger:        logger,
	})
	server := &http.Server{
		Addr: cfg.HTTPAddress,
		Handler: httpapi.RequestIDs(
			httpapi.Logging(logger)(router),
		),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info(
			"server started",
			"address",
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

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error(
			"shutdown server",
			"error",
			err,
		)
	}

	logger.Info("server shutdown completed")
}
