package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	chp "github.com/chromedp/chromedp"
	"github.com/oodzchen/dproject/mocktool"
)

const timeoutDuration int = 300

var articleNum int

func init() {
	const defaultArticleNum int = 16
	flag.IntVar(&articleNum, "article-num", defaultArticleNum, "Create article with specific number")
	flag.IntVar(&articleNum, "an", defaultArticleNum, "Create article with specific number")
}

func main() {
	flag.Parse()

	fmt.Printf("Creating article number: %d\n", articleNum)

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

	ctx, cancel := chp.NewContext(allocCtx, chp.WithLogf(log.Printf))
	defer cancel()

	ctx, tcancel := context.WithTimeout(ctx, time.Duration(timeoutDuration*int(time.Second)))
	defer tcancel()

	err = chp.Run(ctx,
		chp.Navigate(mocktool.ServerURL),
		chp.WaitVisible(`body > footer`),
	)
	if err != nil {
		log.Fatal(err)
	}

	const userNum int = 30
	var userPool []*mocktool.TestUser

	for i := 0; i < userNum; i++ {
		user := mocktool.GenUser()
		fmt.Println("register user: ", user)
		err := chp.Run(ctx,
			mocktool.MustLogout(),
			mocktool.Register(user),
		)
		if err != nil {
			fmt.Printf("register user %v failed: %v\n", user, err)
			continue
		}
		userPool = append(userPool, user)
	}

	failedNum := 0
	for i := 0; i < articleNum; i++ {
		user := userPool[rand.Intn(len(userPool))]
		article := mocktool.GenArticle()
		fmt.Printf("user %v create article \"%s\"\n", user, article.Title)

		err := createSampleArticle(ctx, user, article)
		if err != nil {
			failedNum += 1
			continue
		}
	}

	fmt.Printf("Create article complete. Successed: %d, Failed: %d\n", articleNum-failedNum, failedNum)
}

func createSampleArticle(ctx context.Context, u *mocktool.TestUser, a *mocktool.TestArticle) error {
	return chp.Run(ctx,
		mocktool.MustLogout(),
		// mocktool.Register(u),
		mocktool.Login(u),
		mocktool.CreateArticle(a),
		mocktool.Logout(),
	)
}
