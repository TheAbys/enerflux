package service

import (
	"context"
	"log/slog"

	fetcher "github.com/theabys/enerflux/internal/contract"
	"github.com/theabys/enerflux/internal/datasource/solarlog"
	"github.com/theabys/enerflux/internal/mapper"
	"github.com/theabys/enerflux/internal/repository"
)

type SolarLogService struct {
	Logger  *slog.Logger
	Fetcher fetcher.Fetcher
	Parser  *solarlog.Parser
	Repo    repository.MeasurementRepository
}

func NewSolarLogService(
	l *slog.Logger,
	f fetcher.Fetcher,
	p *solarlog.Parser,
	r repository.MeasurementRepository,
) *SolarLogService {
	return &SolarLogService{
		Logger:  l.With("component", "solarlog-service"),
		Fetcher: f,
		Parser:  p,
		Repo:    r,
	}
}

func (s *SolarLogService) Sync(ctx context.Context) error {
	raw, err := s.Fetcher.Fetch(ctx)
	if err != nil {
		return err
	}

	response, err := s.Parser.Parse(raw)
	if err != nil {
		return err
	}

	measurements := mapper.FromSolarLog(response)

	for _, m := range measurements {
		err := s.Repo.Insert(m)
		if err != nil {
			return err
		}
	}
	s.Logger.Info("sync completed", "inserted", len(measurements))

	return nil
}
