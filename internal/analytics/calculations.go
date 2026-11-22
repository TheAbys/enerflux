package analytics

import (
	"sort"
	"time"
)

// integrated contains a window's estimated amount, interval count, and covered duration.
type integrated struct {
	kWh       float64
	value     float64
	intervals int
	covered   time.Duration
}

// sortedDaily converts date-keyed values to chronologically ordered daily results.
func sortedDaily(values map[string]float64, unit string) []DailyValue {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]DailyValue, 0, len(keys))
	for _, key := range keys {
		result = append(result, DailyValue{Date: key, Value: values[key], Unit: unit})
	}
	return result
}

// dayStart returns midnight at the start of value's local calendar day.
func dayStart(value time.Time, location *time.Location) time.Time {
	local := value.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
}

// maxTime returns the later of two timestamps.
func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// minTime returns the earlier of two timestamps.
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// valuePointer returns a pointer to value for optional response fields.
func valuePointer(value float64) *float64 {
	return &value
}
