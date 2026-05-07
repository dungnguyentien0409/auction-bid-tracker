package tracker

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

var idleTimeout = 30 * time.Minute

type MemoryTracker struct {
	items sync.Map
	users sync.Map
}

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

type userData struct {
	mutex sync.RWMutex
	items map[string]struct{}
}

func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{}
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

func (tracker *MemoryTracker) getItem(itemID string) *itemData {
	value, ok := tracker.items.Load(itemID)
	if !ok {
		value, _ = tracker.items.LoadOrStore(itemID, newItemData(itemID, &tracker.items))
	}
	return value.(*itemData)
}

func (tracker *MemoryTracker) getUser(userID string) *userData {
	value, _ := tracker.users.LoadOrStore(userID, &userData{
		items: make(map[string]struct{}),
	})
	return value.(*userData)
}

func (tracker *MemoryTracker) RecordBid(context context.Context, itemID, userID string, amount float64) (*domain.Bid, error) {
	responseChannel := make(chan bidResponse, 1)
	request := bidRequest{
		userID:          userID,
		amount:          amount,
		responseChannel: responseChannel,
	}

	for {
		item := tracker.getItem(itemID)
		select {
		case item.bidChannel <- request:
		case <-context.Done():
			return nil, context.Err()
		case <-item.doneChannel:
			continue
		}
		break
	}

	var response bidResponse
	select {
	case response = <-responseChannel:
	case <-context.Done():
		return nil, context.Err()
	}

	if response.error != nil {
		return nil, response.error
	}

	user := tracker.getUser(userID)
	user.mutex.Lock()
	user.items[itemID] = struct{}{}
	user.mutex.Unlock()

	return response.bid, nil
}

func (tracker *MemoryTracker) GetWinningBid(context context.Context, itemID string) (*domain.Bid, error) {
	for {
		value, ok := tracker.items.Load(itemID)
		if !ok {
			return nil, domain.ErrNotFound
		}
		item := value.(*itemData)

		responseChannel := make(chan winningResponse, 1)
		select {
		case item.readWinningChannel <- readWinningRequest{responseChannel: responseChannel}:
		case <-context.Done():
			return nil, context.Err()
		case <-item.doneChannel:
			continue
		}

		select {
		case response := <-responseChannel:
			if response.error != nil {
				return nil, response.error
			}
			return response.bid, nil
		case <-context.Done():
			return nil, context.Err()
		}
	}
}

func (tracker *MemoryTracker) GetAllBids(context context.Context, itemID string) ([]domain.Bid, error) {
	for {
		value, ok := tracker.items.Load(itemID)
		if !ok {
			return make([]domain.Bid, 0), nil
		}
		item := value.(*itemData)

		responseChannel := make(chan []domain.Bid, 1)
		select {
		case item.readAllChannel <- readAllRequest{responseChannel: responseChannel}:
		case <-context.Done():
			return nil, context.Err()
		case <-item.doneChannel:
			continue
		}

		select {
		case bids := <-responseChannel:
			return bids, nil
		case <-context.Done():
			return nil, context.Err()
		}
	}
}

func (tracker *MemoryTracker) GetUserItems(context context.Context, userID string) ([]string, error) {
	value, ok := tracker.users.Load(userID)
	if !ok {
		return make([]string, 0), nil
	}
	user := value.(*userData)

	user.mutex.RLock()
	defer user.mutex.RUnlock()

	items := make([]string, 0, len(user.items))
	for itemID := range user.items {
		items = append(items, itemID)
	}

	return items, nil
}
