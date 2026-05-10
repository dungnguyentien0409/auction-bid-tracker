package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

func newTestRedisRepo(t *testing.T) (*RedisRepository, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	t.Cleanup(mr.Close)

	return NewRedisRepository(mr.Addr(), "", 0), mr
}

func TestNewRedisRepository(t *testing.T) {
	t.Parallel()

	repo := NewRedisRepository("localhost:6379", "", 0)

	if repo == nil {
		t.Fatal("expected repository")
	}

	if repo.client == nil {
		t.Fatal("expected redis client")
	}
}

func TestRedisRepository_SaveBid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("save first bid", func(t *testing.T) {
		repo, _ := newTestRedisRepo(t)

		bid := &domain.Bid{
			ID:     "bid_1",
			ItemID: "item_1",
			UserID: "user_1",
			Amount: 100,
		}

		err := repo.SaveBid(ctx, bid)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("save higher bid", func(t *testing.T) {
		repo, _ := newTestRedisRepo(t)

		err := repo.SaveBid(ctx, &domain.Bid{
			ID:     "bid_1",
			ItemID: "item_1",
			UserID: "user_1",
			Amount: 100,
		})
		if err != nil {
			t.Fatalf("save bid error: %v", err)
		}

		err = repo.SaveBid(ctx, &domain.Bid{
			ID:     "bid_2",
			ItemID: "item_1",
			UserID: "user_2",
			Amount: 200,
		})

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("save lower bid returns ErrBidTooLow", func(t *testing.T) {
		repo, _ := newTestRedisRepo(t)

		err := repo.SaveBid(ctx, &domain.Bid{
			ID:     "bid_1",
			ItemID: "item_1",
			UserID: "user_1",
			Amount: 200,
		})
		if err != nil {
			t.Fatalf("save bid error: %v", err)
		}

		err = repo.SaveBid(ctx, &domain.Bid{
			ID:     "bid_2",
			ItemID: "item_1",
			UserID: "user_2",
			Amount: 100,
		})

		if !errors.Is(err, domain.ErrBidTooLow) {
			t.Fatalf("expected ErrBidTooLow, got %v", err)
		}
	})

	t.Run("redis eval error", func(t *testing.T) {
		repo, mr := newTestRedisRepo(t)

		mr.Close()

		err := repo.SaveBid(ctx, &domain.Bid{
			ID:     "bid_1",
			ItemID: "item_1",
			UserID: "user_1",
			Amount: 100,
		})

		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestRedisRepository_GetWinningBid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo, _ := newTestRedisRepo(t)

		err := repo.SaveBid(ctx, &domain.Bid{
			ID:     "bid_1",
			ItemID: "item_1",
			UserID: "user_1",
			Amount: 100,
		})

		if err != nil {
			t.Fatalf("save bid error: %v", err)
		}

		bid, err := repo.GetWinningBid(ctx, "item_1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if bid.ID != "bid_1" {
			t.Fatalf("expected bid_1, got %s", bid.ID)
		}

		if bid.UserID != "user_1" {
			t.Fatalf("expected user_1, got %s", bid.UserID)
		}

		if bid.Amount != 100 {
			t.Fatalf("expected 100, got %v", bid.Amount)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo, _ := newTestRedisRepo(t)

		_, err := repo.GetWinningBid(ctx, "missing")

		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("redis error", func(t *testing.T) {
		repo, mr := newTestRedisRepo(t)

		mr.Close()

		_, err := repo.GetWinningBid(ctx, "item_1")

		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid amount parse", func(t *testing.T) {
		repo, mr := newTestRedisRepo(t)

		mr.HSet("item:item_1", "amount", "invalid")
		mr.HSet("item:item_1", "user_id", "user_1")
		mr.HSet("item:item_1", "bid_id", "bid_1")

		_, err := repo.GetWinningBid(ctx, "item_1")

		if err == nil {
			t.Fatal("expected parse error")
		}
	})
}

func TestRedisRepository_GetAllBids(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo, _ := newTestRedisRepo(t)

		err := repo.SaveBid(ctx, &domain.Bid{
			ID:     "bid_1",
			ItemID: "item_1",
			UserID: "user_1",
			Amount: 100,
		})
		if err != nil {
			t.Fatalf("save bid error: %v", err)
		}

		err = repo.SaveBid(ctx, &domain.Bid{
			ID:     "bid_2",
			ItemID: "item_1",
			UserID: "user_2",
			Amount: 200,
		})
		if err != nil {
			t.Fatalf("save bid error: %v", err)
		}

		bids, err := repo.GetAllBids(ctx, "item_1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(bids) != 2 {
			t.Fatalf("expected 2 bids, got %d", len(bids))
		}
	})

	t.Run("empty", func(t *testing.T) {
		repo, _ := newTestRedisRepo(t)

		bids, err := repo.GetAllBids(ctx, "missing")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(bids) != 0 {
			t.Fatalf("expected empty bids")
		}
	})

	t.Run("skip malformed entry", func(t *testing.T) {
		repo, mr := newTestRedisRepo(t)

		for _, v := range []string{"invalid", "100:user_1"} {
			if _, err := mr.Push("bids:item_1", v); err != nil {
				t.Fatalf("push error: %v", err)
			}
		}

		bids, err := repo.GetAllBids(ctx, "item_1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(bids) != 1 {
			t.Fatalf("expected 1 valid bid, got %d", len(bids))
		}
	})

	t.Run("redis error", func(t *testing.T) {
		repo, mr := newTestRedisRepo(t)

		mr.Close()

		_, err := repo.GetAllBids(ctx, "item_1")

		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestRedisRepository_GetUserItems(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo, _ := newTestRedisRepo(t)

		err := repo.SaveBid(ctx, &domain.Bid{
			ID:     "bid_1",
			ItemID: "item_1",
			UserID: "user_1",
			Amount: 100,
		})
		if err != nil {
			t.Fatalf("save bid error: %v", err)
		}

		items, err := repo.GetUserItems(ctx, "user_1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}

		if items[0] != "item_1" {
			t.Fatalf("expected item_1, got %s", items[0])
		}
	})

	t.Run("empty", func(t *testing.T) {
		repo, _ := newTestRedisRepo(t)

		items, err := repo.GetUserItems(ctx, "missing")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(items) != 0 {
			t.Fatalf("expected empty items")
		}
	})

	t.Run("redis error", func(t *testing.T) {
		repo, mr := newTestRedisRepo(t)

		mr.Close()

		_, err := repo.GetUserItems(ctx, "user_1")

		if err == nil {
			t.Fatal("expected error")
		}
	})
}
