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

func (s *MetricService) GetMetrics(ctx context.Context, options QueryOptions) ([]Metric, error) {
	measurements, err := s.MeasurementService.Find(
		ctx,
		measurements.QueryOptions{
			Filter: measurements.Filter{
				Types: []string{options.Filter.Key},
			},
			Sort: []measurements.Sort{
				{
					Field:     measurements.SortByTimestamp,
					Direction: measurements.SortDescending,
				},
			},
			Limit:  options.Limit,
			Offset: options.Offset,
		},
	)

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
