package service

import (
	"context"
	"log/slog"

	fetcher "github.com/theabys/enerflux/internal/contract"
	"github.com/theabys/enerflux/internal/datasource/eta"
	"github.com/theabys/enerflux/internal/measurements"
)

type EtaService struct {
	Logger             *slog.Logger
	PelletStockFetcher fetcher.Fetcher
	OutsideTempFetcher fetcher.Fetcher
	Parser             *eta.Parser
	Repo               measurements.MeasurementRepository
}

func NewEtaService(
	l *slog.Logger,
	pf fetcher.Fetcher,
	of fetcher.Fetcher,
	p *eta.Parser,
	r measurements.MeasurementRepository,
) *EtaService {
	return &EtaService{
		Logger:             l.With("component", "eta-service"),
		PelletStockFetcher: pf,
		OutsideTempFetcher: of,
		Parser:             p,
		Repo:               r,
	}
}

func (s *EtaService) SyncPelletStock(ctx context.Context) error {
	raw, err := s.PelletStockFetcher.Fetch(ctx)
	if err != nil {
		return err
	}

	response, err := s.Parser.Parse(raw)
	if err != nil {
		return err
	}

	measurements := measurements.FromETA(response)

	for _, m := range measurements {
		err := s.Repo.Insert(m)
		if err != nil {
			return err
		}
	}
	s.Logger.Info("sync completed", "inserted", len(measurements))

	return nil
}

func (s *EtaService) SyncOutsideTemp(ctx context.Context) error {
	raw, err := s.OutsideTempFetcher.Fetch(ctx)
	if err != nil {
		return err
	}

	response, err := s.Parser.Parse(raw)
	if err != nil {
		return err
	}

	measurements := measurements.FromETA(response)

	for _, m := range measurements {
		err := s.Repo.Insert(m)
		if err != nil {
			return err
		}
	}
	s.Logger.Info("sync completed", "inserted", len(measurements))

	return nil
}
