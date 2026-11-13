DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'uniq_measurement'
          AND conrelid = 'measurements'::regclass
    ) THEN
        ALTER TABLE measurements
        ADD CONSTRAINT uniq_measurement UNIQUE (source, ts, type);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_measurements_lookup ON measurements (source, ts, type);
