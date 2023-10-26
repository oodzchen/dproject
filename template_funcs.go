package main

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/oodzchen/dproject/utils"
)

var TmplFuncs = template.FuncMap{
	// "timeAgo":    formatTimeAgo,
	"timeFormat":   utils.FormatTime,
	"intRange":     intRange,
	"placehold":    placehold,
	"joinStrArr":   joinStrArr,
	"upHead":       upCaseHead,
	"downHead":     downCaseHead,
	"pageStr":      pageStr,
	"calcDuration": calcDuration,
	"replaceLink":  replaceLink,
}

func joinStrArr(arr []string, sep string) string {
	return strings.Join(arr, sep)
}

// func formatTimeAgo(t time.Time) string {
// 	// return timeago.English.Format(t)
// 	return timeago.Chinese.Format(t)
// }

func intRange(start, end int) []int {
	// fmt.Println("start: ", start, "end", end)
	n := end - start + 1
	result := make([]int, n)
	for i := 0; i < n; i++ {
		result[i] = start + i
	}
	return result
}

func placehold(data any, placeholcer string) string {
	if data == nil || data == false {
		return placeholcer
	}

	val := reflect.ValueOf(data)

	switch val.Kind() {
	case reflect.Array, reflect.Slice, reflect.String, reflect.Map, reflect.Chan:
		if val.Len() == 0 {
			return placeholcer
		}
		fallthrough
	default:
		return ""
	}
}

func upCaseHead(runeNum int, str string) string {
	runeStr := []rune(str)
	return strings.ToUpper(string(runeStr[:runeNum])) + string(runeStr[runeNum:])
}

func downCaseHead(runeNum int, str string) string {
	runeStr := []rune(str)
	return strings.ToLower(string(runeStr[:runeNum])) + string(runeStr[runeNum:])
}

func pageStr(path string, page int, query url.Values) string {
	if query == nil {
		return fmt.Sprintf("%s?page=%d", path, page)
	}

	var pageStr string
	if page < 1 {
		pageStr = "1"
	} else {
		pageStr = strconv.Itoa(page)
	}

	if query.Has("page") {
		query.Set("page", pageStr)
	} else {
		query.Add("page", pageStr)
	}

	return path + "?" + query.Encode()
}

func calcDuration(start time.Time) string {
	return fmt.Sprintf("%dms", time.Since(start).Milliseconds())
}

const maxDisplayURLLength = 100

// https://stackoverflow.com/a/3809435
var urlRegex = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,4}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`)

func replaceLink(str string) string {
	return urlRegex.ReplaceAllStringFunc(str, func(s string) string {
		shortenUrl := s
		if len(s) > maxDisplayURLLength {
			shortenUrl = string([]rune(s)[:maxDisplayURLLength]) + "..."
		}

		return fmt.Sprintf("<a title=\"%s\" href=\"%s\">%s</a>", s, s, shortenUrl)
	})
}
