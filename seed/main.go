package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	chp "github.com/chromedp/chromedp"
	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/mocktool"
)

const timeoutDuration int = 300

var articleNum int
var userNum int
var envFile string
var showHead bool

var mock *mocktool.Mock

func init() {
	const defaultArticleNum int = 16
	const defaultUserNum int = 30
	const defaultEnvFile string = ".env.local"
	
	flag.IntVar(&articleNum, "an", defaultArticleNum, "Create article with specific number")
	flag.IntVar(&userNum, "un", defaultUserNum, "User(goroutine) number")
	flag.StringVar(&envFile, "e", defaultEnvFile, "ENV file path, default to .env.local")
	flag.BoolVar(&showHead, "h", false, "Show browser head")
}

func main() {
	flag.Parse()

	fmt.Printf("Creating article number: %d\n", articleNum)
	fmt.Printf("User(goroutine) number: %d\n", userNum)
	fmt.Printf("ENV file path: %s\n", envFile)

	cfg, err := config.Parse(envFile)
	if err != nil{
		log.Fatal(err)
	}

	// fmt.Println("App config: ", cfg)
	
	mock = mocktool.NewMock(cfg)

	// fmt.Println("Mock: ", mock)

	startTime := time.Now()

	dir, err := os.MkdirTemp("", "chromedp-temp")
	defer os.RemoveAll(dir)
	if err != nil {
		log.Fatal(err)
	}

	opts := append(chp.DefaultExecAllocatorOptions[:],
		chp.DisableGPU,
		chp.UserDataDir(dir),
		chp.Flag("headless", !showHead),
	)

	allocCtx, cancel := chp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	var userPool []*userCtx
	var wg sync.WaitGroup
	userResults := make(chan *userCtx, userNum)

	for i := 0; i < userNum; i++ {
		wg.Add(1)
		user := mocktool.GenUser()
		go registerUser(allocCtx, user, &wg, userResults)
	}

	go func() {
		wg.Wait()
		close(userResults)
	}()

	for res := range userResults {
		fmt.Println("result from user channel: ", res)
		if res != nil {
			defer res.cancel()
			defer res.tcancel()

			userPool = append(userPool, res)
		}
	}

	fmt.Printf("Register complete. Successed: %d, Failed: %d\n", len(userPool), userNum-len(userPool))

	var aWg sync.WaitGroup
	articleResult := make(chan error, articleNum)
	articleQueue := make(chan *mocktool.TestArticle, articleNum)

	failedNum := 0
	for i := 0; i < len(userPool); i++ {
		aWg.Add(1)
		uCtx := userPool[i]
		go createSampleArticle(&aWg, uCtx.ctx, uCtx.user, articleQueue, articleResult)
	}

	go func() {
		for res := range articleResult {
			fmt.Println("result from article channel: ", res)
			if res != nil {
				failedNum += 1
			}
		}

		fmt.Printf("Create article complete. Successed: %d, Failed: %d\n", articleNum-failedNum, failedNum)

		seedDuration := time.Now().Sub(startTime)
		fmt.Printf("Total duration: %fs\n", seedDuration.Seconds())
	}()

	for i := 0; i < articleNum; i++ {
		// fmt.Println("article queue: ", i)
		articleQueue <- mocktool.GenArticle()
	}
	close(articleQueue)

	aWg.Wait()
	close(articleResult)
}

type userCtx struct {
	ctx     context.Context
	cancel  context.CancelFunc
	tcancel context.CancelFunc
	user    *mocktool.TestUser
}

func registerUser(ctx context.Context, u *mocktool.TestUser, wg *sync.WaitGroup, results chan<- *userCtx) {
	defer wg.Done()
	
	ctx, cancel := chp.NewContext(ctx, chp.WithLogf(log.Printf))

	ctx, tcancel := context.WithTimeout(ctx, time.Duration(timeoutDuration*int(time.Second)))

	fmt.Println("register user: ", u)
	err := chp.Run(ctx,
		chp.Navigate(mock.ServerURL),
		chp.WaitVisible(`body > footer`),
		mock.MustLogout(),
		mock.Register(u),
	)
	
	if err != nil {
		fmt.Printf("register user failed: \n\tuser:%v\n\terror:%v\n", u, err)
		results <- nil
		tcancel()
		return
	}
	results <- &userCtx{ ctx, cancel, tcancel, u }
}

func createSampleArticle(wg *sync.WaitGroup, ctx context.Context, u *mocktool.TestUser, ach <-chan *mocktool.TestArticle, results chan<- error) {
	defer wg.Done()

	for a := range ach {
		fmt.Printf("user %v create article \"%s\"\n", u, a.Title)
		err := chp.Run(ctx,
			mock.MustLogout(),
			mock.Login(u),
			mock.CreateArticle(a),
			mock.Logout(),
		)

		results <- err
	}
}
