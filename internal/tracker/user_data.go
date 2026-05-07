package tracker

import "sync"

type userData struct {
	mutex sync.RWMutex
	items map[string]struct{}
}
