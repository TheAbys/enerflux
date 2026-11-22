package shelly

import (
	"context"
	"log/slog"
	"time"

	"github.com/theabys/enerflux/internal/measurements"
	"github.com/theabys/enerflux/internal/units"
)

type ShellyCollector struct {
	Logger *slog.Logger
	Client Client
	Repo   measurements.MeasurementRepository
}

func NewShellyCollector(
	l *slog.Logger,
	c Client,
	r measurements.MeasurementRepository,
) *ShellyCollector {
	return &ShellyCollector{
		Logger: l.With("component", "shelly-collector"),
		Client: c,
		Repo:   r,
	}
}

func (c *ShellyCollector) Start(ctx context.Context) error {
	c.Logger.Info("ShellyCollector started")
	return c.Client.SubscribeSwitchStatus(
		func(deviceID string, status SwitchStatus) {
			measurements := FromSwitchStatus(
				status,
				deviceID,
			)

			if len(measurements) == 0 {
				return
			}
			c.Logger.Debug("received measurements",
				"payload", measurements,
			)
			if err := c.Repo.InsertMany(ctx, measurements); err != nil {
				c.Logger.Error(
					"failed to store shelly measurements",
					"device", deviceID,
					"error", err,
				)
			}
		},
	)
}

func FromSwitchStatus(status SwitchStatus, deviceID string) []measurements.Measurement {
	ts := time.Now()

	var result []measurements.Measurement

	if status.APower != nil {
		result = append(result, measurements.Measurement{
			TS:     ts,
			Type:   "device.power",
			Value:  *status.APower,
			Unit:   units.Watt,
			Source: deviceID,
		})
	}

	if status.AEnergy != nil {
		result = append(result, measurements.Measurement{
			TS:     ts,
			Type:   "device.energy.total",
			Value:  status.AEnergy.Total,
			Unit:   "Wh",
			Source: deviceID,
		})
	}

	return result
}
