package model

import (
	"fmt"
	"regexp"
	"time"
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
	c := a.CreatedAt
	u := a.UpdatedAt
	a.CreatedAtStr = fmt.Sprintf("%d年%d月%d日 %d时%d分", c.Year(), c.Month(), c.Day(), c.Hour(), c.Minute())
	a.UpdatedAtStr = fmt.Sprintf("%d年%d月%d日 %d时%d分", u.Year(), u.Month(), u.Day(), u.Hour(), u.Minute())
}

func (a *Article) TransformNewlines() {
	re := regexp.MustCompile(`\r`)
	a.Content = re.ReplaceAllString(a.Content, "<br/>")
}
