package api

import (
	"github.com/gin-gonic/gin"
	"github.com/theabys/enerflux/internal/dashboard"
	"github.com/theabys/enerflux/internal/health"
	"github.com/theabys/enerflux/internal/measurements"
	"github.com/theabys/enerflux/internal/metrics"
)

func NewRouter(m *measurements.MeasurementHandler, metricsHandler *metrics.MetricHandler, h *health.HealthHandler, d *dashboard.DashboardHandler) *gin.Engine {
	router := gin.Default()
	router.LoadHTMLGlob("internal/web/templates/*")
	router.Static("/static", "./internal/web/static")

	v1 := router.Group("/api/v1")
	{
		measurements := v1.Group("/measurements")
		{
			measurements.GET("", m.List)
			measurements.GET("/latest", m.GetLatest)
			measurements.POST("", m.CreateMany)
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

	router.GET("/", d.Index)

	return router
}
