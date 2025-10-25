package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/example/intent/backend/internal/database"
	"github.com/example/intent/backend/internal/handlers"
	"github.com/example/intent/backend/internal/logging"
)

func main() {
	logger := logging.NewLogger()
	logger.Info("starting intent backend")

	db, err := setupDatabase(logger)
	if err != nil {
		logger.Error("unable to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      routes(logger, db),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server exited", "error", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("shutting down", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}

func setupDatabase(logger *slog.Logger) (*sql.DB, error) {
	port := 5432
	if value := os.Getenv("DB_PORT"); value != "" {
		fmtPort, err := strconv.Atoi(value)
		if err == nil {
			port = fmtPort
		} else {
			logger.Warn("invalid DB_PORT, using default", "value", value, "error", err)
		}
	}

	cfg := database.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     port,
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		Database: getEnv("DB_NAME", "intent"),
	}

	db, err := database.Open(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func routes(logger *slog.Logger, db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/hello", handlers.HelloHandler(logger, db))
	intentsHandler := handlers.IntentsHandler(logger, db)
	mux.Handle("/api/intents", intentsHandler)
	mux.Handle("/api/intents/", intentsHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	staticDir := getEnv("WEB_ROOT", "./frontend/dist")
	fileServer := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", fileServer)

	return mux
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
