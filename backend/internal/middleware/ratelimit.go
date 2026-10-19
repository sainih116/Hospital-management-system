package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"hms-backend/internal/apperror"
	"hms-backend/internal/response"
)

// visitorStore tracks one token-bucket limiter per client IP, evicting
// entries that have been idle long enough that their bucket would be full
// again anyway. Sized for a single-process (modular monolith) deployment;
// a multi-instance deployment should back this with Redis instead.
type visitorStore struct {
	mu        sync.Mutex
	visitors  map[string]*visitor
	rps       rate.Limit
	burst     int
	idleAfter time.Duration
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newVisitorStore(requestsPerMinute, burst int) *visitorStore {
	store := &visitorStore{
		visitors:  make(map[string]*visitor),
		rps:       rate.Limit(float64(requestsPerMinute) / 60.0),
		burst:     burst,
		idleAfter: 10 * time.Minute,
	}
	go store.cleanupLoop()
	return store
}

func (s *visitorStore) allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.visitors[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(s.rps, s.burst)}
		s.visitors[key] = v
	}
	v.lastSeen = time.Now()
	return v.limiter.Allow()
}

func (s *visitorStore) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-s.idleAfter)
		s.mu.Lock()
		for key, v := range s.visitors {
			if v.lastSeen.Before(cutoff) {
				delete(s.visitors, key)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimit throttles requests per client IP to requestsPerMinute, allowing
// short bursts up to burst. Intended for authentication and other
// high-risk endpoints.
func RateLimit(requestsPerMinute, burst int) gin.HandlerFunc {
	store := newVisitorStore(requestsPerMinute, burst)

	return func(c *gin.Context) {
		if !store.allow(c.ClientIP()) {
			response.Error(c, apperror.ErrTooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}
