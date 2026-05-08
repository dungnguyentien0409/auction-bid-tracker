package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

// BidService implements domain.Tracker using a domain.Repository
type BidService struct {
	repo domain.Repository
}

func NewBidService(repo domain.Repository) *BidService {
	return &BidService{repo: repo}
}

func (s *BidService) RecordBid(ctx context.Context, itemID, userID string, amount float64) (*domain.Bid, error) {
	bid := &domain.Bid{
		ID:        uuid.New().String(),
		ItemID:    itemID,
		UserID:    userID,
		Amount:    amount,
		Timestamp: time.Now().UTC(),
	}

	err := s.repo.SaveBid(ctx, bid)
	if err != nil {
		return nil, err
	}

	return bid, nil
}

func (s *BidService) GetWinningBid(ctx context.Context, itemID string) (*domain.Bid, error) {
	return s.repo.GetWinningBid(ctx, itemID)
}

func (s *BidService) GetAllBids(ctx context.Context, itemID string) ([]domain.Bid, error) {
	return s.repo.GetAllBids(ctx, itemID)
}

func (s *BidService) GetUserItems(ctx context.Context, userID string) ([]string, error) {
	return s.repo.GetUserItems(ctx, userID)
}
