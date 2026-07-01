package main

import (
	"context"
	"time"

	"github.com/theabys/enerflux/internal/api"
	"github.com/theabys/enerflux/internal/config"
	"github.com/theabys/enerflux/internal/datasource/eta"
	"github.com/theabys/enerflux/internal/datasource/solarlog"
	"github.com/theabys/enerflux/internal/logger"
	"github.com/theabys/enerflux/internal/repository"
	"github.com/theabys/enerflux/internal/service"
	"github.com/theabys/enerflux/internal/worker"
)

func main() {
	log := logger.New()
	log.Info("starting EnerFlux")

	cfg := config.Load()

	db, err := repository.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Error("connection to database not possible", "error", err)
	}

	// Repo
	measurementRepo := repository.NewMeasurementRepo(db)

	// SolarLog Client
	solarlogClient := solarlog.NewClient(cfg.SolarLogURL)

	// Parser
	solarlogParser := solarlog.NewParser()

	// Service
	solarLogService := service.NewSolarLogService(log, solarlogClient, solarlogParser, measurementRepo)

	// Worker
	solarLogWorker := worker.NewWorker(log, cfg.PollInterval, solarLogService)
	go solarLogWorker.Start(context.Background())

	// Eta Client
	etaClient := eta.NewClient("http://192.168.178.27:8080/user/var/40/10201/0/0/12015")

	//Parser
	etaParser := eta.NewParser()

	// Service
	etaService := service.NewEtaService(log, etaClient, etaParser, measurementRepo)

	// Worker
	etaWorker := worker.NewWorker(log, cfg.PollInterval+time.Second, etaService)
	go etaWorker.Start(context.Background())

	measurementService := service.NewMeasurementService(log, measurementRepo)
	measurementHandler := api.NewMeasurementHandler(measurementService)
	healthHandler := api.NewHealthHandler()

	// API
	router := api.NewRouter(
		measurementHandler,
		healthHandler,
	)

	log.Info("API started", "port", "8080")
	router.Run(":8080")
}
