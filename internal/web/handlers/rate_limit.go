package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimitEntry tracks the request count within a fixed time window for one key.
type rateLimitEntry struct {
	mu    sync.Mutex
	count int
	start time.Time
}

// RateLimiter returns a Gin middleware that limits each client IP to maxRequests
// within a fixed time window of windowDur. Requests that exceed the limit
// receive HTTP 429 with no body. The limiter is keyed by IP address obtained
// via c.ClientIP() (respects X-Forwarded-For when Gin's trusted proxies are set).
func RateLimiter(maxRequests int, windowDur time.Duration) gin.HandlerFunc {
	var store sync.Map // map[string]*rateLimitEntry
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		v, _ := store.LoadOrStore(ip, &rateLimitEntry{start: now})
		entry := v.(*rateLimitEntry)

		entry.mu.Lock()
		if now.Sub(entry.start) >= windowDur {
			entry.count = 0
			entry.start = now
		}
		entry.count++
		count := entry.count
		entry.mu.Unlock()

		if count > maxRequests {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}
