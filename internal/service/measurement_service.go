package service

import (
	"context"
	"log/slog"

	filter "github.com/theabys/enerflux/internal/measurements"
	"github.com/theabys/enerflux/internal/model"
	"github.com/theabys/enerflux/internal/repository"
)

type MeasurementService struct {
	Logger *slog.Logger
	Repo   repository.MeasurementRepository
}

func NewMeasurementService(logger *slog.Logger, repo repository.MeasurementRepository) *MeasurementService {
	return &MeasurementService{Logger: logger.With("component", "measurement-service"), Repo: repo}
}

func (s *MeasurementService) Create(m model.Measurement) error {
	// hier später Validierung / Business Logic
	s.Logger.Info("insert measurement", "model", m)
	return s.Repo.Insert(m)
}

func (s *MeasurementService) GetAll() ([]model.Measurement, error) {
	return s.Repo.List(5)
}

func (s *MeasurementService) GetLatest() (*model.Measurement, error) {
	return s.Repo.GetLatest()
}

func (s *MeasurementService) GetMetrics(ctx context.Context, filter filter.MeasurementFilter) ([]model.Metric, error) {
	if len(filter.Key) == 0 {
		filter.Key = "default"
	}

	measurements, err := s.Repo.QueryMeasurements(ctx, filter)
	if err != nil {
		return nil, err
	}

	metrics := []model.Metric{}
	for _, measurement := range measurements {
		metrics = append(metrics, model.Metric{
			TS:    measurement.TS,
			Value: measurement.Value,
		})
	}

	return metrics, nil
}
