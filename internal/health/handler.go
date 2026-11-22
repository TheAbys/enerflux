package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type databasePinger interface {
	Ping(context.Context) error
}

type HealthHandler struct {
	database databasePinger
}

func NewHealthHandler(database databasePinger) *HealthHandler {
	return &HealthHandler{database: database}
}

func (h *HealthHandler) Get(c *gin.Context) {
	h.Readiness(c)
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "version": "0.1.0"})
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := h.database.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "database": "unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "version": "0.1.0", "database": "ok"})
}
