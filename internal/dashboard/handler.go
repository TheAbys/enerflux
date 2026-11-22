package dashboard

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct{}

type pageView struct {
	ViteDevServer string
}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

func pageData() pageView {
	return pageView{
		ViteDevServer: strings.TrimRight(os.Getenv("VITE_DEV_SERVER"), "/"),
	}
}

func renderPage(c *gin.Context) {
	c.Header("Cache-Control", "no-cache")
	c.HTML(http.StatusOK, "app.html", pageData())
}

func (h *DashboardHandler) Index(c *gin.Context) {
	renderPage(c)
}

func (h *DashboardHandler) Settings(c *gin.Context) {
	renderPage(c)
}

func (h *DashboardHandler) Pellets(c *gin.Context) {
	renderPage(c)
}

func (h *DashboardHandler) Water(c *gin.Context) {
	renderPage(c)
}

func (h *DashboardHandler) Electricity(c *gin.Context) {
	renderPage(c)
}
