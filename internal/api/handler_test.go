package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/repository"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/service"
)

func TestHandler_RecordBid(t *testing.T) {
	memTracker := service.NewBidService(repository.NewMemoryRepository())
	handler := NewHandler(memTracker)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	reqBody := RecordBidRequest{
		ItemID: "item1",
		UserID: "user1",
		Amount: 100.0,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/bids", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var bid domain.Bid
	_ = json.NewDecoder(w.Body).Decode(&bid)
	if bid.Amount != 100.0 {
		t.Errorf("expected amount 100.0, got %v", bid.Amount)
	}

	reqBodyLow := RecordBidRequest{
		ItemID: "item1",
		UserID: "user2",
		Amount: 50.0,
	}
	bodyLow, _ := json.Marshal(reqBodyLow)
	reqLow := httptest.NewRequest(http.MethodPost, "/bids", bytes.NewReader(bodyLow))
	reqLow.Header.Set("Content-Type", "application/json")
	wLow := httptest.NewRecorder()

	mux.ServeHTTP(wLow, reqLow)

	if wLow.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, wLow.Code)
	}
}

func TestHandler_GetWinningBid(t *testing.T) {
	memTracker := service.NewBidService(repository.NewMemoryRepository())
	handler := NewHandler(memTracker)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	reqBody := RecordBidRequest{
		ItemID: "item1",
		UserID: "user1",
		Amount: 100.0,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/bids", bytes.NewReader(body))
	mux.ServeHTTP(httptest.NewRecorder(), req)

	reqGet := httptest.NewRequest(http.MethodGet, "/items/item1/winning-bid", nil)
	wGet := httptest.NewRecorder()

	mux.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, wGet.Code)
	}

	var bid domain.Bid
	_ = json.NewDecoder(wGet.Body).Decode(&bid)
	if bid.Amount != 100.0 {
		t.Errorf("expected winning bid amount 100.0, got %v", bid.Amount)
	}

	reqNotFound := httptest.NewRequest(http.MethodGet, "/items/item2/winning-bid", nil)
	wNotFound := httptest.NewRecorder()
	mux.ServeHTTP(wNotFound, reqNotFound)

	if wNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, wNotFound.Code)
	}
}

func TestHandler_RecordBid_Invalid(t *testing.T) {
	memTracker := service.NewBidService(repository.NewMemoryRepository())
	handler := NewHandler(memTracker)

	req := httptest.NewRequest(http.MethodPost, "/bids", bytes.NewReader([]byte("{invalid")))
	w := httptest.NewRecorder()
	handler.HandleRecordBid(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	reqBody := RecordBidRequest{Amount: -1}
	body, _ := json.Marshal(reqBody)
	req2 := httptest.NewRequest(http.MethodPost, "/bids", bytes.NewReader(body))
	w2 := httptest.NewRecorder()
	handler.HandleRecordBid(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w2.Code)
	}
}

func TestHandler_GetAllBids(t *testing.T) {
	memTracker := service.NewBidService(repository.NewMemoryRepository())
	handler := NewHandler(memTracker)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	reqGet := httptest.NewRequest(http.MethodGet, "/items/item1/bids", nil)
	wGet := httptest.NewRecorder()
	mux.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", wGet.Code)
	}
}

func TestHandler_GetUserItems(t *testing.T) {
	memTracker := service.NewBidService(repository.NewMemoryRepository())
	handler := NewHandler(memTracker)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	reqGet := httptest.NewRequest(http.MethodGet, "/users/user1/items", nil)
	wGet := httptest.NewRecorder()
	mux.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", wGet.Code)
	}
}

func TestHandler_EmptyPathValues(t *testing.T) {
	memTracker := service.NewBidService(repository.NewMemoryRepository())
	handler := NewHandler(memTracker)

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	w1 := httptest.NewRecorder()
	handler.HandleGetWinningBid(w1, req1)
	if w1.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	w2 := httptest.NewRecorder()
	handler.HandleGetAllBids(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w2.Code)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	w3 := httptest.NewRecorder()
	handler.HandleGetUserItems(w3, req3)
	if w3.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w3.Code)
	}
}

type mockErrorTracker struct{}
func (m *mockErrorTracker) RecordBid(ctx context.Context, itemID, userID string, amount float64) (*domain.Bid, error) { return nil, domain.ErrNotFound }
func (m *mockErrorTracker) GetWinningBid(ctx context.Context, itemID string) (*domain.Bid, error) { return nil, domain.ErrBidTooLow }
func (m *mockErrorTracker) GetAllBids(ctx context.Context, itemID string) ([]domain.Bid, error) { return nil, domain.ErrNotFound }
func (m *mockErrorTracker) GetUserItems(ctx context.Context, userID string) ([]string, error) { return nil, domain.ErrNotFound }

func TestHandler_ErrorCases(t *testing.T) {
	handler := NewHandler(&mockErrorTracker{})
	
	reqBody := RecordBidRequest{ItemID: "i", UserID: "u", Amount: 10}
	body, _ := json.Marshal(reqBody)
	req1 := httptest.NewRequest(http.MethodPost, "/bids", bytes.NewReader(body))
	w1 := httptest.NewRecorder()
	handler.HandleRecordBid(w1, req1)
	if w1.Code != http.StatusInternalServerError {
		t.Errorf("expected 500")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.SetPathValue("id", "i")
	w2 := httptest.NewRecorder()
	handler.HandleGetWinningBid(w2, req2)
	if w2.Code != http.StatusInternalServerError {
		t.Errorf("expected 500")
	}

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.SetPathValue("id", "i")
	w3 := httptest.NewRecorder()
	handler.HandleGetAllBids(w3, req3)
	if w3.Code != http.StatusInternalServerError {
		t.Errorf("expected 500")
	}

	req4 := httptest.NewRequest(http.MethodGet, "/", nil)
	req4.SetPathValue("id", "i")
	w4 := httptest.NewRecorder()
	handler.HandleGetUserItems(w4, req4)
	if w4.Code != http.StatusInternalServerError {
		t.Errorf("expected 500")
	}
}
