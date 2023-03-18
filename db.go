package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func ConnectDB() *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), "postgres://admin:88886666@localhost:8888/discuss")
	if err != nil {
		fmt.Printf("Connect database error: %v\n", err)
		os.Exit(1)
	}
	return conn
}
