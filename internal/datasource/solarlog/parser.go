package solarlog

import (
	"encoding/json"
	"strconv"
	"time"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(data []byte) (Snapshot, error) {

	var raw map[string]any
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return Snapshot{}, err
	}

	section801 := raw["801"].(map[string]any)
	section170 := section801["170"].(map[string]any)

	snap := Snapshot{}

	snap.Timestamp = parseTime(section170["100"])
	snap.PVPower = parseInt(section170["101"])
	snap.PVDCPower = parseInt(section170["102"])

	snap.ACVoltage = parseInt(section170["103"])
	snap.DCVoltage = parseInt(section170["104"])

	snap.YieldToday = parseInt(section170["105"])
	snap.YieldYesterday = parseInt(section170["106"])
	snap.YieldMonth = parseInt(section170["107"])
	snap.YieldYear = parseInt(section170["108"])
	snap.YieldTotal = parseInt(section170["109"])

	snap.ConsumptionPower = parseInt(section170["110"])

	return snap, nil
}

func parseInt(v any) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		return 0
	}
}

func parseTime(v any) time.Time {
	s, ok := v.(string)
	if !ok {
		return time.Time{}
	}

	t, err := time.Parse("02.01.06 15:04:05", s)
	if err != nil {
		return time.Time{}
	}

	return t
}
