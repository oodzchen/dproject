package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/oodzchen/dproject/config"
	i18nc "github.com/oodzchen/dproject/i18n"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/service"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/store/pgstore"
	"github.com/oodzchen/dproject/utils"
	"github.com/redis/go-redis/v9"
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

	if appCfg.Debug {
		runtime.GOMAXPROCS(1)
	}

	// fmt.Printf("App config: %#v\n", appCfg)
	if appCfg.Debug {
		utils.PrintJSONf("App config:\n", appCfg)
	}
	// fmt.Println("DSN: ", os.Getenv("DB_DSN"))

	langFiles := []string{
		"./i18n/active.zh-Hans.toml",
		"./i18n/active.zh-Hant.toml",
		"./i18n/active.ja.toml",
	}

	model.Init()
	i18nCustom := i18nc.New(langFiles)
	model.SetupI18n(i18nCustom)

	permissionData, err := config.ParsePermissionData("./config/permissions.yml")
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Println("permissionData: ", permissionData)
	// fmt.Println("create articler enabled: ", permissionData.Permit("article", "create"))
	// fmt.Println("ban user enabled: ", permissionData.Permit("user", "ban"))

	roleData, err := config.ParseRoleData("./config/roles.yml")
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Println("roleData: ", roleData)
	// fmt.Println("roleData['common_user']: ", roleData["common_user"])
	// fmt.Println("roleData.Get('moderator'): ", roleData.Get("moderator"))
	// fmt.Println("roleData.Get('aaa'): ", roleData.Get("aaa"))

	pg := pgstore.New(&pgstore.DBConfig{
		DSN: appCfg.DB.GetDSN(),
	})

	fmt.Println("connecting database...")
	err = pg.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("connected database successfully")
	defer pg.CloseDB()

	dataStore, err := store.New(pg)
	if err != nil {
		log.Fatal(err)
	}

	redisAddr := net.JoinHostPort(appCfg.Redis.Host, appCfg.Redis.Port)
	fmt.Printf("connecting redis at %s ...\n", redisAddr)
	redisDB := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: appCfg.Redis.User,
		Password: appCfg.Redis.Password,
		DB:       0,
	})
	ctx := context.Background()
	err = redisDB.Ping(ctx).Err()
	if err != nil {
		log.Fatal(err)
	}
	defer redisDB.Close()
	fmt.Println("connected redis successfully")

	permissionSrv := &service.Permission{
		Store:          dataStore,
		PermissionData: permissionData,
		RoleData:       roleData,
	}

	for {
		err := pg.Ping(context.Background())
		// fmt.Println("ping database error: ", err)
		if err == nil {
			err = permissionSrv.InitPermissionTable()
			if err != nil {
				log.Fatal(err)
			}

			err = permissionSrv.InitRoleTable()
			if err != nil {
				log.Fatal(err)
			}

			err = permissionSrv.InitUserRoleTable()
			if err != nil {
				log.Fatal(err)
			}
			break
		}
	}

	appTLS := false
	if os.Getenv("APP_TLS") == "1" {
		appTLS = true
	}

	port := appCfg.AppPort
	addr := fmt.Sprintf(":%d", port)

	sanitizePolicy := bluemonday.UGCPolicy()
	// sanitizePolicy.AllowElements("b")

	mail := service.NewMail(appCfg.SMTP.User, appCfg.SMTP.Password, appCfg.SMTP.Server, appCfg.SMTP.ServerPort, i18nCustom)

	server := &http.Server{
		Addr: addr,
		Handler: (Service(&ServiceConfig{
			sessSecret:     appCfg.SessionSecret,
			store:          dataStore,
			permisisonSrv:  permissionSrv,
			sanitizePolicy: sanitizePolicy,
			i18nCustom:     i18nCustom,
			rdb:            redisDB,
			mail:           mail,
		})),
	}

	tlsManager := NewCertManager()
	if appTLS {
		server.Addr = ":https"
		server.TLSConfig = tlsManager.TLSConfig()
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

	if appTLS {
		go func() {
			log.Fatal(http.ListenAndServe(":http", tlsManager.HTTPHandler(nil)))
		}()
		// fmt.Printf("App listening at https://%s\n", appCfg.DomainName)
		err = server.ListenAndServeTLS("", "")
		// err = http.Serve(autocert.NewListener(appCfg.DomainName), server.Handler)
	} else {
		fmt.Printf("App listening at http://localhost:%d\n", port)
		fmt.Println("Server url is: ", appCfg.GetServerURL())
		err = server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	<-serverCtx.Done()
}
