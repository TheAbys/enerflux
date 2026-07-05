package model

import "time"

type Metric struct {
	TS    time.Time
	Value float64
}
