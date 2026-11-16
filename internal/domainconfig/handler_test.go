package domainconfig

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

var errTestPersistence = errors.New("write failed")

func TestHandlerGetReturnsPersistedSettingsAndNoStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &memoryStore{settings: Settings{TimeZone: "UTC", ElectricityPricePerKWh: floatPointer(.31)}}
	router := gin.New()
	router.GET("/domains", NewHandler(NewService(store)).Get)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/domains", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control=%q, want no-store", response.Header().Get("Cache-Control"))
	}
	var settings Settings
	if err := json.Unmarshal(response.Body.Bytes(), &settings); err != nil {
		t.Fatal(err)
	}
	if settings.TimeZone != "UTC" || settings.ElectricityPricePerKWh == nil || *settings.ElectricityPricePerKWh != .31 {
		t.Fatalf("unexpected settings: %+v", settings)
	}
}

func TestHandlerSaveValidatesPayloadAndReportsPersistenceErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &memoryStore{}
	router := gin.New()
	router.PUT("/domains", NewHandler(NewService(store)).Save)

	badPayload := httptest.NewRecorder()
	router.ServeHTTP(badPayload, httptest.NewRequest(http.MethodPut, "/domains", strings.NewReader("{")))
	if badPayload.Code != http.StatusBadRequest {
		t.Fatalf("malformed payload status=%d, want 400", badPayload.Code)
	}
	invalidSettings := httptest.NewRecorder()
	router.ServeHTTP(invalidSettings, httptest.NewRequest(http.MethodPut, "/domains", strings.NewReader(`{"timeZone":"Bad/Zone"}`)))
	if invalidSettings.Code != http.StatusBadRequest {
		t.Fatalf("invalid settings status=%d, want 400", invalidSettings.Code)
	}
	if store.saves != 0 {
		t.Fatalf("invalid payload reached repository; saves=%d", store.saves)
	}

	validSettings := httptest.NewRecorder()
	router.ServeHTTP(validSettings, httptest.NewRequest(http.MethodPut, "/domains", strings.NewReader(`{"timeZone":"UTC","waterPricePerM3":2,"wastewaterPricePerM3":0}`)))
	if validSettings.Code != http.StatusOK || store.saves != 1 {
		t.Fatalf("valid save status=%d saves=%d body=%s", validSettings.Code, store.saves, validSettings.Body.String())
	}

	store.saveErr = errTestPersistence
	failedSave := httptest.NewRecorder()
	router.ServeHTTP(failedSave, httptest.NewRequest(http.MethodPut, "/domains", strings.NewReader(`{"timeZone":"UTC"}`)))
	if failedSave.Code != http.StatusInternalServerError {
		t.Fatalf("persistence error status=%d, want 500", failedSave.Code)
	}
}
