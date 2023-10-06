package model

const (
	PageThemeLight  string = "light"
	PageThemeDark          = "dark"
	PageThemeSystem        = "system"
)

const (
	PageContentLayoutFull     string = "full"
	PageContentLayoutCentered        = "centered"
)

type UISettings struct {
	Lang          Lang
	Theme         string
	ContentLayout string
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
	Debug                 bool
	DebugUsers            []*User
	BreadCrumbs           []*BreadCrumb
	BrandName             string
	BrandDomainName       string
	Slogan                string
	PermissionEnabledList []string
}
