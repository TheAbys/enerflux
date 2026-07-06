package eta

import "encoding/xml"

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(data []byte) (Eta, error) {
	var r Eta
	err := xml.Unmarshal(data, &r)
	return r, err
}
