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
	i18nc "github.com/oodzchen/dproject/i18n"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/store/pgstore"
	"github.com/oodzchen/dproject/utils"
)

func main() {
	var err error
	testingMode := os.Getenv("TEST")
	if testingMode == "1" || testingMode == "true" {
		err = config.Init(".env.testing")
	} else {
		err = config.InitFromEnv()
	}

	if err != nil {
		log.Fatal(err)
	}

	appCfg := config.Config

	// fmt.Printf("App config: %#v\n", appCfg)
	if appCfg.Debug {
		utils.PrintJSONf("App config:\n", appCfg)
	}
	// fmt.Println("DSN: ", os.Getenv("DB_DSN"))

	i18nc.Init()

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
	addr := fmt.Sprintf(":%d", port)

	model.InitConfidences()
	server := &http.Server{
		Addr: addr,
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

	fmt.Printf("App listening at http://localhost:%d\n", port)
	fmt.Printf("Nginx expose at http://localhost:%d\n", appCfg.AppOuterPort)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	<-serverCtx.Done()
}
