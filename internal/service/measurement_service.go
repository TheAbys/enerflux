package service

import (
	"log/slog"

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
