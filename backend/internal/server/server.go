package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// New builds an *http.Server wrapping the given Gin engine with sane
// timeouts for a production deployment behind a reverse proxy.
func New(addr string, handler *gin.Engine) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// Shutdown gracefully stops the server, allowing in-flight requests up to
// the given timeout to complete.
func Shutdown(ctx context.Context, srv *http.Server, timeout time.Duration) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
