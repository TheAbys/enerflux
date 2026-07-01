package service

import (
	"context"
	"log/slog"

	fetcher "github.com/theabys/enerflux/internal/contract"
	"github.com/theabys/enerflux/internal/datasource/eta"
	"github.com/theabys/enerflux/internal/repository"
)

type EtaService struct {
	Logger  *slog.Logger
	Fetcher fetcher.Fetcher
	Parser  *eta.Parser
	Repo    repository.MeasurementRepository
}

func NewEtaService(
	l *slog.Logger,
	f fetcher.Fetcher,
	p *eta.Parser,
	r repository.MeasurementRepository,
) *EtaService {
	return &EtaService{
		Logger:  l.With("component", "eta-service"),
		Fetcher: f,
		Parser:  p,
		Repo:    r,
	}
}

func (s *EtaService) Sync(ctx context.Context) error {
	raw, err := s.Fetcher.Fetch(ctx)
	if err != nil {
		return err
	}

	response, err := s.Parser.Parse(raw)
	if err != nil {
		return err
	}

	measurements := mapEtaToMeasurements(response)

	for _, m := range measurements {
		err := s.Repo.Insert(m)
		if err != nil {
			return err
		}
	}
	s.Logger.Info("sync completed", "inserted", len(measurements))

	return nil
}
