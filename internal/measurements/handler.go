package measurements

import (
	"fmt"
	"net/http"
	"strconv"
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

func (h *MeasurementHandler) List(c *gin.Context) {
	options, err := parseQueryOptions(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, err := h.Service.Find(c.Request.Context(), options)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, data)
}

func (h *MeasurementHandler) GetLatest(c *gin.Context) {
	options, err := parseQueryOptions(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, err := h.Service.FindLatest(c.Request.Context(), options.Filter)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, data)
}

func (h *MeasurementHandler) CreateMany(c *gin.Context) {
	var payload []MeasurementPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	measurements := ToMeasurements(payload)

	err := h.Service.InsertMany(c.Request.Context(), measurements)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

func parseQueryOptions(c *gin.Context) (QueryOptions, error) {
	options := QueryOptions{
		Filter: Filter{
			Types:   c.QueryArray("type"),
			Sources: c.QueryArray("source"),
		},
	}

	if value := c.Query("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			return QueryOptions{}, fmt.Errorf("invalid limit")
		}
		options.Limit = limit
	}

	if value := c.Query("offset"); value != "" {
		offset, err := strconv.Atoi(value)
		if err != nil {
			return QueryOptions{}, fmt.Errorf("invalid offset")
		}
		options.Offset = offset
	}

	if value := c.Query("from"); value != "" {
		from, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return QueryOptions{}, fmt.Errorf("invalid from timestamp")
		}
		options.Filter.From = &from
	}

	if value := c.Query("to"); value != "" {
		to, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return QueryOptions{}, fmt.Errorf("invalid to timestamp")
		}
		options.Filter.To = &to
	}

	return options, nil
}
