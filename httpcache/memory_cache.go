package httpcache

import (
	"sync"
	"time"
)

// MemoryCache is an implemtation of Cache that stores responses in an in-memory map.
type MemoryCache struct {
	mu     sync.RWMutex
	items  map[string][]byte
	ts     map[string]time.Time
	maxTTL time.Duration
}

// NewMemoryCache returns a new Cache that will store items in an in-memory map
func NewMemoryCache(maxTTL time.Duration) *MemoryCache {
	if maxTTL <= time.Duration(0) {
		panic("maxTTL must be >0")
	}
	c := &MemoryCache{
		items:  make(map[string][]byte),
		ts:     make(map[string]time.Time),
		maxTTL: maxTTL,
	}
	return c
}

// Get returns the []byte representation of the response and true if present, false if not
func (c *MemoryCache) Get(key string) (resp []byte, ok bool) {
	c.mu.RLock()
	resp, ok = c.items[key]
	ts := c.ts[key]
	c.mu.RUnlock()

	if ok && time.Since(ts) > c.maxTTL {
		c.Delete(key)
		return nil, false
	}

	return resp, ok
}

// Set saves response resp to the cache with key
func (c *MemoryCache) Set(key string, resp []byte) {
	c.mu.Lock()
	c.ts[key] = time.Now()
	c.items[key] = resp
	c.mu.Unlock()
}

// Delete removes key from the cache
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.ts, key)
	delete(c.items, key)
	c.mu.Unlock()
}
