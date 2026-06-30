CREATE TABLE IF NOT EXISTS measurements (
    id SERIAL PRIMARY KEY,
    ts TIMESTAMP NOT NULL,
    type TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit TEXT NOT NULL,
    source TEXT
);

CREATE INDEX idx_measurements_ts ON measurements(ts);
CREATE INDEX idx_measurements_type ON measurements(type);