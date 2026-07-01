package config

import "time"

type Config struct {
	DatabaseURL  string
	SolarLogURL  string
	PollInterval time.Duration
}

func Load() *Config {
	return &Config{
		DatabaseURL:  mustGet("DATABASE_URL"),
		SolarLogURL:  mustGet("SOLARLOG_URL"),
		PollInterval: getDuration("POLL_INTERVAL", 1*time.Minute),
	}
}
