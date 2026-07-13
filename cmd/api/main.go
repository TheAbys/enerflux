package main

import (
	"github.com/theabys/enerflux/internal/api"
	"github.com/theabys/enerflux/internal/config"
	"github.com/theabys/enerflux/internal/dashboard"
	"github.com/theabys/enerflux/internal/database"
	"github.com/theabys/enerflux/internal/health"
	"github.com/theabys/enerflux/internal/logger"
	"github.com/theabys/enerflux/internal/measurements"
	"github.com/theabys/enerflux/internal/metrics"
)

func main() {
	log := logger.New()
	log.Info("Starting EnerFlux API")

	cfg := config.Load()

	db, err := database.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Error("connection to database not possible", "error", err)
	}

	// Repo
	measurementRepo := measurements.NewMeasurementRepo(db)

	measurementService := measurements.NewMeasurementService(log, measurementRepo)

	metricService := metrics.NewMetricService(log, measurementService)

	measurementHandler := measurements.NewMeasurementHandler(measurementService)
	healthHandler := health.NewHealthHandler()
	metricHandler := metrics.NewMetricHandler(metricService)
	dashboardHandler := dashboard.NewDashboardHandler()

	// API
	router := api.NewRouter(
		measurementHandler,
		metricHandler,
		healthHandler,
		dashboardHandler,
	)

	log.Info("API started", "port", "8080")
	router.Run(":8080")
}
