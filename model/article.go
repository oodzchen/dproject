package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"github.com/oodzchen/dproject/utils"
	"github.com/xeonx/timeago"
)

const (
	MAX_ARTICLE_TITLE_LEN, MAX_ARTICLE_CONTENT_LEN int = 80, 24000
)

type Article struct {
	Id             int
	Title          string
	AuthorName     string
	AuthorId       int
	Content        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedAtStr   string
	UpdatedAtStr   string
	CreatedTimeAgo string
	UpdatedTimeAgo string
	ReplyTo        int
	Deleted        bool
}

func (a *Article) FormatTimeStr() {
	a.CreatedAtStr = utils.FormatTime(a.CreatedAt, "Y年M月D日 h时m分s秒")
	a.UpdatedAtStr = utils.FormatTime(a.UpdatedAt, "Y年M月D日 h时m分s秒")
	a.CreatedTimeAgo = timeago.English.Format(a.CreatedAt)
	a.UpdatedTimeAgo = timeago.English.Format(a.UpdatedAt)
}

func (a *Article) TransformNewlines() {
	re := regexp.MustCompile(`\r`)
	a.Content = re.ReplaceAllString(a.Content, "<br/>")
}

func (a *Article) Sanitize() {
	p := bluemonday.NewPolicy()
	a.Title = p.Sanitize(a.Title)
	a.Content = p.Sanitize(a.Content)
}

func (a *Article) Valid() error {
	authorId := a.AuthorId
	title := strings.TrimSpace(a.Title)
	content := strings.TrimSpace(a.Content)

	if authorId == 0 {
		return errors.New("author id is required")
	}

	if title == "" {
		return errors.New("article title is required")
	}

	if content == "" {
		return errors.New("article content is required")
	}

	if utf8.RuneCountInString(title) > MAX_ARTICLE_TITLE_LEN {
		return errors.New(fmt.Sprintf("article title limit to %d characters", MAX_ARTICLE_TITLE_LEN))
	}

	if utf8.RuneCountInString(content) > MAX_ARTICLE_CONTENT_LEN {
		return errors.New(fmt.Sprintf("article content limit to %d characters", MAX_ARTICLE_CONTENT_LEN))
	}
	return nil
}
