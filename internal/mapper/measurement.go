package mapper

import (
	"strconv"
	"strings"
	"time"

	"github.com/theabys/enerflux/internal/datasource/eta"
	"github.com/theabys/enerflux/internal/datasource/solarlog"
	"github.com/theabys/enerflux/internal/model"
)

func FromSolarLog(r solarlog.Response) []model.Measurement {
	s := r.Section801.Section170
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
			Type:   "pv.dc.power",
			Value:  float64(s.PVDCPower),
			Unit:   "W",
			Source: "solarlog",
		},

		{
			TS:     ts,
			Type:   "grid.ac.voltage",
			Value:  float64(s.ACVoltage),
			Unit:   "V",
			Source: "solarlog",
		},
		{
			TS:     ts,
			Type:   "grid.dc.voltage",
			Value:  float64(s.DCVoltage),
			Unit:   "V",
			Source: "solarlog",
		},

		{
			TS:     ts,
			Type:   "energy.yield.today",
			Value:  float64(s.YieldToday),
			Unit:   "Wh",
			Source: "solarlog",
		},
		{
			TS:     ts,
			Type:   "energy.yield.yesterday",
			Value:  float64(s.YieldYesterday),
			Unit:   "Wh",
			Source: "solarlog",
		},
		{
			TS:     ts,
			Type:   "energy.yield.month",
			Value:  float64(s.YieldMonth),
			Unit:   "Wh",
			Source: "solarlog",
		},
		{
			TS:     ts,
			Type:   "energy.yield.year",
			Value:  float64(s.YieldYear),
			Unit:   "Wh",
			Source: "solarlog",
		},
		{
			TS:     ts,
			Type:   "energy.yield.total",
			Value:  float64(s.YieldTotal),
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

		{
			TS:     ts,
			Type:   "grid.consumption.today",
			Value:  float64(s.YieldToday), // adjust if SolarLog has real field mapping
			Unit:   "Wh",
			Source: "solarlog",
		},
		{
			TS:     ts,
			Type:   "grid.consumption.yesterday",
			Value:  float64(s.YieldYesterday),
			Unit:   "Wh",
			Source: "solarlog",
		},
		{
			TS:     ts,
			Type:   "grid.consumption.month",
			Value:  float64(s.YieldMonth),
			Unit:   "Wh",
			Source: "solarlog",
		},
		{
			TS:     ts,
			Type:   "grid.consumption.year",
			Value:  float64(s.YieldYear),
			Unit:   "Wh",
			Source: "solarlog",
		},
		{
			TS:     ts,
			Type:   "grid.consumption.total",
			Value:  float64(s.YieldTotal),
			Unit:   "Wh",
			Source: "solarlog",
		},
	}
}
func FromETA(r eta.Eta) []model.Measurement {
	ts := time.Now()

	out := make([]model.Measurement, 0, len(r.Values))

	for _, v := range r.Values {
		out = append(out, model.Measurement{
			TS:     ts,
			Type:   mapETAType(v.Uri),
			Value:  parseCommaFloat(v.StrValue),
			Unit:   v.Unit,
			Source: "eta",
		})
	}

	return out
}

func mapETAType(uri string) string {
	switch uri {
	case "/user/var/120/10601/0/0/12197":
		return "climate.temperature.outside"
	case "/user/var/40/10201/0/0/12015":
		return "heating.pellet.stock"
	default:
		return "unknown"
	}
}

func parseCommaFloat(s string) float64 {
	s = strings.ReplaceAll(s, ",", ".")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
