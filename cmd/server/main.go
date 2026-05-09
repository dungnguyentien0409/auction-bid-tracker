package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/api"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/api/middleware"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/config"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/repository"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/service"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Initialize structured logger
	var logHandler slog.Handler
	if env == "production" {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(logHandler))

	cfg, err := config.Load(env)
	if err != nil {
		slog.Error("failed to load config", "env", env, "error", err)
		os.Exit(1)
	}

	repo := repository.NewMemoryRepository()
	bidService := service.NewBidService(repo)
	apiHandler := api.NewHandler(bidService)

	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)

	// Apply Middlewares (Chain: Recovery)
	var finalHandler http.Handler = mux
	finalHandler = middleware.Recovery(finalHandler)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      finalHandler,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Channel to listen for interrupt or terminate signals from the OS
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Run the server in a separate goroutine so it doesn't block
	go func() {
		slog.Info("server started", "addr", addr, "env", env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Block until a signal is received
	<-quit
	slog.Info("server is shutting down...")

	// Create a context with a 10-second timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Instruct the server to stop accepting new requests and finish existing ones
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server exited properly")
}
