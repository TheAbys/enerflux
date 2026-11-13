package metrics

import "time"

type Metric struct {
	TS     time.Time `json:"TS"`
	Value  float64   `json:"Value"`
	Unit   string    `json:"Unit"`
	Source string    `json:"Source"`
}
