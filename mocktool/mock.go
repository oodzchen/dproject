package mocktool

import (
	"context"
	"errors"

	chp "github.com/chromedp/chromedp"
	"github.com/oodzchen/dproject/config"
)

type Mock struct {
	cfg *config.AppConfig
	TestingPWD string
	ServerURL string
}

func NewMock(cfg *config.AppConfig) *Mock{
	testingPWD := cfg.DB.UserDefaultPassword
	serverURL := cfg.GetServerURL()
	return &Mock{ cfg: cfg, TestingPWD: testingPWD, ServerURL: serverURL }
}

func (mc *Mock)Register(data *TestUser) chp.Tasks {
	return chp.Tasks{
		chp.Navigate(mc.ServerURL),
		chp.WaitVisible(`body>footer`),
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(2) > a:nth-child(1)`, chp.NodeNotVisible),
		chp.WaitVisible(`body>footer`),
		chp.WaitVisible(`input[name="email"]`),
		chp.SetValue(`input[name="email"]`, data.Email),
		chp.SetValue(`input[name="password"]`, mc.TestingPWD),
		chp.SetValue(`input[name="username"]`, data.Name),
		chp.Click(`button[type="submit"]`, chp.NodeVisible),
		chp.WaitVisible(`body>footer`),
	}
}

func (mc *Mock)Login(data *TestUser) chp.Tasks {
	// Logln("login user: ", data)
	return chp.Tasks{
		chp.Navigate(mc.ServerURL),
		chp.WaitVisible(`body>footer`),
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(3) > a:nth-child(1)`),
		chp.SetValue(`input[name="email"]`, data.Email),
		chp.SetValue(`input[name="password"]`, mc.TestingPWD),
		chp.Click(`button[type="submit"]`),
		chp.WaitVisible(`body>footer`),
	}
}

func (mc *Mock)Logout() chp.Tasks {
	return chp.Tasks{
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(3) > a:nth-child(1)`),
		chp.WaitVisible(`body>footer`),
	}
}

func (mc *Mock)MustLogout() chp.Tasks {
	var result string
	return chp.Tasks{
		chp.TextContent(`ul.nav-menu:nth-child(2) > li:nth-child(3) > a:nth-child(1)`, &result),
		chp.ActionFunc(func(ctx context.Context) error {
			// fmt.Println("logout button text: ", result)
			if result == "Logout" {
				return chp.Run(ctx, mc.Logout())
			}
			return nil
		}),
	}
}

func (mc *Mock)CreateArticle(a *TestArticle) chp.Tasks {
	// Logln("create article: \"", a.Title, "\"")
	var result string
	return chp.Tasks{
		chp.Click(`ul.nav-menu:nth-child(2) > li:nth-child(1) > a:nth-child(1)`),
		chp.WaitVisible(`body>footer`),
		chp.SetValue(`body>form>input[name="title"]`, a.Title),
		chp.SetValue(`body>form>textarea[name="content"]`, a.Content),
		chp.Click(`button[type="submit"]`),
		chp.WaitVisible(`body>footer`),
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
