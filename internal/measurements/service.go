package measurements

import (
	"context"
	"log/slog"
)

type MeasurementService struct {
	Logger *slog.Logger
	Repo   MeasurementRepository
}

func NewMeasurementService(logger *slog.Logger, repo MeasurementRepository) *MeasurementService {
	return &MeasurementService{Logger: logger.With("component", "measurement-service"), Repo: repo}
}

func (s *MeasurementService) Create(m Measurement) error {
	// hier später Validierung / Business Logic
	s.Logger.Info("insert measurement", "model", m)
	return s.Repo.Insert(m)
}

func (s *MeasurementService) GetAll(ctx context.Context, filter MeasurementFilter) ([]Measurement, error) {
	if len(filter.Key) == 0 {
		filter.Key = "default"
	}

	return s.Repo.QueryMeasurements(ctx, filter)
}

func (s *MeasurementService) GetLatest() (*Measurement, error) {
	return s.Repo.GetLatest()
}
