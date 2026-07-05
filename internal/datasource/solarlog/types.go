package solarlog

type Response struct {
	Section801 Section801 `json:"801"`
}

type Section801 struct {
	Section170 Section170 `json:"170"`
}

type Section170 struct {
	Time string `json:"100"`

	PVPower   int `json:"101"`
	PVDCPower int `json:"102"`

	ACVoltage int `json:"103"`
	DCVoltage int `json:"104"`

	YieldToday     int `json:"105"`
	YieldYesterday int `json:"106"`
	YieldMonth     int `json:"107"`
	YieldYear      int `json:"108"`
	YieldTotal     int `json:"109"`

	ConsumptionPower int `json:"110"`
}
