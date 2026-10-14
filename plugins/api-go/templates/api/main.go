package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webapp/internal/shell"
)

func main() {
	// Structured JSON from the first line. A log aggregator can filter and
	// group these; the correlation id the server stamps on every request line
	// is what joins them back into one request.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	handler, err := shell.Boot(os.Getenv, logger)
	if err != nil {
		logger.Error("starting the server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Every interface, or loopback only when the local test issuer is running.
	addr := shell.ListenAddr(os.Getenv)

	// Explicit timeouts. The zero-value server has none, so a client that
	// opens a connection and never finishes the request holds it forever.
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// SIGTERM (a container stop) and SIGINT drain in-flight requests, within a
	// bound, instead of cutting them off.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ListenAndServe() }()
	logger.Info("listening", slog.String("addr", addr))

	select {
	case err := <-serveErr:
		logger.Error("server stopped", slog.String("error", err.Error()))
		os.Exit(1)
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	drain, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(drain); err != nil {
		logger.Error("shutdown did not drain", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
