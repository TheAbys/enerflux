package api

import (
	"github.com/gin-gonic/gin"
)

func NewRouter(m *MeasurementHandler, h *HealthHandler) *gin.Engine {
	router := gin.Default()
	v1 := router.Group("/api/v1")
	{
		measurements := v1.Group("/measurements")
		{
			measurements.GET("", m.GetAll)
			measurements.GET("/:id", m.GetLatest)
		}

		health := v1.Group("/health")
		{
			health.GET("", h.Get)
		}
	}

	return router
}
