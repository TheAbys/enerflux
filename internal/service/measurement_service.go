package service

import (
	"github.com/theabys/enerflux/internal/model"
	"github.com/theabys/enerflux/internal/repository"
)

type MeasurementService struct {
	Repo *repository.MeasurementRepo
}

func NewMeasurementService(repo *repository.MeasurementRepo) *MeasurementService {
	return &MeasurementService{Repo: repo}
}

func (s *MeasurementService) Create(m model.Measurement) error {
	// hier später Validierung / Business Logic
	return s.Repo.InsertMeasurement(m)
}

func (s *MeasurementService) GetAll() ([]model.Measurement, error) {
	return s.Repo.List(5)
}

func (s *MeasurementService) GetLatest() (*model.Measurement, error) {
	return s.Repo.GetLatest()
}
