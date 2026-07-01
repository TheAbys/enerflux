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

	response, err := parser.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Section801) == 0 {
		t.Fatal("section801 not parsed")
	}
}
