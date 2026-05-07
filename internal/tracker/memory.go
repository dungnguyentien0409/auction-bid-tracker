package tracker

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

type MemoryTracker struct {
	items sync.Map
	users sync.Map
}

type itemData struct {
	mu         sync.RWMutex
	bids       []domain.Bid
	winningBid *domain.Bid
}

type userData struct {
	mu    sync.RWMutex
	items map[string]struct{}
}

func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{}
}

func (t *MemoryTracker) getItem(itemID string) *itemData {
	val, _ := t.items.LoadOrStore(itemID, &itemData{
		bids: make([]domain.Bid, 0),
	})
	return val.(*itemData)
}

func (t *MemoryTracker) getUser(userID string) *userData {
	val, _ := t.users.LoadOrStore(userID, &userData{
		items: make(map[string]struct{}),
	})
	return val.(*userData)
}

func (t *MemoryTracker) RecordBid(ctx context.Context, itemID, userID string, amount float64) (*domain.Bid, error) {
	item := t.getItem(itemID)

	item.mu.Lock()
	if item.winningBid != nil && amount <= item.winningBid.Amount {
		item.mu.Unlock()
		return nil, domain.ErrBidTooLow
	}

	bid := domain.Bid{
		ID:        uuid.New().String(),
		ItemID:    itemID,
		UserID:    userID,
		Amount:    amount,
		Timestamp: time.Now().UTC(),
	}

	item.bids = append(item.bids, bid)
	item.winningBid = &bid
	item.mu.Unlock()

	user := t.getUser(userID)
	user.mu.Lock()
	user.items[itemID] = struct{}{}
	user.mu.Unlock()

	return &bid, nil
}

func (t *MemoryTracker) GetWinningBid(ctx context.Context, itemID string) (*domain.Bid, error) {
	val, ok := t.items.Load(itemID)
	if !ok {
		return nil, domain.ErrNotFound
	}
	item := val.(*itemData)

	item.mu.RLock()
	defer item.mu.RUnlock()

	if item.winningBid == nil {
		return nil, domain.ErrNotFound
	}

	bidCopy := *item.winningBid
	return &bidCopy, nil
}

func (t *MemoryTracker) GetAllBids(ctx context.Context, itemID string) ([]domain.Bid, error) {
	val, ok := t.items.Load(itemID)
	if !ok {
		return make([]domain.Bid, 0), nil
	}
	item := val.(*itemData)

	item.mu.RLock()
	defer item.mu.RUnlock()

	bidsCopy := make([]domain.Bid, len(item.bids))
	copy(bidsCopy, item.bids)

	return bidsCopy, nil
}

func (t *MemoryTracker) GetUserItems(ctx context.Context, userID string) ([]string, error) {
	val, ok := t.users.Load(userID)
	if !ok {
		return make([]string, 0), nil
	}
	user := val.(*userData)

	user.mu.RLock()
	defer user.mu.RUnlock()

	items := make([]string, 0, len(user.items))
	for itemID := range user.items {
		items = append(items, itemID)
	}

	return items, nil
}
