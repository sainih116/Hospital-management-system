package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimitAllowsWithinBurstAndBlocksAfter(t *testing.T) {
	handler := RateLimit(60, 3) // 1 req/sec sustained, burst of 3

	request := func() (aborted bool, status int) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		c.Request.RemoteAddr = "203.0.113.5:12345"

		handler(c)
		return c.IsAborted(), rec.Code
	}

	for i := 0; i < 3; i++ {
		if aborted, status := request(); aborted {
			t.Fatalf("request %d: expected it to be allowed (within burst), got status %d", i+1, status)
		}
	}

	// The burst is now exhausted; the very next request should be limited.
	aborted, status := request()
	if !aborted {
		t.Fatalf("expected the 4th request to be rate limited")
	}
	if status != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", status)
	}
}

func TestRateLimitTracksClientsIndependently(t *testing.T) {
	handler := RateLimit(60, 1)

	requestFrom := func(ip string) bool {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		c.Request.RemoteAddr = ip + ":12345"
		handler(c)
		return !c.IsAborted()
	}

	if !requestFrom("198.51.100.1") {
		t.Fatalf("expected the first client's first request to be allowed")
	}
	if !requestFrom("198.51.100.2") {
		t.Fatalf("expected a different client's first request to be allowed even though the first client used its burst")
	}
}
