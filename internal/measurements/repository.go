package measurements

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/theabys/enerflux/internal/database"
)

type MeasurementRepository interface {
	InsertMany(ctx context.Context, measurements []Measurement) error
	Find(ctx context.Context, options QueryOptions) ([]Measurement, error)
}

type MeasurementRepo struct {
	Db *database.DB
}

func NewMeasurementRepo(db *database.DB) *MeasurementRepo {
	return &MeasurementRepo{Db: db}
}

func (r *MeasurementRepo) InsertMany(ctx context.Context, measurements []Measurement) error {
	if len(measurements) == 0 {
		return nil
	}

	rows := make([][]any, len(measurements))

	for i, item := range measurements {
		rows[i] = []any{
			item.TS,
			item.Type,
			item.Value,
			item.Unit,
			item.Source,
		}
	}

	_, err := r.Db.CopyFrom(
		ctx,
		pgx.Identifier{"measurements"},
		[]string{"ts", "type", "value", "unit", "source"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("copy measurements: %w", err)
	}

	return nil
}
func (r *MeasurementRepo) Find(ctx context.Context, options QueryOptions) ([]Measurement, error) {
	query := `
		SELECT ts, type, value, unit, source
		FROM measurements
		WHERE 1 = 1
	`

	args := make([]any, 0)
	argIndex := 1

	if len(options.Filter.Types) > 0 {
		query += fmt.Sprintf(" AND type = ANY($%d)", argIndex)
		args = append(args, options.Filter.Types)
		argIndex++
	}

	if len(options.Filter.Sources) > 0 {
		query += fmt.Sprintf(" AND source = ANY($%d)", argIndex)
		args = append(args, options.Filter.Sources)
		argIndex++
	}

	if options.Filter.From != nil {
		query += fmt.Sprintf(" AND ts >= $%d", argIndex)
		args = append(args, *options.Filter.From)
		argIndex++
	}

	if options.Filter.To != nil {
		query += fmt.Sprintf(" AND ts <= $%d", argIndex)
		args = append(args, *options.Filter.To)
		argIndex++
	}

	if len(options.Sort) > 0 {
		orderParts := make([]string, 0, len(options.Sort))

		for _, sort := range options.Sort {
			column, err := sortColumn(sort.Field)
			if err != nil {
				return nil, err
			}

			direction, err := sortDirection(sort.Direction)
			if err != nil {
				return nil, err
			}

			orderParts = append(
				orderParts,
				column+" "+direction,
			)
		}

		query += " ORDER BY " + strings.Join(orderParts, ", ")
	}

	if options.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, options.Limit)
		argIndex++
	}

	if options.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, options.Offset)
	}

	rows, err := r.Db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query measurements: %w", err)
	}
	defer rows.Close()

	measurements := make([]Measurement, 0)

	for rows.Next() {
		var item Measurement

		if err := rows.Scan(
			&item.TS,
			&item.Type,
			&item.Value,
			&item.Unit,
			&item.Source,
		); err != nil {
			return nil, fmt.Errorf("scan measurement: %w", err)
		}

		measurements = append(measurements, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate measurements: %w", err)
	}

	return measurements, nil
}

func sortColumn(field SortField) (string, error) {
	switch field {
	case SortByTimestamp:
		return "ts", nil
	case SortByType:
		return "type", nil
	case SortByValue:
		return "value", nil
	case SortBySource:
		return "source", nil
	default:
		return "", fmt.Errorf("unsupported sort field: %s", field)
	}
}

func sortDirection(direction SortDirection) (string, error) {
	switch direction {
	case SortAscending:
		return "ASC", nil
	case SortDescending:
		return "DESC", nil
	default:
		return "", fmt.Errorf(
			"unsupported sort direction: %s",
			direction,
		)
	}
}
