package sourceconfig

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/theabys/enerflux/internal/config"
)

type Settings struct {
	SolarLogURL                string `json:"solarLogURL"`
	SolarLogPollInterval       string `json:"solarLogPollInterval"`
	EtaPelletstockURL          string `json:"etaPelletstockURL"`
	EtaPelletstockPollInterval string `json:"etaPelletstockPollInterval"`
	EtaOutsidetempURL          string `json:"etaOutsidetempURL"`
	EtaOutsidetempPollInterval string `json:"etaOutsidetempPollInterval"`
}

func FromConfig(cfg *config.Config) Settings {
	return Settings{
		SolarLogURL:                cfg.SolarLogURL,
		SolarLogPollInterval:       cfg.SolarLogPollInterval.String(),
		EtaPelletstockURL:          cfg.EtaPelletstockUrl,
		EtaPelletstockPollInterval: cfg.EtaPelletstockInterval.String(),
		EtaOutsidetempURL:          cfg.EtaOutsidetempUrl,
		EtaOutsidetempPollInterval: cfg.EtaOutsidetempPollInterval.String(),
	}
}

func (s *Settings) NormalizeForRead() error {
	s.SolarLogURL = strings.TrimSpace(s.SolarLogURL)
	s.EtaPelletstockURL = strings.TrimSpace(s.EtaPelletstockURL)
	s.EtaOutsidetempURL = strings.TrimSpace(s.EtaOutsidetempURL)

	var err error
	if s.SolarLogPollInterval, err = normalizeInterval("SolarLog", s.SolarLogPollInterval); err != nil {
		return err
	}
	if s.EtaPelletstockPollInterval, err = normalizeInterval("ETA pellet stock", s.EtaPelletstockPollInterval); err != nil {
		return err
	}
	if s.EtaOutsidetempPollInterval, err = normalizeInterval("ETA outside temperature", s.EtaOutsidetempPollInterval); err != nil {
		return err
	}
	return nil
}

func (s *Settings) NormalizeAndValidate() error {
	if err := s.NormalizeForRead(); err != nil {
		return err
	}
	if err := validateRequiredURL("SolarLog", s.SolarLogURL); err != nil {
		return err
	}
	if err := validateRequiredURL("ETA pellet stock", s.EtaPelletstockURL); err != nil {
		return err
	}
	if err := validateRequiredURL("ETA outside temperature", s.EtaOutsidetempURL); err != nil {
		return err
	}
	return nil
}

func (s Settings) SolarLogInterval() time.Duration {
	return mustParseInterval(s.SolarLogPollInterval)
}

func (s Settings) EtaPelletstockInterval() time.Duration {
	return mustParseInterval(s.EtaPelletstockPollInterval)
}

func (s Settings) EtaOutsidetempInterval() time.Duration {
	return mustParseInterval(s.EtaOutsidetempPollInterval)
}

func validateURL(label, value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s URL must be an absolute HTTP or HTTPS URL", label)
	}
	return nil
}

func validateRequiredURL(label, value string) error {
	if value == "" {
		return fmt.Errorf("%s URL is required", label)
	}
	return validateURL(label, value)
}

func normalizeInterval(label, value string) (string, error) {
	interval, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || interval < time.Second || interval > 24*time.Hour {
		return "", fmt.Errorf("%s poll interval must be between 1s and 24h (for example 30s)", label)
	}
	return interval.String(), nil
}

func mustParseInterval(value string) time.Duration {
	interval, _ := time.ParseDuration(value)
	return interval
}
