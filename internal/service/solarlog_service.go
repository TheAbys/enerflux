package service

import (
	"context"
	"log/slog"

	"github.com/theabys/enerflux/internal/datasource/solarlog"
	"github.com/theabys/enerflux/internal/repository"
)

type SolarLogService struct {
	Logger *slog.Logger
	Client *solarlog.Client
	Parser *solarlog.Parser
	Repo   *repository.MeasurementRepo
}

func NewSolarLogService(
	l *slog.Logger,
	c *solarlog.Client,
	p *solarlog.Parser,
	r *repository.MeasurementRepo,
) *SolarLogService {
	return &SolarLogService{
		Logger: l.With("component", "solarlog-service"),
		Client: c,
		Parser: p,
		Repo:   r,
	}
}

func (s *SolarLogService) Sync(ctx context.Context) error {
	raw, err := s.Client.Fetch(ctx)
	if err != nil {
		return err
	}

	snapshot, err := s.Parser.Parse(raw)
	if err != nil {
		return err
	}

	measurements := MapSnapshotToMeasurements(snapshot)

	for _, m := range measurements {
		err := s.Repo.InsertMeasurement(m)
		if err != nil {
			return err
		}
	}
	s.Logger.Info("sync completed", "inserted", len(measurements))

	return nil
}
