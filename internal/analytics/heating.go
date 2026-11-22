package analytics

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/theabys/enerflux/internal/domainconfig"
	"github.com/theabys/enerflux/internal/units"
)

// Heating calculates pellet consumption, purchase-based costs, stock history, and its forecast.
func (s *Service) Heating(ctx context.Context, from, to time.Time, settings domainconfig.Settings, location *time.Location) (HeatingResponse, error) {
	rows, err := s.query(ctx, []string{pelletStockType}, nil, &to, nil)
	if err != nil {
		return HeatingResponse{}, err
	}
	points := toReadings(rows, location)
	points = normalizeStockPoints(points)
	kgPoints := normalizeStockUnits(points, settings.PelletTankCapacityKg)
	period := pelletUsage(kgPoints, from, to)
	first := firstTime(kgPoints)
	lifetime := pelletUsage(kgPoints, first, to)
	last30From := to.AddDate(0, 0, -30)
	last30 := pelletUsage(kgPoints, last30From, to)
	price, purchases := weightedPurchasePrice(ctx, s, to, location)
	forecast, forecastStatus := s.pelletForecast(ctx, to, settings.PelletTankCapacityKg, location)
	consumed := DomainValue{
		Value:        valuePointer(period.value),
		Unit:         units.Kilogram,
		PricePerUnit: price,
		PriceUnit:    units.EuroPerKilogram,
		Intervals:    period.intervals,
		CoveredHours: period.covered.Hours(),
		Estimate:     true,
		Status:       "Estimated from stock decreases; refills are treated as boundaries.",
	}
	if period.intervals == 0 {
		consumed.Value = nil
		consumed.Status = "Need two stock readings spanning this period."
	} else if price != nil {
		consumed.CostEUR = valuePointer(period.value * *price)
	}
	var lifetimeKg, last30Kg, dailyAverage, last30Cost *float64
	if lifetime.intervals > 0 {
		lifetimeKg = valuePointer(lifetime.value)
	}
	if last30.intervals > 0 {
		last30Kg = valuePointer(last30.value)
		if last30.covered > 0 {
			dailyAverage = valuePointer(last30.value / (last30.covered.Hours() / 24))
		}
		if price != nil {
			last30Cost = valuePointer(last30.value * *price)
		}
	}
	stock := latestReading(points, to)
	if stock != nil {
		if kg, ok := stockToKg(*stock, settings.PelletTankCapacityKg); ok {
			stock.Value = kg
			stock.Unit = units.Kilogram
		}
	}
	stockReadings := normalizedStockReadings(points, settings.PelletTankCapacityKg)
	for index := 1; index < len(points); index++ {
		previous, previousOK := stockToKg(points[index-1], settings.PelletTankCapacityKg)
		current, currentOK := stockToKg(points[index], settings.PelletTankCapacityKg)
		if previousOK && currentOK && current > previous {
			for readingIndex := range stockReadings {
				if stockReadings[readingIndex].TS.Equal(points[index].TS) && stockReadings[readingIndex].Source == points[index].Source {
					stockReadings[readingIndex].EventType = "stock_increase"
					break
				}
			}
		}
	}
	return HeatingResponse{
		Window:                    Window{From: from, To: to},
		CurrentStock:              stock,
		Consumed:                  consumed,
		LifetimeConsumedKg:        lifetimeKg,
		AverageDaily30DaysKg:      dailyAverage,
		ConsumedLast30DaysKg:      last30Kg,
		ConsumedLast30DaysCostEUR: last30Cost,
		Last30DaysIntervals:       last30.intervals,
		Last30DaysCoveredHours:    last30.covered.Hours(),
		ForecastNextHourKg:        forecast,
		ForecastStatus:            forecastStatus,
		AveragePurchasePriceEUR:   price,
		StockReadings:             filterReadings(stockReadings, from, to),
		Daily:                     pelletDaily(kgPoints, from, to, location),
		Purchases:                 purchases,
	}, nil
}

// pelletUsage assumes consumption is uniform between consecutive stock readings and prorates each
// decrease over its overlap with the requested window. Increases are treated as refills and excluded.
func pelletUsage(points []ReadingPoint, from, to time.Time) integrated {
	var total integrated
	for index := 1; index < len(points); index++ {
		left, leftOK := stockToKg(points[index-1], nil)
		right, rightOK := stockToKg(points[index], nil)
		startTime, endTime := points[index-1].TS, points[index].TS
		span := endTime.Sub(startTime)
		if !leftOK || !rightOK || span <= 0 || right >= left {
			continue
		}
		start, end := maxTime(startTime, from), minTime(endTime, to)
		if !end.After(start) {
			continue
		}
		total.value += (left - right) * float64(end.Sub(start)) / float64(span)
		total.intervals++
		total.covered += end.Sub(start)
	}
	return total
}

