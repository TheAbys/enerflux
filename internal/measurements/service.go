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

func (s *MeasurementService) Insert(ctx context.Context, measurement Measurement) error {
	// hier später Validierung / Business Logic
	s.Logger.Info("insert measurement", "model", measurement)
	return s.InsertMany(ctx, []Measurement{measurement})
}

func (s *MeasurementService) InsertMany(ctx context.Context, measurements []Measurement) error {
	s.Logger.Info("insert many measurements", "model", measurements)
	return s.Repo.InsertMany(ctx, measurements)
}

func (s *MeasurementService) Find(ctx context.Context, options QueryOptions) ([]Measurement, error) {
	if options.Limit == 0 {
		options.Limit = 100
	}

	return s.Repo.Find(ctx, options)
}

func (s *MeasurementService) FindLatest(ctx context.Context, filter Filter) (*Measurement, error) {

	measurements, err := s.Repo.Find(ctx, QueryOptions{
		Filter: filter,
		Sort: []Sort{
			{
				Field:     SortByTimestamp,
				Direction: SortDescending,
			},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, err
	}

	if len(measurements) == 0 {
		return nil, nil
	}

	return &measurements[0], nil
}
