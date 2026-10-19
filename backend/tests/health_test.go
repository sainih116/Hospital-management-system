// Package tests holds cross-cutting integration tests that exercise the
// HTTP router end-to-end (as opposed to unit tests colocated with each
// package).
package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"hms-backend/internal/config"
	"hms-backend/internal/response"
	"hms-backend/internal/server"
)

func newTestRouter() http.Handler {
	cfg := &config.Config{
		Env:            "test",
		AllowedOrigins: []string{"http://localhost:5173"},
	}

	return server.NewRouter(server.Dependencies{
		Config:      cfg,
		Logger:      zap.NewNop(),
		MongoClient: nil,
		MongoDB:     nil,
		RedisClient: nil,
		Version:     "test",
	})
}

func TestHealthLiveEndpoint(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if !body.Success {
		t.Fatalf("expected success=true, got false")
	}

	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatalf("expected X-Request-Id header to be set")
	}
}

func TestHealthReadyEndpointWithoutDependencies(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// With no Mongo/Redis clients configured, there are no dependencies to
	// check, so readiness should report success.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNotFoundRoute(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}
