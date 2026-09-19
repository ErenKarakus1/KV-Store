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

func NewStore(capacity int) *Store {
	return &Store{
		data:     make(map[string]*entry),
		lru:      list.New(),
		capacity: capacity,
	}
}

func (s *Store) deleteLocked(key string) bool {
	e, ok := s.data[key]
	if !ok {
		return false
	}
	s.lru.Remove(e.lruNode)
	delete(s.data, key)
	return true
}

func (s *Store) evictLRULocked() bool {
	oldest := s.lru.Back()
	if oldest == nil {
		return false
	}
	key := oldest.Value.(string)
	return s.deleteLocked(key)
}

func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if currentEntry, ok := s.data[key]; !ok {
		node := s.lru.PushFront(key)
		e := entry{
			key:       key,
			value:     value,
			hasExpiry: false,
			lruNode:   node,
		}
		s.data[key] = &e
		if s.capacity > 0 && len(s.data) > s.capacity {
			s.evictLRULocked()
		}
	} else {
		currentEntry.hasExpiry = false
		currentEntry.value = value
		currentEntry.expiresAt = time.Time{}
		s.lru.MoveToFront(currentEntry.lruNode)
	}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok {
		return "", false
	}
	if !e.hasExpiry || e.expiresAt.After(time.Now()) {
		s.lru.MoveToFront(e.lruNode)
		return e.value, true
	}
	s.deleteLocked(key)
	return "", false
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteLocked(key)
}

func (s *Store) Exists(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok {
		return false
	}
	if !e.hasExpiry || e.expiresAt.After(time.Now()) {
		return true
	}
	s.deleteLocked(key)
	return false
}
