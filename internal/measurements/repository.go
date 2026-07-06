package measurements

import (
	"context"

	"github.com/theabys/enerflux/internal/database"
)

type MeasurementRepository interface {
	Insert(m Measurement) error
	List(limit int) ([]Measurement, error)
	GetLatest() (*Measurement, error)
	QueryMeasurements(ctx context.Context, filter MeasurementFilter) ([]Measurement, error)
}

type MeasurementRepo struct {
	Db *database.DB
}

func NewMeasurementRepo(db *database.DB) *MeasurementRepo {
	return &MeasurementRepo{Db: db}
}

func (repo *MeasurementRepo) Insert(m Measurement) error {
	_, err := repo.Db.Exec(
		context.Background(),
		`INSERT INTO measurements (ts, type, value, unit, source)
		 VALUES ($1, $2, $3, $4, $5)`,
		m.TS, m.Type, m.Value, m.Unit, m.Source,
	)
	return err
}

func (repo *MeasurementRepo) List(limit int) ([]Measurement, error) {
	rows, err := repo.Db.Query(
		context.Background(),
		`SELECT id, ts, type, value, unit, source
		 FROM measurements
		 ORDER BY ts DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Measurement

	for rows.Next() {
		var m Measurement
		err := rows.Scan(&m.ID, &m.TS, &m.Type, &m.Value, &m.Unit, &m.Source)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}

	return result, nil
}

func (repo *MeasurementRepo) QueryMeasurements(ctx context.Context, filter MeasurementFilter) ([]Measurement, error) {
	rows, err := repo.Db.Query(
		context.Background(),
		`SELECT id, ts, type, value, unit, source
		 FROM measurements
		 WHERE type = $1
		 ORDER BY ts DESC`,
		filter.Key,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Measurement

	for rows.Next() {
		var m Measurement
		err := rows.Scan(&m.ID, &m.TS, &m.Type, &m.Value, &m.Unit, &m.Source)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}

	return result, nil
}

func (repo *MeasurementRepo) GetLatest() (*Measurement, error) {
	rows, err := repo.Db.Query(
		context.Background(),
		`SELECT id, ts, type, value, unit, source
		 FROM measurements
		 ORDER BY ts DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result *Measurement

	for rows.Next() {
		var m Measurement
		err := rows.Scan(&m.ID, &m.TS, &m.Type, &m.Value, &m.Unit, &m.Source)
		if err != nil {
			return nil, err
		}
		result = &m
	}

	return result, nil
}
