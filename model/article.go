package model

import (
	"regexp"
	"time"

	"github.com/oodzchen/dproject/utils"
)

type Article struct {
	Id           int
	Title        string
	AuthorName   string
	AuthorId     int
	Content      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedAtStr string
	UpdatedAtStr string
}

func (a *Article) FormatTimeStr() {
	a.CreatedAtStr = utils.FormatTime(a.CreatedAt, "Y年M月D日 h时m分s秒")
	a.UpdatedAtStr = utils.FormatTime(a.UpdatedAt, "Y年M月D日 h时m分s秒")
}

func (a *Article) TransformNewlines() {
	re := regexp.MustCompile(`\r`)
	a.Content = re.ReplaceAllString(a.Content, "<br/>")
}
