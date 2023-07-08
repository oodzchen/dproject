package main

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type DB struct {
	DSN  string
	Conn *pgx.Conn
}

func (db *DB) Connect() error {
	conn, err := pgx.Connect(context.Background(), db.DSN)
	if err != nil {
		return err
	}
	db.Conn = conn
	return nil
}

func (db *DB) Close() error {
	if db.Conn != nil {
		return db.Conn.Close(context.Background())
	}
	return nil
}
