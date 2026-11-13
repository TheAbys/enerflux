package sourceconfig

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
		CREATE TABLE IF NOT EXISTS source_configuration (
			id SMALLINT PRIMARY KEY CHECK (id = 1),
			solarlog_url TEXT NOT NULL,
			solarlog_poll_interval TEXT NOT NULL,
			eta_pelletstock_url TEXT NOT NULL,
			eta_pelletstock_poll_interval TEXT NOT NULL,
			eta_outsidetemp_url TEXT NOT NULL,
			eta_outsidetemp_poll_interval TEXT NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("create source configuration table: %w", err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, fallback Settings) (Settings, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT solarlog_url, solarlog_poll_interval,
		       eta_pelletstock_url, eta_pelletstock_poll_interval,
		       eta_outsidetemp_url, eta_outsidetemp_poll_interval
		FROM source_configuration
		WHERE id = 1
	`)
	if err != nil {
		return Settings{}, fmt.Errorf("read source configuration: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return Settings{}, fmt.Errorf("read source configuration: %w", err)
		}
		return fallback, nil
	}

	var settings Settings
	if err := rows.Scan(
		&settings.SolarLogURL,
		&settings.SolarLogPollInterval,
		&settings.EtaPelletstockURL,
		&settings.EtaPelletstockPollInterval,
		&settings.EtaOutsidetempURL,
		&settings.EtaOutsidetempPollInterval,
	); err != nil {
		return Settings{}, fmt.Errorf("scan source configuration: %w", err)
	}
	return settings, nil
}

func (r *Repository) Save(ctx context.Context, settings Settings) error {
	_, err := r.DB.Exec(ctx, `
		INSERT INTO source_configuration (
			id, solarlog_url, solarlog_poll_interval,
			eta_pelletstock_url, eta_pelletstock_poll_interval,
			eta_outsidetemp_url, eta_outsidetemp_poll_interval, updated_at
		)
		VALUES (1, $1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			solarlog_url = EXCLUDED.solarlog_url,
			solarlog_poll_interval = EXCLUDED.solarlog_poll_interval,
			eta_pelletstock_url = EXCLUDED.eta_pelletstock_url,
			eta_pelletstock_poll_interval = EXCLUDED.eta_pelletstock_poll_interval,
			eta_outsidetemp_url = EXCLUDED.eta_outsidetemp_url,
			eta_outsidetemp_poll_interval = EXCLUDED.eta_outsidetemp_poll_interval,
			updated_at = CURRENT_TIMESTAMP
	`,
		settings.SolarLogURL,
		settings.SolarLogPollInterval,
		settings.EtaPelletstockURL,
		settings.EtaPelletstockPollInterval,
		settings.EtaOutsidetempURL,
		settings.EtaOutsidetempPollInterval,
	)
	if err != nil {
		return fmt.Errorf("save source configuration: %w", err)
	}
	return nil
}
