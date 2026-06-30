package config

import "time"

type Config struct {
	DatabaseURL  string
	SolarLogURL  string
	PollInterval time.Duration
}

func Load() *Config {
	return &Config{
		DatabaseURL:  "postgres://enerflux:enerflux@localhost:5432/enerflux",
		SolarLogURL:  "http://solar-log/getjp",
		PollInterval: 1 * time.Minute,
	}
}
