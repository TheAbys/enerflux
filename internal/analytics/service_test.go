package analytics

import (
	"context"
	"errors"
	"math"
	"sort"
	"testing"
	"time"

	"github.com/theabys/enerflux/internal/domainconfig"
	"github.com/theabys/enerflux/internal/measurements"
)

type memoryMeasurementStore struct {
	rows []measurements.Measurement
	err  error
}

func (s *memoryMeasurementStore) Find(_ context.Context, options measurements.QueryOptions) ([]measurements.Measurement, error) {
	if s.err != nil {
		return nil, s.err
	}
	rows := make([]measurements.Measurement, 0, len(s.rows))
	for _, row := range s.rows {
		if len(options.Filter.Types) > 0 && !contains(options.Filter.Types, row.Type) {
			continue
		}
		if len(options.Filter.Sources) > 0 && !contains(options.Filter.Sources, row.Source) {
			continue
		}
		if options.Filter.From != nil && row.TS.Before(*options.Filter.From) {
			continue
		}
		if options.Filter.To != nil && row.TS.After(*options.Filter.To) {
			continue
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].TS.Before(rows[j].TS)
	})
	if options.Limit > 0 && len(rows) > options.Limit {
		rows = rows[:options.Limit]
	}
	return rows, nil
}

type memoryDomainConfigStore struct {
	settings SettingsForTest
	err      error
	saved    *domainconfig.Settings
}

type SettingsForTest = domainconfig.Settings

func (s *memoryDomainConfigStore) Get(context.Context) (domainconfig.Settings, error) {
	if s.err != nil {
		return domainconfig.Settings{}, s.err
	}
	return s.settings, nil
}

func (s *memoryDomainConfigStore) Save(_ context.Context, settings domainconfig.Settings) error {
	if s.err != nil {
		return s.err
	}
	s.saved = &settings
	s.settings = settings
	return nil
}

func ptr(value float64) *float64 { return &value }

func point(at time.Time, value float64, unit, source string) ReadingPoint {
	return ReadingPoint{TS: at, Value: value, Unit: unit, Source: source}
}

func measurement(at time.Time, kind string, value float64, unit, source string) measurements.Measurement {
	return measurements.Measurement{TS: at, Type: kind, Value: value, Unit: unit, Source: source}
}

