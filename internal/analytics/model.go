package analytics

import "time"

type Window struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type ReadingPoint struct {
	TS        time.Time `json:"ts"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Source    string    `json:"source"`
	Note      string    `json:"note,omitempty"`
	EventType string    `json:"eventType,omitempty"`
}

type DailyValue struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type Purchase struct {
	TS           time.Time `json:"ts"`
	QuantityKg   *float64  `json:"quantityKg"`
	TotalCostEUR *float64  `json:"totalCostEur"`
	CostPerKgEUR *float64  `json:"costPerKgEur"`
	Note         string    `json:"note,omitempty"`
}

type DomainValue struct {
	Value        *float64 `json:"value"`
	Unit         string   `json:"unit"`
	CostEUR      *float64 `json:"costEur"`
	PricePerUnit *float64 `json:"pricePerUnit"`
	PriceUnit    string   `json:"priceUnit"`
	Intervals    int      `json:"intervals"`
	CoveredHours float64  `json:"coveredHours"`
	Estimate     bool     `json:"estimate"`
	Status       string   `json:"status"`
}

type WaterResponse struct {
	Window         Window         `json:"window"`
	CurrentReading *ReadingPoint  `json:"currentReading"`
	Consumed       DomainValue    `json:"consumed"`
	Readings       []ReadingPoint `json:"readings"`
	Daily          []DailyValue   `json:"daily"`
}

type ElectricityResponse struct {
	Window        Window         `json:"window"`
	CurrentPower  *ReadingPoint  `json:"currentPower"`
	Consumed      DomainValue    `json:"consumed"`
	PowerReadings []ReadingPoint `json:"powerReadings"`
	Daily         []DailyValue   `json:"daily"`
}

type HeatingResponse struct {
	Window                    Window         `json:"window"`
	CurrentStock              *ReadingPoint  `json:"currentStock"`
	Consumed                  DomainValue    `json:"consumed"`
	LifetimeConsumedKg        *float64       `json:"lifetimeConsumedKg"`
	AverageDaily30DaysKg      *float64       `json:"averageDaily30DaysKg"`
	ConsumedLast30DaysKg      *float64       `json:"consumedLast30DaysKg"`
	ConsumedLast30DaysCostEUR *float64       `json:"consumedLast30DaysCostEur"`
	Last30DaysIntervals       int            `json:"last30DaysIntervals"`
	Last30DaysCoveredHours    float64        `json:"last30DaysCoveredHours"`
	ForecastNextHourKg        *float64       `json:"forecastNextHourKg"`
	ForecastStatus            string         `json:"forecastStatus"`
	AveragePurchasePriceEUR   *float64       `json:"averagePurchasePriceEurPerKg"`
	StockReadings             []ReadingPoint `json:"stockReadings"`
	Daily                     []DailyValue   `json:"daily"`
	Purchases                 []Purchase     `json:"purchases"`
}

type OverviewResponse struct {
	GeneratedAt time.Time           `json:"generatedAt"`
	TimeZone    string              `json:"timeZone"`
	Electricity ElectricityResponse `json:"electricity"`
	Water       WaterResponse       `json:"water"`
	Heating     HeatingResponse     `json:"heating"`
	TotalCost   CostOverview        `json:"totalCost"`
}

type CostOverview struct {
	Window         Window   `json:"window"`
	TotalEUR       *float64 `json:"totalEur"`
	ElectricityEUR *float64 `json:"electricityEur"`
	WaterEUR       *float64 `json:"waterEur"`
	HeatingEUR     *float64 `json:"heatingEur"`
	Complete       bool     `json:"complete"`
	MissingDomains []string `json:"missingDomains"`
}
