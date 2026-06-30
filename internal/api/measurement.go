package api

import (
	"net/http"
	"time"

	"github.com/theabys/enerflux/internal/model"
	"github.com/theabys/enerflux/internal/service"

	"github.com/gin-gonic/gin"
)

type MeasurementHandler struct {
	Service *service.MeasurementService
}

func NewMeasurementHandler(s *service.MeasurementService) *MeasurementHandler {
	return &MeasurementHandler{Service: s}
}

type CreateMeasurementRequest struct {
	TS     time.Time `json:"ts"`
	Type   string    `json:"type"`
	Value  float64   `json:"value"`
	Unit   string    `json:"unit"`
	Source string    `json:"source"`
}

func (h *MeasurementHandler) GetAll(c *gin.Context) {

	data, err := h.Service.GetAll()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, data)
}

func (h *MeasurementHandler) GetLatest(c *gin.Context) {

	data, err := h.Service.GetLatest()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, data)
}

func (h *MeasurementHandler) Create(c *gin.Context) {
	var req CreateMeasurementRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.Service.Create(model.Measurement{
		TS:     req.TS,
		Type:   req.Type,
		Value:  req.Value,
		Unit:   req.Unit,
		Source: req.Source,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
