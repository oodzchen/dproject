package model

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type articleData struct {
	Title           string
	AuthorId        int
	Content         string
	Depth           int
	Link            string
	CategoryFrontId string
}

func TestArticleValid(t *testing.T) {
	mockTitle := gofakeit.Sentence(MAX_ARTICLE_TITLE_LEN)
	largeText, err := os.ReadFile("testdata/rfc791.txt")
	if err != nil {
		t.Errorf("read file content error: %v", err)
	}

	tests := []struct {
		desc     string
		in       *articleData
		isUpdate bool
		valid    bool
	}{
		{
			desc: "All valid",
			in: &articleData{
				"This is Title",
				1,
				"This is content",
				0,
				"https://test.com",
				"general",
			},
			isUpdate: false,
			valid:    true,
		},
		{
			desc: "Title is required",
			in: &articleData{
				" ",
				1,
				"This is content",
				0,
				"https://test.com",
				"general",
			},
			isUpdate: false,
			valid:    false,
		},
		{
			desc: "Author is required",
			in: &articleData{
				"This is Title",
				0,
				"This is content",
				0,
				"https://test.com",
				"general",
			},
			isUpdate: false,
			valid:    false,
		},
		{
			desc: "Content is optional",
			in: &articleData{
				"This is Title",
				1,
				" ",
				0,
				"https://test.com",
				"general",
			},
			isUpdate: false,
			valid:    true,
		},
		{
			desc: "Title limit",
			in: &articleData{
				mockTitle,
				1,
				"This is content",
				0,
				"https://test.com",
				"general",
			},
			isUpdate: false,
			valid:    false,
		},
		{
			desc: "Content limit",
			in: &articleData{
				"This is Title",
				1,
				string(largeText),
				0,
				"https://test.com",
				"general",
			},
			isUpdate: false,
			valid:    false,
		},
		{
			desc: "Update root article without title",
			in: &articleData{
				"",
				1,
				"This is Content",
				0,
				"https://test.com",
				"general",
			},
			isUpdate: true,
			valid:    false,
		},
		{
			desc: "Update reply without title",
			in: &articleData{
				"",
				0,
				"This is Content",
				1,
				"https://test.com",
				"general",
			},
			isUpdate: true,
			valid:    true,
		},
		{
			desc: "Update without author",
			in: &articleData{
				"",
				0,
				"This is Content",
				1,
				"https://test.com",
				"general",
			},
			isUpdate: true,
			valid:    true,
		},
		{
			desc: "Update without content",
			in: &articleData{
				"This is Title",
				0,
				"",
				1,
				"https://test.com",
				"general",
			},
			isUpdate: true,
			valid:    false,
		},
		{
			desc: "URL is optinal",
			in: &articleData{
				"This is Title",
				1,
				"",
				0,
				"",
				"general",
			},
			isUpdate: false,
			valid:    true,
		},
		{
			desc: "URL format",
			in: &articleData{
				"This is Title",
				1,
				"",
				0,
				"abc.com",
				"general",
			},
			isUpdate: false,
			valid:    false,
		},
		{
			desc: "Category is required",
			in: &articleData{
				"This is Title",
				1,
				"This is content",
				0,
				"https://test.com",
				"",
			},
			isUpdate: false,
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			a := &Article{
				Title:           tt.in.Title,
				AuthorId:        tt.in.AuthorId,
				Content:         tt.in.Content,
				ReplyDepth:      tt.in.Depth,
				Link:            tt.in.Link,
				CategoryFrontId: tt.in.CategoryFrontId,
			}
			err := a.Valid(tt.isUpdate)
			got := err == nil
			want := tt.valid

			if got != want {
				t.Errorf("article: %+v \nvalidate result should be %t, but got %t, error: %v", tt.in, want, got, err)
			}
		})
	}
}

