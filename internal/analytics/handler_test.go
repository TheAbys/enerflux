package analytics

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/theabys/enerflux/internal/domainconfig"
	"github.com/theabys/enerflux/internal/measurements"
)

func TestParseWindowDefaultsAndRejectsInvalidBounds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{name: "defaults", wantStatus: http.StatusNoContent},
		{name: "invalid from", query: "?from=not-a-time", wantStatus: http.StatusBadRequest},
		{name: "invalid to", query: "?to=not-a-time", wantStatus: http.StatusBadRequest},
		{name: "reversed", query: "?from=2026-02-02T00:00:00Z&to=2026-02-01T00:00:00Z", wantStatus: http.StatusBadRequest},
		{name: "valid explicit range", query: "?from=2026-02-01T00:00:00Z&to=2026-02-02T00:00:00Z", wantStatus: http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/", func(c *gin.Context) {
				_, _, ok := parseWindow(c)
				if ok {
					c.Status(http.StatusNoContent)
				}
			})
			request := httptest.NewRequest(http.MethodGet, "/"+test.query, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d, want %d, body=%s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestDomainAnalyticsHandlersReturnStructuredResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	rows := []measurements.Measurement{
		measurement(now.Add(-10*time.Minute), gridPowerType, 1000, "W", "solarlog"),
		measurement(now, gridPowerType, 2000, "W", "solarlog"),
		measurement(now.Add(-2*time.Hour), waterType, 10, "m³", "manual"),
		measurement(now, waterType, 12, "m³", "manual"),
		measurement(now.Add(-24*time.Hour), pelletStockType, 1000, "kg", "manual"),
		measurement(now, pelletStockType, 900, "kg", "manual"),
	}
	store := &memoryMeasurementStore{rows: rows}
	settings := domainconfig.Settings{
		TimeZone:               "UTC",
		ElectricityPricePerKWh: ptr(.3),
		WaterPricePerM3:        ptr(2),
		WastewaterPricePerM3:   ptr(1),
	}
	service := NewService(store, NewTestDomainConfig(settings))
	handler := NewHandler(service, "")
	router := gin.New()
	router.GET("/overview", handler.Overview)
	router.GET("/water", handler.Water)
	router.GET("/electricity", handler.Electricity)
	router.GET("/heating", handler.Heating)
	query := "?from=" + now.Add(-48*time.Hour).Format(time.RFC3339) + "&to=" + now.Format(time.RFC3339)
	for _, path := range []string{"/overview", "/water", "/electricity", "/heating"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path+query, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d, body=%s", response.Code, response.Body.String())
			}
			var payload map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}
			if len(payload) == 0 {
				t.Fatal("expected a non-empty JSON response")
			}
		})
	}
}

func TestWaterHandlerReturnsBadRequestForInvalidWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := NewService(&memoryMeasurementStore{}, NewTestDomainConfig(domainconfig.Settings{TimeZone: "UTC"}))
	handler := NewHandler(service, "")
	router := gin.New()
	router.GET("/water", handler.Water)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/water?from=bad", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", response.Code)
	}
}

func TestAnalyticsHandlerReportsStorageFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &memoryMeasurementStore{err: errors.New("database unavailable")}
	service := NewService(store, NewTestDomainConfig(domainconfig.Settings{TimeZone: "UTC"}))
	router := gin.New()
	router.GET("/water", NewHandler(service, "").Water)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/water", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
}

func TestHomeAssistantEndpointRequiresBearerTokenAndExportsCalculatedValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC().Truncate(time.Minute)
	rows := []measurements.Measurement{
		measurement(now.Add(-10*time.Minute), gridPowerType, 1000, "W", "solarlog"),
		measurement(now, gridPowerType, 1000, "W", "solarlog"),
		measurement(now.Add(-31*24*time.Hour), waterType, 10, "m³", "manual"),
		measurement(now, waterType, 12, "m³", "manual"),
		measurement(now.Add(-24*time.Hour), pelletStockType, 1000, "kg", "manual"),
		measurement(now, pelletStockType, 900, "kg", "manual"),
		measurement(now.Add(-48*time.Hour), purchaseQuantityType, 1000, "kg", "manual-purchase:one"),
		measurement(now.Add(-48*time.Hour), purchaseCostType, 300, "EUR", "manual-purchase:one"),
	}
	settings := domainconfig.Settings{
		TimeZone:               "UTC",
		ElectricityPricePerKWh: ptr(.3),
		WaterPricePerM3:        ptr(2),
		WastewaterPricePerM3:   ptr(1),
	}
	service := NewService(&memoryMeasurementStore{rows: rows}, NewTestDomainConfig(settings))
	router := gin.New()
	router.GET("/sensors", NewHandler(service, "a-long-secret-token").HomeAssistant)

	for _, test := range []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "wrong", token: "wrong-token", wantStatus: http.StatusUnauthorized},
		{name: "valid", token: "a-long-secret-token", wantStatus: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/sensors", nil)
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d, want %d, body=%s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantStatus != http.StatusOK {
				return
			}
			var payload map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}
			for _, key := range []string{"generated_at", "time_zone", "electricity_consumed_today_kwh", "water_consumed_last_30_days_m3", "pellets_consumed_last_30_days_kg", "total_cost_last_30_days_eur"} {
				if _, ok := payload[key]; !ok {
					t.Errorf("missing Home Assistant value %q", key)
				}
			}
		})
	}

	disabled := httptest.NewRecorder()
	router = gin.New()
	router.GET("/sensors", NewHandler(service, "").HomeAssistant)
	router.ServeHTTP(disabled, httptest.NewRequest(http.MethodGet, "/sensors", nil))
	if disabled.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured export status=%d, want 503", disabled.Code)
	}
}
