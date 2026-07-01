package solarlog

import "testing"

func TestParser_Parse(t *testing.T) {
	parser := NewParser()

	data := []byte(`{
        "801":{
            "170":{
                "100":"29.06.26 21:01:15",
                "101":137,
                "102":111,
                "105":51079,
                "106":53380,
                "109":56021703,
                "110":605
            }
        }
    }`)

	snapshot, err := parser.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.Timestamp.IsZero() {
		t.Fatal("timestamp not parsed")
	}

	if snapshot.PVPower != 137 {
		t.Fatalf("expected PVPower 137, got %d", snapshot.PVPower)
	}
	if snapshot.PVDCPower != 111 {
		t.Fatalf("expected PVDCPower 111, got %d", snapshot.PVDCPower)
	}
	if snapshot.ACVoltage != 0 {
		t.Fatalf("expected ACVoltage 0, got %d", snapshot.ACVoltage)
	}
	if snapshot.DCVoltage != 0 {
		t.Fatalf("expected DCVoltage 0, got %d", snapshot.DCVoltage)
	}

	if snapshot.YieldToday != 51079 {
		t.Fatalf("expected YieldToday 51079, got %d", snapshot.YieldToday)
	}
	if snapshot.YieldTotal != 56021703 {
		t.Fatalf("expected YieldTotal 56021703, got %d", snapshot.YieldTotal)
	}
	if snapshot.YieldYesterday != 53380 {
		t.Fatalf("expected YieldYesterday 53380, got %d", snapshot.YieldYesterday)
	}

	if snapshot.ConsumptionPower != 605 {
		t.Fatalf("expected ConsumptionPower 605, got %d", snapshot.ConsumptionPower)
	}

	if snapshot.InstalledPower != 0 {
		t.Fatalf("expected InstalledPower 0, got %d", snapshot.InstalledPower)
	}
}
