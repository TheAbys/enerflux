package measurements

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type MeasurementHandler struct {
	Service *MeasurementService
}

func NewMeasurementHandler(s *MeasurementService) *MeasurementHandler {
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

	// TODO: parse query param and build filter
	filter := MeasurementFilter{}

	data, err := h.Service.GetAll(c.Request.Context(), filter)
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

func (h *MeasurementHandler) CreateMultiple(c *gin.Context) {
	var payload []MeasurementPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	measurements := ToMeasurements(payload)

	// TODO do one big insert instead of multiple small ones?
	for _, measurement := range measurements {
		err := h.Service.Create(measurement)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"status": "ok"})
}

func ToMeasurements(payloads []MeasurementPayload) []Measurement {
	items := make([]Measurement, len(payloads))

	for i := range payloads {
		items[i] = Measurement(payloads[i])
	}

	return items
}
