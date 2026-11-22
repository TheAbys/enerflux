package main

import (
	"context"

	"github.com/theabys/enerflux/internal/analytics"
	"github.com/theabys/enerflux/internal/api"
	"github.com/theabys/enerflux/internal/config"
	"github.com/theabys/enerflux/internal/dashboard"
	"github.com/theabys/enerflux/internal/database"
	"github.com/theabys/enerflux/internal/domainconfig"
	"github.com/theabys/enerflux/internal/health"
	"github.com/theabys/enerflux/internal/logger"
	"github.com/theabys/enerflux/internal/measurements"
	"github.com/theabys/enerflux/internal/metrics"
	"github.com/theabys/enerflux/internal/sourceconfig"
)

func main() {
	log := logger.New()
	log.Info("Starting EnerFlux API")

	cfg := config.Load()

	db, err := database.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Error("connection to database not possible", "error", err)
		return
	}

	sourceConfigRepository := sourceconfig.NewRepository(db)
	if err := sourceConfigRepository.EnsureSchema(context.Background()); err != nil {
		log.Error("source configuration storage not available", "error", err)
		return
	}
	sourceConfigService := sourceconfig.NewService(sourceConfigRepository, sourceconfig.FromConfig(cfg))
	domainConfigRepository := domainconfig.NewRepository(db)
	if err := domainConfigRepository.EnsureSchema(context.Background()); err != nil {
		log.Error("domain configuration storage not available", "error", err)
		return
	}
	domainConfigService := domainconfig.NewService(domainConfigRepository)

	// Repo
	measurementRepo := measurements.NewMeasurementRepo(db)
	if err := measurementRepo.EnsureSchema(context.Background()); err != nil {
		log.Error("measurement storage not available", "error", err)
		return
	}

	measurementService := measurements.NewMeasurementService(log, measurementRepo)

	metricService := metrics.NewMetricService(log, measurementService)
	analyticsService := analytics.NewService(measurementRepo, domainConfigService)

	measurementHandler := measurements.NewMeasurementHandler(measurementService)
	healthHandler := health.NewHealthHandler(db)
	metricHandler := metrics.NewMetricHandler(metricService)
	sourceConfigHandler := sourceconfig.NewHandler(sourceConfigService)
	domainConfigHandler := domainconfig.NewHandler(domainConfigService)
	analyticsHandler := analytics.NewHandler(analyticsService, cfg.HomeAssistantReadToken)
	dashboardHandler := dashboard.NewDashboardHandler()

	// API
	router := api.NewRouter(
		measurementHandler,
		metricHandler,
		sourceConfigHandler,
		domainConfigHandler,
		analyticsHandler,
		healthHandler,
		dashboardHandler,
	)

	log.Info("API started", "port", cfg.ApiPort)
	router.Run(cfg.ApiPort)
}
