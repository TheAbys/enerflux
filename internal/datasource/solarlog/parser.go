package solarlog

import (
	"encoding/json"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(data []byte) (Response, error) {

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Response{}, err
	}

	section801, ok := raw["801"].(map[string]any)
	if !ok {
		return Response{}, nil
	}

	return Response{
		Section801: section801,
	}, nil
}
