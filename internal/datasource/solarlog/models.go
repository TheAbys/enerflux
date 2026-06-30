package solarlog

import "time"

type Snapshot struct {
	Timestamp time.Time

	PVPower   int
	PVDCPower int

	ACVoltage int
	DCVoltage int

	YieldToday     int
	YieldYesterday int
	YieldMonth     int
	YieldYear      int
	YieldTotal     int

	ConsumptionPower     int
	ConsumptionToday     int
	ConsumptionYesterday int
	ConsumptionMonth     int
	ConsumptionYear      int
	ConsumptionTotal     int

	InstalledPower int
}
