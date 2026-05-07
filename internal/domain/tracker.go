package domain

import (
	"context"
	"errors"
)

var (
	ErrBidTooLow = errors.New("bid amount must be higher than current winning bid")
	ErrNotFound  = errors.New("not found")
)

type Tracker interface {
	RecordBid(ctx context.Context, itemID, userID string, amount float64) (*Bid, error)
	GetWinningBid(ctx context.Context, itemID string) (*Bid, error)
	GetAllBids(ctx context.Context, itemID string) ([]Bid, error)
	GetUserItems(ctx context.Context, userID string) ([]string, error)
}
