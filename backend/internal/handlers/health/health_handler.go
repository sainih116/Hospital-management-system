// Package health exposes liveness/readiness endpoints used by
// orchestrators (Docker, Kubernetes) and uptime monitors.
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"hms-backend/internal/response"
)

// Handler serves health/readiness endpoints.
type Handler struct {
	mongoClient *mongo.Client
	redisClient *redis.Client
	version     string
}

// NewHandler builds a health Handler. redisClient may be nil when the
// Redis cache layer is disabled.
func NewHandler(mongoClient *mongo.Client, redisClient *redis.Client, version string) *Handler {
	return &Handler{mongoClient: mongoClient, redisClient: redisClient, version: version}
}

// dependencyStatus reports the health of a single downstream dependency.
type dependencyStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// livePayload is the response body for the liveness probe.
type livePayload struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// readyPayload is the response body for the readiness probe.
type readyPayload struct {
	Status       string                      `json:"status"`
	Version      string                      `json:"version"`
	Dependencies map[string]dependencyStatus `json:"dependencies"`
}

// Live godoc
// @Summary      Liveness probe
// @Description  Reports whether the process is running. Does not check downstream dependencies.
// @Tags         health
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Router       /health [get]
func (h *Handler) Live(c *gin.Context) {
	response.OK(c, livePayload{Status: "ok", Version: h.version})
}

// Ready godoc
// @Summary      Readiness probe
// @Description  Reports whether the service and its dependencies (MongoDB, Redis) are ready to serve traffic.
// @Tags         health
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Failure      503  {object}  response.Envelope
// @Router       /health/ready [get]
func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	deps := map[string]dependencyStatus{}
	overallOK := true

	if h.mongoClient != nil {
		if err := h.mongoClient.Ping(ctx, readpref.Primary()); err != nil {
			deps["mongodb"] = dependencyStatus{Status: "down", Error: err.Error()}
			overallOK = false
		} else {
			deps["mongodb"] = dependencyStatus{Status: "up"}
		}
	}

	if h.redisClient != nil {
		if err := h.redisClient.Ping(ctx).Err(); err != nil {
			deps["redis"] = dependencyStatus{Status: "down", Error: err.Error()}
			overallOK = false
		} else {
			deps["redis"] = dependencyStatus{Status: "up"}
		}
	}

	payload := readyPayload{Version: h.version, Dependencies: deps}

	if !overallOK {
		payload.Status = "unavailable"
		c.JSON(http.StatusServiceUnavailable, response.Envelope{
			Success: false,
			Data:    payload,
		})
		return
	}

	payload.Status = "ok"
	response.OK(c, payload)
}
