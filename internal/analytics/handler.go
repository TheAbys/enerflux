package analytics

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service   *Service
	ReadToken string
}

func NewHandler(service *Service, readToken string) *Handler {
	return &Handler{Service: service, ReadToken: readToken}
}

func (h *Handler) Overview(c *gin.Context) {
	data, err := h.Service.Overview(c.Request.Context(), time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not calculate domain overview"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, data)
}

func (h *Handler) Water(c *gin.Context) {
	from, to, ok := parseWindow(c)
	if !ok {
		return
	}
	data, err := h.Service.GetWater(c.Request.Context(), from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not calculate water usage"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, data)
}

func (h *Handler) Electricity(c *gin.Context) {
	from, to, ok := parseWindow(c)
	if !ok {
		return
	}
	data, err := h.Service.GetElectricity(c.Request.Context(), from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not calculate electricity usage"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, data)
}

func (h *Handler) Heating(c *gin.Context) {
	from, to, ok := parseWindow(c)
	if !ok {
		return
	}
	data, err := h.Service.GetHeating(c.Request.Context(), from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not calculate pellet heating usage"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, data)
}

type homeAssistantResponse struct {
	GeneratedAt                        time.Time `json:"generated_at"`
	TimeZone                           string    `json:"time_zone"`
	ElectricityConsumedTodayKWh        *float64  `json:"electricity_consumed_today_kwh"`
	ElectricityCostTodayEUR            *float64  `json:"electricity_cost_today_eur"`
	ElectricityCostLast30DaysEUR       *float64  `json:"electricity_cost_last_30_days_eur"`
	WaterConsumedLast30DaysM3          *float64  `json:"water_consumed_last_30_days_m3"`
	WaterCostLast30DaysEUR             *float64  `json:"water_cost_last_30_days_eur"`
	WaterMeterTotalM3                  *float64  `json:"water_meter_total_m3"`
	PelletStockKg                      *float64  `json:"pellet_stock_kg"`
	PelletsConsumedLast30DaysKg        *float64  `json:"pellets_consumed_last_30_days_kg"`
	PelletAveragePerCoveredDay30DaysKg *float64  `json:"pellet_average_per_covered_day_30_days_kg"`
	PelletForecastNextHourKg           *float64  `json:"pellet_forecast_next_hour_kg"`
	PelletCostLast30DaysEUR            *float64  `json:"pellet_cost_last_30_days_eur"`
	TotalCostLast30DaysEUR             *float64  `json:"total_cost_last_30_days_eur"`
}

func (h *Handler) HomeAssistant(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if h.ReadToken == "" {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Home Assistant export is not configured"})
		return
	}
	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	if len(authorization) < 7 || !strings.EqualFold(authorization[:7], "Bearer ") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return
	}
	provided := strings.TrimSpace(authorization[7:])
	if subtle.ConstantTimeCompare([]byte(h.ReadToken), []byte(provided)) != 1 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid bearer token"})
		return
	}

	data, err := h.Service.Overview(c.Request.Context(), time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not calculate Home Assistant sensor values"})
		return
	}
	response := homeAssistantResponse{
		GeneratedAt:                        data.GeneratedAt,
		TimeZone:                           data.TimeZone,
		ElectricityConsumedTodayKWh:        data.Electricity.Consumed.Value,
		ElectricityCostTodayEUR:            data.Electricity.Consumed.CostEUR,
		ElectricityCostLast30DaysEUR:       data.TotalCost.ElectricityEUR,
		WaterConsumedLast30DaysM3:          data.Water.Consumed.Value,
		WaterCostLast30DaysEUR:             data.Water.Consumed.CostEUR,
		PelletsConsumedLast30DaysKg:        data.Heating.ConsumedLast30DaysKg,
		PelletAveragePerCoveredDay30DaysKg: data.Heating.AverageDaily30DaysKg,
		PelletForecastNextHourKg:           data.Heating.ForecastNextHourKg,
		PelletCostLast30DaysEUR:            data.Heating.ConsumedLast30DaysCostEUR,
		TotalCostLast30DaysEUR:             data.TotalCost.TotalEUR,
	}
	if data.Water.CurrentReading != nil {
		if value, ok := convertWater(*data.Water.CurrentReading); ok {
			response.WaterMeterTotalM3 = &value
		}
	}
	if data.Heating.CurrentStock != nil && data.Heating.CurrentStock.Unit == "kg" {
		response.PelletStockKg = &data.Heating.CurrentStock.Value
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, response)
}

func parseWindow(c *gin.Context) (time.Time, time.Time, bool) {
	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now
	if value := c.Query("from"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from timestamp; expected RFC3339"})
			return time.Time{}, time.Time{}, false
		}
		from = parsed
	}
	if value := c.Query("to"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to timestamp; expected RFC3339"})
			return time.Time{}, time.Time{}, false
		}
		to = parsed
	}
	if from.After(to) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from timestamp must not be after to timestamp"})
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}
