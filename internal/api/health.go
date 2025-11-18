package api

import (
	"net/http"
	"time"

	"haystack-lite/internal/storage"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	store     *storage.Store
	startTime time.Time
}

func NewHealthHandler(store *storage.Store) *HealthHandler {
	return &HealthHandler{
		store:     store,
		startTime: time.Now(),
	}
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	status := h.store.Status()

	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"storage": status,
		"uptime":  time.Since(h.startTime).Seconds(),
	})
}

func (h *HealthHandler) Health(c *gin.Context) {
	status := h.store.Status()

	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(h.startTime).Seconds(),
		"storage": gin.H{
			"total_files":  status["total_files"],
			"active_files": status["active_files"],
			"total_size":   status["total_size"],
			"volumes":      status["volume_count"],
		},
	}

	c.JSON(http.StatusOK, health)
}
