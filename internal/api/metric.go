package api

import (
	"github.com/gin-gonic/gin"
	filter "github.com/theabys/enerflux/internal/measurements"
	"github.com/theabys/enerflux/internal/service"
)

type MetricHandler struct {
	Service *service.MeasurementService
}

func NewMetricHandler(s *service.MeasurementService) *MetricHandler {
	return &MetricHandler{Service: s}
}

func (h *MetricHandler) GetMetrics(c *gin.Context) {

	key := c.Query("key")
	if len(key) == 0 {
		c.JSON(400, gin.H{"error": "missing or empty key query parameter"})
		return
	}

	filter := filter.MeasurementFilter{
		Key: key,
	}
	data, err := h.Service.GetMetrics(c.Request.Context(), filter)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, data)
}
