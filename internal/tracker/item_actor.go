package tracker

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

type bidRequest struct {
	userID          string
	amount          float64
	responseChannel chan bidResponse
}

type bidResponse struct {
	bid   *domain.Bid
	error error
}

type readWinningRequest struct {
	responseChannel chan winningResponse
}

type winningResponse struct {
	bid   *domain.Bid
	error error
}

type readAllRequest struct {
	responseChannel chan []domain.Bid
}

type itemData struct {
	identifier         string
	bids               []domain.Bid
	winningBid         *domain.Bid
	bidChannel         chan bidRequest
	readWinningChannel chan readWinningRequest
	readAllChannel     chan readAllRequest
	doneChannel        chan struct{}
}

func newItemData(identifier string, trackerItems *sync.Map) *itemData {
	item := &itemData{
		identifier:         identifier,
		bids:               make([]domain.Bid, 0),
		bidChannel:         make(chan bidRequest),
		readWinningChannel: make(chan readWinningRequest),
		readAllChannel:     make(chan readAllRequest),
		doneChannel:        make(chan struct{}),
	}
	go item.run(trackerItems)
	return item
}

func (item *itemData) run(trackerItems *sync.Map) {
	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			trackerItems.Delete(item.identifier)
			close(item.doneChannel)
			return

		case request := <-item.bidChannel:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(idleTimeout)

			if item.winningBid != nil && request.amount <= item.winningBid.Amount {
				request.responseChannel <- bidResponse{error: domain.ErrBidTooLow}
				continue
			}

			bid := domain.Bid{
				ID:        uuid.New().String(),
				ItemID:    item.identifier,
				UserID:    request.userID,
				Amount:    request.amount,
				Timestamp: time.Now().UTC(),
			}

			item.bids = append(item.bids, bid)
			item.winningBid = &bid
			request.responseChannel <- bidResponse{bid: &bid}

		case request := <-item.readWinningChannel:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(idleTimeout)

			if item.winningBid == nil {
				request.responseChannel <- winningResponse{error: domain.ErrNotFound}
			} else {
				bidCopy := *item.winningBid
				request.responseChannel <- winningResponse{bid: &bidCopy}
			}

		case request := <-item.readAllChannel:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(idleTimeout)

			bidsCopy := make([]domain.Bid, len(item.bids))
			copy(bidsCopy, item.bids)
			request.responseChannel <- bidsCopy
		}
	}
}
