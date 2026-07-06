package eta

import "encoding/xml"

type Eta struct {
	XMLName xml.Name `xml:"eta"`
	Version string   `xml:"version,attr"`
	Values  []Value  `xml:"value"`
}

type Value struct {
	Uri           string `xml:"uri,attr"`
	StrValue      string `xml:"strValue,attr"`
	Unit          string `xml:"unit,attr"`
	DecPlaces     string `xml:"decPlaces,attr"`
	ScaleFactor   string `xml:"scaleFactor,attr"`
	AdvTextOffset string `xml:"advTextOffset,attr"`
	Text          string `xml:",chardata"`
}
