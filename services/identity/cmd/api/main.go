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

	"borrowhub/services/identity/internal/config"
	"borrowhub/services/identity/internal/httpapi"
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

	var ready atomic.Bool

	mux := http.NewServeMux()

	mux.HandleFunc(
		"GET /health/live",
		func(w http.ResponseWriter, r *http.Request) {
			httpapi.WriteJSON(
				w,
				r,
				http.StatusOK,
				map[string]string{
					"status": "ok",
				},
			)
		},
	)

	mux.HandleFunc(
		"GET /health/ready",
		func(w http.ResponseWriter, r *http.Request) {
			if !ready.Load() {
				httpapi.WriteError(
					w,
					r,
					http.StatusServiceUnavailable,
					"SERVICE_NOT_READY",
					"Service belum siap menerima request.",
					nil,
				)
				return
			}

			httpapi.WriteJSON(
				w,
				r,
				http.StatusOK,
				map[string]string{
					"status": "ok",
				},
			)
		},
	)

	mux.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			httpapi.WriteError(
				w,
				r,
				http.StatusNotFound,
				"ROUTE_NOT_FOUND",
				"Route tidak ditemukan.",
				nil,
			)
		},
	)

	server := &http.Server{
		Addr: cfg.HTTPAddress,
		Handler: httpapi.RequestIDs(
			httpapi.Logging(logger)(mux),
		),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		<-ctx.Done()

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
			"server stopped",
			"error",
			err,
		)
		os.Exit(1)
	}
}
