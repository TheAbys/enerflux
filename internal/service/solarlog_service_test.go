package service

import (
	"context"
	"testing"

	"github.com/theabys/enerflux/internal/datasource/solarlog"
	"github.com/theabys/enerflux/internal/logger"
	"github.com/theabys/enerflux/internal/measurements"
	filter "github.com/theabys/enerflux/internal/measurements"
)

type mockFetcher struct{}

func (m *mockFetcher) Fetch(ctx context.Context) ([]byte, error) {
	return []byte(`{
        "801":{
            "170":{
                "100":"29.06.26 21:01:15",
                "101":137,
                "102":111,
                "105":51079
            }
        }
    }`), nil
}

type mockRepo struct {
	called       int
	measurements []measurements.Measurement
}

func (repo *mockRepo) Insert(m measurements.Measurement) error {
	repo.called++
	repo.measurements = append(repo.measurements, m)
	return nil
}

func (repo *mockRepo) List(limit int) ([]measurements.Measurement, error) {
	return nil, nil
}

func (repo *mockRepo) GetLatest() (*measurements.Measurement, error) {
	return nil, nil
}

func (repo *mockRepo) QueryMeasurements(ctx context.Context, filter filter.MeasurementFilter) ([]measurements.Measurement, error) {
	return nil, nil
}

func TestSolarLogService_Sync(t *testing.T) {

	fetcher := &mockFetcher{}
	repo := &mockRepo{}

	parser := solarlog.NewParser()

	svc := NewSolarLogService(logger.New(), fetcher, parser, repo)

	err := svc.Sync(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.called != 15 {
		t.Fatalf("expected repo to be called once, got %d", repo.called)
	}
	if len(repo.measurements) == 15 && repo.measurements[0].Type != "pv.power" {
		t.Fatalf("expected first measurement to be pv.power")
	}

}
