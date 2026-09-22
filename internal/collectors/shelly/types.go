package shelly

type SwitchStatus struct {
	ID      int      `json:"id"`
	Output  bool     `json:"output"`
	APower  *float64 `json:"apower"`
	Voltage *float64 `json:"voltage"`
	Current *float64 `json:"current"`

	AEnergy *struct {
		Total float64 `json:"total"`
	} `json:"aenergy"`
}
