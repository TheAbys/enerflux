package worker

import (
	"context"
	"log/slog"
	"time"
)

type Syncer interface {
	Sync(ctx context.Context) error
}
type Worker struct {
	Logger   *slog.Logger
	Interval time.Duration
	Syncer   Syncer
}

func NewWorker(logger *slog.Logger, interval time.Duration, syncer Syncer) *Worker {
	return &Worker{
		Logger:   logger.With("component", "worker"),
		Interval: interval,
		Syncer:   syncer,
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.Logger.Info("worker initialised", "intervall", w.Interval)
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for range ticker.C {
		w.Logger.Info("worker tick started")
		err := w.Syncer.Sync(ctx)
		if err != nil {
			w.Logger.Error("sync execution failed", "error", err)
		}
		w.Logger.Info("worker tick finished")
	}
}