func closeTo(t *testing.T, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("got %.10f, want %.10f (tolerance %.10f)", got, want, tolerance)
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func testLocation(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	return location
}

func TestIntegratePowerUsesTrapezoidAndUnitConversion(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := make([]ReadingPoint, 0, 7)
	for index := 0; index <= 6; index++ {
		value := 1000 + float64(index)*1000/6
		unit := "W"
		if index%2 == 1 {
			value /= 1000
			unit = "kW"
		}
		points = append(points, point(start.Add(time.Duration(index)*10*time.Minute), value, unit, "solarlog"))
	}
	got := integratePower(points, start, start.Add(time.Hour))
	closeTo(t, got.kWh, 1.5, 1e-9)
	if got.intervals != 6 || got.covered != time.Hour {
		t.Fatalf("unexpected interval coverage: %+v", got)
	}
}

func TestIntegratePowerClipsRangeAndRejectsLongGaps(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := []ReadingPoint{
		point(start, 1000, "W", "one"),
		point(start.Add(15*time.Minute), 3000, "W", "one"),
		point(start.Add(31*time.Minute), 3000, "W", "one"),
	}
	got := integratePower(points, start.Add(7*time.Minute+30*time.Second), start.Add(15*time.Minute))
	closeTo(t, got.kWh, .3125, 1e-9)
	if got.intervals != 1 {
		t.Fatalf("expected exactly the clipped 15 minute segment, got %d", got.intervals)
	}
	tooLong := integratePower(points[1:], start.Add(15*time.Minute), start.Add(31*time.Minute))
	if tooLong.intervals != 0 || tooLong.kWh != 0 {
		t.Fatalf("expected gap over 15 minutes to be omitted, got %+v", tooLong)
	}
}

func TestIntegratePowerClampsNegativePower(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := integratePower([]ReadingPoint{
		point(start, -1000, "W", "one"),
		point(start.Add(10*time.Minute), 1000, "W", "one"),
	}, start, start.Add(10*time.Minute))
	closeTo(t, got.kWh, 1.0/12.0, 1e-9)
}

func TestPowerDailySplitsAtLocalMidnight(t *testing.T) {
	location := time.UTC
	start := time.Date(2026, 1, 1, 23, 55, 0, 0, location)
	points := []ReadingPoint{
		point(start, 1000, "W", "one"),
		point(start.Add(10*time.Minute), 1000, "W", "one"),
	}
	daily := powerDaily(points, start, start.Add(10*time.Minute), location)
	if len(daily) != 2 {
		t.Fatalf("expected two local day buckets, got %+v", daily)
	}
	if daily[0].Date != "2026-01-01" || daily[1].Date != "2026-01-02" {
		t.Fatalf("unexpected day keys: %+v", daily)
	}
	closeTo(t, daily[0].Value, 1.0/12.0, 1e-9)
	closeTo(t, daily[1].Value, 1.0/12.0, 1e-9)
}

func TestMeterUsageUsesBaselineAndProratesRange(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := []ReadingPoint{
		point(start, 10, "m³", "manual"),
		point(start.Add(2*time.Hour), 14, "m³", "manual"),
		point(start.Add(3*time.Hour), 2, "m³", "manual"), // reset is ignored
		point(start.Add(4*time.Hour), 3000, "L", "manual"),
		point(start.Add(5*time.Hour), 4000, "L", "manual"),
	}
	got := meterUsage(points, start.Add(time.Hour), start.Add(5*time.Hour))
	closeTo(t, got.value, 4, 1e-9) // 2 m³ clipped from the first interval plus two 1 m³ intervals
	if got.intervals != 3 {
		t.Fatalf("expected two valid intervals, got %+v", got)
	}
}

func TestMeterDailySplitsAcrossLocalMidnight(t *testing.T) {
	start := time.Date(2026, 1, 1, 23, 55, 0, 0, time.UTC)
	daily := meterDaily([]ReadingPoint{
		point(start, 0, "m3", "manual"),
		point(start.Add(10*time.Minute), 1, "m3", "manual"),
	}, start, start.Add(10*time.Minute), time.UTC, "m³")
	if len(daily) != 2 {
		t.Fatalf("expected two water day buckets, got %+v", daily)
	}
	closeTo(t, daily[0].Value, .5, 1e-9)
	closeTo(t, daily[1].Value, .5, 1e-9)
}

func TestNormalizeStockPointsPrefersETAWhenSourcesAreNear(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := normalizeStockPoints([]ReadingPoint{
		point(start, 1200, "kg", "manual"),
		point(start.Add(10*time.Minute), 1198, "kg", etaSource),
		point(start.Add(40*time.Minute), 1190, "kg", "manual"),
	})
	if len(got) != 2 {
		t.Fatalf("expected close cross-source rows to deduplicate, got %+v", got)
	}
	if got[0].Source != etaSource || got[0].Value != 1198 {
		t.Fatalf("expected ETA sample to win, got %+v", got[0])
	}
	if got[1].Source != "manual" {
		t.Fatalf("expected distant manual point to remain, got %+v", got[1])
	}
}

func TestStockToKgConversions(t *testing.T) {
	tests := []struct {
		name     string
		point    ReadingPoint
		capacity *float64
		want     float64
		ok       bool
	}{
		{"kg", point(time.Time{}, 12, "kg", "manual"), nil, 12, true},
		{"grams", point(time.Time{}, 2500, "g", "manual"), nil, 2.5, true},
		{"tonnes", point(time.Time{}, 1.25, "t", "manual"), nil, 1250, true},
		{"percentage with capacity", point(time.Time{}, 25, "%", etaSource), ptr(4000), 1000, true},
		{"percentage without capacity", point(time.Time{}, 25, "%", etaSource), nil, 0, false},
		{"unknown unit", point(time.Time{}, 25, "lb", etaSource), ptr(4000), 0, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := stockToKg(test.point, test.capacity)
			if ok != test.ok {
				t.Fatalf("ok=%v, want %v", ok, test.ok)
			}
			if ok {
				closeTo(t, got, test.want, 1e-9)
			}
		})
	}
}

func TestPelletUsageIgnoresRefillsAndClipsWindow(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := []ReadingPoint{
		point(start, 1000, "kg", "manual"),
		point(start.Add(10*time.Hour), 900, "kg", "manual"),
		point(start.Add(20*time.Hour), 2800, "kg", "manual"), // refill boundary
		point(start.Add(30*time.Hour), 2700, "kg", "manual"),
	}
	got := pelletUsage(points, start.Add(5*time.Hour), start.Add(30*time.Hour))
	closeTo(t, got.value, 150, 1e-9)
	if got.intervals != 2 || got.covered != 15*time.Hour {
		t.Fatalf("unexpected pellet coverage: %+v", got)
	}
}

func TestPelletDailySplitsAtLocalMidnight(t *testing.T) {
	start := time.Date(2026, 1, 1, 23, 55, 0, 0, time.UTC)
	daily := pelletDaily([]ReadingPoint{
		point(start, 100, "kg", "manual"),
		point(start.Add(10*time.Minute), 90, "kg", "manual"),
	}, start, start.Add(10*time.Minute), time.UTC)
	if len(daily) != 2 {
		t.Fatalf("expected two pellet day buckets, got %+v", daily)
	}
	closeTo(t, daily[0].Value, 5, 1e-9)
	closeTo(t, daily[1].Value, 5, 1e-9)
}

