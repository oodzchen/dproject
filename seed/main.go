package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/mocktool"
	"github.com/oodzchen/dproject/service"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/store/pgstore"
)

const timeoutDuration int = 300

var articleNum int
var userNum int
var envFile string
var showHead bool

var mock *mocktool.Mock

var replyCmd *flag.FlagSet
var replyNum int
var replyNumOfReply int
var replyToId int
var maxLevel int

func init() {
	const defaultArticleNum int = 16
	const defaultUserNum int = 30
	const defaultEnvFile string = ".env.testing"

	flag.IntVar(&articleNum, "an", defaultArticleNum, "Create article with specific number")
	flag.IntVar(&userNum, "un", defaultUserNum, "User(goroutine) number")
	flag.StringVar(&envFile, "e", defaultEnvFile, "ENV file path, default to .env.testing")
	flag.BoolVar(&showHead, "h", false, "Show browser head")

	replyCmd = flag.NewFlagSet("reply", flag.ExitOnError)
	replyCmd.IntVar(&replyNum, "n", 10, "Number of replies")
	replyCmd.IntVar(&replyToId, "t", 0, "Article id to reply")
	replyCmd.IntVar(&maxLevel, "l", 0, "Max nested reply level")
	replyCmd.IntVar(&replyNumOfReply, "rnr", 1, "reply number of reply")
}

func main() {
	flag.Parse()

	_, err := os.Stat(envFile)

	if os.IsNotExist(err) {
		err = config.InitFromEnv()
	} else {
		fmt.Printf("ENV file path: %s\n", envFile)
		err = config.Init(envFile)
	}

	if err != nil {
		log.Fatal(err)
	}
	cfg := config.Config

	// fmt.Println("App config: ", cfg)

	mock = mocktool.NewMock(cfg)

	// fmt.Println("Mock: ", mock)

	startTime := time.Now()

	// ----------------------- by headless brwoser -------------------------------
	// dir, err := os.MkdirTemp("", "chromedp-temp")
	// defer os.RemoveAll(dir)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// opts := append(chp.DefaultExecAllocatorOptions[:],
	// 	chp.DisableGPU,
	// 	chp.UserDataDir(dir),
	// 	chp.Flag("headless", !showHead),
	// )

	// allocCtx, cancel := chp.NewExecAllocator(context.Background(), opts...)
	// defer cancel()

	// ----------------------- by database operations -------------------------------
	pg := pgstore.New(&pgstore.DBConfig{
		DSN: cfg.DB.GetDSN(),
	})

	err = pg.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer pg.CloseDB()

	err = pg.InitModules()
	if err != nil {
		log.Fatal(err)
	}

	dataStore := &store.Store{
		Activity:   pg.Activity,
		Article:    pg.Article,
		Message:    pg.Message,
		Permission: pg.Permission,
		Role:       pg.Role,
		User:       pg.User,
	}

	policy := bluemonday.UGCPolicy()
	userSrv := &service.User{Store: dataStore, SantizePolicy: policy}
	articleSrv := &service.Article{Store: dataStore, SantizePolicy: policy}

	fmt.Println("os.Args", os.Args)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "reply":
			replyCmd.Parse(os.Args[2:])
			fmt.Println("Reply to article", replyToId, "with", replyNum, "replies")
			fmt.Println("Max level: ", maxLevel)
			fmt.Println("Reply number of reply: ", replyNumOfReply)
			if replyToId == 0 {
				log.Fatal("Article id is required")
			}
			replyArticle(userSrv, articleSrv)
		default:
			seedArticles(userSrv, articleSrv, startTime)
		}
	} else {
		seedArticles(userSrv, articleSrv, startTime)
	}
}
