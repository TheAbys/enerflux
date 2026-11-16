package analytics

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/theabys/enerflux/internal/domainconfig"
	"github.com/theabys/enerflux/internal/measurements"
)

const (
	waterType            = "water.meter.reading"
	gridPowerType        = "grid.consumption.power"
	pelletStockType      = "heating.pellet.stock"
	purchaseQuantityType = "heating.pellet.purchase.quantity"
	purchaseCostType     = "heating.pellet.purchase.cost"
	etaSource            = "eta"
	manualSource         = "manual"
	maxPowerGap          = 15 * time.Minute
	maxForecastAge       = 20 * time.Minute
	forecastWindow       = 6 * time.Hour
	minimumForecastAge   = 2 * time.Hour
)

type MeasurementStore interface {
	Find(context.Context, measurements.QueryOptions) ([]measurements.Measurement, error)
}

type Service struct {
	Measurements  MeasurementStore
	Configuration *domainconfig.Service
}

func NewService(store MeasurementStore, configuration *domainconfig.Service) *Service {
	return &Service{Measurements: store, Configuration: configuration}
}

func (s *Service) Overview(ctx context.Context, now time.Time) (OverviewResponse, error) {
	settings, location, err := s.settings(ctx)
	if err != nil {
		return OverviewResponse{}, err
	}
	now = now.In(location)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	costFrom := now.AddDate(0, 0, -30)
	electricityFrom := dayStart

	electricity, err := s.Electricity(ctx, electricityFrom, now, settings, location)
	if err != nil {
		return OverviewResponse{}, err
	}
	electricity30, err := s.Electricity(ctx, costFrom, now, settings, location)
	if err != nil {
		return OverviewResponse{}, err
	}
	water, err := s.Water(ctx, costFrom, now, settings, location)
	if err != nil {
		return OverviewResponse{}, err
	}
	heating, err := s.Heating(ctx, costFrom, now, settings, location)
	if err != nil {
		return OverviewResponse{}, err
	}
	costs := costOverview(costFrom, now, electricity30.Consumed.CostEUR, water.Consumed.CostEUR, heating.ConsumedLast30DaysCostEUR)
	return OverviewResponse{GeneratedAt: now, TimeZone: location.String(), Electricity: electricity, Water: water, Heating: heating, TotalCost: costs}, nil
}

func costOverview(from, to time.Time, electricity, water, heating *float64) CostOverview {
	result := CostOverview{
		Window:         Window{From: from, To: to},
		ElectricityEUR: electricity,
		WaterEUR:       water,
		HeatingEUR:     heating,
		Complete:       true,
		MissingDomains: make([]string, 0),
	}
	values := []struct {
		name  string
		value *float64
	}{{"electricity", electricity}, {"water", water}, {"pellet heating", heating}}
	total := 0.0
	for _, item := range values {
		if item.value == nil {
			result.Complete = false
			result.MissingDomains = append(result.MissingDomains, item.name)
			continue
		}
		total += *item.value
	}
	if result.Complete {
		result.TotalEUR = valuePointer(total)
	}
	return result
}

func (s *Service) GetWater(ctx context.Context, from, to time.Time) (WaterResponse, error) {
	settings, location, err := s.settings(ctx)
	if err != nil {
		return WaterResponse{}, err
	}
	return s.Water(ctx, from.In(location), to.In(location), settings, location)
}

func (s *Service) GetElectricity(ctx context.Context, from, to time.Time) (ElectricityResponse, error) {
	settings, location, err := s.settings(ctx)
	if err != nil {
		return ElectricityResponse{}, err
	}
	return s.Electricity(ctx, from.In(location), to.In(location), settings, location)
}

func (s *Service) GetHeating(ctx context.Context, from, to time.Time) (HeatingResponse, error) {
	settings, location, err := s.settings(ctx)
	if err != nil {
		return HeatingResponse{}, err
	}
	return s.Heating(ctx, from.In(location), to.In(location), settings, location)
}

func (s *Service) settings(ctx context.Context) (domainconfig.Settings, *time.Location, error) {
	settings, err := s.Configuration.Get(ctx)
	if err != nil {
		return domainconfig.Settings{}, nil, err
	}
	location, err := time.LoadLocation(settings.TimeZone)
	if err != nil {
		return domainconfig.Settings{}, nil, fmt.Errorf("load configured time zone: %w", err)
	}
	return settings, location, nil
}

