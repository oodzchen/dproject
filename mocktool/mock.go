package mocktool

import (
	"context"
	"errors"

	chp "github.com/chromedp/chromedp"
	"github.com/oodzchen/dproject/config"
)

type Mock struct {
	cfg        *config.AppConfig
	TestingPWD string
	ServerURL  string
}

func NewMock(cfg *config.AppConfig) *Mock {
	testingPWD := cfg.DB.UserDefaultPassword
	// fmt.Println("test password: ", testingPWD)
	serverURL := cfg.GetServerURL()
	return &Mock{cfg: cfg, TestingPWD: testingPWD, ServerURL: serverURL}
}

func (mc *Mock) Register(data *TestUser) chp.Tasks {
	return chp.Tasks{
		chp.Navigate(mc.ServerURL),
		mc.WaitFooterReady(),
		chp.Click(`ul.nav-menu:nth-child(2) > li > a[href^="/register"]`, chp.NodeNotVisible),
		mc.WaitFooterReady(),
		chp.WaitReady(`input[name="email"]`),
		chp.SetValue(`input[name="email"]`, data.Email),
		chp.SetValue(`input[name="password"]`, mc.TestingPWD),
		chp.SetValue(`input[name="username"]`, data.Name),
		chp.Click(`body>form>button[type="submit"]`),
		mc.WaitFooterReady(),
	}
}

func (mc *Mock) Login(data *TestUser) chp.Tasks {
	// Logln("login user: ", data)
	return chp.Tasks{
		chp.Navigate(mc.ServerURL),
		mc.WaitFooterReady(),
		chp.Click(`ul.nav-menu:nth-child(2) > li > a[href^="/login"]`),
		chp.SetValue(`input[name="email"]`, data.Email),
		chp.SetValue(`input[name="password"]`, mc.TestingPWD),
		chp.Click(`#login-form>button[type="submit"]`),
		mc.WaitFooterReady(),
	}
}

func (mc *Mock) Logout() chp.Tasks {
	return chp.Tasks{
		chp.Click(`ul.nav-menu:nth-child(2) > li > form[action="/logout"] > button[type="submit"]`),
		mc.WaitFooterReady(),
	}
}

func (mc *Mock) MustLogout() chp.Tasks {
	var result string
	return chp.Tasks{
		chp.TextContent(`ul.nav-menu:nth-child(2) > li:nth-child(4) > a`, &result),
		chp.ActionFunc(func(ctx context.Context) error {
			// fmt.Println("logout button text: ", result)
			if result == "Logout" {
				return chp.Run(ctx, mc.Logout())
			}
			return nil
		}),
	}
}

func (mc *Mock) CreateArticle(a *TestArticle) chp.Tasks {
	// Logln("create article: \"", a.Title, "\"")
	var result string
	return chp.Tasks{
		chp.Click(`ul.nav-menu:nth-child(2) > li > a[href^="/articles/new"]`),
		mc.WaitFooterReady(),
		chp.SetValue(`#title`, a.Title),
		chp.SetValue(`#content`, a.Content),
		chp.Click(`body>form>button[type="submit"]`),
		mc.WaitFooterReady(),
		chp.TextContent(`body>article>h1`, &result),
		chp.ActionFunc(func(ctx context.Context) error {
			// Logln("\nnew article: ", a.Title)
			// Logln("resulteText: \n", result)
			if result != a.Title {
				return errors.New("create article failed, new article title incorrect")
			}
			return nil
		}),
	}
}

func (mc *Mock) WaitFooterReady() chp.Tasks {
	return chp.Tasks{
		chp.WaitReady(`body > footer`, chp.ByQuery),
	}
}
