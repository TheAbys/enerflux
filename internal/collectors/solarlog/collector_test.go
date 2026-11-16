package solarlog

import (
	"context"
	"testing"

	"github.com/theabys/enerflux/internal/logger"
	"github.com/theabys/enerflux/internal/measurements"
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

func (repo *mockRepo) InsertMany(ctx context.Context, m []measurements.Measurement) error {
	repo.called++
	repo.measurements = append(repo.measurements, m...)
	return nil
}

func (repo *mockRepo) FindLatest(ctx context.Context) (*measurements.Measurement, error) {
	return nil, nil
}

func (repo *mockRepo) Find(ctx context.Context, queryOptions measurements.QueryOptions) ([]measurements.Measurement, error) {
	return nil, nil
}

func TestSolarLogCollector_Sync(t *testing.T) {

	fetcher := &mockFetcher{}
	repo := &mockRepo{}

	parser := NewParser()

	svc := NewSolarLogCollector(logger.New(), fetcher, parser, repo)

	items, err := svc.Sync(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 15 {
		t.Fatalf("expected 15 measurements from SolarLog, got %d", len(items))
	}
	if repo.called != 0 {
		t.Fatalf("collector should return measurements for the uploader, not persist them directly; called %d times", repo.called)
	}
	if items[0].Type != "pv.power" {
		t.Fatalf("expected first measurement to be pv.power")
	}

}
