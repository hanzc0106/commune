package main

import (
	"context"
	"errors"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hanzc0106/commune/apps/api/internal/app"
	"github.com/hanzc0106/commune/apps/api/internal/config"
	"github.com/hanzc0106/commune/apps/api/internal/db"
	apphttp "github.com/hanzc0106/commune/apps/api/internal/http"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database_open_failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	service := app.NewService(pool)
	handler := apphttp.NewHandler(apphttp.Options{
		StaticDir:  cfg.StaticDir,
		APIHandler: apphttp.NewAPI(service),
	})

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := &stdhttp.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := serveHTTP(runCtx, server, logger, 10*time.Second); err != nil {
		logger.Error("server_failed", "error", err)
		os.Exit(1)
	}
}

func serveHTTP(ctx context.Context, server *stdhttp.Server, logger *slog.Logger, shutdownTimeout time.Duration) error {
	errCh := make(chan error, 1)
	go func() {
		logger.Info("server_starting", "addr", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, stdhttp.ErrServerClosed) {
			logger.Info("server_stopped", "addr", server.Addr)
			return nil
		}
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	err := <-errCh
	if err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
		return err
	}

	logger.Info("server_stopped", "addr", server.Addr)
	return nil
}
