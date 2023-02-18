package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func ConnectDB() *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:888888@localhost:8888")
	if err != nil {
		fmt.Printf("Connect database error: %v\n", err)
		os.Exit(1)
	}
	return conn
}
