package tracker

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

func TestMemoryTracker_RecordBid(t *testing.T) {
	tracker := NewMemoryTracker()
	ctx := context.Background()

	bid1, err := tracker.RecordBid(ctx, "item1", "user1", 100.0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bid1.Amount != 100.0 {
		t.Errorf("expected bid amount 100.0, got %v", bid1.Amount)
	}

	_, err = tracker.RecordBid(ctx, "item1", "user2", 50.0)
	if err != domain.ErrBidTooLow {
		t.Errorf("expected ErrBidTooLow, got %v", err)
	}

	bid3, err := tracker.RecordBid(ctx, "item1", "user3", 150.0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bid3.Amount != 150.0 {
		t.Errorf("expected bid amount 150.0, got %v", bid3.Amount)
	}

	winning, err := tracker.GetWinningBid(ctx, "item1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if winning.ID != bid3.ID {
		t.Errorf("expected winning bid ID %s, got %s", bid3.ID, winning.ID)
	}

	allBids, err := tracker.GetAllBids(ctx, "item1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(allBids) != 2 {
		t.Errorf("expected 2 bids, got %d", len(allBids))
	}
}

func TestMemoryTracker_Concurrency(t *testing.T) {
	tracker := NewMemoryTracker()
	ctx := context.Background()
	var wg sync.WaitGroup

	numBids := 1000
	item1 := "item_concurrent"

	for i := 1; i <= numBids; i++ {
		wg.Add(1)
		go func(amount float64) {
			defer wg.Done()
			user := fmt.Sprintf("user_%d", int(amount))
			_, _ = tracker.RecordBid(ctx, item1, user, amount)
		}(float64(i))
	}

	wg.Wait()

	winning, err := tracker.GetWinningBid(ctx, item1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if winning.Amount != float64(numBids) {
		t.Errorf("expected winning bid amount %d, got %v", numBids, winning.Amount)
	}
}

func BenchmarkMemoryTracker_RecordBid(b *testing.B) {
	tracker := NewMemoryTracker()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		amount := 1.0
		for pb.Next() {
			amount += 1.0
			itemID := fmt.Sprintf("item_%d", int(amount)%100)
			userID := fmt.Sprintf("user_%d", int(amount)%1000)
			_, _ = tracker.RecordBid(ctx, itemID, userID, amount)
		}
	})
}

func TestMemoryTracker_EdgeCases(t *testing.T) {
	tracker := NewMemoryTracker()
	ctx := context.Background()

	bids, err := tracker.GetAllBids(ctx, "nonexist")
	if err != nil {
		t.Fatalf("expected no error")
	}
	if len(bids) != 0 {
		t.Errorf("expected empty bids")
	}

	items, err := tracker.GetUserItems(ctx, "nonexist")
	if err != nil {
		t.Fatalf("expected no error")
	}
	if len(items) != 0 {
		t.Errorf("expected empty items")
	}

	_, _ = tracker.RecordBid(ctx, "item1", "user1", 100)
	items, err = tracker.GetUserItems(ctx, "user1")
	if err != nil || len(items) != 1 || items[0] != "item1" {
		t.Errorf("expected 1 item 'item1'")
	}

	tracker.items.Store("empty_item", &itemData{})
	_, err = tracker.GetWinningBid(ctx, "empty_item")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound for item with no bids")
	}
}
