package analytics

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/theabys/enerflux/internal/units"
)

// weightedPurchasePrice pairs purchase quantity and cost readings by source, then divides the sum
// of valid purchase costs by their combined quantity to return a weighted EUR/kg price.
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
			if strings.EqualFold(strings.TrimSpace(row.Unit), units.Euro) || row.Unit == units.EuroSymbol {
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

// quantityKg converts kilogram and tonne purchase quantities to kilograms.
func quantityKg(value float64, unit string) (*float64, bool) {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case units.Kilogram, "kilogram", "kilograms", "":
		return valuePointer(value), true
	case "t", "ton", "tonne", "tonnes":
		return valuePointer(value * 1000), true
	default:
		return nil, false
	}
}
