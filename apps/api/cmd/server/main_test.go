package main

import (
	"context"
	"io"
	"log/slog"
	stdhttp "net/http"
	"testing"
	"time"
)

func TestServeHTTPStopsCleanlyWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := &stdhttp.Server{
		Addr:    "127.0.0.1:0",
		Handler: stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {}),
	}

	done := make(chan error, 1)
	go func() {
		done <- serveHTTP(ctx, server, slog.New(slog.NewTextHandler(io.Discard, nil)), 2*time.Second)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveHTTP returned error after context cancellation: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveHTTP did not stop after context cancellation")
	}
}
