package worker

import (
	"context"
	"log"
	"time"
)

type Syncer interface {
	Sync(ctx context.Context) error
}
type Worker struct {
	Interval time.Duration
	Syncer   Syncer
}

func NewWorker(interval time.Duration, syncer Syncer) *Worker {
	return &Worker{
		Interval: interval,
		Syncer:   syncer,
	}
}

func (w *Worker) Start(ctx context.Context) {

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for range ticker.C {
		err := w.Syncer.Sync(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}

}
