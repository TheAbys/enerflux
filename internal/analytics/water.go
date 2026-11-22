package analytics

import (
	"context"
	"strings"
	"time"

	"github.com/theabys/enerflux/internal/domainconfig"
	"github.com/theabys/enerflux/internal/units"
)

// Water calculates water usage from cumulative meter readings and applies configured water rates.
func (s *Service) Water(ctx context.Context, from, to time.Time, settings domainconfig.Settings, location *time.Location) (WaterResponse, error) {
	rows, err := s.query(ctx, []string{waterType}, nil, &to, nil)
	if err != nil {
		return WaterResponse{}, err
	}
	points := toReadings(rows, location)
	result := meterUsage(points, from, to)
	price := sumPrices(settings.WaterPricePerM3, settings.WastewaterPricePerM3)
	consumed := DomainValue{
		Value:        valuePointer(result.value),
		Unit:         units.CubicMeter,
		PricePerUnit: price,
		PriceUnit:    units.EuroPerCubicMeter,
		Intervals:    result.intervals,
		CoveredHours: result.covered.Hours(),
		Estimate:     true,
		Status:       "Estimated by spreading meter changes across the time between readings.",
	}
	if result.intervals == 0 {
		consumed.Value = nil
		consumed.Status = "Need consecutive water-meter readings overlapping this period."
	} else if price != nil {
		consumed.CostEUR = valuePointer(result.value * *price)
	}
	daily := meterDaily(points, from, to, location, units.CubicMeter)
	return WaterResponse{
		Window:         Window{From: from, To: to},
		CurrentReading: latestReading(points, to),
		Consumed:       consumed,
		Readings:       filterReadings(points, from, to),
		Daily:          daily,
	}, nil
}

// meterUsage assumes consumption is uniform between consecutive cumulative readings and prorates
// each increase over its overlap with the requested window. Decreases, which can indicate a reset
// or meter replacement, are excluded.
func meterUsage(points []ReadingPoint, from, to time.Time) integrated {
	var total integrated
	for index := 1; index < len(points); index++ {
		leftValue, leftOK := convertWater(points[index-1])
		rightValue, rightOK := convertWater(points[index])
		left, right := points[index-1].TS, points[index].TS
		span := right.Sub(left)
		if !leftOK || !rightOK || span <= 0 || rightValue < leftValue {
			continue
		}
		start, end := maxTime(left, from), minTime(right, to)
		if !end.After(start) {
			continue
		}
		total.value += (rightValue - leftValue) * float64(end.Sub(start)) / float64(span)
		total.intervals++
		total.covered += end.Sub(start)
	}
	return total
}

// meterDaily assumes consumption is uniform between consecutive meter readings and splits each
// valid increase across the local calendar days it overlaps.
func meterDaily(points []ReadingPoint, from, to time.Time, location *time.Location, unit string) []DailyValue {
	buckets := map[string]float64{}
	for index := 1; index < len(points); index++ {
		leftValue, leftOK := convertWater(points[index-1])
		rightValue, rightOK := convertWater(points[index])
		left, right := points[index-1].TS, points[index].TS
		span := right.Sub(left)
		if !leftOK || !rightOK || span <= 0 || rightValue < leftValue {
			continue
		}
		start, end := maxTime(left, from), minTime(right, to)
		if !end.After(start) {
			continue
		}
		for dayStart := dayStart(start, location); dayStart.Before(end); {
			next := dayStart.AddDate(0, 0, 1)
			partStart, partEnd := maxTime(start, dayStart), minTime(end, next)
			if partEnd.After(partStart) {
				buckets[dayStart.Format("2006-01-02")] += (rightValue - leftValue) * float64(partEnd.Sub(partStart)) / float64(span)
			}
			dayStart = next
		}
	}
	return sortedDaily(buckets, unit)
}

// convertWater converts cubic-meter and liter readings to cubic meters.
func convertWater(point ReadingPoint) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(point.Unit)) {
	case units.CubicMeter, units.CubicMeterASCII:
		return point.Value, true
	case "l", "liter", "litre", "liters", "litres":
		return point.Value / 1000, true
	default:
		return 0, false
	}
}

// sumPrices returns the sum only when both component rates are configured.
func sumPrices(first, second *float64) *float64 {
	if first == nil || second == nil {
		return nil
	}
	return valuePointer(*first + *second)
}
