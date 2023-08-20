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
	"github.com/oodzchen/dproject/mocktool"
)

const timeoutDuration int = 300

var articleNum int
var parallelNum int

func init() {
	const defaultArticleNum int = 16
	const defaultParallel int = 30
	flag.IntVar(&articleNum, "article-num", defaultArticleNum, "Create article with specific number")
	flag.IntVar(&articleNum, "an", defaultArticleNum, "Create article with specific number")
	flag.IntVar(&parallelNum, "parallel", defaultParallel, "Goroutine number")
	flag.IntVar(&parallelNum, "p", defaultParallel, "Goroutine number")
}

func main() {
	flag.Parse()

	fmt.Printf("Creating article number: %d\n", articleNum)
	fmt.Printf("Parallel number: %d\n", parallelNum)

	startTime := time.Now()

	dir, err := os.MkdirTemp("", "chromedp-example")
	defer os.RemoveAll(dir)
	if err != nil {
		log.Fatal(err)
	}

	opts := append(chp.DefaultExecAllocatorOptions[:],
		chp.DisableGPU,
		chp.UserDataDir(dir),
		chp.Flag("headless", true),
	)

	allocCtx, cancel := chp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	userNum := parallelNum
	var userPool []*userCtx
	var wg sync.WaitGroup
	userResults := make(chan *userCtx)

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
		// uCtx := userPool[i]
		// article := mocktool.GenArticle()
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
	uCtx := new(userCtx)
	uCtx.user = u

	ctx, cancel := chp.NewContext(ctx, chp.WithLogf(log.Printf))

	uCtx.cancel = cancel
	// defer cancel()

	ctx, tcancel := context.WithTimeout(ctx, time.Duration(timeoutDuration*int(time.Second)))
	uCtx.tcancel = tcancel
	// defer tcancel()

	uCtx.ctx = ctx

	defer wg.Done()

	fmt.Println("register user: ", u)
	err := chp.Run(ctx,
		chp.Navigate(mocktool.ServerURL),
		chp.WaitVisible(`body > footer`),
		mocktool.MustLogout(),
		mocktool.Register(u),
	)
	if err != nil {
		results <- nil
		fmt.Printf("register user failed: \n\tuser:%v\n\terror:%v\n", u, err)
		return
	}
	results <- uCtx
}

func createSampleArticle(wg *sync.WaitGroup, ctx context.Context, u *mocktool.TestUser, ach <-chan *mocktool.TestArticle, results chan<- error) {
	defer wg.Done()

	for a := range ach {
		fmt.Printf("user %v create article \"%s\"\n", u, a.Title)
		err := chp.Run(ctx,
			mocktool.MustLogout(),
			mocktool.Login(u),
			mocktool.CreateArticle(a),
			mocktool.Logout(),
		)

		results <- err
	}
}
