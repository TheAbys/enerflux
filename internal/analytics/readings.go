package analytics

import (
	"sort"
	"time"

	"github.com/theabys/enerflux/internal/measurements"
)

// toReadings converts measurements to local-time reading points ordered by timestamp.
func toReadings(rows []measurements.Measurement, location *time.Location) []ReadingPoint {
	result := make([]ReadingPoint, 0, len(rows))
	for _, row := range rows {
		result = append(result, ReadingPoint{TS: row.TS.In(location), Value: row.Value, Unit: row.Unit, Source: row.Source, Note: row.Note})
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].TS.Before(result[j].TS) })
	return result
}

// groupReadings returns timestamp-ordered series grouped by source.
func groupReadings(points []ReadingPoint) [][]ReadingPoint {
	groups := map[string][]ReadingPoint{}
	order := make([]string, 0)
	for _, point := range points {
		if _, ok := groups[point.Source]; !ok {
			order = append(order, point.Source)
		}
		groups[point.Source] = append(groups[point.Source], point)
	}
	result := make([][]ReadingPoint, 0, len(groups))
	for _, key := range order {
		series := groups[key]
		sort.SliceStable(series, func(i, j int) bool { return series[i].TS.Before(series[j].TS) })
		result = append(result, series)
	}
	return result
}

// filterReadings returns points whose timestamps are within the inclusive window.
func filterReadings(points []ReadingPoint, from, to time.Time) []ReadingPoint {
	result := make([]ReadingPoint, 0)
	for _, point := range points {
		if !point.TS.Before(from) && !point.TS.After(to) {
			result = append(result, point)
		}
	}
	return result
}

// latestReading returns the latest point at or before to, or nil when none exists.
func latestReading(points []ReadingPoint, to time.Time) *ReadingPoint {
	for index := len(points) - 1; index >= 0; index-- {
		if !points[index].TS.After(to) {
			point := points[index]
			return &point
		}
	}
	return nil
}

// firstTime returns the first point's timestamp, or the zero time for an empty series.
func firstTime(points []ReadingPoint) time.Time {
	if len(points) == 0 {
		return time.Time{}
	}
	return points[0].TS
}
