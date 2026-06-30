package model

import "time"

type Measurement struct {
	ID     int
	TS     time.Time
	Type   string
	Value  float64
	Unit   string
	Source string
}
