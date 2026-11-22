package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/theabys/enerflux/internal/domainconfig"
	"github.com/theabys/enerflux/internal/measurements"
)

const (
	waterType            = "water.meter.reading"
	gridPowerType        = "grid.consumption.power"
	pelletStockType      = "heating.pellet.stock"
	purchaseQuantityType = "heating.pellet.purchase.quantity"
	purchaseCostType     = "heating.pellet.purchase.cost"
	etaSource            = "eta"
	manualSource         = "manual"
	maxPowerGap          = 15 * time.Minute
	maxForecastAge       = 20 * time.Minute
	forecastWindow       = 6 * time.Hour
	minimumForecastAge   = 2 * time.Hour
)

// MeasurementStore provides ordered measurement queries to the analytics service.
type MeasurementStore interface {
	Find(context.Context, measurements.QueryOptions) ([]measurements.Measurement, error)
}

// Service calculates resource usage, costs, and forecasts from stored measurements.
type Service struct {
	Measurements  MeasurementStore
	Configuration *domainconfig.Service
}

// NewService creates an analytics service backed by a measurement store and domain configuration.
func NewService(store MeasurementStore, configuration *domainconfig.Service) *Service {
	return &Service{Measurements: store, Configuration: configuration}
}

// Overview returns current electricity, water, and heating summaries plus their combined cost.
func (s *Service) Overview(ctx context.Context, now time.Time) (OverviewResponse, error) {
	settings, location, err := s.settings(ctx)
	if err != nil {
		return OverviewResponse{}, err
	}
	now = now.In(location)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	costFrom := now.AddDate(0, 0, -30)
	electricityFrom := dayStart

	electricity, err := s.Electricity(ctx, electricityFrom, now, settings, location)
	if err != nil {
		return OverviewResponse{}, err
	}
	electricity30, err := s.Electricity(ctx, costFrom, now, settings, location)
	if err != nil {
		return OverviewResponse{}, err
	}
	water, err := s.Water(ctx, costFrom, now, settings, location)
	if err != nil {
		return OverviewResponse{}, err
	}
	heating, err := s.Heating(ctx, costFrom, now, settings, location)
	if err != nil {
		return OverviewResponse{}, err
	}
	costs := costOverview(costFrom, now, electricity30.Consumed.CostEUR, water.Consumed.CostEUR, heating.ConsumedLast30DaysCostEUR)
	return OverviewResponse{GeneratedAt: now, TimeZone: location.String(), Electricity: electricity, Water: water, Heating: heating, TotalCost: costs}, nil
}

// costOverview sums domain costs only when every domain has a known estimate.
func costOverview(from, to time.Time, electricity, water, heating *float64) CostOverview {
	result := CostOverview{
		Window:         Window{From: from, To: to},
		ElectricityEUR: electricity,
		WaterEUR:       water,
		HeatingEUR:     heating,
		Complete:       true,
		MissingDomains: make([]string, 0),
	}
	values := []struct {
		name  string
		value *float64
	}{{"electricity", electricity}, {"water", water}, {"pellet heating", heating}}
	total := 0.0
	for _, item := range values {
		if item.value == nil {
			result.Complete = false
			result.MissingDomains = append(result.MissingDomains, item.name)
			continue
		}
		total += *item.value
	}
	if result.Complete {
		result.TotalEUR = valuePointer(total)
	}
	return result
}

// GetWater loads the configured time zone and calculates water usage for the requested window.
func (s *Service) GetWater(ctx context.Context, from, to time.Time) (WaterResponse, error) {
	settings, location, err := s.settings(ctx)
	if err != nil {
		return WaterResponse{}, err
	}
	return s.Water(ctx, from.In(location), to.In(location), settings, location)
}

// GetElectricity loads the configured time zone and calculates electricity usage for the requested window.
func (s *Service) GetElectricity(ctx context.Context, from, to time.Time) (ElectricityResponse, error) {
	settings, location, err := s.settings(ctx)
	if err != nil {
		return ElectricityResponse{}, err
	}
	return s.Electricity(ctx, from.In(location), to.In(location), settings, location)
}

// GetHeating loads the configured time zone and calculates pellet usage for the requested window.
func (s *Service) GetHeating(ctx context.Context, from, to time.Time) (HeatingResponse, error) {
	settings, location, err := s.settings(ctx)
	if err != nil {
		return HeatingResponse{}, err
	}
	return s.Heating(ctx, from.In(location), to.In(location), settings, location)
}

func (s *Service) settings(ctx context.Context) (domainconfig.Settings, *time.Location, error) {
	settings, err := s.Configuration.Get(ctx)
	if err != nil {
		return domainconfig.Settings{}, nil, err
	}
	location, err := time.LoadLocation(settings.TimeZone)
	if err != nil {
		return domainconfig.Settings{}, nil, fmt.Errorf("load configured time zone: %w", err)
	}
	return settings, location, nil
}

// query loads measurements in ascending timestamp order for the requested types and filters.
func (s *Service) query(ctx context.Context, types []string, from, to *time.Time, sources []string) ([]measurements.Measurement, error) {
	options := measurements.QueryOptions{
		Filter: measurements.Filter{Types: types, Sources: sources, From: from, To: to},
		Sort:   []measurements.Sort{{Field: measurements.SortByTimestamp, Direction: measurements.SortAscending}},
	}
	rows, err := s.Measurements.Find(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("load analytics measurements: %w", err)
	}
	return rows, nil
}
