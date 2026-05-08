package service

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/repository"
)

func setupService() *BidService {
	repo := repository.NewMemoryRepository()
	return NewBidService(repo)
}

func TestBidService_RecordBid(t *testing.T) {
	service := setupService()
	backgroundContext := context.Background()

	bid1, err := service.RecordBid(backgroundContext, "item1", "user1", 100.0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bid1.Amount != 100.0 {
		t.Errorf("expected bid amount 100.0, got %v", bid1.Amount)
	}

	_, err = service.RecordBid(backgroundContext, "item1", "user2", 50.0)
	if err != domain.ErrBidTooLow {
		t.Errorf("expected ErrBidTooLow, got %v", err)
	}

	bid3, err := service.RecordBid(backgroundContext, "item1", "user3", 150.0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bid3.Amount != 150.0 {
		t.Errorf("expected bid amount 150.0, got %v", bid3.Amount)
	}

	winningBid, err := service.GetWinningBid(backgroundContext, "item1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if winningBid.ID != bid3.ID {
		t.Errorf("expected winning bid ID %s, got %s", bid3.ID, winningBid.ID)
	}

	allBids, err := service.GetAllBids(backgroundContext, "item1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(allBids) != 2 {
		t.Errorf("expected 2 bids, got %d", len(allBids))
	}
}

func TestBidService_Concurrency(t *testing.T) {
	service := setupService()
	backgroundContext := context.Background()
	var waitGroup sync.WaitGroup

	numberOfBids := 1000
	item1 := "item_concurrent"

	for index := 1; index <= numberOfBids; index++ {
		waitGroup.Add(1)
		go func(amount float64) {
			defer waitGroup.Done()
			user := fmt.Sprintf("user_%d", int(amount))
			_, _ = service.RecordBid(backgroundContext, item1, user, amount)
		}(float64(index))
	}

	waitGroup.Wait()

	winningBid, err := service.GetWinningBid(backgroundContext, item1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if winningBid.Amount != float64(numberOfBids) {
		t.Errorf("expected winning bid amount %d, got %v", numberOfBids, winningBid.Amount)
	}
}

func BenchmarkBidService_RecordBid(b *testing.B) {
	service := setupService()
	backgroundContext := context.Background()

	b.ResetTimer()
	b.RunParallel(func(parallelBenchmark *testing.PB) {
		amount := 1.0
		for parallelBenchmark.Next() {
			amount += 1.0
			itemID := fmt.Sprintf("item_%d", int(amount)%100)
			userID := fmt.Sprintf("user_%d", int(amount)%1000)
			_, _ = service.RecordBid(backgroundContext, itemID, userID, amount)
		}
	})
}

func TestBidService_EdgeCases(t *testing.T) {
	service := setupService()
	backgroundContext := context.Background()

	bids, err := service.GetAllBids(backgroundContext, "nonexist")
	if err != nil {
		t.Fatalf("expected no error")
	}
	if len(bids) != 0 {
		t.Errorf("expected empty bids")
	}

	items, err := service.GetUserItems(backgroundContext, "nonexist")
	if err != nil {
		t.Fatalf("expected no error")
	}
	if len(items) != 0 {
		t.Errorf("expected empty items")
	}

	_, _ = service.RecordBid(backgroundContext, "item1", "user1", 100)
	items, err = service.GetUserItems(backgroundContext, "user1")
	if err != nil || len(items) != 1 || items[0] != "item1" {
		t.Errorf("expected 1 item 'item1'")
	}

	_, err = service.GetWinningBid(backgroundContext, "empty_item")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound for item with no bids")
	}
}
