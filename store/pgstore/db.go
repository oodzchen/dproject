package pgstore

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	DSN string
	// Conn *pgx.Conn
	Pool *pgxpool.Pool
}

func (db *DB) Connect() error {
	// conn, err := pgx.Connect(context.Background(), db.DSN)
	dbpool, err := pgxpool.New(context.Background(), db.DSN)
	if err != nil {
		return err
	}
	// db.Conn = conn
	db.Pool = dbpool
	return nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}