func TestElectricityServiceIntegratesAndPricesPower(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []measurements.Measurement{
		measurement(start, gridPowerType, 1000, "W", "solarlog"),
		measurement(start.Add(15*time.Minute), gridPowerType, 2000, "W", "solarlog"),
	}
	store := &memoryMeasurementStore{rows: rows}
	settings := domainconfig.Settings{TimeZone: "UTC", ElectricityPricePerKWh: ptr(.3)}
	service := NewService(store, NewTestDomainConfig(settings))
	result, err := service.Electricity(context.Background(), start, start.Add(15*time.Minute), settings, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	closeTo(t, *result.Consumed.Value, .375, 1e-9)
	closeTo(t, *result.Consumed.CostEUR, .1125, 1e-9)
	if result.Consumed.Intervals != 1 || result.CurrentPower == nil || result.CurrentPower.Value != 2000 {
		t.Fatalf("unexpected electricity response: %+v", result)
	}
}

func TestWaterServiceRequiresBothRatesForCost(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store := &memoryMeasurementStore{rows: []measurements.Measurement{
		measurement(start, waterType, 10, "m³", "manual"),
		measurement(start.Add(2*time.Hour), waterType, 14, "m³", "manual"),
	}}
	settings := domainconfig.Settings{TimeZone: "UTC", WaterPricePerM3: ptr(2), WastewaterPricePerM3: ptr(3)}
	service := NewService(store, NewTestDomainConfig(settings))
	result, err := service.Water(context.Background(), start.Add(time.Hour), start.Add(2*time.Hour), settings, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	closeTo(t, *result.Consumed.Value, 2, 1e-9)
	closeTo(t, *result.Consumed.CostEUR, 10, 1e-9)
	settings.WastewaterPricePerM3 = nil
	if sumPrices(settings.WaterPricePerM3, settings.WastewaterPricePerM3) != nil {
		t.Fatal("expected missing wastewater rate to leave water cost unavailable")
	}
}

func TestHeatingServiceCalculatesConsumptionPurchaseCostAndCoverageAverage(t *testing.T) {
	to := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	store := &memoryMeasurementStore{rows: []measurements.Measurement{
		measurement(to.Add(-48*time.Hour), pelletStockType, 1000, "kg", "manual"),
		measurement(to.Add(-24*time.Hour), pelletStockType, 900, "kg", "manual"),
		measurement(to, pelletStockType, 800, "kg", "manual"),
		measurement(to.Add(-72*time.Hour), purchaseQuantityType, 1000, "kg", "manual-purchase:one"),
		measurement(to.Add(-72*time.Hour), purchaseCostType, 300, "EUR", "manual-purchase:one"),
	}}
	settings := domainconfig.Settings{TimeZone: "UTC"}
	service := NewService(store, NewTestDomainConfig(settings))
	result, err := service.Heating(context.Background(), to.Add(-48*time.Hour), to, settings, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if result.CurrentStock == nil || result.CurrentStock.Value != 800 || result.LifetimeConsumedKg == nil {
		t.Fatalf("missing stock/lifetime values: %+v", result)
	}
	closeTo(t, *result.Consumed.Value, 200, 1e-9)
	closeTo(t, *result.LifetimeConsumedKg, 200, 1e-9)
	closeTo(t, *result.AveragePurchasePriceEUR, .3, 1e-9)
	closeTo(t, *result.Consumed.CostEUR, 60, 1e-9)
	closeTo(t, *result.AverageDaily30DaysKg, 100, 1e-9)
	if result.Last30DaysIntervals != 2 || result.Last30DaysCoveredHours != 48 {
		t.Fatalf("unexpected 30-day measurement coverage: %+v", result)
	}
	if len(result.Purchases) != 1 || result.Purchases[0].CostPerKgEUR == nil {
		t.Fatalf("expected grouped purchase summary: %+v", result.Purchases)
	}
	if result.ForecastNextHourKg != nil {
		t.Fatalf("manual readings must not create the ETA forecast: %v", *result.ForecastNextHourKg)
	}
}

func TestHeatingForecastNeedsFreshContinuousTwoHourEtaHistory(t *testing.T) {
	to := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	rows := make([]measurements.Measurement, 0, 13)
	for index := 0; index <= 12; index++ {
		rows = append(rows, measurement(to.Add(-2*time.Hour+time.Duration(index)*10*time.Minute), pelletStockType, 1000-float64(index)*2, "kg", etaSource))
	}
	store := &memoryMeasurementStore{rows: rows}
	service := NewService(store, NewTestDomainConfig(domainconfig.Settings{TimeZone: "UTC"}))
	forecast, status := service.pelletForecast(context.Background(), to, nil, time.UTC)
	if forecast == nil {
		t.Fatalf("expected forecast, got nil (%s)", status)
	}
	closeTo(t, *forecast, 12, 1e-9)

	forecast, status = service.pelletForecast(context.Background(), to.Add(21*time.Minute), nil, time.UTC)
	if forecast != nil || status != "Latest ETA stock reading is older than 20 minutes." {
		t.Fatalf("expected stale data to reject forecast; got %v (%s)", forecast, status)
	}
	forecast, status = service.pelletForecast(context.Background(), to.Add(30*time.Minute), nil, time.UTC)
	if forecast != nil || status != "Latest ETA stock reading is older than 20 minutes." {
		t.Fatalf("expected expired window to reject forecast; got %v (%s)", forecast, status)
	}
}

func TestWeightedPurchasePriceGroupsPairsAndIgnoresIncompleteOrFuturePurchases(t *testing.T) {
	to := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	store := &memoryMeasurementStore{rows: []measurements.Measurement{
		measurement(to.Add(-72*time.Hour), purchaseQuantityType, 1000, "kg", "purchase-one"),
		measurement(to.Add(-72*time.Hour), purchaseCostType, 300, "EUR", "purchase-one"),
		measurement(to.Add(-48*time.Hour), purchaseQuantityType, .5, "t", "purchase-two"),
		measurement(to.Add(-48*time.Hour), purchaseCostType, 200, "€", "purchase-two"),
		measurement(to.Add(-24*time.Hour), purchaseQuantityType, 500, "kg", "incomplete"),
		measurement(to.Add(time.Hour), purchaseQuantityType, 1000, "kg", "future"),
		measurement(to.Add(time.Hour), purchaseCostType, 100, "EUR", "future"),
	}}
	service := NewService(store, NewTestDomainConfig(domainconfig.Settings{TimeZone: "UTC"}))
	price, purchases := weightedPurchasePrice(context.Background(), service, to, time.UTC)
	if price == nil {
		t.Fatal("expected weighted price from two complete purchases")
	}
	closeTo(t, *price, 500.0/1500.0, 1e-9)
	if len(purchases) != 3 {
		t.Fatalf("expected two complete and one incomplete visible purchase, got %+v", purchases)
	}
	if purchases[0].TS.After(to) {
		t.Fatalf("future purchase must be excluded, newest returned was %s", purchases[0].TS)
	}
}

func TestHeatingServiceMarksRefillReadingsAsBoundaries(t *testing.T) {
	to := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	store := &memoryMeasurementStore{rows: []measurements.Measurement{
		measurement(to.Add(-24*time.Hour), pelletStockType, 1000, "kg", "manual"),
		measurement(to.Add(-12*time.Hour), pelletStockType, 1800, "kg", "manual"),
		measurement(to, pelletStockType, 1700, "kg", "manual"),
	}}
	service := NewService(store, NewTestDomainConfig(domainconfig.Settings{TimeZone: "UTC"}))
	result, err := service.Heating(context.Background(), to.Add(-24*time.Hour), to, domainconfig.Settings{TimeZone: "UTC"}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.StockReadings) != 3 || result.StockReadings[1].EventType != "stock_increase" {
		t.Fatalf("expected the stock increase to be marked by backend: %+v", result.StockReadings)
	}
	closeTo(t, *result.Consumed.Value, 100, 1e-9)
}

func TestCostOverviewOnlyReturnsTotalWhenAllDomainsHaveCosts(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(30 * 24 * time.Hour)
	complete := costOverview(from, to, ptr(10), ptr(2), ptr(5))
	if !complete.Complete || complete.TotalEUR == nil {
		t.Fatalf("expected complete total: %+v", complete)
	}
	closeTo(t, *complete.TotalEUR, 17, 1e-9)
	incomplete := costOverview(from, to, ptr(10), nil, ptr(5))
	if incomplete.Complete || incomplete.TotalEUR != nil || len(incomplete.MissingDomains) != 1 || incomplete.MissingDomains[0] != "water" {
		t.Fatalf("expected missing water cost to suppress total: %+v", incomplete)
	}
}

func TestQueryErrorsAreWrapped(t *testing.T) {
	store := &memoryMeasurementStore{err: errors.New("database unavailable")}
	service := NewService(store, NewTestDomainConfig(domainconfig.Settings{TimeZone: "UTC"}))
	_, err := service.query(context.Background(), []string{waterType}, nil, nil, nil)
	if err == nil || err.Error() == "database unavailable" {
		t.Fatalf("expected analytics context in error, got %v", err)
	}
}

func NewTestDomainConfig(settings domainconfig.Settings) *domainconfig.Service {
	return domainconfig.NewService(&memoryDomainConfigStore{settings: settings})
}
