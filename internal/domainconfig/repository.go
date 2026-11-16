package domainconfig

import (
	"context"
	"fmt"

	"github.com/theabys/enerflux/internal/database"
)

type Repository struct {
	DB *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) EnsureSchema(ctx context.Context) error {
	_, err := r.DB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS domain_configuration (
			id SMALLINT PRIMARY KEY CHECK (id = 1),
			time_zone TEXT NOT NULL DEFAULT 'Europe/Berlin',
			electricity_price_per_kwh DOUBLE PRECISION CHECK (electricity_price_per_kwh IS NULL OR electricity_price_per_kwh >= 0),
			water_price_per_m3 DOUBLE PRECISION CHECK (water_price_per_m3 IS NULL OR water_price_per_m3 >= 0),
			wastewater_price_per_m3 DOUBLE PRECISION CHECK (wastewater_price_per_m3 IS NULL OR wastewater_price_per_m3 >= 0),
			pellet_tank_capacity_kg DOUBLE PRECISION CHECK (pellet_tank_capacity_kg IS NULL OR pellet_tank_capacity_kg > 0),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("ensure domain configuration schema: %w", err)
	}
	if _, err := r.DB.Exec(ctx, `INSERT INTO domain_configuration (id) VALUES (1) ON CONFLICT (id) DO NOTHING`); err != nil {
		return fmt.Errorf("initialize domain configuration: %w", err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context) (Settings, error) {
	var settings Settings
	var electricity, water, wastewater, capacity float64
	rows, err := r.DB.Query(ctx, `
		SELECT time_zone,
		       COALESCE(electricity_price_per_kwh, -1),
		       COALESCE(water_price_per_m3, -1),
		       COALESCE(wastewater_price_per_m3, -1),
		       COALESCE(pellet_tank_capacity_kg, -1)
		FROM domain_configuration WHERE id = 1
	`)
	if err != nil {
		return Settings{}, fmt.Errorf("read domain configuration: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return Settings{}, fmt.Errorf("read domain configuration: %w", err)
		}
		return DefaultSettings(), nil
	}
	if err := rows.Scan(&settings.TimeZone, &electricity, &water, &wastewater, &capacity); err != nil {
		return Settings{}, fmt.Errorf("scan domain configuration: %w", err)
	}
	settings.ElectricityPricePerKWh = optional(electricity)
	settings.WaterPricePerM3 = optional(water)
	settings.WastewaterPricePerM3 = optional(wastewater)
	settings.PelletTankCapacityKg = optional(capacity)
	return settings, nil
}

func (r *Repository) Save(ctx context.Context, settings Settings) error {
	_, err := r.DB.Exec(ctx, `
		INSERT INTO domain_configuration (
			id, time_zone, electricity_price_per_kwh, water_price_per_m3,
			wastewater_price_per_m3, pellet_tank_capacity_kg, updated_at
		) VALUES (1, $1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			time_zone = EXCLUDED.time_zone,
			electricity_price_per_kwh = EXCLUDED.electricity_price_per_kwh,
			water_price_per_m3 = EXCLUDED.water_price_per_m3,
			wastewater_price_per_m3 = EXCLUDED.wastewater_price_per_m3,
			pellet_tank_capacity_kg = EXCLUDED.pellet_tank_capacity_kg,
			updated_at = CURRENT_TIMESTAMP
	`, settings.TimeZone, settings.ElectricityPricePerKWh, settings.WaterPricePerM3, settings.WastewaterPricePerM3, settings.PelletTankCapacityKg)
	if err != nil {
		return fmt.Errorf("save domain configuration: %w", err)
	}
	return nil
}

func optional(value float64) *float64 {
	if value < 0 {
		return nil
	}
	return &value
}
