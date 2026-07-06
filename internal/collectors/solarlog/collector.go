package solarlog

import (
	"context"
	"log/slog"
	"time"

	"github.com/theabys/enerflux/internal/collectors"
	"github.com/theabys/enerflux/internal/measurements"
)

type SolarLogCollector struct {
	Logger  *slog.Logger
	Fetcher collectors.Fetcher
	Parser  *Parser
	Repo    measurements.MeasurementRepository
}

func NewSolarLogCollector(
	l *slog.Logger,
	f collectors.Fetcher,
	p *Parser,
	r measurements.MeasurementRepository,
) *SolarLogCollector {
	return &SolarLogCollector{
		Logger:  l.With("component", "solarlog-collector"),
		Fetcher: f,
		Parser:  p,
		Repo:    r,
	}
}

func (s *SolarLogCollector) Sync(ctx context.Context) error {
	raw, err := s.Fetcher.Fetch(ctx)
	if err != nil {
		return err
	}

	response, err := s.Parser.Parse(raw)
	if err != nil {
		return err
	}

	measurements := FromSolarLog(response)

	for _, m := range measurements {
		err := s.Repo.Insert(m)
		if err != nil {
			return err
		}
	}
	s.Logger.Info("sync completed", "inserted", len(measurements))

	return nil
}

func FromSolarLog(r Response) []measurements.Measurement {
	s := r.Section801.Section170
	ts := time.Now()

	return []measurements.Measurement{
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
