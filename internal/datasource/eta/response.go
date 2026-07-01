package eta

type Response struct {
	Values []Value
}

type Value struct {
	URI         string
	StrValue    string
	Unit        string
	ScaleFactor int
	RawValue    int
}
