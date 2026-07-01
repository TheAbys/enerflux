package eta

import (
	"encoding/xml"
	"strconv"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

type xmlRoot struct {
	Values []xmlValue `xml:"value"`
}

type xmlValue struct {
	URI         string `xml:"uri,attr"`
	StrValue    string `xml:"strValue,attr"`
	Unit        string `xml:"unit,attr"`
	ScaleFactor string `xml:"scaleFactor,attr"`
	Value       string `xml:",chardata"`
}

func (p *Parser) Parse(data []byte) (Response, error) {

	var root xmlRoot

	if err := xml.Unmarshal(data, &root); err != nil {
		return Response{}, err
	}

	resp := Response{
		Values: make([]Value, 0, len(root.Values)),
	}

	for _, v := range root.Values {

		raw, err := strconv.Atoi(v.Value)
		if err != nil {
			continue
		}

		scale := 1
		if v.ScaleFactor != "" {
			if s, err := strconv.Atoi(v.ScaleFactor); err == nil {
				scale = s
			}
		}

		resp.Values = append(resp.Values, Value{
			URI:         v.URI,
			StrValue:    v.StrValue,
			Unit:        v.Unit,
			ScaleFactor: scale,
			RawValue:    raw,
		})
	}

	return resp, nil
}
