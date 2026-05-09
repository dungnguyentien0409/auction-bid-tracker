package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

func TestMemoryRepository_SaveBid(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	t.Run("first bid on item", func(t *testing.T) {
		bid := &domain.Bid{ItemID: "item1", UserID: "user1", Amount: 100.0}
		err := repo.SaveBid(ctx, bid)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("higher bid on same item", func(t *testing.T) {
		bid := &domain.Bid{ItemID: "item1", UserID: "user2", Amount: 150.0}
		err := repo.SaveBid(ctx, bid)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("lower bid on same item", func(t *testing.T) {
		bid := &domain.Bid{ItemID: "item1", UserID: "user3", Amount: 120.0}
		err := repo.SaveBid(ctx, bid)
		if err != domain.ErrBidTooLow {
			t.Errorf("expected ErrBidTooLow, got %v", err)
		}
	})
}

func TestMemoryRepository_GetWinningBid(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetWinningBid(ctx, "non-existent")
		if err != domain.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("found", func(t *testing.T) {
		_ = repo.SaveBid(ctx, &domain.Bid{ItemID: "item1", UserID: "user1", Amount: 100.0})
		bid, err := repo.GetWinningBid(ctx, "item1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if bid.UserID != "user1" {
			t.Errorf("expected user1, got %s", bid.UserID)
		}
	})
}

func TestMemoryRepository_GetAllBids(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	t.Run("empty item", func(t *testing.T) {
		bids, err := repo.GetAllBids(ctx, "item-empty")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(bids) != 0 {
			t.Errorf("expected 0 bids, got %d", len(bids))
		}
	})

	t.Run("multiple bids on item", func(t *testing.T) {
		_ = repo.SaveBid(ctx, &domain.Bid{ItemID: "item2", UserID: "user1", Amount: 100.0})
		_ = repo.SaveBid(ctx, &domain.Bid{ItemID: "item2", UserID: "user2", Amount: 200.0})
		_ = repo.SaveBid(ctx, &domain.Bid{ItemID: "item2", UserID: "user3", Amount: 300.0})

		bids, err := repo.GetAllBids(ctx, "item2")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(bids) != 3 {
			t.Errorf("expected 3 bids, got %d", len(bids))
		}
		if bids[0].Amount != 100.0 || bids[1].Amount != 200.0 || bids[2].Amount != 300.0 {
			t.Error("bids are not in the expected order")
		}
	})
}

func TestMemoryRepository_GetUserItems(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	t.Run("user with no bids", func(t *testing.T) {
		items, err := repo.GetUserItems(ctx, "newuser")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
	})

	t.Run("user with multiple items", func(t *testing.T) {
		_ = repo.SaveBid(ctx, &domain.Bid{ItemID: "item-a", UserID: "user-x", Amount: 10.0})
		_ = repo.SaveBid(ctx, &domain.Bid{ItemID: "item-b", UserID: "user-x", Amount: 20.0})
		_ = repo.SaveBid(ctx, &domain.Bid{ItemID: "item-c", UserID: "user-y", Amount: 30.0})

		items, err := repo.GetUserItems(ctx, "user-x")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(items) != 2 {
			t.Errorf("expected 2 items, got %d", len(items))
		}
	})
}

func TestMemoryRepository_Concurrency(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	
	const numGoroutines = 50
	const numBidsPerGoroutine = 50
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(userID int) {
			defer wg.Done()
			for j := 0; j < numBidsPerGoroutine; j++ {
				_ = repo.SaveBid(ctx, &domain.Bid{ItemID: "item1", UserID: "user", Amount: float64(j)})
				_ = repo.SaveBid(ctx, &domain.Bid{ItemID: "item2", UserID: "user", Amount: float64(j)})
			}
		}(i)
	}
	
	wg.Wait()
}

func TestMemoryRepository_EdgeCases(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	t.Run("item exists but has no bids", func(t *testing.T) {
		// Manually inject an empty item record
		repo.items.Store("empty-item", &itemRecord{})
		
		// Test GetWinningBid
		_, err := repo.GetWinningBid(ctx, "empty-item")
		if err != domain.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}

		// Test GetAllBids
		bids, err := repo.GetAllBids(ctx, "empty-item")
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if len(bids) != 0 {
			t.Errorf("expected 0 bids, got %d", len(bids))
		}
	})

	t.Run("user exists but has no items", func(t *testing.T) {
		// Manually inject an empty user record
		repo.users.Store("empty-user", &userRecord{items: make(map[string]struct{})})

		items, err := repo.GetUserItems(ctx, "empty-user")
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
	})
}

