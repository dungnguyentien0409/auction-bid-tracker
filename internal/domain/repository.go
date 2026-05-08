package domain

import "context"

type Repository interface {
	// SaveBid saves a new bid. It returns ErrBidTooLow if the bid is not higher
	// than the current winning bid for the item. It should also record the item
	// for the user.
	SaveBid(ctx context.Context, bid *Bid) error

	// GetWinningBid returns the highest bid for a given item.
	// Returns ErrNotFound if no bids exist for the item.
	GetWinningBid(ctx context.Context, itemID string) (*Bid, error)

	// GetAllBids returns all bids for a given item, ordered chronologically or by amount.
	GetAllBids(ctx context.Context, itemID string) ([]Bid, error)

	// GetUserItems returns a list of item IDs that the user has bid on.
	GetUserItems(ctx context.Context, userID string) ([]string, error)
}
