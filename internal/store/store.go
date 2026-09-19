package store

import (
	"container/list"
	"sync"
	"time"
)

type expiryHeap []expiryItem

type expiryItem struct {
	key       string
	expiresAt time.Time
}

type entry struct {
	key       string
	value     string
	expiresAt time.Time
	hasExpiry bool
	lruNode   *list.Element
}

type Store struct {
	data       map[string]*entry
	mu         sync.Mutex
	expiryHeap expiryHeap
	lru        *list.List
	capacity   int
}
