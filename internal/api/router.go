package api

import (
	"github.com/gin-gonic/gin"
	"github.com/theabys/enerflux/internal/health"
	"github.com/theabys/enerflux/internal/measurements"
	"github.com/theabys/enerflux/internal/metrics"
)

func NewRouter(m *measurements.MeasurementHandler, metricsHandler *metrics.MetricHandler, h *health.HealthHandler) *gin.Engine {
	router := gin.Default()
	v1 := router.Group("/api/v1")
	{
		measurements := v1.Group("/measurements")
		{
			measurements.GET("", m.GetAll)
			measurements.GET("/:id", m.GetLatest)
		}

		metrics := v1.Group("/metrics")
		{
			metrics.GET("/", metricsHandler.GetMetrics)
		}

		health := v1.Group("/health")
		{
			health.GET("", h.Get)
		}
	}

	return router
}
