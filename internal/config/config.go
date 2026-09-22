package config

import "time"

type Config struct {
	ApiUrl                     string
	ApiPort                    string
	DatabaseURL                string
	SolarLogURL                string
	SolarLogPollInterval       time.Duration
	EtaPelletstockUrl          string
	EtaPelletstockInterval     time.Duration
	EtaOutsidetempUrl          string
	EtaOutsidetempPollInterval time.Duration
	MQTTBrokerURL              string
	MQTTClientID               string
}

func Load() *Config {
	return &Config{
		ApiUrl:  mustGet("API_URL"),
		ApiPort: mustGet("API_PORT"),

		DatabaseURL:          mustGet("DATABASE_URL"),
		SolarLogURL:          mustGet("SOLARLOG_URL"),
		SolarLogPollInterval: getDuration("SOLARLOG_POLLINTERVAL", 1*time.Minute),

		EtaPelletstockUrl:          mustGet("ETA_PELLETSTOCK_URL"),
		EtaPelletstockInterval:     getDuration("ETA_PELLETSTOCK_POLLINTERVAL", 1*time.Minute),
		EtaOutsidetempUrl:          mustGet("ETA_OUTSIDETEMP_URL"),
		EtaOutsidetempPollInterval: getDuration("ETA_OUTSIDETEMP_POLLINTERVAL", 1*time.Minute),

		MQTTBrokerURL: mustGet("MQTT_BROKER_URL"),
		MQTTClientID:  mustGet("MQTT_CLIENT_ID"),
	}
}