func (s *Service) query(ctx context.Context, types []string, from, to *time.Time, sources []string) ([]measurements.Measurement, error) {
	options := measurements.QueryOptions{
		Filter: measurements.Filter{Types: types, Sources: sources, From: from, To: to},
		Sort:   []measurements.Sort{{Field: measurements.SortByTimestamp, Direction: measurements.SortAscending}},
	}
	rows, err := s.Measurements.Find(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("load analytics measurements: %w", err)
	}
	return rows, nil
}

func (s *Service) Electricity(ctx context.Context, from, to time.Time, settings domainconfig.Settings, location *time.Location) (ElectricityResponse, error) {
	queryFrom := from.Add(-maxPowerGap)
	rows, err := s.query(ctx, []string{gridPowerType}, &queryFrom, &to, nil)
	if err != nil {
		return ElectricityResponse{}, err
	}
	points := toReadings(rows, location)
	result := integratePower(points, from, to, location)
	consumed := DomainValue{
		Value:        valuePointer(result.kWh),
		Unit:         "kWh",
		PricePerUnit: settings.ElectricityPricePerKWh,
		PriceUnit:    "EUR/kWh",
		Intervals:    result.intervals,
		CoveredHours: result.covered.Hours(),
		Estimate:     true,
		Status:       "Estimated from measured grid power; gaps over 15 minutes are omitted.",
	}
	if result.intervals == 0 {
		consumed.Value = nil
		consumed.Status = "No complete grid-power intervals in this period."
	} else if settings.ElectricityPricePerKWh != nil {
		consumed.CostEUR = valuePointer(result.kWh * *settings.ElectricityPricePerKWh)
	}
	daily := powerDaily(points, from, to, location)
	windowPoints := filterReadings(points, from, to)
	return ElectricityResponse{
		Window:        Window{From: from, To: to},
		CurrentPower:  latestReading(points, to),
		Consumed:      consumed,
		PowerReadings: windowPoints,
		Daily:         daily,
	}, nil
}

func (s *Service) Water(ctx context.Context, from, to time.Time, settings domainconfig.Settings, location *time.Location) (WaterResponse, error) {
	rows, err := s.query(ctx, []string{waterType}, nil, &to, nil)
	if err != nil {
		return WaterResponse{}, err
	}
	points := toReadings(rows, location)
	result := meterUsage(points, from, to, location, "m³")
	price := sumPrices(settings.WaterPricePerM3, settings.WastewaterPricePerM3)
	consumed := DomainValue{
		Value:        valuePointer(result.value),
		Unit:         "m³",
		PricePerUnit: price,
		PriceUnit:    "EUR/m³",
		Intervals:    result.intervals,
		CoveredHours: result.covered.Hours(),
		Estimate:     true,
		Status:       "Estimated by spreading meter changes across the time between readings.",
	}
	if result.intervals == 0 {
		consumed.Value = nil
		consumed.Status = "Need consecutive water-meter readings overlapping this period."
	} else if price != nil {
		consumed.CostEUR = valuePointer(result.value * *price)
	}
	daily := meterDaily(points, from, to, location, "m³")
	return WaterResponse{
		Window:         Window{From: from, To: to},
		CurrentReading: latestReading(points, to),
		Consumed:       consumed,
		Readings:       filterReadings(points, from, to),
		Daily:          daily,
	}, nil
}

