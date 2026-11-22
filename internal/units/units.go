// Package units contains canonical unit symbols shared by collection and analytics code.
package units

const (
	// CubicMeter is the canonical display symbol for water volume.
	CubicMeter = "m³"
	// CubicMeterASCII is the ASCII spelling accepted for imported measurements.
	CubicMeterASCII = "m3"
	// Kilogram is the canonical display symbol for pellet mass.
	Kilogram = "kg"
	// KilowattHour is the canonical display symbol for energy.
	KilowattHour = "kWh"
	// Watt is the canonical display symbol for electrical power.
	Watt = "W"
	// Kilowatt is the canonical display symbol for larger power values.
	Kilowatt = "kW"
	// Euro is the canonical currency code.
	Euro = "EUR"
	// EuroSymbol is the common symbol accepted for currency measurements.
	EuroSymbol = "€"

	// EuroPerCubicMeter is the display unit for water prices.
	EuroPerCubicMeter = Euro + "/" + CubicMeter
	// EuroPerKilogram is the display unit for pellet prices.
	EuroPerKilogram = Euro + "/" + Kilogram
	// EuroPerKilowattHour is the display unit for electricity prices.
	EuroPerKilowattHour = Euro + "/" + KilowattHour
)
