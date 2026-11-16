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
	c.HTML(http.StatusOK, "dashboard.html", gin.H{"ActiveNav": "overview"})
}

func (h *DashboardHandler) Settings(c *gin.Context) {
	c.HTML(http.StatusOK, "settings.html", gin.H{"ActiveNav": "settings"})
}

func (h *DashboardHandler) Pellets(c *gin.Context) {
	c.HTML(http.StatusOK, "pellets.html", gin.H{"ActiveNav": "heating"})
}

func (h *DashboardHandler) Water(c *gin.Context) {
	c.HTML(http.StatusOK, "water.html", gin.H{"ActiveNav": "water"})
}

func (h *DashboardHandler) Electricity(c *gin.Context) {
	c.HTML(http.StatusOK, "electricity.html", gin.H{"ActiveNav": "electricity"})
}
