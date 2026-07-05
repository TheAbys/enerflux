package config

import "time"

type Config struct {
	DatabaseURL       string
	SolarLogURL       string
	EtaPelletstockUrl string
	EtaOutsidetempUrl string
	PollInterval      time.Duration
}

func Load() *Config {
	return &Config{
		DatabaseURL:       mustGet("DATABASE_URL"),
		SolarLogURL:       mustGet("SOLARLOG_URL"),
		EtaPelletstockUrl: mustGet("ETA_PELLETSTOCK_URL"),
		EtaOutsidetempUrl: mustGet("ETA_OUTSIDETEMP_URL"),

		PollInterval: getDuration("POLL_INTERVAL", 1*time.Minute),
	}
}
