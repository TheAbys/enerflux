package config

import "time"

type Config struct {
	DatabaseURL  string
	SolarLogURL  string
	PollInterval time.Duration
}

func Load() *Config {
	return &Config{
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://enerflux:enerflux@localhost:5432/enerflux"),
		SolarLogURL:  getEnv("SOLARLOG_URL", "http://solar-log/getjp"),
		PollInterval: getDuration("POLL_INTERVAL", 1*time.Minute),
	}
}
