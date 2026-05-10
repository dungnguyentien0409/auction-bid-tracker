package api

import (
	"encoding/json"
	"net/http"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

type Handler struct {
	tracker domain.Tracker
}

func NewHandler(tracker domain.Tracker) *Handler {
	return &Handler{tracker: tracker}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.HandleHealth)
	mux.HandleFunc("POST /bids", h.HandleRecordBid)
	mux.HandleFunc("GET /items/{id}/winning-bid", h.HandleGetWinningBid)
	mux.HandleFunc("GET /items/{id}/bids", h.HandleGetAllBids)
	mux.HandleFunc("GET /users/{id}/items", h.HandleGetUserItems)
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

type RecordBidRequest struct {
	ItemID string  `json:"item_id"`
	UserID string  `json:"user_id"`
	Amount float64 `json:"amount"`
}

func (h *Handler) HandleRecordBid(w http.ResponseWriter, r *http.Request) {
	var req RecordBidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ItemID == "" || req.UserID == "" || req.Amount <= 0 {
		http.Error(w, "invalid request: missing fields or amount <= 0", http.StatusBadRequest)
		return
	}

	bid, err := h.tracker.RecordBid(r.Context(), req.ItemID, req.UserID, req.Amount)
	if err != nil {
		if err == domain.ErrBidTooLow {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(bid)
}

func (h *Handler) HandleGetWinningBid(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("id")
	if itemID == "" {
		http.Error(w, "missing item id", http.StatusBadRequest)
		return
	}

	bid, err := h.tracker.GetWinningBid(r.Context(), itemID)
	if err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "winning bid not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bid)
}

func (h *Handler) HandleGetAllBids(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("id")
	if itemID == "" {
		http.Error(w, "missing item id", http.StatusBadRequest)
		return
	}

	bids, err := h.tracker.GetAllBids(r.Context(), itemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bids)
}

func (h *Handler) HandleGetUserItems(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, "missing user id", http.StatusBadRequest)
		return
	}

	items, err := h.tracker.GetUserItems(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}
