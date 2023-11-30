package model

import (
	"errors"
	"fmt"
	"html"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/microcosm-cc/bluemonday"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	MAX_ARTICLE_TITLE_LEN, MAX_ARTICLE_CONTENT_LEN int = 255, 24000
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

type ArticleReactCounts map[string]int

type CurrUserState struct {
	VoteType     VoteType
	NullVoteType pgtype.Text
	Saved        bool
	ReactFrontId string
	Subscribed   bool
}

func (cus *CurrUserState) FormatNullValues() {
	if cus.VoteType == "" && cus.NullVoteType.Valid {
		cus.VoteType = VoteType(cus.NullVoteType.String)
	}
}

type ArticleSortType string

const (
	ReplySortBest  ArticleSortType = "best"
	ListSortOldest                 = "oldest"
	ListSortLatest                 = "latest"
	ListSortBest                   = "list_best"
	ListSortHot                    = "list_hot"
)

var articleSortMap = map[ArticleSortType]bool{
	ReplySortBest:  true,
	ListSortOldest: true,
	ListSortLatest: true,
	ListSortBest:   true,
	ListSortHot:    true,
}

func ValidArticleSort(sortType string) bool {
	return articleSortMap[ArticleSortType(sortType)]
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

func NewArticleList(list []*Article, sortType ArticleSortType, currPage, pageSize, total int) *ArticleList {
	return &ArticleList{
		List:      list,
		SortType:  sortType,
		CurrPage:  currPage,
		PageSize:  pageSize,
		Total:     total,
		TotalPage: CeilInt(total, pageSize),
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
	if start < 0 {
		start = 0
	}
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
		return al.List[i].CreatedAt.Compare(al.List[j].CreatedAt) > 0
	case ListSortOldest:
		return al.List[i].CreatedAt.Compare(al.List[j].CreatedAt) <= 0
	case ListSortBest:
		return al.List[i].ListWeight > al.List[j].ListWeight
	case ListSortHot:
		return al.List[i].ParticipateCount > al.List[j].ParticipateCount
	default:
		return al.List[i].Weight > al.List[j].Weight
	}
}

func (al *ArticleList) Swap(i, j int) {
	al.List[i], al.List[j] = al.List[j], al.List[i]
}

func (al *ArticleList) Remove(id int) {
	al.List = deleteElement(al.List, func(item *Article) bool {
		return item.Id == id
	})
}

func deleteElement[T any](slice []T, fn func(T) bool) []T {
	var res []T
	for _, item := range slice {
		if toDelete := fn(item); toDelete {
			continue
		}
		res = append(res, item)
	}
	return res
}

type Article struct {
	Id                        int
	Title                     string
	Link                      string
	NullTitle                 pgtype.Text
	AuthorName                string
	AuthorId                  int
	Content                   string
	Summary                   string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	CreatedAtStr              string
	UpdatedAtStr              string
	ReplyToId                 int
	ReplyToArticle            *Article
	Deleted                   bool
	Replies                   *ArticleList
	ReplyDepth                int
	ReplyRootArticleId        int
	NullReplyRootArticleTitle pgtype.Text
	ReplyRootArticleTitle     string
	DisplayTitle              string // only for display
	TotalReplyCount           int
	ChildrenCount             int
	VoteUp                    int
	VoteDown                  int
	VoteScore                 int
	Weight                    float64 // weight in replise
	ListWeight                float64 // weight in list page
	ParticipateCount          int
	CurrUserState             *CurrUserState
	Reacts                    []*ArticleReact
	ReactCounts               ArticleReactCounts
	ShowScore                 bool
	TmpParent                 *Article // Only for temporary use, to avoid circular reference errors
	CategoryFrontId           string
	Category                  *Category
	Locked                    bool
	Pinned                    bool
	NullPinnedExpireAt        pgtype.Timestamp
	PinnedExpireAt            time.Time
}

type ArticleReact struct {
	Id        int
	Emoji     string
	FrontId   string
	Describe  string
	CreatedAt time.Time
}

type ArticleLog struct {
	Id                      int
	PrimaryArticleId        int
	CreatedAt               time.Time
	OperatorId              int
	Operator                *User
	CurrEditTime            time.Time
	PrevEditTime            time.Time
	PrevArticle             *Article
	CurrArticle             *Article
	VersionNum              int
	TitleDelta              string
	URLDelta                string
	ContentDelta            string
	CategoryFrontIdDelta    string
	TitleDiffHTML           string
	URLDiffHTML             string
	ContentDiffHTML         string
	CategoryFrontIdDiffHTML string
	IsHidden                bool
}

type ArticleLogList struct {
	List []*ArticleLog
}

func (all *ArticleLogList) Len() int {
	return len(all.List)
}

func (all *ArticleLogList) Less(i, j int) bool {
	return all.List[i].VersionNum > all.List[j].VersionNum
}

func (all *ArticleLogList) Swap(i, j int) {
	all.List[i], all.List[j] = all.List[j], all.List[i]
}

func GenArticleDiffsFromDelta(dmp *diffmatchpatch.DiffMatchPatch, headArticle *Article, list []*ArticleLog) ([]*ArticleLog, error) {
	if len(list) == 0 {
		return list, nil
	}

	alList := &ArticleLogList{
		List: list,
	}

	sort.Sort(alList)

	// for _, item := range alList.List {
	// 	fmt.Println("version num:", item.VersionNum)
	// }

	currArticle := headArticle
	var tempArticle *Article
	for _, log := range alList.List {
		// fmt.Printf("log %#v\n", log)
		// fmt.Println("recover version num:", log.VersionNum)

		tempArticle = new(Article)

		if log.TitleDelta != "" {
			titleDiffs, err := dmp.DiffFromDelta(currArticle.Title, log.TitleDelta)
			if err != nil {
				return nil, err
			}
			tempArticle.Title = dmp.DiffText2(titleDiffs)
			log.TitleDiffHTML = dmp.DiffPrettyHtml(reverseDiffs(dmp, titleDiffs))
			// fmt.Println("title diff text:", dmp.DiffPrettyText(titleDiffs))
		} else {
			tempArticle.Title = currArticle.Title
		}

		if log.URLDelta != "" {
			urlDiffs, err := dmp.DiffFromDelta(currArticle.Link, log.URLDelta)
			if err != nil {
				return nil, err
			}
			tempArticle.Link = dmp.DiffText2(urlDiffs)
			log.URLDiffHTML = dmp.DiffPrettyHtml(reverseDiffs(dmp, urlDiffs))
			// fmt.Println("link diff text:", dmp.DiffPrettyText(urlDiffs))
		} else {
			tempArticle.Link = currArticle.Link
		}

		if log.ContentDelta != "" {
			contentDiffs, err := dmp.DiffFromDelta(currArticle.Content, log.ContentDelta)
			if err != nil {
				return nil, err
			}
			tempArticle.Content = dmp.DiffText2(contentDiffs)
			log.ContentDiffHTML = dmp.DiffPrettyHtml(reverseDiffs(dmp, contentDiffs))
			// fmt.Println("content diff text:", dmp.DiffPrettyText(contentDiffs))
		} else {
			tempArticle.Content = currArticle.Content
		}

		if log.CategoryFrontIdDelta != "" {
			categoryFrontDiffs, err := dmp.DiffFromDelta(currArticle.CategoryFrontId, log.CategoryFrontIdDelta)
			if err != nil {
				return nil, err
			}
			tempArticle.CategoryFrontId = dmp.DiffText2(categoryFrontDiffs)
			log.CategoryFrontIdDiffHTML = dmp.DiffPrettyHtml(reverseDiffs(dmp, categoryFrontDiffs))
			// fmt.Println("category front id diff text:", dmp.DiffPrettyText(categoryFrontDiffs))
		} else {
			tempArticle.CategoryFrontId = currArticle.CategoryFrontId
		}

		log.CurrArticle = currArticle
		log.PrevArticle = tempArticle

		currArticle = tempArticle
	}

	return alList.List, nil
}

func reverseDiffs(dmp *diffmatchpatch.DiffMatchPatch, diffs []diffmatchpatch.Diff) []diffmatchpatch.Diff {
	return dmp.DiffMain(dmp.DiffText2(diffs), dmp.DiffText1(diffs), false)
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

	if a.PinnedExpireAt.IsZero() && a.NullPinnedExpireAt.Valid {
		a.PinnedExpireAt = a.NullPinnedExpireAt.Time
		// err := a.NullPinnedExpireAt.Scan(&a.PinnedExpireAt)
		// if err != nil {
		// 	fmt.Println("Format pinned expired time null value error:", err)
		// }
	}
}

// func (a *Article) FormatUserStateNullValues() {
// 	if a.CurrUserState != nil {
// 		a.CurrUserState.FormatNullValues()
// 	}
// }

func (a *Article) FormatReactCounts() {
	countsMap := make(map[string]int)

	for _, react := range a.Reacts {
		if count, ok := countsMap[react.FrontId]; ok {
			countsMap[react.FrontId] = count + 1
		} else {
			countsMap[react.FrontId] = 1
		}
	}

	if len(countsMap) > 0 {
		a.ReactCounts = countsMap
	}
}

func (a *Article) UpdateDisplayTitle() {
	var displayTitle string
	if a.ReplyDepth == 0 {
		displayTitle = a.Title
	} else if a.ReplyDepth == 1 {
		displayTitle = fmt.Sprintf("%s: %s", translator.LocalTpl("Re"), a.ReplyRootArticleTitle)
	} else {
		displayTitle = fmt.Sprintf("%s × %d: %s", translator.LocalTpl("Re"), a.ReplyDepth, a.ReplyRootArticleTitle)
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
	a.CreatedAtStr = translator.TimeAgo.Format(a.CreatedAt)
	a.UpdatedAtStr = translator.TimeAgo.Format(a.UpdatedAt)
}

// func (a *Article) TransformNewlines() {
// 	re := regexp.MustCompile(`\r`)
// 	a.Content = re.ReplaceAllString(a.Content, "<br/>")
// }

func (a *Article) Sanitize(p *bluemonday.Policy) {
	// a.Title = p.Sanitize(a.Title)
	// a.Content = p.Sanitize(a.Content)
	a.Title = p.Sanitize(a.Title)
	a.Link = p.Sanitize(a.Link)
	a.Content = html.EscapeString(a.Content)
}

func articleValidErr(str string) error {
	return errors.Join(AppErrArticleValidFailed, errors.New(", "+str))
}

func (a *Article) TrimSpace() {
	a.Title = strings.TrimSpace(a.Title)
	a.Link = strings.TrimSpace(a.Link)
}

var reArticleURLStr = `^(https?|ftp)://[^\s/$.?#].[^\s]*$`

func (a *Article) Valid(isUpdate bool) error {
	isReply := a.ReplyDepth > 0 || a.ReplyToId != 0
	authorId := a.AuthorId
	title := strings.TrimSpace(a.Title)
	content := strings.TrimSpace(a.Content)
	link := strings.TrimSpace(a.Link)

	if !isUpdate && authorId == 0 {
		return articleValidErr(translator.LocalTpl("Required", "FieldNames", translator.LocalTpl("Author")))
	}

	if !isReply {
		if title == "" {
			return articleValidErr(translator.LocalTpl("Required", "FieldNames", translator.LocalTpl("ArticleTitle")))
		}

		if utf8.RuneCountInString(title) > MAX_ARTICLE_TITLE_LEN {
			return articleValidErr(translator.LocalTpl("NotExceed", "FieldNames", translator.LocalTpl("ArticleTitle"), "Num", MAX_ARTICLE_TITLE_LEN))
		}

		if link != "" {
			if !regexp.MustCompile(reArticleURLStr).Match([]byte(link)) {
				return articleValidErr(translator.LocalTpl("FormatError", "FieldNames", translator.LocalTpl("URL")))
			}
		}

		if a.CategoryFrontId == "" {
			return articleValidErr(translator.LocalTpl("Required", "FieldNames", translator.LocalTpl("Category")))
		}

	} else {
		if content == "" {
			return articleValidErr(translator.LocalTpl("Required", "FieldNames", translator.LocalTpl("ArticleContent")))
		}
	}

	if utf8.RuneCountInString(content) > MAX_ARTICLE_CONTENT_LEN {
		return articleValidErr(translator.LocalTpl("NotExceed", "FieldNames", translator.LocalTpl("ArticleContent"), "Num", MAX_ARTICLE_CONTENT_LEN))
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

func CalcArticleReplyWeight(voteUpCount, voteDownCount int) float64 {
	return confidence(voteUpCount, voteDownCount)
}

func CalcArticleListWeight(voteUpCount, voteDownCount int, createdAt time.Time) float64 {
	return hot(voteUpCount, voteDownCount, createdAt)
}

func (a *Article) CheckShowScore(loginedUserId int) {
	// a.VoteScore = a.VoteUp - a.VoteDown - 1
	if time.Now().Sub(a.CreatedAt).Hours() > 1 || a.AuthorId == loginedUserId {
		a.ShowScore = true
	} else {
		a.ShowScore = false
	}
}

func (a *Article) UpdatePinnedState() {
	// fmt.Println("pinned compare:", a.PinnedExpireAt.Compare(time.Now()))
	if a.PinnedExpireAt.IsZero() {
		a.Pinned = false
	} else {
		a.Pinned = a.PinnedExpireAt.Compare(time.Now()) > -1
	}

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

	// fmt.Println("confidence len: ", len(confidences))
}

func confidence(ups, downs int) float64 {
	// fmt.Println("confidence len: ", len(confidences))
	// fmt.Println("ups:", ups, "downs:", downs)
	if (ups + downs) == 0 {
		return 0
	} else if (ups < upRange) && (downs < downRange) {
		return confidences[downs+ups*downRange]
	} else {
		return doConfidence(ups, downs)
	}
}
