package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"url-shortener/internal/api"
	"url-shortener/internal/config"
	"url-shortener/internal/storage/memory"
	"url-shortener/internal/storage/postgres"

	"github.com/MatusOllah/slogcolor"
	"golang.org/x/sync/errgroup"
)

const (
	envLocal = "local"
	envProd  = "prod"

	shutdownTimeout = 10 * time.Second
)

func main() {
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, log); err != nil {
		log.Error("service exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("service stopped")
}

func run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	log.Info("starting the app", slog.String("storage", cfg.StorageType))

	store, closeStore, err := newStore(ctx, cfg)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	defer closeStore()

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Host + ":" + strconv.Itoa(cfg.HTTPServer.Port),
		Handler:      api.NewServer(log, cfg.BaseURL, store),
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Info("listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		log.Info("shutting down gracefully")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	})

	return g.Wait()
}

func newStore(ctx context.Context, cfg *config.Config) (api.Store, func(), error) {
	switch cfg.StorageType {
	case config.StoragePostgres:
		st, err := postgres.New(ctx, cfg.Postgres.DSN())
		if err != nil {
			return nil, nil, err
		}
		return st, func() { _ = st.Close() }, nil

	case config.StorageMemory, "":
		return memory.New(), func() {}, nil

	default:
		return nil, nil, fmt.Errorf("unknown storage type %q (want %q or %q)",
			cfg.StorageType, config.StorageMemory, config.StoragePostgres)
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	opts := *slogcolor.DefaultOptions

	switch env {
	case envLocal:
		opts.Level = slog.LevelDebug
		log = slog.New(slogcolor.NewHandler(os.Stderr, &opts))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		opts.Level = slog.LevelInfo
		log = slog.New(slogcolor.NewHandler(os.Stderr, &opts))
	}

	return log
}
