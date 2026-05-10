package repository

import (
	"context"
	"fmt"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(addr string, password string, db int) *RedisRepository {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisRepository{client: rdb}
}

// Lua script to atomically compare and set the winning bid, and record history/user items
var saveBidLua = `
local item_id = KEYS[1]
local item_key = "item:" .. item_id
local user_id = ARGV[1]
local amount = tonumber(ARGV[2])
local bid_id = ARGV[3]
local user_items_key = "user:" .. user_id .. ":items"
local bids_history_key = "bids:" .. item_id

local current_amount = tonumber(redis.call("HGET", item_key, "amount") or "0")

if amount > current_amount then
    -- Update winning bid
    redis.call("HSET", item_key, "user_id", user_id, "amount", amount, "bid_id", bid_id)
    -- Add to user's bid items
    redis.call("SADD", user_items_key, item_id)
    -- Add to bid history (simplified: just amount)
    redis.call("RPUSH", bids_history_key, amount .. ":" .. user_id)
    return 1
end

return 0
`

func (r *RedisRepository) SaveBid(ctx context.Context, bid *domain.Bid) error {
	res, err := r.client.Eval(ctx, saveBidLua, []string{bid.ItemID}, bid.UserID, bid.Amount, bid.ID).Result()
	if err != nil {
		return fmt.Errorf("failed to execute lua script: %w", err)
	}

	if res.(int64) == 0 {
		return domain.ErrBidTooLow
	}

	return nil
}

func (r *RedisRepository) GetWinningBid(ctx context.Context, itemID string) (*domain.Bid, error) {
	itemKey := "item:" + itemID
	data, err := r.client.HGetAll(ctx, itemKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get winning bid: %w", err)
	}

	if len(data) == 0 {
		return nil, domain.ErrNotFound
	}

	var amount float64
	if _, err := fmt.Sscanf(data["amount"], "%f", &amount); err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}

	return &domain.Bid{
		ID:     data["bid_id"],
		ItemID: itemID,
		UserID: data["user_id"],
		Amount: amount,
	}, nil
}

func (r *RedisRepository) GetAllBids(ctx context.Context, itemID string) ([]domain.Bid, error) {
	bidsKey := "bids:" + itemID
	results, err := r.client.LRange(ctx, bidsKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all bids: %w", err)
	}

	bids := make([]domain.Bid, 0, len(results))
	for _, res := range results {
		var amount float64
		var userID string
		if _, err := fmt.Sscanf(res, "%f:%s", &amount, &userID); err != nil {
			continue // Skip malformed entries
		}
		bids = append(bids, domain.Bid{
			ItemID: itemID,
			UserID: userID,
			Amount: amount,
		})
	}

	return bids, nil
}

func (r *RedisRepository) GetUserItems(ctx context.Context, userID string) ([]string, error) {
	userItemsKey := "user:" + userID + ":items"
	items, err := r.client.SMembers(ctx, userItemsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user items: %w", err)
	}
	return items, nil
}