func (s *Service) Heating(ctx context.Context, from, to time.Time, settings domainconfig.Settings, location *time.Location) (HeatingResponse, error) {
	rows, err := s.query(ctx, []string{pelletStockType}, nil, &to, nil)
	if err != nil {
		return HeatingResponse{}, err
	}
	points := toReadings(rows, location)
	points = normalizeStockPoints(points)
	kgPoints := normalizeStockUnits(points, settings.PelletTankCapacityKg)
	period := pelletUsage(kgPoints, from, to, location)
	first := firstTime(kgPoints)
	lifetime := pelletUsage(kgPoints, first, to, location)
	last30From := to.AddDate(0, 0, -30)
	last30 := pelletUsage(kgPoints, last30From, to, location)
	price, purchases := weightedPurchasePrice(ctx, s, to, location)
	forecast, forecastStatus := s.pelletForecast(ctx, to, settings.PelletTankCapacityKg, location)
	consumed := DomainValue{
		Value:        valuePointer(period.value),
		Unit:         "kg",
		PricePerUnit: price,
		PriceUnit:    "EUR/kg",
		Intervals:    period.intervals,
		CoveredHours: period.covered.Hours(),
		Estimate:     true,
		Status:       "Estimated from stock decreases; refills are treated as boundaries.",
	}
	if period.intervals == 0 {
		consumed.Value = nil
		consumed.Status = "Need two stock readings spanning this period."
	} else if price != nil {
		consumed.CostEUR = valuePointer(period.value * *price)
	}
	var lifetimeKg, last30Kg, dailyAverage, last30Cost *float64
	if lifetime.intervals > 0 {
		lifetimeKg = valuePointer(lifetime.value)
	}
	if last30.intervals > 0 {
		last30Kg = valuePointer(last30.value)
		if last30.covered > 0 {
			dailyAverage = valuePointer(last30.value / (last30.covered.Hours() / 24))
		}
		if price != nil {
			last30Cost = valuePointer(last30.value * *price)
		}
	}
	stock := latestReading(points, to)
	if stock != nil {
		if kg, ok := stockToKg(*stock, settings.PelletTankCapacityKg); ok {
			stock.Value = kg
			stock.Unit = "kg"
		}
	}
	stockReadings := normalizedStockReadings(points, settings.PelletTankCapacityKg)
	for index := 1; index < len(points); index++ {
		previous, previousOK := stockToKg(points[index-1], settings.PelletTankCapacityKg)
		current, currentOK := stockToKg(points[index], settings.PelletTankCapacityKg)
		if previousOK && currentOK && current > previous {
			for readingIndex := range stockReadings {
				if stockReadings[readingIndex].TS.Equal(points[index].TS) && stockReadings[readingIndex].Source == points[index].Source {
					stockReadings[readingIndex].EventType = "stock_increase"
					break
				}
			}
		}
	}
	return HeatingResponse{
		Window:                    Window{From: from, To: to},
		CurrentStock:              stock,
		Consumed:                  consumed,
		LifetimeConsumedKg:        lifetimeKg,
		AverageDaily30DaysKg:      dailyAverage,
		ConsumedLast30DaysKg:      last30Kg,
		ConsumedLast30DaysCostEUR: last30Cost,
		Last30DaysIntervals:       last30.intervals,
		Last30DaysCoveredHours:    last30.covered.Hours(),
		ForecastNextHourKg:        forecast,
		ForecastStatus:            forecastStatus,
		AveragePurchasePriceEUR:   price,
		StockReadings:             filterReadings(stockReadings, from, to),
		Daily:                     pelletDaily(kgPoints, from, to, location),
		Purchases:                 purchases,
	}, nil
}

type integrated struct {
	kWh       float64
	value     float64
	intervals int
	covered   time.Duration
}

func integratePower(points []ReadingPoint, from, to time.Time, location *time.Location) integrated {
	groups := groupReadings(points)
	var total integrated
	for _, series := range groups {
		for index := 1; index < len(series); index++ {
			left, leftOK := powerW(series[index-1])
			right, rightOK := powerW(series[index])
			start, end := series[index-1].TS, series[index].TS
			span := end.Sub(start)
			if !leftOK || !rightOK || span <= 0 || span > maxPowerGap {
				continue
			}
			start = maxTime(start, from)
			end = minTime(end, to)
			if !end.After(start) {
				continue
			}
			startFraction := float64(start.Sub(series[index-1].TS)) / float64(span)
			endFraction := float64(end.Sub(series[index-1].TS)) / float64(span)
			startW := math.Max(0, left+(right-left)*startFraction)
			endW := math.Max(0, left+(right-left)*endFraction)
			total.kWh += ((startW + endW) / 2) * end.Sub(start).Hours() / 1000
			total.intervals++
			total.covered += end.Sub(start)
		}
	}
	return total
}

