package domainconfig

import (
	"fmt"
	"math"
	"time"
	_ "time/tzdata"
)

type Settings struct {
	TimeZone               string   `json:"timeZone"`
	ElectricityPricePerKWh *float64 `json:"electricityPricePerKWh"`
	WaterPricePerM3        *float64 `json:"waterPricePerM3"`
	WastewaterPricePerM3   *float64 `json:"wastewaterPricePerM3"`
	PelletTankCapacityKg   *float64 `json:"pelletTankCapacityKg"`
}

func DefaultSettings() Settings {
	return Settings{TimeZone: "Europe/Berlin"}
}

func (s *Settings) Validate() error {
	if s.TimeZone == "" {
		s.TimeZone = "Europe/Berlin"
	}
	if _, err := time.LoadLocation(s.TimeZone); err != nil {
		return fmt.Errorf("time zone must be a valid IANA time zone")
	}
	for _, field := range []struct {
		name  string
		value *float64
	}{
		{"electricity price per kWh", s.ElectricityPricePerKWh},
		{"water price per m³", s.WaterPricePerM3},
		{"wastewater price per m³", s.WastewaterPricePerM3},
	} {
		if field.value != nil && (math.IsNaN(*field.value) || math.IsInf(*field.value, 0) || *field.value < 0) {
			return fmt.Errorf("%s cannot be negative", field.name)
		}
	}
	if s.PelletTankCapacityKg != nil && (math.IsNaN(*s.PelletTankCapacityKg) || math.IsInf(*s.PelletTankCapacityKg, 0) || *s.PelletTankCapacityKg <= 0) {
		return fmt.Errorf("pellet tank capacity must be greater than zero")
	}
	return nil
}
