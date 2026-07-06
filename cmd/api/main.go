package main

import (
	"context"

	"github.com/theabys/enerflux/internal/api"
	"github.com/theabys/enerflux/internal/config"
	"github.com/theabys/enerflux/internal/database"
	"github.com/theabys/enerflux/internal/datasource/eta"
	"github.com/theabys/enerflux/internal/datasource/solarlog"
	"github.com/theabys/enerflux/internal/health"
	"github.com/theabys/enerflux/internal/logger"
	"github.com/theabys/enerflux/internal/measurements"
	"github.com/theabys/enerflux/internal/metrics"
	"github.com/theabys/enerflux/internal/service"
	"github.com/theabys/enerflux/internal/worker"
)

func main() {
	log := logger.New()
	log.Info("starting EnerFlux")

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
	solarLogService := service.NewSolarLogService(log, solarlogClient, solarlogParser, measurementRepo)

	//Parser
	etaParser := eta.NewParser()

	// Service
	etaService := service.NewEtaService(log, eta.NewClient(cfg.EtaPelletstockUrl), eta.NewClient(cfg.EtaOutsidetempUrl), etaParser, measurementRepo)

	// Scheduler
	scheduler := worker.NewScheduler(log)
	scheduler.Add(worker.Job{
		Name:     "solarlog",
		Interval: cfg.SolarLogPollInterval,
		Run:      solarLogService.Sync,
	})
	scheduler.Add(worker.Job{
		Name:     "eta-pelletstock",
		Interval: cfg.EtaPelletstockInterval,
		Run:      etaService.SyncPelletStock,
	})
	scheduler.Add(worker.Job{
		Name:     "eta-outsidetemp",
		Interval: cfg.EtaOutsidetempPollInterval,
		Run:      etaService.SyncOutsideTemp,
	})
	scheduler.Start(context.Background())

	measurementService := measurements.NewMeasurementService(log, measurementRepo)

	metricService := metrics.NewMetricService(log, measurementService)

	measurementHandler := measurements.NewMeasurementHandler(measurementService)
	healthHandler := health.NewHealthHandler()
	metricHandler := metrics.NewMetricHandler(metricService)

	// API
	router := api.NewRouter(
		measurementHandler,
		metricHandler,
		healthHandler,
	)

	log.Info("API started", "port", "8080")
	router.Run(":8080")
}
