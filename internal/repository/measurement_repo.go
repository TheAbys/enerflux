package repository

import (
	"context"

	"github.com/theabys/enerflux/internal/model"
)

type MeasurementRepo struct {
	Db *DB
}

func NewMeasurementRepo(db *DB) *MeasurementRepo {
	return &MeasurementRepo{Db: db}
}

func (repo *MeasurementRepo) InsertMeasurement(m model.Measurement) error {
	_, err := repo.Db.Conn.Exec(
		context.Background(),
		`INSERT INTO measurements (ts, type, value, unit, source)
		 VALUES ($1, $2, $3, $4, $5)`,
		m.TS, m.Type, m.Value, m.Unit, m.Source,
	)
	return err
}

func (repo *MeasurementRepo) List(limit int) ([]model.Measurement, error) {
	rows, err := repo.Db.Conn.Query(
		context.Background(),
		`SELECT id, ts, type, value, unit, source
		 FROM measurements
		 ORDER BY ts DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.Measurement

	for rows.Next() {
		var m model.Measurement
		err := rows.Scan(&m.ID, &m.TS, &m.Type, &m.Value, &m.Unit, &m.Source)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}

	return result, nil
}

func (repo *MeasurementRepo) GetLatest() (*model.Measurement, error) {
	rows, err := repo.Db.Conn.Query(
		context.Background(),
		`SELECT id, ts, type, value, unit, source
		 FROM measurements
		 ORDER BY ts DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result *model.Measurement

	for rows.Next() {
		var m model.Measurement
		err := rows.Scan(&m.ID, &m.TS, &m.Type, &m.Value, &m.Unit, &m.Source)
		if err != nil {
			return nil, err
		}
		result = &m
	}

	return result, nil
}
