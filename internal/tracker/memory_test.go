package tracker

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

func TestMemoryTracker_RecordBid(t *testing.T) {
	tracker := NewMemoryTracker()
	backgroundContext := context.Background()

	bid1, err := tracker.RecordBid(backgroundContext, "item1", "user1", 100.0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bid1.Amount != 100.0 {
		t.Errorf("expected bid amount 100.0, got %v", bid1.Amount)
	}

	_, err = tracker.RecordBid(backgroundContext, "item1", "user2", 50.0)
	if err != domain.ErrBidTooLow {
		t.Errorf("expected ErrBidTooLow, got %v", err)
	}

	bid3, err := tracker.RecordBid(backgroundContext, "item1", "user3", 150.0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bid3.Amount != 150.0 {
		t.Errorf("expected bid amount 150.0, got %v", bid3.Amount)
	}

	winningBid, err := tracker.GetWinningBid(backgroundContext, "item1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if winningBid.ID != bid3.ID {
		t.Errorf("expected winning bid ID %s, got %s", bid3.ID, winningBid.ID)
	}

	allBids, err := tracker.GetAllBids(backgroundContext, "item1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(allBids) != 2 {
		t.Errorf("expected 2 bids, got %d", len(allBids))
	}
}

func TestMemoryTracker_Concurrency(t *testing.T) {
	tracker := NewMemoryTracker()
	backgroundContext := context.Background()
	var waitGroup sync.WaitGroup

	numberOfBids := 1000
	item1 := "item_concurrent"

	for index := 1; index <= numberOfBids; index++ {
		waitGroup.Add(1)
		go func(amount float64) {
			defer waitGroup.Done()
			user := fmt.Sprintf("user_%d", int(amount))
			_, _ = tracker.RecordBid(backgroundContext, item1, user, amount)
		}(float64(index))
	}

	waitGroup.Wait()

	winningBid, err := tracker.GetWinningBid(backgroundContext, item1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if winningBid.Amount != float64(numberOfBids) {
		t.Errorf("expected winning bid amount %d, got %v", numberOfBids, winningBid.Amount)
	}
}

func BenchmarkMemoryTracker_RecordBid(b *testing.B) {
	tracker := NewMemoryTracker()
	backgroundContext := context.Background()

	b.ResetTimer()
	b.RunParallel(func(parallelBenchmark *testing.PB) {
		amount := 1.0
		for parallelBenchmark.Next() {
			amount += 1.0
			itemID := fmt.Sprintf("item_%d", int(amount)%100)
			userID := fmt.Sprintf("user_%d", int(amount)%1000)
			_, _ = tracker.RecordBid(backgroundContext, itemID, userID, amount)
		}
	})
}

func TestMemoryTracker_EdgeCases(t *testing.T) {
	tracker := NewMemoryTracker()
	backgroundContext := context.Background()

	bids, err := tracker.GetAllBids(backgroundContext, "nonexist")
	if err != nil {
		t.Fatalf("expected no error")
	}
	if len(bids) != 0 {
		t.Errorf("expected empty bids")
	}

	items, err := tracker.GetUserItems(backgroundContext, "nonexist")
	if err != nil {
		t.Fatalf("expected no error")
	}
	if len(items) != 0 {
		t.Errorf("expected empty items")
	}

	_, _ = tracker.RecordBid(backgroundContext, "item1", "user1", 100)
	items, err = tracker.GetUserItems(backgroundContext, "user1")
	if err != nil || len(items) != 1 || items[0] != "item1" {
		t.Errorf("expected 1 item 'item1'")
	}

	tracker.items.Store("empty_item", newItemData("empty_item", &tracker.items))
	_, err = tracker.GetWinningBid(backgroundContext, "empty_item")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound for item with no bids")
	}
}

func TestMemoryTracker_IdleTimeout(t *testing.T) {
	// Temporarily override the timeout for this test
	originalTimeout := idleTimeout
	idleTimeout = 10 * time.Millisecond
	defer func() { idleTimeout = originalTimeout }()

	tracker := NewMemoryTracker()
	backgroundContext := context.Background()

	// Create an item
	_, _ = tracker.RecordBid(backgroundContext, "timeout_item", "user1", 100)

	// Wait longer than the timeout
	time.Sleep(50 * time.Millisecond)

	// Item should be gone from the map
	_, ok := tracker.items.Load("timeout_item")
	if ok {
		t.Errorf("expected item to be deleted from sync.Map")
	}

	// Another bid should recreate it transparently
	_, err := tracker.RecordBid(backgroundContext, "timeout_item", "user2", 200)
	if err != nil {
		t.Errorf("expected no error recreating item, got %v", err)
	}

	winningBid, err := tracker.GetWinningBid(backgroundContext, "timeout_item")
	if err != nil || winningBid.Amount != 200 {
		t.Errorf("expected winning bid to be 200, got %v", winningBid)
	}
}
