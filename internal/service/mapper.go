package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/theabys/enerflux/internal/datasource/eta"
	"github.com/theabys/enerflux/internal/datasource/solarlog"
	"github.com/theabys/enerflux/internal/model"
)

func mapSolarLogToMeasurements(r solarlog.Response) []model.Measurement {

	section170 := r.Section801["170"].(map[string]any)

	ts := time.Now()

	return []model.Measurement{

		{
			TS:     ts,
			Type:   "pv.power",
			Value:  float64(parseInt(section170["101"])),
			Unit:   "W",
			Source: "solarlog",
		},

		{
			TS:     ts,
			Type:   "pv.energy.day",
			Value:  float64(parseInt(section170["105"])),
			Unit:   "Wh",
			Source: "solarlog",
		},

		{
			TS:     ts,
			Type:   "grid.consumption.power",
			Value:  float64(parseInt(section170["110"])),
			Unit:   "W",
			Source: "solarlog",
		},
	}
}

func mapEtaPelletStockToMeasurements(r eta.Response) []model.Measurement {

	value := r.Values[0]

	ts := time.Now()

	return []model.Measurement{
		{
			TS:     ts,
			Type:   "eta.pellet.stock",
			Value:  float64(parseInt(value.StrValue)),
			Unit:   value.Unit,
			Source: "eta",
		},
	}
}

func mapEtaOutsideTempToMeasurements(r eta.Response) []model.Measurement {

	value := r.Values[0]

	ts := time.Now()

	return []model.Measurement{
		{
			TS:     ts,
			Type:   "eta.sensor.outside",
			Value:  parseCommaFloat(value.StrValue),
			Unit:   value.Unit,
			Source: "eta",
		},
	}
}

func parseInt(v any) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		return 0
	}
}

func parseTime(v any) time.Time {
	s, ok := v.(string)
	if !ok {
		return time.Time{}
	}

	t, err := time.Parse("02.01.06 15:04:05", s)
	if err != nil {
		return time.Time{}
	}

	return t
}

func parseCommaFloat(s string) float64 {
	s = strings.ReplaceAll(s, ",", ".")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
