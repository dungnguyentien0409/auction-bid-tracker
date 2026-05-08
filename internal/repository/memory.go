package repository

import (
	"context"
	"sync"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string][]domain.Bid
	users map[string]map[string]struct{}
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		items: make(map[string][]domain.Bid),
		users: make(map[string]map[string]struct{}),
	}
}

func (r *MemoryRepository) SaveBid(ctx context.Context, bid *domain.Bid) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	bids := r.items[bid.ItemID]

	if len(bids) > 0 {
		highestBid := bids[len(bids)-1]
		if bid.Amount <= highestBid.Amount {
			return domain.ErrBidTooLow
		}
	}

	r.items[bid.ItemID] = append(bids, *bid)

	if r.users[bid.UserID] == nil {
		r.users[bid.UserID] = make(map[string]struct{})
	}
	r.users[bid.UserID][bid.ItemID] = struct{}{}

	return nil
}

func (r *MemoryRepository) GetWinningBid(ctx context.Context, itemID string) (*domain.Bid, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bids := r.items[itemID]
	if len(bids) == 0 {
		return nil, domain.ErrNotFound
	}

	winningBid := bids[len(bids)-1]
	return &winningBid, nil
}

func (r *MemoryRepository) GetAllBids(ctx context.Context, itemID string) ([]domain.Bid, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bids := r.items[itemID]
	if len(bids) == 0 {
		return []domain.Bid{}, nil
	}

	// Make a copy to avoid race conditions when the caller reads the slice
	res := make([]domain.Bid, len(bids))
	copy(res, bids)
	return res, nil
}

func (r *MemoryRepository) GetUserItems(ctx context.Context, userID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userItems := r.users[userID]
	if len(userItems) == 0 {
		return []string{}, nil
	}

	res := make([]string, 0, len(userItems))
	for itemID := range userItems {
		res = append(res, itemID)
	}
	return res, nil
}
