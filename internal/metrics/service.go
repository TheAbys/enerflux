package metrics

import (
	"context"
	"log/slog"

	"github.com/theabys/enerflux/internal/measurements"
)

type MetricService struct {
	Logger             *slog.Logger
	MeasurementService *measurements.MeasurementService
}

func NewMetricService(logger *slog.Logger, service *measurements.MeasurementService) *MetricService {
	return &MetricService{Logger: logger.With("component", "metrics-service"), MeasurementService: service}
}

func (s *MetricService) GetMetrics(ctx context.Context, filter measurements.MeasurementFilter) ([]Metric, error) {
	if len(filter.Key) == 0 {
		filter.Key = "default"
	}

	measurements, err := s.MeasurementService.GetAll(ctx, filter)

	if err != nil {
		return nil, err
	}

	metrics := []Metric{}
	for _, measurement := range measurements {
		metrics = append(metrics, Metric{
			TS:    measurement.TS,
			Value: measurement.Value,
		})
	}

	return metrics, nil
}