func powerDaily(points []ReadingPoint, from, to time.Time, location *time.Location) []DailyValue {
	buckets := map[string]float64{}
	for _, series := range groupReadings(points) {
		for index := 1; index < len(series); index++ {
			leftW, leftOK := powerW(series[index-1])
			rightW, rightOK := powerW(series[index])
			left, right := series[index-1].TS, series[index].TS
			span := right.Sub(left)
			if !leftOK || !rightOK || span <= 0 || span > maxPowerGap {
				continue
			}
			start, end := maxTime(left, from), minTime(right, to)
			if !end.After(start) {
				continue
			}
			for dayStart := dayStart(start, location); dayStart.Before(end); {
				next := dayStart.AddDate(0, 0, 1)
				partStart, partEnd := maxTime(start, dayStart), minTime(end, next)
				if partEnd.After(partStart) {
					f0 := float64(partStart.Sub(left)) / float64(span)
					f1 := float64(partEnd.Sub(left)) / float64(span)
					p0 := math.Max(0, leftW+(rightW-leftW)*f0)
					p1 := math.Max(0, leftW+(rightW-leftW)*f1)
					key := dayStart.Format("2006-01-02")
					buckets[key] += ((p0 + p1) / 2) * partEnd.Sub(partStart).Hours() / 1000
				}
				dayStart = next
			}
		}
	}
	return sortedDaily(buckets, "kWh")
}

func meterUsage(points []ReadingPoint, from, to time.Time, location *time.Location, unit string) integrated {
	var total integrated
	for index := 1; index < len(points); index++ {
		leftValue, leftOK := convertWater(points[index-1])
		rightValue, rightOK := convertWater(points[index])
		left, right := points[index-1].TS, points[index].TS
		span := right.Sub(left)
		if !leftOK || !rightOK || span <= 0 || rightValue < leftValue {
			continue
		}
		start, end := maxTime(left, from), minTime(right, to)
		if !end.After(start) {
			continue
		}
		total.value += (rightValue - leftValue) * float64(end.Sub(start)) / float64(span)
		total.intervals++
		total.covered += end.Sub(start)
	}
	return total
}

func meterDaily(points []ReadingPoint, from, to time.Time, location *time.Location, unit string) []DailyValue {
	buckets := map[string]float64{}
	for index := 1; index < len(points); index++ {
		leftValue, leftOK := convertWater(points[index-1])
		rightValue, rightOK := convertWater(points[index])
		left, right := points[index-1].TS, points[index].TS
		span := right.Sub(left)
		if !leftOK || !rightOK || span <= 0 || rightValue < leftValue {
			continue
		}
		start, end := maxTime(left, from), minTime(right, to)
		if !end.After(start) {
			continue
		}
		for dayStart := dayStart(start, location); dayStart.Before(end); {
			next := dayStart.AddDate(0, 0, 1)
			partStart, partEnd := maxTime(start, dayStart), minTime(end, next)
			if partEnd.After(partStart) {
				buckets[dayStart.Format("2006-01-02")] += (rightValue - leftValue) * float64(partEnd.Sub(partStart)) / float64(span)
			}
			dayStart = next
		}
	}
	return sortedDaily(buckets, unit)
}

func pelletUsage(points []ReadingPoint, from, to time.Time, location *time.Location) integrated {
	var total integrated
	for index := 1; index < len(points); index++ {
		left, leftOK := stockToKg(points[index-1], nil)
		right, rightOK := stockToKg(points[index], nil)
		startTime, endTime := points[index-1].TS, points[index].TS
		span := endTime.Sub(startTime)
		if !leftOK || !rightOK || span <= 0 || right >= left {
			continue
		}
		start, end := maxTime(startTime, from), minTime(endTime, to)
		if !end.After(start) {
			continue
		}
		total.value += (left - right) * float64(end.Sub(start)) / float64(span)
		total.intervals++
		total.covered += end.Sub(start)
	}
	return total
}

func pelletDaily(points []ReadingPoint, from, to time.Time, location *time.Location) []DailyValue {
	buckets := map[string]float64{}
	for index := 1; index < len(points); index++ {
		left, leftOK := stockToKg(points[index-1], nil)
		right, rightOK := stockToKg(points[index], nil)
		startTime, endTime := points[index-1].TS, points[index].TS
		span := endTime.Sub(startTime)
		if !leftOK || !rightOK || span <= 0 || right >= left {
			continue
		}
		start, end := maxTime(startTime, from), minTime(endTime, to)
		if !end.After(start) {
			continue
		}
		for dayStart := dayStart(start, location); dayStart.Before(end); {
			next := dayStart.AddDate(0, 0, 1)
			partStart, partEnd := maxTime(start, dayStart), minTime(end, next)
			if partEnd.After(partStart) {
				share := (left - right) * float64(partEnd.Sub(partStart)) / float64(span)
				buckets[dayStart.Format("2006-01-02")] += share
			}
			dayStart = next
		}
	}
	return sortedDaily(buckets, "kg")
}

