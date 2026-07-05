package config

import "time"

type Config struct {
	DatabaseURL                string
	SolarLogURL                string
	SolarLogPollInterval       time.Duration
	EtaPelletstockUrl          string
	EtaPelletstockInterval     time.Duration
	EtaOutsidetempUrl          string
	EtaOutsidetempPollInterval time.Duration
}

func Load() *Config {
	return &Config{
		DatabaseURL:          mustGet("DATABASE_URL"),
		SolarLogURL:          mustGet("SOLARLOG_URL"),
		SolarLogPollInterval: getDuration("SOLARLOG_POLLINTERVAL", 1*time.Minute),

		EtaPelletstockUrl:          mustGet("ETA_PELLETSTOCK_URL"),
		EtaPelletstockInterval:     getDuration("ETA_PELLETSTOCK_POLLINTERVAL", 1*time.Minute),
		EtaOutsidetempUrl:          mustGet("ETA_OUTSIDETEMP_URL"),
		EtaOutsidetempPollInterval: getDuration("ETA_OUTSIDETEMP_POLLINTERVAL", 1*time.Minute),
	}
}
