package main

import (
	"log/slog"
	"net/http"
	"os"
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

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

	logger.Info("listening", slog.String("addr", addr))
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
