package model

import "net/url"

const (
	PageThemeLight  string = "light"
	PageThemeDark          = "dark"
	PageThemeSystem        = "system"
)

const (
	PageContentLayoutFull     string = "full"
	PageContentLayoutCentered        = "centered"
)

var DefaultUiSettings = &UISettings{
	Lang:           LangEn,
	Theme:          PageThemeSystem,
	ContentLayout:  PageContentLayoutCentered,
	FontSize:       16,
	FontSizeCustom: false,
}

type UISettings struct {
	Lang           Lang
	Theme          string
	ContentLayout  string
	FontSize       int
	FontSizeCustom bool
}

type BreadCrumb struct {
	Path string
	Name string
}

type PageData struct {
	Title                 string
	Data                  any
	TipMsg                []string
	LoginedUser           *User
	JSONStr               string
	CSRFField             string
	UISettings            *UISettings
	RoutePath             string
	RouteQuery            url.Values
	RouteRawQuery         string
	Debug                 bool
	DebugUsers            []*User
	BreadCrumbs           []*BreadCrumb
	BrandName             string
	BrandDomainName       string
	Slogan                string
	PermissionEnabledList []string
	MessageCount          int
}