// pelletDaily assumes a uniform stock decrease between readings and splits valid decreases across
// the local calendar days they overlap. Refill intervals are omitted.
func pelletDaily(points []ReadingPoint, from, to time.Time, location *time.Location) []DailyValue {
	buckets := map[string]float64{}
	for index := 1; index < len(points); index++ {
		left, leftOK := stockToKg(points[index-1], nil)
		right, rightOK := stockToKg(points[index], nil)
		startTime, endTime := points[index-1].TS, points[index].TS
		span := endTime.Sub(startTime)
		if !leftOK || !rightOK || span <= 0 || right >= left {
			continue
		}
		start, end := maxTime(startTime, from), minTime(endTime, to)
		if !end.After(start) {
			continue
		}
		for dayStart := dayStart(start, location); dayStart.Before(end); {
			next := dayStart.AddDate(0, 0, 1)
			partStart, partEnd := maxTime(start, dayStart), minTime(end, next)
			if partEnd.After(partStart) {
				share := (left - right) * float64(partEnd.Sub(partStart)) / float64(span)
				buckets[dayStart.Format("2006-01-02")] += share
			}
			dayStart = next
		}
	}
	return sortedDaily(buckets, units.Kilogram)
}

// pelletForecast estimates a per-hour pellet usage rate from recent ETA stock measurements. It
// uses the latest contiguous series, requires at least two hours of observations, and rejects a
// latest reading older than maxForecastAge.
func (s *Service) pelletForecast(ctx context.Context, to time.Time, capacity *float64, location *time.Location) (*float64, string) {
	from := to.Add(-forecastWindow)
	rows, err := s.query(ctx, []string{pelletStockType}, &from, &to, []string{etaSource})
	if err != nil {
		return nil, "Forecast unavailable because recent ETA readings could not be loaded."
	}
	points := normalizeStockPoints(toReadings(rows, location))
	points = normalizeStockUnits(points, capacity)
	if len(points) < 2 {
		return nil, "Need at least two recent ETA stock readings."
	}
	continuousStart := len(points) - 1
	for continuousStart > 0 && points[continuousStart].TS.Sub(points[continuousStart-1].TS) <= maxForecastAge {
		continuousStart--
	}
	points = points[continuousStart:]
	if len(points) < 2 {
		return nil, "Recent ETA stock readings are too far apart."
	}
	latest := points[len(points)-1]
	if to.Sub(latest.TS) > maxForecastAge {
		return nil, "Latest ETA stock reading is older than 20 minutes."
	}
	first := points[0]
	observed := latest.TS.Sub(first.TS)
	if observed < minimumForecastAge {
		return nil, "Need at least two hours of ETA stock history."
	}
	usage := pelletUsage(points, points[0].TS, to)
	if usage.intervals == 0 {
		return nil, "No declining ETA stock intervals in the forecast window."
	}
	return valuePointer(usage.value / observed.Hours()), "Estimated from the recent ETA stock decline."
}

// normalizeStockPoints sorts readings and prefers ETA values when unlike sources report near-simultaneously.
func normalizeStockPoints(points []ReadingPoint) []ReadingPoint {
	points = append([]ReadingPoint(nil), points...)
	sort.SliceStable(points, func(i, j int) bool { return points[i].TS.Before(points[j].TS) })
	result := make([]ReadingPoint, 0, len(points))
	for _, point := range points {
		previous := len(result) - 1
		if previous >= 0 && result[previous].Source != point.Source && math.Abs(float64(point.TS.Sub(result[previous].TS))) <= float64(15*time.Minute) {
			if point.Source == etaSource && result[previous].Source != etaSource {
				result[previous] = point
			}
			continue
		}
		result = append(result, point)
	}
	return result
}

// normalizeStockUnits converts supported stock readings to kilograms and drops unconvertible units.
func normalizeStockUnits(points []ReadingPoint, capacity *float64) []ReadingPoint {
	result := make([]ReadingPoint, 0, len(points))
	for _, point := range points {
		if kg, ok := stockToKg(point, capacity); ok {
			point.Value = kg
			point.Unit = units.Kilogram
			result = append(result, point)
		}
	}
	return result
}

// normalizedStockReadings converts supported readings to kilograms while retaining unconvertible readings.
func normalizedStockReadings(points []ReadingPoint, capacity *float64) []ReadingPoint {
	result := make([]ReadingPoint, 0, len(points))
	for _, point := range points {
		if kg, ok := stockToKg(point, capacity); ok {
			point.Value = kg
			point.Unit = units.Kilogram
		}
		result = append(result, point)
	}
	return result
}

// stockToKg converts a supported stock unit to kilograms, using capacity for percentage readings.
func stockToKg(point ReadingPoint, capacity *float64) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(point.Unit)) {
	case units.Kilogram, "kilogram", "kilograms", "":
		return point.Value, true
	case "g", "gram", "grams":
		return point.Value / 1000, true
	case "t", "ton", "tonne", "tonnes":
		return point.Value * 1000, true
	case "%", "percent", "percentage":
		if capacity == nil || *capacity <= 0 {
			return 0, false
		}
		return point.Value * *capacity / 100, true
	default:
		return 0, false
	}
}
