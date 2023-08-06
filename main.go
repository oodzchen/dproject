package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/store/pgstore"
)

func main() {
	err := godotenv.Load(".env.local")
	if err != nil {
		log.Fatal(err)
	}

	pg := pgstore.New(&pgstore.DBConfig{
		DSN: os.Getenv("DB_DSN"),
	})

	err = pg.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer pg.CloseDB()

	dataStore, err := store.New(pg)

	server := NewServer(&ServerConfig{
		sessSecret: os.Getenv("SESSION_SECRET"),
		store:      dataStore,
	})

	port := os.Getenv("PORT")
	fmt.Printf("Listening at http://localhost%v\n", port)
	log.Fatal(http.ListenAndServe(port, server))
}
