package domainconfig

import (
	"context"
	"errors"
	"math"
	"testing"
)

func TestSettingsValidateAcceptsConfiguredAndOptionalFields(t *testing.T) {
	settings := Settings{
		TimeZone:               "Europe/Berlin",
		ElectricityPricePerKWh: floatPointer(.32),
		WaterPricePerM3:        floatPointer(2.1),
		WastewaterPricePerM3:   floatPointer(3.4),
		PelletTankCapacityKg:   floatPointer(6000),
	}
	if err := settings.Validate(); err != nil {
		t.Fatalf("valid settings rejected: %v", err)
	}
	optional := Settings{}
	if err := optional.Validate(); err != nil {
		t.Fatalf("optional blank rates should be valid: %v", err)
	}
	if optional.TimeZone != "Europe/Berlin" {
		t.Fatalf("empty time zone should default to Europe/Berlin, got %q", optional.TimeZone)
	}
}

func TestSettingsValidateRejectsInvalidTimeZoneRatesAndCapacity(t *testing.T) {
	cases := []struct {
		name     string
		settings Settings
	}{
		{name: "unknown time zone", settings: Settings{TimeZone: "Mars/Olympus"}},
		{name: "negative electricity rate", settings: Settings{TimeZone: "UTC", ElectricityPricePerKWh: floatPointer(-.01)}},
		{name: "negative water rate", settings: Settings{TimeZone: "UTC", WaterPricePerM3: floatPointer(-1)}},
		{name: "negative wastewater rate", settings: Settings{TimeZone: "UTC", WastewaterPricePerM3: floatPointer(-1)}},
		{name: "zero tank capacity", settings: Settings{TimeZone: "UTC", PelletTankCapacityKg: floatPointer(0)}},
		{name: "nan rate", settings: Settings{TimeZone: "UTC", ElectricityPricePerKWh: floatPointer(math.NaN())}},
		{name: "infinite capacity", settings: Settings{TimeZone: "UTC", PelletTankCapacityKg: floatPointer(math.Inf(1))}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := test.settings.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

type memoryStore struct {
	settings Settings
	getErr   error
	saveErr  error
	saves    int
}

func (s *memoryStore) Get(context.Context) (Settings, error) {
	if s.getErr != nil {
		return Settings{}, s.getErr
	}
	return s.settings, nil
}

func (s *memoryStore) Save(_ context.Context, settings Settings) error {
	s.saves++
	if s.saveErr != nil {
		return s.saveErr
	}
	s.settings = settings
	return nil
}

func TestServiceGetDefaultsMissingTimezoneAndPropagatesErrors(t *testing.T) {
	service := NewService(&memoryStore{settings: Settings{ElectricityPricePerKWh: floatPointer(.25)}})
	settings, err := service.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if settings.TimeZone != "Europe/Berlin" {
		t.Fatalf("expected default timezone, got %q", settings.TimeZone)
	}
	storeErr := errors.New("database unavailable")
	service = NewService(&memoryStore{getErr: storeErr})
	if _, err := service.Get(context.Background()); !errors.Is(err, storeErr) {
		t.Fatalf("expected underlying error %v, got %v", storeErr, err)
	}
}

func TestServiceSaveValidatesBeforePersistence(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	if err := service.Save(context.Background(), Settings{TimeZone: "UTC", ElectricityPricePerKWh: floatPointer(.3)}); err != nil {
		t.Fatalf("valid save failed: %v", err)
	}
	if store.saves != 1 || store.settings.TimeZone != "UTC" {
		t.Fatalf("expected valid settings persisted, got %+v", store)
	}
	if err := service.Save(context.Background(), Settings{TimeZone: "Invalid/Zone"}); err == nil {
		t.Fatal("expected invalid settings to fail")
	}
	if store.saves != 1 {
		t.Fatalf("invalid settings reached persistence; saves=%d", store.saves)
	}
	store.saveErr = errors.New("write failed")
	if err := service.Save(context.Background(), Settings{TimeZone: "UTC"}); !errors.Is(err, store.saveErr) {
		t.Fatalf("expected persistence error, got %v", err)
	}
}

func floatPointer(value float64) *float64 { return &value }
