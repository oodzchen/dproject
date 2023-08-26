package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/store/pgstore"
	"github.com/oodzchen/dproject/utils"
)

func main() {
	err := config.Init(".env.local")
	if err != nil {
		log.Fatal(err)
	}

	appCfg := config.Config

	// fmt.Printf("App config: %#v\n", appCfg)
	if appCfg.Debug {
		utils.PrintJSONf("App config:\n", appCfg)
	}
	// fmt.Println("DSN: ", os.Getenv("DB_DSN"))

	pg := pgstore.New(&pgstore.DBConfig{
		DSN: appCfg.DB.GetDSN(),
	})

	err = pg.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer pg.CloseDB()

	dataStore, err := store.New(pg)
	if err != nil {
		log.Fatal(err)
	}

	port := appCfg.Port
	server := &http.Server{
		Addr: fmt.Sprintf("0.0.0.0:%d", port),
		Handler: (Service(&ServiceConfig{
			sessSecret: appCfg.SessionSecret,
			store:      dataStore,
		})),
	}

	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		shutdownCtx, cancel := context.WithTimeout(serverCtx, 3*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("gracefual shutdown time out, force exit")
			}
		}()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
		serverStopCtx()
	}()

	fmt.Printf("Listening at http://localhost%d\n", port)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	// port := os.Getenv("PORT")
	// log.Fatal(http.ListenAndServe(port, server))
	<-serverCtx.Done()
}
