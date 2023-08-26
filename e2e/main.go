package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	chp "github.com/chromedp/chromedp"
	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/mocktool"
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

func init() {
	const defaultTimeoutDuration int = 6
	const defaultSowHead = false
	const defaultEnvFile = ".env.local"
	
	flag.BoolVar(&showHead, "h", defaultSowHead, "Show browser head")
	flag.IntVar(&timeoutDuration, "t", defaultTimeoutDuration, "Timeout duration")
	flag.StringVar(&envFile, "e", defaultEnvFile, "App env file")
}

func main() {
	startTime := time.Now()
	dir, err := os.MkdirTemp("", "chromedp-example")
	defer os.RemoveAll(dir)
	mocktool.LogErrf("create temp dir failed:%v", err)

	flag.Parse()

	fmt.Printf("Show browser head: %t\n", showHead)
	fmt.Printf("Timeout duration: %ds\n", timeoutDuration)
	fmt.Printf("ENV file: %s\n", envFile)

	cfg, err := config.Parse(envFile)
	if err != nil{
		log.Fatal(err)
	}
	
	mock = mocktool.NewMock(cfg)

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
		chp.WaitVisible(`body>footer`),
		chp.Click(`ol>li:first-child>a`, chp.NodeVisible),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`body>article>section`, &content),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(content) == 0 {
				return errors.New("article content is empty")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	err = runTasks("add new as anonymous", ctx,
		chp.Navigate(mock.ServerURL),
		chp.WaitVisible(`body>footer`),
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(1) > a:nth-child(1)`, chp.NodeVisible),
		chp.WaitVisible(`body>footer`),
		chp.WaitVisible(`#password`, chp.ByID),
	)
	mocktool.LogFailed(err)

	newUser := mocktool.GenUser()
	var resultText string
	err = runTasks("register new user", ctx,
		mock.Register(newUser),
		chp.TextContent(`#page-tip>span`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			mocktool.Logln("new user: ", newUser)
			if len(resultText) == 0 {
				return errors.New("empty register success message")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	// if regexp.MustCompile(``).Match([]byte(resultText)){
	// 	mocktool.LogErrf(msg string, err error)
	// }

	err = runTasks("register duplicate user", ctx,
		mock.Register(newUser),
		chp.TextContent(`#err-msg`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(resultText) == 0 {
				return errors.New("empty register duplicate message")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	err = runTasks("login", ctx,
		mock.Login(newUser),
		chp.TextContent(`ul.nav-menu:nth-child(2) > li:nth-child(2) > a:nth-child(1)`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(resultText) == 0 || resultText != newUser.Name {
				return errors.New("user name incorrect")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	err = runTasks("logout", ctx,
		mock.Logout(),
		chp.TextContent(`ul.nav-menu:nth-child(2) > li:nth-child(3) > a:nth-child(1)`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != "Login" {
				return errors.New("No login button")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	err = runTasks("user profile", ctx,
		mock.Login(newUser),
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(2) > a:nth-child(1)`),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`body > table:nth-child(6) > tbody:nth-child(1) > tr:nth-child(1) > td:nth-child(1)`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(resultText) == 0 || resultText != "Joined At" {
				return errors.New("user data is empty")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	err = runTasks("create article as anonymous", ctx,
		mock.Logout(),
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(1) > a:nth-child(1)`),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`button[type=submit]`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != "Login" {
				return errors.New("No login button")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	newArticle := mocktool.GenArticle()
	err = runTasks("create article as logined user", ctx,
		mock.Login(newUser),
		mock.CreateArticle(newArticle),
	)
	mocktool.LogFailed(err)

	editArticle := mocktool.GenArticle()
	err = runTasks("edit article", ctx,
		chp.Click(`body > article:nth-child(5) > div:nth-child(4) > small:nth-child(1) > a:nth-child(1)`),
		chp.WaitVisible(`body>footer`),
		chp.SetValue(`textarea[name="content"]`, editArticle.Content),
		chp.Click(`button[type="submit"]`),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`body > article:nth-child(5) > section:nth-child(3)`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != editArticle.Content {
				return errors.New("content not match after edit")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	err = runTasks("visit profile post list", ctx,
		chp.Click(`ul.nav-menu:last-child > li:nth-child(2) > a:nth-child(1)`),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`body > ul:nth-child(8) > li:nth-child(1) > div:last-child`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(resultText) == 0 {
				return errors.New("user post list is empty")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	newReply := gofakeit.Sentence(5 + rand.Intn(10))
	err = runTasks("reply article", ctx,
		chp.Click(`body > ul:nth-child(8) > li:last-child > div:first-child > a`),
		chp.WaitVisible(`body>footer`),
		chp.SetValue(`#content`, newReply, chp.ByID),
		chp.Click(`#reply_form>button[type="submit"]`),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`ul.replies > li:last-child > article > section`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			mocktool.Logln("new reply: ", newReply)
			mocktool.Logln("resultText: ", resultText)
			if resultText != newReply {
				return errors.New("reply content is incorrect")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	newReply = gofakeit.Sentence(5 + rand.Intn(10))
	err = runTasks("reply comment", ctx,
		chp.WaitVisible(`body>footer`),
		chp.Click(`ul.replies > li:nth-child(1) > article > section+div > small:last-child > a`),
		chp.WaitVisible(`body>footer`),
		chp.SetValue(`#reply_form>textarea[name="content"]`, newReply),
		chp.Click(`#reply_form>button[type="submit"]`),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`ul.replies > li:last-child > article:nth-child(1) > section:nth-child(2)`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			mocktool.Logln("new reply: ", newReply)
			mocktool.Logln("resultText: ", resultText)
			if resultText != newReply {
				return errors.New("reply content is incorrect")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	err = runTasks("delete article", ctx,
		chp.Click(`ul.nav-menu:last-child > li:nth-child(2) > a:nth-child(1)`),
		chp.WaitVisible(`body>footer`),
		chp.Click(`body > ul:nth-child(8) > li:last-child > div:first-child > a`),
		chp.WaitVisible(`body>footer`),
		chp.Click(`.btn-del`),
		chp.WaitVisible(`body>footer`),
		chp.SetValue(`body>form>input[name="confirm_del"]`, "yes"),
		chp.Click(`body>form>button[type=submit]`),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`body>article>i`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != "<Deleted>" {
				return errors.New("delete article failed")
			}
			return nil
		}),
	)
	mocktool.LogFailed(err)

	testingDuration := time.Now().Sub(startTime)
	fmt.Printf("OK, all pass! Testing duration: %fs\n", testingDuration.Seconds())
}

func runTasks(name string, ctx context.Context, actions ...chp.Action) error {
	mocktool.Logln("Task: " + name)

	err := chp.Run(ctx, actions...)

	if err != nil {
		return err
	}
	mocktool.Logln("PASS: ", name)
	mocktool.Logln("")
	return nil
}
