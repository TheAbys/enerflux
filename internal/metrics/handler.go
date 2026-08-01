package metrics

import (
	"fmt"
	"strconv"

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

	// TODO solve in parsing function
	key := c.Query("key")
	if len(key) == 0 {
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

	return options, nil
}
