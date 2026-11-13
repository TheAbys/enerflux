package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Syncer interface {
	Sync(ctx context.Context) error
}

type Scheduler struct {
	Logger          *slog.Logger
	Jobs            []Job
	intervalUpdates map[string]chan time.Duration
	triggers        map[string]chan struct{}
	mu              sync.RWMutex
}
type Job struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
}

func NewScheduler(logger *slog.Logger) *Scheduler {
	return &Scheduler{
		Logger:          logger.With("component", "scheduler"),
		intervalUpdates: make(map[string]chan time.Duration),
		triggers:        make(map[string]chan struct{}),
	}
}

func (s *Scheduler) Add(job Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Jobs = append(s.Jobs, job)
	s.intervalUpdates[job.Name] = make(chan time.Duration, 1)
	s.triggers[job.Name] = make(chan struct{}, 1)
}

func (s *Scheduler) Start(ctx context.Context) {
	for _, job := range s.Jobs {
		s.mu.RLock()
		updates := s.intervalUpdates[job.Name]
		triggers := s.triggers[job.Name]
		s.mu.RUnlock()
		go s.runJob(ctx, job, updates, triggers)
	}
}

func (s *Scheduler) UpdateInterval(name string, interval time.Duration) bool {
	if interval <= 0 {
		return false
	}

	s.mu.RLock()
	updates := s.intervalUpdates[name]
	s.mu.RUnlock()
	if updates == nil {
		return false
	}

	select {
	case updates <- interval:
	default:
		select {
		case <-updates:
		default:
		}
		updates <- interval
	}
	return true
}

func (s *Scheduler) Trigger(name string) bool {
	s.mu.RLock()
	triggers := s.triggers[name]
	s.mu.RUnlock()
	if triggers == nil {
		return false
	}

	select {
	case triggers <- struct{}{}:
	default:
	}
	return true
}

func (s *Scheduler) runJob(ctx context.Context, job Job, intervalUpdates <-chan time.Duration, triggers <-chan struct{}) {
	s.Logger.Info("job initialised", "name", job.Name, "interval", job.Interval)
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()
	run := func() {
		s.Logger.Info("job tick started", "name", job.Name)
		if err := job.Run(ctx); err != nil {
			s.Logger.Error("sync execution failed", "name", job.Name, "error", err)
		}
		s.Logger.Info("job tick finished", "name", job.Name)
	}

	for {
		select {
		case <-ticker.C:
			run()

		case <-triggers:
			run()

		case interval := <-intervalUpdates:
			ticker.Reset(interval)
			s.Logger.Info("job interval updated", "name", job.Name, "interval", interval)

		case <-ctx.Done():
			s.Logger.Info("job stopped", "name", job.Name)
			return
		}
	}
}
