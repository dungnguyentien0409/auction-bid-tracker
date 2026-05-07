package tracker

import (
	"context"
	"sync"
	"time"

	"github.com/dungnguyentien0409/auction-bid-tracker/internal/domain"
)

var idleTimeout = 30 * time.Minute

type MemoryTracker struct {
	items sync.Map
	users sync.Map
}

func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{}
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
