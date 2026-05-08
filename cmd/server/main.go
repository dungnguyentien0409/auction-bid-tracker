package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/api"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/repository"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/service"
)

func main() {
	repo := repository.NewMemoryRepository()
	bidService := service.NewBidService(repo)
	handler := api.NewHandler(bidService)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Channel to listen for interrupt or terminate signals from the OS
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Run the server in a separate goroutine so it doesn't block
	go func() {
		log.Println("server started on :8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Block until a signal is received
	<-quit
	log.Println("server is shutting down...")

	// Create a context with a 10-second timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Instruct the server to stop accepting new requests and finish existing ones
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited properly")
}
