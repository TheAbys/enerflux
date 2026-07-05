ALTER TABLE measurements
ADD CONSTRAINT uniq_measurement UNIQUE (source, ts, type);

CREATE INDEX idx_measurements_lookup ON measurements (source, ts, type);