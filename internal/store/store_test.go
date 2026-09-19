package store

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestStoreConcurrentSetGetDelete(t *testing.T) {
	s := NewStore(0)

	const workers = 32
	const opsPerWorker = 500

	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		worker := worker
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				key := fmt.Sprintf("key-%d-%d", worker, i%64)
				value := fmt.Sprintf("value-%d-%d", worker, i)

				s.Set(key, value)

				if got, ok := s.Get(key); !ok || got == "" {
					t.Errorf("Get(%q) = %q, %v; want non-empty value, true", key, got, ok)
				}

				if !s.Exists(key) {
					t.Errorf("Exists(%q) = false; want true", key)
				}

				if i%5 == 0 {
					s.Delete(key)
				}
			}
		}()
	}
	wg.Wait()
}

func TestStoreConcurrentLRUCapacity(t *testing.T) {
	const capacity = 50
	s := NewStore(capacity)

	const workers = 16
	const opsPerWorker = 300

	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		worker := worker
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				key := fmt.Sprintf("key-%d", (worker*opsPerWorker+i)%200)
				s.Set(key, "value")
				s.Get(key)
				s.Exists(key)
			}
		}()
	}
	wg.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.data) > capacity {
		t.Fatalf("len(data) = %d; want <= %d", len(s.data), capacity)
	}
	if s.lru.Len() != len(s.data) {
		t.Fatalf("lru.Len() = %d; want %d", s.lru.Len(), len(s.data))
	}
	for key, e := range s.data {
		if e.lruNode == nil {
			t.Fatalf("entry %q has nil lruNode", key)
		}
		if got := e.lruNode.Value.(string); got != key {
			t.Fatalf("entry %q lruNode has key %q", key, got)
		}
	}
}

func TestStoreConcurrentTTLAndCleanup(t *testing.T) {
	s := NewStore(0)
	stop := s.StartCleanup(time.Millisecond)
	defer stop()

	const workers = 16
	const opsPerWorker = 200

	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		worker := worker
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				key := fmt.Sprintf("ttl-%d-%d", worker, i%32)
				if !s.SetWithTTL(key, "value", 2*time.Millisecond) {
					t.Errorf("SetWithTTL(%q) returned false", key)
				}
				s.Get(key)
				s.Exists(key)
				if i%7 == 0 {
					s.Delete(key)
				}
			}
		}()
	}
	wg.Wait()

	time.Sleep(20 * time.Millisecond)

	s.mu.Lock()
	s.cleanupExpiredLocked(time.Now())
	defer s.mu.Unlock()

	for key, e := range s.data {
		if e.hasExpiry && !e.expiresAt.After(time.Now()) {
			t.Fatalf("expired key %q remains after cleanup", key)
		}
	}
}
