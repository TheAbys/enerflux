package metrics

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type MetricHandler struct {
	Service *MetricService
}

func NewMetricHandler(s *MetricService) *MetricHandler {
	return &MetricHandler{Service: s}
}

func (h *MetricHandler) GetMetrics(c *gin.Context) {
	options, err := parseQueryOptions(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if len(options.Filter.Key) == 0 {
		c.JSON(400, gin.H{"error": "missing or empty key query parameter"})
		return
	}

	data, err := h.Service.GetMetrics(c.Request.Context(), options)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, data)
}

func parseQueryOptions(c *gin.Context) (QueryOptions, error) {
	options := QueryOptions{
		Filter: Filter{
			Key: c.Query("key"),
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
			return QueryOptions{}, fmt.Errorf("invalid from timestamp: expected RFC3339")
		}
		options.Filter.From = &from
	}

	if value := c.Query("to"); value != "" {
		to, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return QueryOptions{}, fmt.Errorf("invalid to timestamp: expected RFC3339")
		}
		options.Filter.To = &to
	}

	if options.Filter.From != nil && options.Filter.To != nil && options.Filter.From.After(*options.Filter.To) {
		return QueryOptions{}, fmt.Errorf("from timestamp must not be after to timestamp")
	}

	return options, nil
}
