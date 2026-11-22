package analytics

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/theabys/enerflux/internal/domainconfig"
	"github.com/theabys/enerflux/internal/units"
)

// Electricity calculates grid energy, its estimated cost, and daily energy totals for a window.
func (s *Service) Electricity(ctx context.Context, from, to time.Time, settings domainconfig.Settings, location *time.Location) (ElectricityResponse, error) {
	queryFrom := from.Add(-maxPowerGap)
	rows, err := s.query(ctx, []string{gridPowerType}, &queryFrom, &to, nil)
	if err != nil {
		return ElectricityResponse{}, err
	}
	points := toReadings(rows, location)
	result := integratePower(points, from, to)
	consumed := DomainValue{
		Value:        valuePointer(result.kWh),
		Unit:         units.KilowattHour,
		PricePerUnit: settings.ElectricityPricePerKWh,
		PriceUnit:    units.EuroPerKilowattHour,
		Intervals:    result.intervals,
		CoveredHours: result.covered.Hours(),
		Estimate:     true,
		Status:       "Estimated from measured grid power; gaps over 15 minutes are omitted.",
	}
	if result.intervals == 0 {
		consumed.Value = nil
		consumed.Status = "No complete grid-power intervals in this period."
	} else if settings.ElectricityPricePerKWh != nil {
		consumed.CostEUR = valuePointer(result.kWh * *settings.ElectricityPricePerKWh)
	}
	daily := powerDaily(points, from, to, location)
	windowPoints := filterReadings(points, from, to)
	return ElectricityResponse{
		Window:        Window{From: from, To: to},
		CurrentPower:  latestReading(points, to),
		Consumed:      consumed,
		PowerReadings: windowPoints,
		Daily:         daily,
	}, nil
}

// integratePower assumes a linear change between consecutive readings and applies trapezoidal
// integration to estimate kWh. It clips intervals to the requested window, converts W/kW to W,
// clamps negative power to zero, and skips intervals longer than maxPowerGap.
func integratePower(points []ReadingPoint, from, to time.Time) integrated {
	groups := groupReadings(points)
	var total integrated
	for _, series := range groups {
		for index := 1; index < len(series); index++ {
			left, leftOK := powerW(series[index-1])
			right, rightOK := powerW(series[index])
			start, end := series[index-1].TS, series[index].TS
			span := end.Sub(start)
			if !leftOK || !rightOK || span <= 0 || span > maxPowerGap {
				continue
			}
			start = maxTime(start, from)
			end = minTime(end, to)
			if !end.After(start) {
				continue
			}
			startFraction := float64(start.Sub(series[index-1].TS)) / float64(span)
			endFraction := float64(end.Sub(series[index-1].TS)) / float64(span)
			startW := math.Max(0, left+(right-left)*startFraction)
			endW := math.Max(0, left+(right-left)*endFraction)
			total.kWh += ((startW + endW) / 2) * end.Sub(start).Hours() / 1000
			total.intervals++
			total.covered += end.Sub(start)
		}
	}
	return total
}

// powerDaily splits valid trapezoidal power intervals across local calendar days.
func powerDaily(points []ReadingPoint, from, to time.Time, location *time.Location) []DailyValue {
	buckets := map[string]float64{}
	for _, series := range groupReadings(points) {
		for index := 1; index < len(series); index++ {
			leftW, leftOK := powerW(series[index-1])
			rightW, rightOK := powerW(series[index])
			left, right := series[index-1].TS, series[index].TS
			span := right.Sub(left)
			if !leftOK || !rightOK || span <= 0 || span > maxPowerGap {
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
					f0 := float64(partStart.Sub(left)) / float64(span)
					f1 := float64(partEnd.Sub(left)) / float64(span)
					p0 := math.Max(0, leftW+(rightW-leftW)*f0)
					p1 := math.Max(0, leftW+(rightW-leftW)*f1)
					key := dayStart.Format("2006-01-02")
					buckets[key] += ((p0 + p1) / 2) * partEnd.Sub(partStart).Hours() / 1000
				}
				dayStart = next
			}
		}
	}
	return sortedDaily(buckets, units.KilowattHour)
}

// powerW converts supported watt and kilowatt readings to watts.
func powerW(point ReadingPoint) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(point.Unit)) {
	case strings.ToLower(units.Watt), "watt", "watts":
		return point.Value, true
	case strings.ToLower(units.Kilowatt), "kilowatt", "kilowatts":
		return point.Value * 1000, true
	default:
		return 0, false
	}
}
