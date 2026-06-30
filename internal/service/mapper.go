package service

import (
	"time"

	"github.com/theabys/enerflux/internal/datasource/solarlog"
	"github.com/theabys/enerflux/internal/model"
)

func MapSnapshotToMeasurements(s solarlog.Snapshot) []model.Measurement {

	ts := time.Now()

	return []model.Measurement{

		{
			TS:     ts,
			Type:   "pv.power",
			Value:  float64(s.PVPower),
			Unit:   "W",
			Source: "solarlog",
		},

		{
			TS:     ts,
			Type:   "pv.energy.day",
			Value:  float64(s.YieldToday),
			Unit:   "Wh",
			Source: "solarlog",
		},

		{
			TS:     ts,
			Type:   "grid.consumption.power",
			Value:  float64(s.ConsumptionPower),
			Unit:   "W",
			Source: "solarlog",
		},
	}
}
