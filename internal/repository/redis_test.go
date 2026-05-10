package repository

import (
	"context"
	"testing"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
	"github.com/redis/go-redis/v9"
)

func TestRedisRepository(t *testing.T) {
	// Skip if Redis is not available locally
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available on localhost:6379, skipping integration test")
	}
	defer func() { _ = client.Close() }()

	repo := NewRedisRepository("localhost:6379", "", 0)

	t.Run("SaveBid and GetWinningBid", func(t *testing.T) {
		// Clean up before test
		_ = client.Del(ctx, "item:item_1", "bids:item_1", "user:user_1:items").Err()

		bid := &domain.Bid{ItemID: "item_1", UserID: "user_1", Amount: 100.0}
		err := repo.SaveBid(ctx, bid)
		if err != nil {
			t.Fatalf("failed to save bid: %v", err)
		}

		winning, err := repo.GetWinningBid(ctx, "item_1")
		if err != nil {
			t.Fatalf("failed to get winning bid: %v", err)
		}
		if winning.Amount != 100.0 || winning.UserID != "user_1" {
			t.Errorf("expected 100.0 by user_1, got %v by %s", winning.Amount, winning.UserID)
		}

		// Higher bid should succeed
		higherBid := &domain.Bid{ItemID: "item_1", UserID: "user_2", Amount: 150.0}
		_ = repo.SaveBid(ctx, higherBid)
		winning, _ = repo.GetWinningBid(ctx, "item_1")
		if winning.Amount != 150.0 {
			t.Errorf("expected 150.0, got %v", winning.Amount)
		}

		// Lower bid should not update
		lowerBid := &domain.Bid{ItemID: "item_1", UserID: "user_3", Amount: 120.0}
		_ = repo.SaveBid(ctx, lowerBid)
		winning, _ = repo.GetWinningBid(ctx, "item_1")
		if winning.Amount != 150.0 {
			t.Errorf("expected 150.0 to remain, got %v", winning.Amount)
		}
	})

	t.Run("GetAllBids and GetUserItems", func(t *testing.T) {
		items, _ := repo.GetUserItems(ctx, "user_1")
		if len(items) != 1 || items[0] != "item_1" {
			t.Errorf("expected item_1 in user items, got %v", items)
		}

		allBids, _ := repo.GetAllBids(ctx, "item_1")
		if len(allBids) < 2 {
			t.Errorf("expected at least 2 bids in history, got %d", len(allBids))
		}
	})
}
