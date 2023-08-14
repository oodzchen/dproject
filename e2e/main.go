package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	chp "github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

func logFailed(err error) {
	logErrf("FAILED: %v", err)
}

func logErrf(msg string, err error) {
	if err != nil {
		log.Fatalf(msg, err)
	}
}

func logln(data ...any) {
	fmt.Println(data...)
}

// func deepEqual(want, got any) {
// 	if !reflect.DeepEqual(want, got) {
// 		panic(fmt.Sprintf("want %v, bug got %v\n", want, got))
// 	}
// }

const TIMEOUT_DURATION = 5 * time.Second
const TESTING_PWD string = `123!@#abc`
const SERVER_URL string = `http://localhost:3000`

func main() {
	startTime := time.Now()
	dir, err := os.MkdirTemp("", "chromedp-example")
	defer os.RemoveAll(dir)
	logErrf("create temp dir failed:%v", err)

	opts := append(chp.DefaultExecAllocatorOptions[:],
		chp.DisableGPU,
		chp.UserDataDir(dir),
		chp.Flag("headless", true),
	)

	allocCtx, cancel := chp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chp.NewContext(allocCtx, chp.WithLogf(log.Printf))
	defer cancel()

	ctx, tcancel := context.WithTimeout(ctx, TIMEOUT_DURATION)
	defer tcancel()

	var content string
	err = runTasks("visit article and get content", ctx,
		chp.Navigate(SERVER_URL),
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
	logFailed(err)

	err = runTasks("add new as anonymous", ctx,
		chp.Navigate(SERVER_URL),
		chp.WaitVisible(`body>footer`),
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(1) > a:nth-child(1)`, chp.NodeVisible),
		chp.WaitVisible(`body>footer`),
		chp.WaitVisible(`#password`, chp.ByID),
	)
	logFailed(err)

	newUser := getRandUser()
	var resultText string
	err = runTasks("register new user", ctx,
		register(newUser),
		chp.TextContent(`#page-tip>span`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(resultText) == 0 {
				return errors.New("empty register success message")
			}
			return nil
		}),
	)
	logFailed(err)

	// if regexp.MustCompile(``).Match([]byte(resultText)){
	// 	logErrf(msg string, err error)
	// }

	err = runTasks("register duplicate user", ctx,
		register(newUser),
		chp.TextContent(`#err-msg`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(resultText) == 0 {
				return errors.New("empty register duplicate message")
			}
			return nil
		}),
	)
	logFailed(err)

	err = runTasks("login", ctx,
		login(newUser),
		chp.TextContent(`ul.nav-menu:nth-child(2) > li:nth-child(2) > a:nth-child(1)`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(resultText) == 0 || resultText != newUser.name {
				return errors.New("user name incorrect")
			}
			return nil
		}),
	)
	logFailed(err)

	err = runTasks("logout", ctx,
		logout(),
		chp.TextContent(`ul.nav-menu:nth-child(2) > li:nth-child(3) > a:nth-child(1)`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != "Login" {
				return errors.New("No login button")
			}
			return nil
		}),
	)
	logFailed(err)

	err = runTasks("user profile", ctx,
		login(newUser),
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
	logFailed(err)

	err = runTasks("create article as anonymous", ctx,
		logout(),
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
	logFailed(err)

	newArticle := genArticle()
	logln("new article: ", newArticle.title)
	err = runTasks("create article as logined user", ctx,
		login(newUser),
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(1) > a:nth-child(1)`),
		chp.WaitVisible(`body>footer`),
		chp.SetValue(`input[name="title"]`, newArticle.title),
		chp.SetValue(`textarea[name="content"]`, newArticle.content),
		chp.Click(`button[type="submit"]`),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`h1`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != newArticle.title {
				return errors.New("new article title incorrect")
			}
			return nil
		}),
	)
	logFailed(err)

	editArticle := genArticle()
	err = runTasks("edit article", ctx,
		chp.Click(`body > article:nth-child(5) > div:nth-child(4) > small:nth-child(1) > a:nth-child(1)`),
		chp.WaitVisible(`body>footer`),
		chp.SetValue(`textarea[name="content"]`, editArticle.content),
		chp.Click(`button[type="submit"]`),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`body > article:nth-child(5) > section:nth-child(3)`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != editArticle.content {
				return errors.New("content not match after edit")
			}
			return nil
		}),
	)
	logFailed(err)

	err = runTasks("visit profile post list", ctx,
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(2) > a:nth-child(1)`),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`body > ul:nth-child(8) > li:nth-child(1) > div:nth-child(2)`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if len(resultText) == 0 {
				return errors.New("user post list is empty")
			}
			return nil
		}),
	)
	logFailed(err)

	newReply := gofakeit.Sentence(5 + rand.Intn(10))
	logln("new reply:", newReply)
	err = runTasks("reply article", ctx,
		chp.NavigateBack(),
		chp.SetValue(`textarea[name="content"]`, newReply),
		chp.Click(`#reply_form>button[type="submit"]`),
		chp.WaitVisible(`body>footer`),
		chp.TextContent(`ul.replies:nth-child(8) > li:last-child > article:nth-child(1) > section:nth-child(2)`, &resultText),
		chp.ActionFunc(func(ctx context.Context) error {
			if resultText != newReply {
				return errors.New("reply content incorrect")
			}
			return nil
		}),
	)
	logFailed(err)

	testingDuration := time.Now().Sub(startTime)
	fmt.Printf("OK, all pass! Testing duration: %fs\n", testingDuration.Seconds())
}

func runTasks(name string, ctx context.Context, actions ...chp.Action) error {
	logln("Task: " + name)

	err := chp.Run(ctx, actions...)

	if err != nil {
		return err
	}
	logln("PASS: ", name)
	logln("")
	return nil
}

func register(data *testUser) chp.Tasks {
	return chp.Tasks{
		chp.Navigate(SERVER_URL),
		chp.WaitVisible(`body>footer`),
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(2) > a:nth-child(1)`, chp.NodeNotVisible),
		chp.WaitVisible(`body>footer`),
		chp.WaitVisible(`input[name="email"]`),
		chp.SetValue(`input[name="email"]`, data.email),
		chp.SetValue(`input[name="password"]`, TESTING_PWD),
		chp.SetValue(`input[name="username"]`, data.name),
		chp.Click(`button[type="submit"]`, chp.NodeVisible),
		chp.WaitVisible(`body>footer`),
	}
}

func login(data *testUser) chp.Tasks {
	return chp.Tasks{
		chp.Navigate(SERVER_URL),
		chp.WaitVisible(`body>footer`),
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(3) > a:nth-child(1)`),
		chp.SetValue(`input[name="email"]`, data.email),
		chp.SetValue(`input[name="password"]`, TESTING_PWD),
		chp.Click(`button[type="submit"]`),
		chp.WaitVisible(`body>footer`),
	}
}

func logout() chp.Tasks {
	return chp.Tasks{
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(3) > a:nth-child(1)`),
		chp.WaitVisible(`body>footer`),
	}
}
