package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/theabys/enerflux/internal/analytics"
	"github.com/theabys/enerflux/internal/dashboard"
	"github.com/theabys/enerflux/internal/domainconfig"
	"github.com/theabys/enerflux/internal/health"
	"github.com/theabys/enerflux/internal/measurements"
	"github.com/theabys/enerflux/internal/metrics"
	"github.com/theabys/enerflux/internal/sourceconfig"
)

func NewRouter(m *measurements.MeasurementHandler, metricsHandler *metrics.MetricHandler, sourceConfigHandler *sourceconfig.Handler, domainConfigHandler *domainconfig.Handler, analyticsHandler *analytics.Handler, h *health.HealthHandler, d *dashboard.DashboardHandler) *gin.Engine {
	router := gin.Default()
	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "no-cache")
		}
		c.Next()
	})
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

		configuration := v1.Group("/configuration")
		{
			configuration.GET("/sources", sourceConfigHandler.Get)
			configuration.PUT("/sources", sourceConfigHandler.Save)
			configuration.GET("/domains", domainConfigHandler.Get)
			configuration.PUT("/domains", domainConfigHandler.Save)
		}

		analyticsRoutes := v1.Group("/analytics")
		{
			analyticsRoutes.GET("/overview", analyticsHandler.Overview)
			analyticsRoutes.GET("/water", analyticsHandler.Water)
			analyticsRoutes.GET("/electricity", analyticsHandler.Electricity)
			analyticsRoutes.GET("/heating", analyticsHandler.Heating)
		}

		v1.GET("/integrations/home-assistant/sensors", analyticsHandler.HomeAssistant)

		health := v1.Group("/health")
		{
			health.GET("", h.Get)
			health.GET("/live", h.Liveness)
			health.GET("/ready", h.Readiness)
		}
	}

	router.GET("/", d.Index)
	router.GET("/water", d.Water)
	router.GET("/electricity", d.Electricity)
	router.GET("/heating", d.Pellets)
	router.GET("/pellets", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/heating") })
	router.GET("/settings", d.Settings)

	return router
}
