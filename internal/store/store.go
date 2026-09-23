package store

import (
	"container/heap"
	"container/list"
	"math"
	"strconv"
	"sync"
	"time"
)

type expiryHeap []expiryItem

func (h expiryHeap) Len() int {
	return len(h)
}

func (h expiryHeap) Less(i, j int) bool {
	return h[i].expiresAt.Before(h[j].expiresAt)
}

func (h expiryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *expiryHeap) Push(x any) {
	item := x.(expiryItem)
	*h = append(*h, item)
}

func (h *expiryHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

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
	now := time.Now()
	s.cleanupExpiredLocked(now)
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
	now := time.Now()
	e, ok := s.data[key]
	if !ok {
		return "", false
	}
	if !e.hasExpiry || e.expiresAt.After(now) {
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
	now := time.Now()
	e, ok := s.data[key]
	if !ok {
		return false
	}
	if !e.hasExpiry || e.expiresAt.After(now) {
		return true
	}
	s.deleteLocked(key)
	return false
}

func (s *Store) SetWithTTL(key, value string, ttl time.Duration) bool {
	if ttl <= 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.cleanupExpiredLocked(now)
	expiresAt := now.Add(ttl)
	e, ok := s.data[key]
	if !ok {
		node := s.lru.PushFront(key)
		newEntry := entry{
			key:       key,
			value:     value,
			hasExpiry: true,
			lruNode:   node,
			expiresAt: expiresAt,
		}
		s.data[key] = &newEntry
		heap.Push(&s.expiryHeap, expiryItem{key: key, expiresAt: expiresAt})
		if s.capacity > 0 && len(s.data) > s.capacity {
			s.evictLRULocked()
		}
		return true
	}
	e.value = value
	e.hasExpiry = true
	e.expiresAt = expiresAt
	s.lru.MoveToFront(e.lruNode)
	heap.Push(&s.expiryHeap, expiryItem{key: key, expiresAt: expiresAt})
	return true
}

func (s *Store) cleanupExpiredLocked(now time.Time) {
	for len(s.expiryHeap) > 0 {
		item := s.expiryHeap[0]
		if item.expiresAt.After(now) {
			return
		}
		heap.Pop(&s.expiryHeap)
		current, ok := s.data[item.key]
		if !ok {
			continue
		}
		if !current.hasExpiry {
			continue
		}
		if !current.expiresAt.Equal(item.expiresAt) {
			continue
		}
		s.deleteLocked(item.key)
	}
}

func (s *Store) StartCleanup(interval time.Duration) func() {
	if interval <= 0 {
		return func() {}
	}
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				s.mu.Lock()
				s.cleanupExpiredLocked(time.Now())
				s.mu.Unlock()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(done) }) }
}

func (s *Store) Increment(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.cleanupExpiredLocked(now)
	e, ok := s.data[key]
	if !ok || (e.hasExpiry && !e.expiresAt.After(now)) {
		if ok {
			s.deleteLocked(key)
		}
		node := s.lru.PushFront(key)
		e := entry{
			key:       key,
			value:     "1",
			hasExpiry: false,
			lruNode:   node,
		}
		s.data[key] = &e
		if s.capacity > 0 && len(s.data) > s.capacity {
			s.evictLRULocked()
		}
		return "1", true
	}
	strValue := e.value
	intValue, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		return "", false
	}
	if intValue == math.MaxInt64 {
		return "", false
	}
	intValue++
	updatedStrValue := strconv.FormatInt(intValue, 10)
	e.value = updatedStrValue
	s.lru.MoveToFront(e.lruNode)
	return updatedStrValue, true
}

func (s *Store) SetNX(key, value string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.cleanupExpiredLocked(now)
	ent, ok := s.data[key]
	if ok && (ent.hasExpiry && !ent.expiresAt.After(now)) {
		s.deleteLocked(key)
	} else if ok {
		return false
	}
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
	return true
}
