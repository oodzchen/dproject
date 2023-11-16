package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	chp "github.com/chromedp/chromedp"
	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/mocktool"
	"github.com/oodzchen/dproject/service"
)

func seedArticles(userSrv *service.User, articleSrv *service.Article, startTime time.Time) {
	fmt.Printf("Creating article number: %d\n", articleNum)
	fmt.Printf("User(goroutine) number: %d\n", userNum)

	var userPool []*userCtx
	var wg sync.WaitGroup
	userResults := make(chan *userCtx, userNum)

	for i := 0; i < userNum; i++ {
		wg.Add(1)
		user := mocktool.GenUser()
		// go browserRegisterUser(allocCtx, user, &wg, userResults)
		go srvRegisterUser(context.Background(), userSrv, user, &wg, userResults)
	}

	go func() {
		wg.Wait()
		close(userResults)
	}()

	for res := range userResults {
		fmt.Println("result from user channel: ", res.userId)
		if res != nil {
			defer res.cancel()
			defer res.tcancel()

			userPool = append(userPool, res)
		}
	}

	fmt.Printf("Register complete. Successed: %d, Failed: %d\n", len(userPool), userNum-len(userPool))

	var aWg sync.WaitGroup
	articleResult := make(chan *articleRes, articleNum)
	articleQueue := make(chan *mocktool.TestArticle, articleNum)

	failedNum := 0
	for i := 0; i < len(userPool); i++ {
		aWg.Add(1)
		uCtx := userPool[i]
		// go browserCreateSampleArticle(&aWg, uCtx.ctx, uCtx.user, articleQueue, articleResult)
		go srvCreateArticle(articleSrv, &aWg, uCtx.ctx, uCtx.userId, articleQueue, articleResult)

	}

	go func() {
		for res := range articleResult {
			if res.err != nil {
				fmt.Println("result from article error: ", res.err)
				failedNum += 1
			} else {
				fmt.Println("result from article channel: ", res.articleId)
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

type articleRes struct {
	articleId int
	err       error
}

type userCtx struct {
	ctx     context.Context
	cancel  context.CancelFunc
	tcancel context.CancelFunc
	user    *mocktool.TestUser
	userId  int
}

func srvRegisterUser(ctx context.Context, srv *service.User, u *mocktool.TestUser, wg *sync.WaitGroup, results chan<- *userCtx) {
	defer wg.Done()
	ctx, cancel := chp.NewContext(ctx, chp.WithLogf(log.Printf))
	ctx, tcancel := context.WithTimeout(ctx, time.Duration(timeoutDuration*int(time.Second)))

	fmt.Println("register user: ", u)
	id, err := srv.Register(u.Email, config.Config.DB.UserDefaultPassword, u.Name)
	if err != nil {
		fmt.Printf("register user failed: \n\tuser:%v\n\terror:%+v\n", u, err)
		results <- nil
		tcancel()
		return
	}

	results <- &userCtx{ctx, cancel, tcancel, u, id}
}

func browserRegisterUser(ctx context.Context, u *mocktool.TestUser, wg *sync.WaitGroup, results chan<- *userCtx) {
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
	results <- &userCtx{ctx, cancel, tcancel, u, 0}
}

func srvCreateArticle(srv *service.Article, wg *sync.WaitGroup, ctx context.Context, authorId int, ach <-chan *mocktool.TestArticle, results chan<- *articleRes) {
	defer wg.Done()

	for a := range ach {
		fmt.Printf("user %v create article \"%s\"\n", authorId, a.Title)
		id, err := srv.Create(a.Title, a.URL, a.Content, authorId, 0, "general")
		results <- &articleRes{id, err}
	}
}

func browserCreateSampleArticle(wg *sync.WaitGroup, ctx context.Context, u *mocktool.TestUser, ach <-chan *mocktool.TestArticle, results chan<- error) {
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
