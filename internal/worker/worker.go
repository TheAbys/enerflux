package worker

import (
	"context"
	"log/slog"
	"time"
)

type Syncer interface {
	Sync(ctx context.Context) error
}

type Scheduler struct {
	Logger *slog.Logger
	Jobs   []Job
}
type Job struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
}

func NewScheduler(logger *slog.Logger) *Scheduler {
	return &Scheduler{
		Logger: logger.With("component", "scheduler"),
	}
}

func (s *Scheduler) Add(job Job) {
	s.Jobs = append(s.Jobs, job)
}

func (s *Scheduler) Start(ctx context.Context) {
	for _, job := range s.Jobs {
		go s.runJob(ctx, job)
	}
}

func (s *Scheduler) runJob(ctx context.Context, job Job) {
	s.Logger.Info("job initialised", "name", job.Name, "interval", job.Interval)
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	for range ticker.C {
		s.Logger.Info("job tick started", "name", job.Name)
		err := job.Run(ctx)
		if err != nil {
			s.Logger.Error("sync execution failed", "name", job.Name, "error", err)
		}
		s.Logger.Info("job tick finished", "name", job.Name)
	}
}