func (s *Service) pelletForecast(ctx context.Context, to time.Time, capacity *float64, location *time.Location) (*float64, string) {
	from := to.Add(-forecastWindow)
	rows, err := s.query(ctx, []string{pelletStockType}, &from, &to, []string{etaSource})
	if err != nil {
		return nil, "Forecast unavailable because recent ETA readings could not be loaded."
	}
	points := normalizeStockPoints(toReadings(rows, location))
	points = normalizeStockUnits(points, capacity)
	if len(points) < 2 {
		return nil, "Need at least two recent ETA stock readings."
	}
	continuousStart := len(points) - 1
	for continuousStart > 0 && points[continuousStart].TS.Sub(points[continuousStart-1].TS) <= maxForecastAge {
		continuousStart--
	}
	points = points[continuousStart:]
	if len(points) < 2 {
		return nil, "Recent ETA stock readings are too far apart."
	}
	latest := points[len(points)-1]
	if to.Sub(latest.TS) > maxForecastAge {
		return nil, "Latest ETA stock reading is older than 20 minutes."
	}
	first := points[0]
	observed := latest.TS.Sub(first.TS)
	if observed < minimumForecastAge {
		return nil, "Need at least two hours of ETA stock history."
	}
	usage := pelletUsage(points, points[0].TS, to, location)
	if usage.intervals == 0 {
		return nil, "No declining ETA stock intervals in the forecast window."
	}
	return valuePointer(usage.value / observed.Hours()), "Estimated from the recent ETA stock decline."
}

func weightedPurchasePrice(ctx context.Context, s *Service, to time.Time, location *time.Location) (*float64, []Purchase) {
	rows, err := s.query(ctx, []string{purchaseQuantityType, purchaseCostType}, nil, &to, nil)
	if err != nil {
		return nil, nil
	}
	type purchaseGroup struct {
		ts       time.Time
		quantity *float64
		cost     *float64
		note     string
	}
	groups := map[string]*purchaseGroup{}
	for _, row := range rows {
		key := row.Source
		group := groups[key]
		if group == nil {
			group = &purchaseGroup{ts: row.TS.In(location)}
			groups[key] = group
		}
		if row.TS.Before(group.ts) {
			group.ts = row.TS.In(location)
		}
		if group.note == "" && row.Note != "" {
			group.note = row.Note
		}
		switch row.Type {
		case purchaseQuantityType:
			if value, ok := quantityKg(row.Value, row.Unit); ok {
				group.quantity = value
			}
		case purchaseCostType:
			if strings.EqualFold(strings.TrimSpace(row.Unit), "EUR") || row.Unit == "€" {
				group.cost = valuePointer(row.Value)
			}
		}
	}
	purchases := make([]Purchase, 0, len(groups))
	var totalQuantity, totalCost float64
	for _, group := range groups {
		purchase := Purchase{TS: group.ts, QuantityKg: group.quantity, TotalCostEUR: group.cost, Note: group.note}
		if group.quantity != nil && *group.quantity > 0 && group.cost != nil && *group.cost >= 0 {
			unitPrice := *group.cost / *group.quantity
			purchase.CostPerKgEUR = valuePointer(unitPrice)
			totalQuantity += *group.quantity
			totalCost += *group.cost
		}
		purchases = append(purchases, purchase)
	}
	sort.Slice(purchases, func(i, j int) bool { return purchases[i].TS.After(purchases[j].TS) })
	if totalQuantity == 0 {
		return nil, purchases
	}
	return valuePointer(totalCost / totalQuantity), purchases
}

func normalizeStockPoints(points []ReadingPoint) []ReadingPoint {
	points = append([]ReadingPoint(nil), points...)
	sort.SliceStable(points, func(i, j int) bool { return points[i].TS.Before(points[j].TS) })
	result := make([]ReadingPoint, 0, len(points))
	for _, point := range points {
		previous := len(result) - 1
		if previous >= 0 && result[previous].Source != point.Source && math.Abs(float64(point.TS.Sub(result[previous].TS))) <= float64(15*time.Minute) {
			if point.Source == etaSource && result[previous].Source != etaSource {
				result[previous] = point
			}
			continue
		}
		result = append(result, point)
	}
	return result
}

