package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/theabys/enerflux/internal/measurements"
)

type MetricHandler struct {
	Service *MetricService
}

func NewMetricHandler(s *MetricService) *MetricHandler {
	return &MetricHandler{Service: s}
}

func (h *MetricHandler) GetMetrics(c *gin.Context) {

	key := c.Query("key")
	if len(key) == 0 {
		c.JSON(400, gin.H{"error": "missing or empty key query parameter"})
		return
	}

	filter := measurements.MeasurementFilter{
		Key: key,
	}
	data, err := h.Service.GetMetrics(c.Request.Context(), filter)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, data)
}
