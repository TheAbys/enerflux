package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/theabys/enerflux/internal/agent"
	"github.com/theabys/enerflux/internal/collectors/eta"
	"github.com/theabys/enerflux/internal/collectors/shelly"
	"github.com/theabys/enerflux/internal/collectors/solarlog"
	"github.com/theabys/enerflux/internal/config"
	"github.com/theabys/enerflux/internal/database"
	"github.com/theabys/enerflux/internal/logger"
	"github.com/theabys/enerflux/internal/measurements"
	"github.com/theabys/enerflux/internal/sourceconfig"
	"github.com/theabys/enerflux/internal/worker"
)

type activeSourceSettings struct {
	mu       sync.RWMutex
	settings sourceconfig.Settings
}

func (s *activeSourceSettings) Get() sourceconfig.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *activeSourceSettings) Set(settings sourceconfig.Settings) {
	s.mu.Lock()
	s.settings = settings
	s.mu.Unlock()
}

func main() {
	log := logger.New()
	log.Info("Starting EnerFlux agent")

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
	settings, err := sourceConfigService.Get(context.Background())
	if err != nil {
		log.Error("source configuration not available", "error", err)
		return
	}
	activeSettings := &activeSourceSettings{settings: settings}
	if settings.SolarLogURL == "" {
		log.Warn("SolarLog source is not configured")
	}
	if settings.EtaPelletstockURL == "" {
		log.Warn("ETA pellet stock source is not configured")
	}
	if settings.EtaOutsidetempURL == "" {
		log.Warn("ETA outside temperature source is not configured")
	}

	measurementRepo := measurements.NewMeasurementRepo(db)
	if err := measurementRepo.EnsureSchema(context.Background()); err != nil {
		log.Error("measurement storage not available", "error", err)
		return
	}
	solarlogParser := solarlog.NewParser()
	etaParser := eta.NewParser()
	uploader := agent.NewHTTPUploader(cfg.ApiUrl)

	scheduler := worker.NewScheduler(log)
	scheduler.Add(worker.Job{
		Name:     "solarlog",
		Interval: settings.SolarLogInterval(),
		Run: func(ctx context.Context) error {
			current := activeSettings.Get()
			if current.SolarLogURL == "" {
				return nil
			}
			collector := solarlog.NewSolarLogCollector(log, solarlog.NewClient(current.SolarLogURL), solarlogParser, measurementRepo)
			items, err := collector.Sync(ctx)
			if err != nil {
				return err
			}
			return uploader.Upload(ctx, agent.ToPayloads(items))
		},
	})
	scheduler.Add(worker.Job{
		Name:     "eta-pelletstock",
		Interval: settings.EtaPelletstockInterval(),
		Run: func(ctx context.Context) error {
			current := activeSettings.Get()
			if current.EtaPelletstockURL == "" {
				return nil
			}
			collector := eta.NewEtaCollector(log, eta.NewClient(current.EtaPelletstockURL), eta.NewClient(current.EtaOutsidetempURL), etaParser, measurementRepo)
			items, err := collector.SyncPelletStock(ctx)
			if err != nil {
				return err
			}
			return uploader.Upload(ctx, agent.ToPayloads(items))
		},
	})
	scheduler.Add(worker.Job{
		Name:     "eta-outsidetemp",
		Interval: settings.EtaOutsidetempInterval(),
		Run: func(ctx context.Context) error {
			current := activeSettings.Get()
			if current.EtaOutsidetempURL == "" {
				return nil
			}
			collector := eta.NewEtaCollector(log, eta.NewClient(current.EtaPelletstockURL), eta.NewClient(current.EtaOutsidetempURL), etaParser, measurementRepo)
			items, err := collector.SyncOutsideTemp(ctx)
			if err != nil {
				return err
			}
			return uploader.Upload(ctx, agent.ToPayloads(items))
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)
	go watchSourceConfiguration(ctx, log, sourceConfigService, activeSettings, scheduler)

	shellyClient := shelly.NewClient(cfg.MQTTBrokerURL, cfg.MQTTClientID)
	if err := shellyClient.Connect(); err != nil {
		log.Error("failed to connect to mqtt broker", "error", err)
		os.Exit(1)
	}
	defer shellyClient.Close()
	shellyCollector := shelly.NewShellyCollector(log, shellyClient, measurementRepo)
	if err := shellyCollector.Start(ctx); err != nil {
		log.Error("failed to start shelly collector", "error", err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Info("Shutting down EnerFlux agent")
	cancel()
	log.Info("EnerFlux agent stopped")
}

func watchSourceConfiguration(
	ctx context.Context,
	log *slog.Logger,
	service *sourceconfig.Service,
	active *activeSourceSettings,
	scheduler *worker.Scheduler,
) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			settings, err := service.Get(ctx)
			if err != nil {
				log.Warn("failed to reload source configuration", "error", err)
				continue
			}

			previous := active.Get()
			if previous == settings {
				continue
			}

			solarLogURLChanged := previous.SolarLogURL != settings.SolarLogURL
			pelletstockURLChanged := previous.EtaPelletstockURL != settings.EtaPelletstockURL
			outsidetempURLChanged := previous.EtaOutsidetempURL != settings.EtaOutsidetempURL

			if previous.SolarLogPollInterval != settings.SolarLogPollInterval {
				scheduler.UpdateInterval("solarlog", settings.SolarLogInterval())
			}
			if previous.EtaPelletstockPollInterval != settings.EtaPelletstockPollInterval {
				scheduler.UpdateInterval("eta-pelletstock", settings.EtaPelletstockInterval())
			}
			if previous.EtaOutsidetempPollInterval != settings.EtaOutsidetempPollInterval {
				scheduler.UpdateInterval("eta-outsidetemp", settings.EtaOutsidetempInterval())
			}

			active.Set(settings)
			if solarLogURLChanged {
				scheduler.Trigger("solarlog")
			}
			if pelletstockURLChanged {
				scheduler.Trigger("eta-pelletstock")
			}
			if outsidetempURLChanged {
				scheduler.Trigger("eta-outsidetemp")
			}
			log.Info("source configuration reloaded")

		case <-ctx.Done():
			return
		}
	}
}
