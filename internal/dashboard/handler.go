package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

func (h *DashboardHandler) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", nil)
}
