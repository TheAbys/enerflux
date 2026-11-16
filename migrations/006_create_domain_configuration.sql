CREATE TABLE IF NOT EXISTS domain_configuration (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    time_zone TEXT NOT NULL DEFAULT 'Europe/Berlin',
    electricity_price_per_kwh DOUBLE PRECISION CHECK (electricity_price_per_kwh IS NULL OR electricity_price_per_kwh >= 0),
    water_price_per_m3 DOUBLE PRECISION CHECK (water_price_per_m3 IS NULL OR water_price_per_m3 >= 0),
    wastewater_price_per_m3 DOUBLE PRECISION CHECK (wastewater_price_per_m3 IS NULL OR wastewater_price_per_m3 >= 0),
    pellet_tank_capacity_kg DOUBLE PRECISION CHECK (pellet_tank_capacity_kg IS NULL OR pellet_tank_capacity_kg > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO domain_configuration (id) VALUES (1)
ON CONFLICT (id) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_measurements_type_ts ON measurements(type, ts);
CREATE INDEX IF NOT EXISTS idx_measurements_type_source_ts ON measurements(type, source, ts);
