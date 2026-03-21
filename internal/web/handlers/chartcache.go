package handlers

import (
	"fmt"
	"sync"
	"time"
)

// chartCacheTTL controls how long public chart data remains cached.
// At most one DB hit per (username, monitorID, span) per minute regardless
// of how many concurrent visitors request the same public status page.
const chartCacheTTL = 60 * time.Second

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// chartCache is a lightweight in-memory TTL store for pre-serialised chart
// JSON responses.  It is used only for the public (unauthenticated) endpoint;
// authenticated owners always receive fresh realtime data.
type chartCache struct {
	m    sync.Map
	once sync.Once
}

func newChartCache() *chartCache {
	c := &chartCache{}
	c.once.Do(func() { go c.evictLoop() })
	return c
}

// chartCacheKey returns the canonical lookup key.
// The \x00 separator cannot appear in any of the three fields.
func chartCacheKey(username, monitorID, span string) string {
	return fmt.Sprintf("%s\x00%s\x00%s", username, monitorID, span)
}

// get returns the cached bytes and true when a valid non-expired entry exists.
func (c *chartCache) get(key string) ([]byte, bool) {
	v, ok := c.m.Load(key)
	if !ok {
		return nil, false
	}
	e := v.(cacheEntry)
	if time.Now().After(e.expiresAt) {
		c.m.Delete(key)
		return nil, false
	}
	return e.data, true
}

// set stores data under key with the given TTL.
func (c *chartCache) set(key string, data []byte, ttl time.Duration) {
	c.m.Store(key, cacheEntry{data: data, expiresAt: time.Now().Add(ttl)})
}

// evictLoop runs in a background goroutine and purges expired entries every
// minute to prevent unbounded memory growth.
func (c *chartCache) evictLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.m.Range(func(k, v any) bool {
			if now.After(v.(cacheEntry).expiresAt) {
				c.m.Delete(k)
			}
			return true
		})
	}
}
