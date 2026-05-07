package main

import (
	"log"
	"net/http"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/api"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/tracker"
)

func main() {
	memTracker := tracker.NewMemoryTracker()
	handler := api.NewHandler(memTracker)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	log.Println("server started on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
