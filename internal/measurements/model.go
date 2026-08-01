package measurements

import "time"

type Measurement struct {
	ID     int       `json:"id"`
	TS     time.Time `json:"ts"`
	Type   string    `json:"type"`
	Value  float64   `json:"value"`
	Unit   string    `json:"unit"`
	Source string    `json:"source"`
}

type MeasurementPayload Measurement
