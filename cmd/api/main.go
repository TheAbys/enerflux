package main

import (
	"context"
	"log"

	"github.com/theabys/enerflux/internal/api"
	"github.com/theabys/enerflux/internal/config"
	"github.com/theabys/enerflux/internal/datasource/solarlog"
	"github.com/theabys/enerflux/internal/repository"
	"github.com/theabys/enerflux/internal/service"
	"github.com/theabys/enerflux/internal/worker"
)

func main() {

	cfg := config.Load()

	db, err := repository.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	// Repo
	measurementRepo := repository.NewMeasurementRepo(db)

	// SolarLog Client
	solarlogClient := solarlog.NewClient(cfg.SolarLogURL)

	// Parser
	solarlogParser := solarlog.NewParser()

	// Service
	solarLogService := service.NewSolarLogService(solarlogClient, solarlogParser, measurementRepo)

	// Worker
	solarLogWorker := worker.NewWorker(cfg.PollInterval, solarLogService)

	go solarLogWorker.Start(context.Background())

	measurementService := service.NewMeasurementService(measurementRepo)
	measurementHandler := api.NewMeasurementHandler(measurementService)
	healthHandler := api.NewHealthHandler()

	// API
	router := api.NewRouter(
		measurementHandler,
		healthHandler,
	)

	router.Run(":8080")
}
