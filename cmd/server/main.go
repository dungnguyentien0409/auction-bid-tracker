package main

import (
	"log"
	"net/http"

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

	log.Println("server started on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
