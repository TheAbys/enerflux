package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/theabys/enerflux/internal/agent"
	"github.com/theabys/enerflux/internal/collectors/eta"
	"github.com/theabys/enerflux/internal/collectors/solarlog"
	"github.com/theabys/enerflux/internal/config"
	"github.com/theabys/enerflux/internal/database"
	"github.com/theabys/enerflux/internal/logger"
	"github.com/theabys/enerflux/internal/measurements"
	"github.com/theabys/enerflux/internal/worker"
)

func main() {
	log := logger.New()
	log.Info("Starting EnerFlux agent")

	cfg := config.Load()

	db, err := database.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Error("connection to database not possible", "error", err)
	}

	// Repo
	measurementRepo := measurements.NewMeasurementRepo(db)

	// SolarLog Client
	solarlogClient := solarlog.NewClient(cfg.SolarLogURL)

	// Parser
	solarlogParser := solarlog.NewParser()

	// Service
	solarLogCollector := solarlog.NewSolarLogCollector(log, solarlogClient, solarlogParser, measurementRepo)

	//Parser
	etaParser := eta.NewParser()

	// Service
	etaCollector := eta.NewEtaCollector(log, eta.NewClient(cfg.EtaPelletstockUrl), eta.NewClient(cfg.EtaOutsidetempUrl), etaParser, measurementRepo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	uploader := agent.NewHTTPUploader("http://localhost:8080/")

	// Scheduler
	scheduler := worker.NewScheduler(log)
	scheduler.Add(worker.Job{
		Name:     "solarlog",
		Interval: cfg.SolarLogPollInterval,
		Run: func(ctx context.Context) error {
			items, err := solarLogCollector.Sync(ctx)
			if err != nil {
				return err
			}

			return uploader.Upload(ctx, agent.ToPayloads(items))
		},
	})
	scheduler.Add(worker.Job{
		Name:     "eta-pelletstock",
		Interval: cfg.EtaPelletstockInterval,
		Run: func(ctx context.Context) error {
			items, err := etaCollector.SyncPelletStock(ctx)
			if err != nil {
				return err
			}

			return uploader.Upload(ctx, agent.ToPayloads(items))
		},
	})
	scheduler.Add(worker.Job{
		Name:     "eta-outsidetemp",
		Interval: cfg.EtaOutsidetempPollInterval,
		Run: func(ctx context.Context) error {
			items, err := etaCollector.SyncOutsideTemp(ctx)
			if err != nil {
				return err
			}

			return uploader.Upload(ctx, agent.ToPayloads(items))
		},
	})
	scheduler.Start(ctx)

	// Wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(
		sig,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-sig

	log.Info("Shutting down EnerFlux agent")

	// Trigger graceful shutdown
	cancel()

	log.Info("EnerFlux agent stopped")
}
