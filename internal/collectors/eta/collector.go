package eta

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/theabys/enerflux/internal/collectors"
	"github.com/theabys/enerflux/internal/measurements"
)

type EtaCollector struct {
	Logger             *slog.Logger
	PelletStockFetcher collectors.Fetcher
	OutsideTempFetcher collectors.Fetcher
	Parser             *Parser
	Repo               measurements.MeasurementRepository
}

func NewEtaCollector(
	l *slog.Logger,
	pf collectors.Fetcher,
	of collectors.Fetcher,
	p *Parser,
	r measurements.MeasurementRepository,
) *EtaCollector {
	return &EtaCollector{
		Logger:             l.With("component", "eta-collector"),
		PelletStockFetcher: pf,
		OutsideTempFetcher: of,
		Parser:             p,
		Repo:               r,
	}
}

func (s *EtaCollector) SyncPelletStock(ctx context.Context) error {
	raw, err := s.PelletStockFetcher.Fetch(ctx)
	if err != nil {
		return err
	}

	response, err := s.Parser.Parse(raw)
	if err != nil {
		return err
	}

	measurements := FromETA(response)

	for _, m := range measurements {
		err := s.Repo.Insert(m)
		if err != nil {
			return err
		}
	}
	s.Logger.Info("sync completed", "inserted", len(measurements))

	return nil
}

func (s *EtaCollector) SyncOutsideTemp(ctx context.Context) error {
	raw, err := s.OutsideTempFetcher.Fetch(ctx)
	if err != nil {
		return err
	}

	response, err := s.Parser.Parse(raw)
	if err != nil {
		return err
	}

	measurements := FromETA(response)

	for _, m := range measurements {
		err := s.Repo.Insert(m)
		if err != nil {
			return err
		}
	}
	s.Logger.Info("sync completed", "inserted", len(measurements))

	return nil
}

func FromETA(r Eta) []measurements.Measurement {
	ts := time.Now()

	out := make([]measurements.Measurement, 0, len(r.Values))

	for _, v := range r.Values {
		out = append(out, measurements.Measurement{
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
