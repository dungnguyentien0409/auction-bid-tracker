//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/api"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/repository"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/service"
)

func setupServer() (*httptest.Server, *service.BidService) {
	repo := repository.NewMemoryRepository()
	bidService := service.NewBidService(repo)
	handler := api.NewHandler(bidService)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	server := httptest.NewServer(mux)
	return server, bidService
}

func TestIntegration_AuctionJourney(t *testing.T) {
	server, _ := setupServer()
	defer server.Close()

	itemID := "vintage_car"
	userA := "user_a"
	userB := "user_b"

	// 1. User A bids 100
	reqBody1 := map[string]interface{}{
		"item_id": itemID,
		"user_id": userA,
		"amount":  100.0,
	}
	bodyBytes, _ := json.Marshal(reqBody1)
	resp, err := http.Post(server.URL+"/bids", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("failed to post bid: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %v", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 2. User B bids 50 (Too Low)
	reqBody2 := map[string]interface{}{
		"item_id": itemID,
		"user_id": userB,
		"amount":  50.0,
	}
	bodyBytes, _ = json.Marshal(reqBody2)
	resp, err = http.Post(server.URL+"/bids", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("failed to post bid: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409 for low bid, got %v", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 3. User B bids 150 (Success)
	reqBody3 := map[string]interface{}{
		"item_id": itemID,
		"user_id": userB,
		"amount":  150.0,
	}
	bodyBytes, _ = json.Marshal(reqBody3)
	resp, err = http.Post(server.URL+"/bids", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("failed to post bid: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %v", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 4. Get Winning Bid (Should be 150)
	resp, err = http.Get(server.URL + "/items/" + itemID + "/winning-bid")
	if err != nil {
		t.Fatalf("failed to get winning bid: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %v", resp.StatusCode)
	}
	var winningBid domain.Bid
	if err := json.NewDecoder(resp.Body).Decode(&winningBid); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	_ = resp.Body.Close()

	if winningBid.Amount != 150.0 {
		t.Errorf("expected winning bid amount 150.0, got %v", winningBid.Amount)
	}
	if winningBid.UserID != userB {
		t.Errorf("expected winning user user_b, got %v", winningBid.UserID)
	}

	// 5. Get All Bids for Item
	resp, err = http.Get(server.URL + "/items/" + itemID + "/bids")
	if err != nil {
		t.Fatalf("failed to get all bids: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %v", resp.StatusCode)
	}
	var allBids []domain.Bid
	if err := json.NewDecoder(resp.Body).Decode(&allBids); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	_ = resp.Body.Close()

	if len(allBids) != 2 {
		t.Errorf("expected 2 valid bids, got %v", len(allBids))
	}

	// 6. Get User A Items
	resp, err = http.Get(server.URL + "/users/" + userA + "/items")
	if err != nil {
		t.Fatalf("failed to get user items: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %v", resp.StatusCode)
	}
	var items []string
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	_ = resp.Body.Close()

	if len(items) != 1 || items[0] != itemID {
		t.Errorf("expected 1 item for user_a matching %s, got %v", itemID, items)
	}
}

func TestIntegration_Endpoints(t *testing.T) {
	t.Run("POST_Bid_Success", func(t *testing.T) {
		t.Parallel()
		server, _ := setupServer()
		defer server.Close()

		reqBody := map[string]interface{}{
			"item_id": "item_1",
			"user_id": "user_1",
			"amount":  100.0,
		}
		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(server.URL+"/bids", "application/json", bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected 201, got %v", resp.StatusCode)
		}
		_ = resp.Body.Close()
	})

	t.Run("POST_Bid_TooLow", func(t *testing.T) {
		t.Parallel()
		server, bidService := setupServer()
		defer server.Close()

		// Pre-seed a bid directly into service
		_, _ = bidService.RecordBid(context.Background(), "item_2", "user_1", 200.0)

		reqBody := map[string]interface{}{
			"item_id": "item_2",
			"user_id": "user_2",
			"amount":  100.0, // Lower than 200
		}
		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(server.URL+"/bids", "application/json", bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("expected 409, got %v", resp.StatusCode)
		}
		_ = resp.Body.Close()
	})

	t.Run("GET_WinningBid_Empty", func(t *testing.T) {
		t.Parallel()
		server, _ := setupServer()
		defer server.Close()

		resp, err := http.Get(server.URL + "/items/empty_item/winning-bid")
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %v", resp.StatusCode)
		}
		_ = resp.Body.Close()
	})

	t.Run("GET_AllBids_Empty", func(t *testing.T) {
		t.Parallel()
		server, _ := setupServer()
		defer server.Close()

		resp, err := http.Get(server.URL + "/items/empty_item/bids")
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}
		
		var bids []domain.Bid
		_ = json.NewDecoder(resp.Body).Decode(&bids)
		if len(bids) != 0 {
			t.Errorf("expected empty array, got len %v", len(bids))
		}
		_ = resp.Body.Close()
	})

	t.Run("GET_UserItems_Empty", func(t *testing.T) {
		t.Parallel()
		server, _ := setupServer()
		defer server.Close()

		resp, err := http.Get(server.URL + "/users/empty_user/items")
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}
		
		var items []string
		_ = json.NewDecoder(resp.Body).Decode(&items)
		if len(items) != 0 {
			t.Errorf("expected empty array, got len %v", len(items))
		}
		_ = resp.Body.Close()
	})
}
