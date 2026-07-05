package solarlog

import "testing"

func TestParser_Parse(t *testing.T) {
	parser := NewParser()

	data := []byte(`{"801":{"170":{"100":"29.06.26 20:57:15","101":178,"102":156,"103":0,"104":350,"105":51068,"106":53380,"107":1222310,"108":5235078,"109":56021692,"110":577,"111":11284,"112":16805,"113":363237,"114":2240951,"115":20285012,"116":9000}}}`)

	response, err := parser.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Section801.Section170.PVPower != 178 {
		t.Fatal("PVPower not parsed")
	}
	if response.Section801.Section170.YieldTotal != 56021692 {
		t.Fatal("YieldTotal not parsed")
	}
}
