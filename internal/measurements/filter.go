package filter

import "time"

type MeasurementFilter struct {
	Key        string
	From       time.Time
	To         time.Time
	Interval   time.Duration
	Datasource string
}
