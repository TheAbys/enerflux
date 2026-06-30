package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type DB struct {
	Conn *pgx.Conn
}

func NewDB(connString string) (*DB, error) {
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, err
	}

	return &DB{Conn: conn}, nil
}

func (db *DB) Ping() error {
	var result int
	err := db.Conn.QueryRow(context.Background(), "SELECT 1").Scan(&result)
	if err != nil {
		return err
	}

	if result != 1 {
		return fmt.Errorf("unexpected ping result: %d", result)
	}

	return nil
}
