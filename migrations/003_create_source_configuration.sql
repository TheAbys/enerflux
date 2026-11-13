CREATE TABLE IF NOT EXISTS source_configuration (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    solarlog_url TEXT NOT NULL,
    solarlog_poll_interval TEXT NOT NULL,
    eta_pelletstock_url TEXT NOT NULL,
    eta_pelletstock_poll_interval TEXT NOT NULL,
    eta_outsidetemp_url TEXT NOT NULL,
    eta_outsidetemp_poll_interval TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
