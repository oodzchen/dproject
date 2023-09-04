package model

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
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

type ArticleSortType string

const (
	ReplySortBest  ArticleSortType = "best"
	ListSortLatest                 = "latest"
	ListSortBest                   = "list_best"
)

var replySortMap = map[ArticleSortType]bool{
	ReplySortBest:  true,
	ListSortLatest: true,
	ListSortBest:   true,
}

func ValidReplySort(sortType string) bool {
	return replySortMap[ArticleSortType(sortType)]
}

type ArticleList struct {
	List      []*Article
	SortType  ArticleSortType
	CurrPage  int
	PageSize  int
	Total     int
	TotalPage int
}

func CeilInt(a, b int) int {
	return int(math.Ceil(float64(a) / float64(b)))
}

func NewArticleList(list []*Article, sortType ArticleSortType, currPage, pageSize int) *ArticleList {
	return &ArticleList{
		List:      list,
		SortType:  sortType,
		CurrPage:  currPage,
		PageSize:  pageSize,
		Total:     len(list),
		TotalPage: CeilInt(len(list), pageSize),
	}
}

func (al *ArticleList) Sort(sortType ArticleSortType) []*Article {
	al.SortType = sortType
	sort.Sort(al)
	return al.List
}

func (al *ArticleList) PagingList(page, pageSize int) []*Article {
	if page < 1 {
		page = 1
	}
	if page > al.TotalPage {
		page = al.TotalPage
	}

	start := pageSize * (page - 1)
	end := start + pageSize
	if end > len(al.List) {
		end = len(al.List)
	}
	al.CurrPage = page
	al.PageSize = pageSize
	return al.List[start:end]
}

func (al *ArticleList) Len() int {
	if al != nil && al.List != nil {
		return len(al.List)
	}
	return 0
}

func (al *ArticleList) Less(i, j int) bool {
	switch al.SortType {
	case ListSortLatest:
		compare := al.List[i].CreatedAt.Compare(al.List[j].CreatedAt)
		return compare > 0
	case ListSortBest:
		return al.List[i].ListWeight > al.List[j].ListWeight
	default:
		return al.List[i].Weight > al.List[j].Weight
	}
}

func (al *ArticleList) Swap(i, j int) {
	al.List[i], al.List[j] = al.List[j], al.List[i]
}

type Article struct {
	Id                        int
	Title                     string
	NullTitle                 pgtype.Text
	AuthorName                string
	AuthorId                  int
	Content                   string
	Summary                   string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	CreatedAtStr              string
	UpdatedAtStr              string
	ReplyTo                   int
	Deleted                   bool
	Replies                   *ArticleList
	ReplyDepth                int
	ReplyRootArticleId        int
	NullReplyRootArticleTitle pgtype.Text
	ReplyRootArticleTitle     string
	DisplayTitle              string // only for display
	TotalReplyCount           int
	VoteUp                    int
	VoteDown                  int
	VoteScore                 int
	Weight                    float64 // weight in replise
	ListWeight                float64 // weight in list page
	CurrUserState             *CurrUserState
	// SortType                  ArticleSortType
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

var ErrValidArticleFailed = errors.New("valid article failed")

func articleValidErr(str string) error {
	return errors.Join(ErrValidArticleFailed, errors.New(str))
}

func (a *Article) Valid(isUpdate bool) error {
	isReply := a.ReplyDepth > 0 || a.ReplyTo != 0
	authorId := a.AuthorId
	title := strings.TrimSpace(a.Title)
	content := strings.TrimSpace(a.Content)

	if !isUpdate && authorId == 0 {
		return articleValidErr("author id is required")
	}

	if !isReply {
		if title == "" {
			return articleValidErr("article title is required")
		}

		if utf8.RuneCountInString(title) > MAX_ARTICLE_TITLE_LEN {
			return articleValidErr(fmt.Sprintf("article title limit to %d characters", MAX_ARTICLE_TITLE_LEN))
		}
	}

	if content == "" {
		return articleValidErr("article content is required")
	}

	if utf8.RuneCountInString(content) > MAX_ARTICLE_CONTENT_LEN {
		return articleValidErr(fmt.Sprintf("article content limit to %d characters", MAX_ARTICLE_CONTENT_LEN))
	}
	return nil
}

func (a *Article) CalcScore() {
	a.VoteScore = a.VoteUp - a.VoteDown - 1
}

const gravity float64 = 1.6

func (a *Article) CalcWeight() {
	cf := confidence(a.VoteUp, a.VoteDown)
	// fmt.Println("confidence array: ", confidences)
	// fmt.Println("confidence: ", cf)
	a.Weight = cf

	a.ListWeight = hot(a.VoteUp, a.VoteDown, a.CreatedAt)
}

// First commit  Mon Feb 13 00:11:53 2023 +0800
var projectStartDate = time.Date(2023, time.Month(2), 13, 0, 11, 53, 0, time.Local)

// https://github.com/reddit-archive/reddit/blob/master/r2/r2/lib/db/_sorts.pyx
func hot(ups, downs int, date time.Time) float64 {
	s := float64(ups - downs)
	order := math.Log10(math.Max(math.Abs(s), 1))

	// fmt.Println("order: ", order)

	var sign float64
	if s > 0 {
		sign = 1
	} else if s < 0 {
		sign = -1
	} else {
		sign = 0
	}

	seconds := date.Sub(projectStartDate).Seconds()
	// fmt.Println("seconds/45000: ", seconds/45000)
	return math.Round((sign*order+seconds/45000)*1e7) / 1e7
}

// cpdef double _confidence(int ups, int downs):
//     """The confidence sort.
//        http://www.evanmiller.org/how-not-to-sort-by-average-rating.html"""
//     cdef float n = ups + downs

//     if n == 0:
//         return 0

//     cdef float z = 1.281551565545 # 80% confidence
//     cdef float p = float(ups) / n

//     left = p + 1/(2*n)*z*z
//     right = z*sqrt(p*(1-p)/n + z*z/(4*n*n))
//     under = 1+1/n*z*z

//     return (left - right) / under

// cdef int up_range = 400
// cdef int down_range = 100
// cdef list _confidences = []
// for ups in xrange(up_range):
//     for downs in xrange(down_range):
//         _confidences.append(_confidence(ups, downs))
// def confidence(int ups, int downs):
//     if ups + downs == 0:
//         return 0
//     elif ups < up_range and downs < down_range:
//         return _confidences[downs + ups * down_range]
//     else:
//         return _confidence(ups, downs)

func doConfidence(ups, downs int) float64 {
	n := float64(ups + downs)

	if n == 0 {
		return 0
	}

	z := 1.281551565545 // 80% confidence
	p := float64(ups) / n

	left := p + 1/(2*n)*z*z
	right := z * math.Sqrt(p*(1-p)/n+z*z/(4*n*n))
	under := 1 + 1/n*z*z

	return (left - right) / under
}

const upRange = 400
const downRange = 100

var confidences []float64

func InitConfidences() {
	for i := 0; i < upRange; i++ {
		for j := 0; j < downRange; j++ {
			confidences = append(confidences, doConfidence(i, j))
		}
	}
}

func confidence(ups, downs int) float64 {
	if (ups + downs) == 0 {
		return 0
	} else if (ups < upRange) && (downs < downRange) {
		return confidences[downs+ups*downRange]
	} else {
		return doConfidence(ups, downs)
	}
}
