package repository

import (
	"context"
	"sync"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

type itemRecord struct {
	mu   sync.RWMutex
	bids []domain.Bid
}

type userRecord struct {
	mu    sync.RWMutex
	items map[string]struct{}
}

type MemoryRepository struct {
	items sync.Map // maps string -> *itemRecord
	users sync.Map // maps string -> *userRecord
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{}
}

func (r *MemoryRepository) SaveBid(ctx context.Context, bid *domain.Bid) error {
	// 1. Save bid with item-level lock
	itemVal, _ := r.items.LoadOrStore(bid.ItemID, &itemRecord{})
	item := itemVal.(*itemRecord)

	item.mu.Lock()
	if len(item.bids) > 0 {
		highestBid := item.bids[len(item.bids)-1]
		if bid.Amount <= highestBid.Amount {
			item.mu.Unlock()
			return domain.ErrBidTooLow
		}
	}
	item.bids = append(item.bids, *bid)
	item.mu.Unlock()

	// 2. Track user item with user-level lock
	userVal, _ := r.users.LoadOrStore(bid.UserID, &userRecord{
		items: make(map[string]struct{}),
	})
	user := userVal.(*userRecord)

	user.mu.Lock()
	user.items[bid.ItemID] = struct{}{}
	user.mu.Unlock()

	return nil
}

func (r *MemoryRepository) GetWinningBid(ctx context.Context, itemID string) (*domain.Bid, error) {
	itemVal, ok := r.items.Load(itemID)
	if !ok {
		return nil, domain.ErrNotFound
	}
	item := itemVal.(*itemRecord)

	item.mu.RLock()
	defer item.mu.RUnlock()

	if len(item.bids) == 0 {
		return nil, domain.ErrNotFound
	}

	winningBid := item.bids[len(item.bids)-1]
	return &winningBid, nil
}

func (r *MemoryRepository) GetAllBids(ctx context.Context, itemID string) ([]domain.Bid, error) {
	itemVal, ok := r.items.Load(itemID)
	if !ok {
		return []domain.Bid{}, nil
	}
	item := itemVal.(*itemRecord)

	item.mu.RLock()
	defer item.mu.RUnlock()

	if len(item.bids) == 0 {
		return []domain.Bid{}, nil
	}

	res := make([]domain.Bid, len(item.bids))
	copy(res, item.bids)
	return res, nil
}

func (r *MemoryRepository) GetUserItems(ctx context.Context, userID string) ([]string, error) {
	userVal, ok := r.users.Load(userID)
	if !ok {
		return []string{}, nil
	}
	user := userVal.(*userRecord)

	user.mu.RLock()
	defer user.mu.RUnlock()

	if len(user.items) == 0 {
		return []string{}, nil
	}

	res := make([]string, 0, len(user.items))
	for itemID := range user.items {
		res = append(res, itemID)
	}
	return res, nil
}
