package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	chp "github.com/chromedp/chromedp"
	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/mocktool"
	mt "github.com/oodzchen/dproject/mocktool"
	"github.com/pkg/errors"
)

// func deepEqual(want, got any) {
// 	if !reflect.DeepEqual(want, got) {
// 		panic(fmt.Sprintf("want %v, bug got %v\n", want, got))
// 	}
// }

var showHead bool
var timeoutDuration int

var envFile string
var mock *mocktool.Mock

var successRE = regexp.MustCompile("/successfully/")

func init() {
	const defaultTimeoutDuration int = 6
	const defaultSowHead = false
	const defaultEnvFile = ".env.testing"

	flag.BoolVar(&showHead, "h", defaultSowHead, "Show browser head")
	flag.IntVar(&timeoutDuration, "t", defaultTimeoutDuration, "Timeout duration")
	flag.StringVar(&envFile, "e", defaultEnvFile, "App env file")
}

func main() {
	startTime := time.Now()
	dir, err := os.MkdirTemp("", "chromedp-example")
	defer os.RemoveAll(dir)
	mt.LogErrf("create temp dir failed:%v", err)

	flag.Parse()

	fmt.Printf("Show browser head: %t\n", showHead)
	fmt.Printf("Timeout duration: %ds\n", timeoutDuration)
	fmt.Printf("ENV file: %s\n", envFile)

	_, err = os.Stat(envFile)

	var cfg *config.AppConfig
	if os.IsNotExist(err) {
		cfg, err = config.ParseFromEnv()
	} else {
		fmt.Printf("ENV file path: %s\n", envFile)
		cfg, err = config.Parse(envFile)
	}
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Debug mode: ", cfg.Debug)
	fmt.Println("Testing mode: ", cfg.Testing)

	mock = mt.NewMock(cfg)

	fmt.Printf("Server URL: %s\n", mock.ServerURL)

	opts := append(chp.DefaultExecAllocatorOptions[:],
		chp.DisableGPU,
		chp.UserDataDir(dir),
		chp.Flag("headless", !showHead),
	)

	allocCtx, cancel := chp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chp.NewContext(allocCtx, chp.WithLogf(log.Printf))
	defer cancel()

	ctx, tcancel := context.WithTimeout(ctx, time.Duration(timeoutDuration*int(time.Second)))
	defer tcancel()

	var content string
	err = runTasks("visit article and get content", ctx,
		chp.Navigate(mock.ServerURL),
		mock.WaitFooterReady(),
		chp.Click(`ol>li .article-list__title`, chp.NodeReady),
		mock.WaitFooterReady(),
		chp.TextContent(`body>article>section`, &content),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(content) == 0 {
				return errors.New("article content is empty")
			}
			return nil
		}),
	)
	mt.LogFailed(err)

	err = runTasks("add new as anonymous", ctx,
		chp.Navigate(mock.ServerURL),
		mock.WaitFooterReady(),
		chp.Click(`ul.nav-menu:nth-child(2) > li > a[href^="/articles/new"]`, chp.NodeReady),
		mock.WaitFooterReady(),
		chp.WaitReady(`#password`, chp.ByID),
	)
	mt.LogFailed(err)

	newUser := mt.GenUser()
	var resultText string
	err = runTasks("register new user", ctx,
		mock.Register(newUser),
		chp.SetValue(`input[name="code"]`, config.SuperCode),
		chp.Click(`body>form>button[type="submit"]`),
		mock.WaitFooterReady(),
		chp.TextContent(`#page-tip>span`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			mt.Logln("new user: ", newUser)
			if len(resultText) == 0 {
				return errors.New("empty register success message")
			}
			return nil
		}),
	)
	mt.LogFailed(err)

	// if regexp.MustCompile(``).Match([]byte(resultText)){
	// 	mt.LogErrf(msg string, err error)
	// }

	err = runTasks("register duplicate user", ctx,
		mock.Register(newUser),
		chp.TextContent(`#err-msg`, &resultText, chp.ByID),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(resultText) == 0 {
				return errors.New("empty register duplicate message")
			}
			return nil
		}),
	)
	mt.LogFailed(err)

	err = runTasks("login", ctx,
		mock.Login(newUser),
		chp.TextContent(`ul.nav-menu:nth-child(2) > li > a[href^="/users/"]`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(resultText) == 0 {
				return errors.New("user name incorrect")
			}
			return nil
		}),
	)
	mt.LogFailed(err)

	err = runTasks("logout", ctx,
		mock.Logout(),
		chp.TextContent(`ul.nav-menu:nth-child(2) > li > a[href^="/login"]`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != "Login" {
				return errors.New("No login button")
			}
			return nil
		}),
	)
	mt.LogFailed(err)

	err = runTasks("user profile", ctx,
		mock.Login(newUser),
		chp.Click(`ul.nav-menu:nth-child(2) > li > a[href^="/users/"]`),
		mock.WaitFooterReady(),
	)
	mt.LogFailed(err)

	err = runTasks("create article as anonymous", ctx,
		mock.Logout(),
		chp.Click(`ul.nav-menu:nth-child(2) > li > a[href^="/articles/new"]`),
		mock.WaitFooterReady(),
		chp.TextContent(`button[type=submit]`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != "Login" {
				return errors.New("No login button")
			}
			return nil
		}),
	)
	mt.LogFailed(err)

	newArticle := mt.GenArticle()
	err = runTasks("create article as logined user", ctx,
		mock.Login(newUser),
		mock.CreateArticle(newArticle),
	)
	mt.LogFailed(err)

	editArticle := mt.GenArticle()
	var resultTitle string
	err = runTasks("edit article title and content", ctx,
		chp.Click(`body > article > div > small > a.btn-edit`),
		mock.WaitFooterReady(),
		chp.SetValue(`input[name="title"]`, editArticle.Title),
		chp.SetValue(`textarea[name="content"]`, editArticle.Content),
		chp.Click(`body > form > button[type="submit"]`),
		mock.WaitFooterReady(),
		chp.TextContent(`body > article > h1`, &resultTitle),
		chp.TextContent(`body > article > section`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != editArticle.Content || resultTitle != editArticle.Title {
				return errors.New("content not match after edit")
			}
			return nil
		}),
	)
	mt.LogFailed(err)

	err = runTasks("visit profile post list", ctx,
		chp.Click(`ul.nav-menu:nth-child(2) > li > a[href^="/users/"]`),
		mock.WaitFooterReady(),
		chp.TextContent(`body > ul > li:nth-child(1) > div:last-child`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(resultText) == 0 {
				return errors.New("user post list is empty")
			}
			return nil
		}),
	)
	mt.LogFailed(err)

	newReply := gofakeit.Sentence(5 + rand.Intn(10))
	err = runTasks("reply article", ctx,
		chp.Navigate(mock.ServerURL),
		chp.Click(`body > .tabs > a[title^="Hot"]`),
		mock.WaitFooterReady(),
		chp.Click(`body > .article-list > li .article-list__title`),
		mock.WaitFooterReady(),
		chp.Click(`body > article > .article-operation a[title="Reply"]`),
		mock.WaitFooterReady(),
		chp.SetValue(`#content`, newReply, chp.ByID),
		chp.Click(`#reply_form button[type="submit"]`),
		mock.WaitFooterReady(),
		chp.TextContent(`#page-tip>span`, &resultText, chp.ByQuery),
		chp.ActionFunc(func(ctx context.Context) error {
			if successRE.Match([]byte(resultText)) {
				return errors.New("reply article failed")
			}
			return nil
		}),
	)
	mt.LogFailed(err)

	// var currUrl string
	newReply = gofakeit.Sentence(5 + rand.Intn(10))
	err = runTasks("reply comment", ctx,
		chp.Navigate(mock.ServerURL),
		chp.Click(`body > .tabs > a[title^="Hot"]`),
		mock.WaitFooterReady(),
		chp.Click(`body > .article-list > li .article-list__title`),
		mock.WaitFooterReady(),
		chp.Click(`#replies-box > li:first-child > article > .article-operation a[title="Reply"]`),
		mock.WaitFooterReady(),
		chp.SetValue(`#content`, newReply, chp.ByID),
		chp.Click(`#reply_form button[type="submit"]`),
		mock.WaitFooterReady(),
		chp.TextContent(`#page-tip>span`, &resultText, chp.ByQuery),
		chp.ActionFunc(func(ctx context.Context) error {
			if successRE.Match([]byte(resultText)) {
				return errors.New("reply comment failed")
			}
			return nil
		}),
	)
	mt.LogFailed(err)

	err = runTasks("delete article", ctx,
		chp.Click(`ul.nav-menu:nth-child(2) > li > a[href^="/users/"]`),
		mock.WaitFooterReady(),
		chp.Click(`body > ul > li:last-child > div:first-child > a`),
		mock.WaitFooterReady(),
		chp.Click(`.btn-del`),
		mock.WaitFooterReady(),
		chp.Click(`body>form>button[type=submit]`),
		mock.WaitFooterReady(),
		chp.TextContent(`body>article>i`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != "<Deleted>" {
				return errors.New("delete article failed")
			}
			return nil
		}),
	)
	mt.LogFailed(err)

	testingDuration := time.Now().Sub(startTime)
	fmt.Printf("OK, all pass! Testing duration: %fs\n", testingDuration.Seconds())
}

func runTasks(name string, ctx context.Context, actions ...chp.Action) error {
	mt.Logln("Task: " + name)

	err := chp.Run(ctx, actions...)

	if err != nil {
		return err
	}
	mt.Logln("PASS: ", name)
	mt.Logln("")
	return nil
}
