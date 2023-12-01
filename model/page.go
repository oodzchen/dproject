package model

import (
	"net/url"
	"time"
)

const (
	PageThemeLight  string = "light"
	PageThemeDark          = "dark"
	PageThemeSystem        = "system"
)

const (
	PageContentLayoutFull     string = "full"
	PageContentLayoutCentered        = "centered"
)

const (
	RepliesLayoutTree string = "tree"
	RepliesLayoutTile        = "tile"
)

var DefaultUiSettings = &UISettings{
	Lang:           LangEn,
	Theme:          PageThemeSystem,
	ContentLayout:  PageContentLayoutCentered,
	RepliesLayout:  RepliesLayoutTree,
	FontSize:       14,
	FontSizeCustom: false,
	ShowEmoji:      true,
}

type UISettings struct {
	Lang           Lang   `json:"lang"`
	Theme          string `json:"theme"`
	ContentLayout  string `json:"content_layout"`
	RepliesLayout  string `json:"replies_layout"`
	FontSize       int    `json:"font_size"`
	FontSizeCustom bool   `json:"font_size_custom"`
	ShowEmoji      bool   `json:"show_emoji"`
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
	RespStart             time.Time
	RenderStart           time.Time
}
