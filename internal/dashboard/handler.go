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

func (h *DashboardHandler) Settings(c *gin.Context) {
	c.HTML(http.StatusOK, "settings.html", nil)
}

func (h *DashboardHandler) Pellets(c *gin.Context) {
	c.HTML(http.StatusOK, "pellets.html", nil)
}