func normalizeStockUnits(points []ReadingPoint, capacity *float64) []ReadingPoint {
	result := make([]ReadingPoint, 0, len(points))
	for _, point := range points {
		if kg, ok := stockToKg(point, capacity); ok {
			point.Value = kg
			point.Unit = "kg"
			result = append(result, point)
		}
	}
	return result
}

func normalizedStockReadings(points []ReadingPoint, capacity *float64) []ReadingPoint {
	result := make([]ReadingPoint, 0, len(points))
	for _, point := range points {
		if kg, ok := stockToKg(point, capacity); ok {
			point.Value = kg
			point.Unit = "kg"
		}
		result = append(result, point)
	}
	return result
}

func stockToKg(point ReadingPoint, capacity *float64) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(point.Unit)) {
	case "kg", "kilogram", "kilograms", "":
		return point.Value, true
	case "g", "gram", "grams":
		return point.Value / 1000, true
	case "t", "ton", "tonne", "tonnes":
		return point.Value * 1000, true
	case "%", "percent", "percentage":
		if capacity == nil || *capacity <= 0 {
			return 0, false
		}
		return point.Value * *capacity / 100, true
	default:
		return 0, false
	}
}

func quantityKg(value float64, unit string) (*float64, bool) {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "kg", "kilogram", "kilograms", "":
		return valuePointer(value), true
	case "t", "ton", "tonne", "tonnes":
		return valuePointer(value * 1000), true
	default:
		return nil, false
	}
}

func convertWater(point ReadingPoint) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(point.Unit)) {
	case "m³", "m3":
		return point.Value, true
	case "l", "liter", "litre", "liters", "litres":
		return point.Value / 1000, true
	default:
		return 0, false
	}
}

func powerW(point ReadingPoint) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(point.Unit)) {
	case "w", "watt", "watts":
		return point.Value, true
	case "kw", "kilowatt", "kilowatts":
		return point.Value * 1000, true
	default:
		return 0, false
	}
}

func sumPrices(first, second *float64) *float64 {
	if first == nil || second == nil {
		return nil
	}
	return valuePointer(*first + *second)
}

func toReadings(rows []measurements.Measurement, location *time.Location) []ReadingPoint {
	result := make([]ReadingPoint, 0, len(rows))
	for _, row := range rows {
		result = append(result, ReadingPoint{TS: row.TS.In(location), Value: row.Value, Unit: row.Unit, Source: row.Source, Note: row.Note})
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].TS.Before(result[j].TS) })
	return result
}

func groupReadings(points []ReadingPoint) [][]ReadingPoint {
	groups := map[string][]ReadingPoint{}
	order := make([]string, 0)
	for _, point := range points {
		if _, ok := groups[point.Source]; !ok {
			order = append(order, point.Source)
		}
		groups[point.Source] = append(groups[point.Source], point)
	}
	result := make([][]ReadingPoint, 0, len(groups))
	for _, key := range order {
		series := groups[key]
		sort.SliceStable(series, func(i, j int) bool { return series[i].TS.Before(series[j].TS) })
		result = append(result, series)
	}
	return result
}

func filterReadings(points []ReadingPoint, from, to time.Time) []ReadingPoint {
	result := make([]ReadingPoint, 0)
	for _, point := range points {
		if !point.TS.Before(from) && !point.TS.After(to) {
			result = append(result, point)
		}
	}
	return result
}

func latestReading(points []ReadingPoint, to time.Time) *ReadingPoint {
	for index := len(points) - 1; index >= 0; index-- {
		if !points[index].TS.After(to) {
			point := points[index]
			return &point
		}
	}
	return nil
}

func firstTime(points []ReadingPoint) time.Time {
	if len(points) == 0 {
		return time.Time{}
	}
	return points[0].TS
}

func sortedDaily(values map[string]float64, unit string) []DailyValue {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]DailyValue, 0, len(keys))
	for _, key := range keys {
		result = append(result, DailyValue{Date: key, Value: values[key], Unit: unit})
	}
	return result
}

func dayStart(value time.Time, location *time.Location) time.Time {
	local := value.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func valuePointer(value float64) *float64 {
	return &value
}
