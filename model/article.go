package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/microcosm-cc/bluemonday"
	"github.com/oodzchen/dproject/utils"
)

const (
	MAX_ARTICLE_TITLE_LEN, MAX_ARTICLE_CONTENT_LEN int = 80, 24000
)

type VoteType string

const (
	VoteTypeUp   VoteType = "up"
	VoteTypeDown          = "down"
)

var validVoteType = map[VoteType]bool{
	VoteTypeUp:   true,
	VoteTypeDown: true,
}

func IsValidVoteType(t VoteType) bool {
	return validVoteType[t]
}

type CurrUserState struct {
	VoteType     VoteType
	NullVoteType pgtype.Text
}

func (cus *CurrUserState) FormatNullValues() {
	if cus.VoteType == "" && cus.NullVoteType.Valid {
		cus.VoteType = VoteType(cus.NullVoteType.String)
	}
}

type ReplySortType string

const (
	ReplySortBest   ReplySortType = "best"
	ReplySortLatest               = "latest"
)

var replySortMap = map[ReplySortType]bool{
	ReplySortBest:   true,
	ReplySortLatest: true,
}

func ValidReplySort(sortType string) bool {
	return replySortMap[ReplySortType(sortType)]
}

type Article struct {
	Id           int
	Title        string
	NullTitle    pgtype.Text
	AuthorName   string
	AuthorId     int
	Content      string
	Summary      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedAtStr string
	UpdatedAtStr string
	ReplyTo      int
	// ReplyToTitle              string
	// NullReplyToTitle          pgtype.Text
	Deleted                   bool
	Replies                   []*Article
	ReplyDepth                int
	ReplyRootArticleId        int
	NullReplyRootArticleTitle pgtype.Text
	ReplyRootArticleTitle     string
	DisplayTitle              string // only for display
	TotalReplyCount           int
	VoteScore                 int
	Weight                    int
	CurrUserState             *CurrUserState
	SortType                  ReplySortType
}

func (a *Article) FormatNullValues() {
	if a.Title == "" && a.NullTitle.Valid {
		a.Title = a.NullTitle.String
	}

	// if a.ReplyToTitle == "" && a.NullReplyToTitle.Valid {
	// 	a.ReplyToTitle = a.NullReplyToTitle.String
	// }

	if a.ReplyRootArticleTitle == "" && a.NullReplyRootArticleTitle.Valid {
		a.ReplyRootArticleTitle = a.NullReplyRootArticleTitle.String
	}

	if a.CurrUserState != nil {
		a.CurrUserState.FormatNullValues()
	}
}

func (a *Article) UpdateDisplayTitle() {
	var displayTitle string
	if a.ReplyDepth == 0 {
		displayTitle = a.Title
	} else if a.ReplyDepth == 1 {
		displayTitle = fmt.Sprintf("Re: %s", a.ReplyRootArticleTitle)
	} else {
		displayTitle = fmt.Sprintf("Re × %d: %s", a.ReplyDepth, a.ReplyRootArticleTitle)
	}

	a.DisplayTitle = displayTitle
}

func (a *Article) FormatDeleted() {
	if a.Deleted {
		a.Content = ""
	}
}

func (a *Article) GenSummary(strLen int) {
	if utf8.RuneCountInString(a.Content) > strLen {
		a.Summary = string([]rune(a.Content)[:strLen])
	} else {
		a.Summary = a.Content
	}
}

func (a *Article) FormatTimeStr() {
	a.CreatedAtStr = utils.FormatTime(a.CreatedAt, "Y年M月D日 h时m分s秒")
	a.UpdatedAtStr = utils.FormatTime(a.UpdatedAt, "Y年M月D日 h时m分s秒")
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

func (a *Article) Valid(isUpdate bool) error {
	isReply := a.ReplyDepth > 0 || a.ReplyTo != 0
	authorId := a.AuthorId
	title := strings.TrimSpace(a.Title)
	content := strings.TrimSpace(a.Content)

	if !isUpdate && authorId == 0 {
		return errors.New("author id is required")
	}

	if !isReply {
		if title == "" {
			return errors.New("article title is required")
		}

		if utf8.RuneCountInString(title) > MAX_ARTICLE_TITLE_LEN {
			return errors.New(fmt.Sprintf("article title limit to %d characters", MAX_ARTICLE_TITLE_LEN))
		}
	}

	if content == "" {
		return errors.New("article content is required")
	}

	if utf8.RuneCountInString(content) > MAX_ARTICLE_CONTENT_LEN {
		return errors.New(fmt.Sprintf("article content limit to %d characters", MAX_ARTICLE_CONTENT_LEN))
	}
	return nil
}

func (a *Article) CalcWeight() {
	weight := 0
	weight += a.VoteScore

	a.Weight = weight
}

func (a *Article) Len() int {
	return len(a.Replies)
}

func (a *Article) Less(i, j int) bool {
	switch a.SortType {
	case ReplySortLatest:
		compare := a.Replies[i].CreatedAt.Compare(a.Replies[j].CreatedAt)
		return compare > 0
	default:
		return a.Replies[i].Weight > a.Replies[j].Weight
	}

}

func (a *Article) Swap(i, j int) {
	a.Replies[i], a.Replies[j] = a.Replies[j], a.Replies[i]
}