func TestDeleteElement(t *testing.T) {
	tests := []struct {
		in       []any
		deleteFn func(any) bool
		want     []any
	}{
		{
			in: []any{"a", "b", "c"},
			deleteFn: func(a any) bool {
				return a == "b"
			},
			want: []any{"a", "c"},
		},
		{
			in: []any{1, 2, 3},
			deleteFn: func(a any) bool {
				return a == 3
			},
			want: []any{1, 2},
		},
	}

	for idx, tt := range tests {
		t.Run(fmt.Sprintf("delete item %d", idx), func(t *testing.T) {
			if got := deleteElement(tt.in, tt.deleteFn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("want %v, but got %v", tt.want, got)
			}
		})
	}
}

func TestGenArticleDiffsFromDelta(t *testing.T) {
	dmp := diffmatchpatch.New()
	article0 := Article{
		Title:           "This is title",
		Link:            "https://www.example.com/abc",
		CategoryFrontId: "general",
		Content:         "This is content",
	}

	var logList []*ArticleLog

	article1 := article0
	article1.Title += " 111"
	article1TitleDelta := dmp.DiffToDelta(dmp.DiffMain(article1.Title, article0.Title, false))
	logList = append(logList, &ArticleLog{VersionNum: 1, PrevArticle: &article0, CurrArticle: &article1, TitleDelta: article1TitleDelta})

	article2 := article1
	article2.Link = "https://www.example.com/"
	article2LinkDelta := dmp.DiffToDelta(dmp.DiffMain(article2.Link, article1.Link, false))
	logList = append(logList, &ArticleLog{VersionNum: 2, PrevArticle: &article1, CurrArticle: &article2, URLDelta: article2LinkDelta})

	article3 := article2
	article3.CategoryFrontId = "internet"
	article3CategoryFrontDelta := dmp.DiffToDelta(dmp.DiffMain(article3.CategoryFrontId, article2.CategoryFrontId, false))
	logList = append(logList, &ArticleLog{VersionNum: 3, PrevArticle: &article2, CurrArticle: &article3, CategoryFrontIdDelta: article3CategoryFrontDelta})

	article4 := article3
	article4.Content += " 222"
	article4ContentDelta := dmp.DiffToDelta(dmp.DiffMain(article4.Content, article3.Content, false))
	logList = append(logList, &ArticleLog{VersionNum: 4, PrevArticle: &article3, CurrArticle: &article4, ContentDelta: article4ContentDelta})

	alList := &ArticleLogList{List: logList}
	sort.Sort(alList)
	logList = alList.List

	for _, item := range logList {
		// fmt.Println("version num:", item.VersionNum)
		// currArticle := item.CurrArticle
		// fmt.Println("delta title, url, category, content:", item.TitleDelta, item.URLDelta, item.CategoryFrontIdDelta, item.ContentDelta)
		// fmt.Println("title, url, category, content:", currArticle.Title, currArticle.Link, currArticle.CategoryFrontId, currArticle.Content)
		if item.ContentDelta != "" {
			contentDiffs, err := dmp.DiffFromDelta(item.CurrArticle.Content, item.ContentDelta)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println("content diff text:", dmp.DiffPrettyText(contentDiffs))
		}
	}

	logList1, err := GenArticleDiffsFromDelta(dmp, &article4, logList)
	if err != nil {
		t.Errorf("generate article diffs from delta error: %v", err)
		return
	}

	comparedFields := []string{"Title", "Link", "Content", "CategoryFrontId"}

	for idx, log1 := range logList1 {
		a1 := logList[idx].CurrArticle
		a2 := log1.CurrArticle

		for _, field := range comparedFields {
			val1, err := GetFieldValue(a1, field)
			if err != nil {
				t.Fatal(err)
			}
			val2, err := GetFieldValue(a2, field)
			if err != nil {
				t.Fatal(err)
			}

			if val1 != val2 {
				t.Errorf("recover %s value failed, want %s, but got %s", field, val1, val2)
			}
		}
	}
}

func GetFieldValue(s interface{}, fieldName string) (interface{}, error) {
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	fieldVal := val.FieldByName(fieldName)
	if !fieldVal.IsValid() {
		return nil, fmt.Errorf("No such field: %s in obj", fieldName)
	}

	return fieldVal.Interface(), nil
}
