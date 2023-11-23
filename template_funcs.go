package main

import (
	"fmt"
	"html"
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
	"getDomain":    getDomain,
}

func joinStrArr(arr []string, sep string) string {
	return strings.Join(arr, sep)
}

// func formatTimeAgo(t time.Time) string {
// 	// return timeago.English.Format(t)
// 	return timeago.Chinese.Format(t)
// }

const (
	pageMiddleNum = 5
	pageEndNum    = 5
)

func intRange(start, end, curr int) []int {
	// fmt.Println("start: ", start, "end", end)
	n := end - start + 1
	all := make([]int, n)
	for i := 0; i < n; i++ {
		all[i] = start + i
	}

	halfMiddleNum := (pageMiddleNum / 2)
	currIndex := curr - 1
	var result []int

	if n > pageMiddleNum+pageEndNum*2 {
		middleLeft := currIndex - halfMiddleNum
		middleRight := currIndex + halfMiddleNum + 1

		leftOverley := pageEndNum + halfMiddleNum
		rightOverley := end - 1 - pageEndNum - halfMiddleNum

		if currIndex < leftOverley || currIndex > rightOverley {
			result = append(result, all[:leftOverley]...)
			result = append(result, 0)
			result = append(result, all[rightOverley:]...)
		} else {
			result = append(result, all[:pageEndNum]...)
			result = append(result, 0)
			result = append(result, all[middleLeft:middleRight]...)
			result = append(result, 0)
			result = append(result, all[n-pageEndNum:]...)
		}
	} else {
		result = all
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
	rawStr := html.UnescapeString(str)
	idxArr := urlRegex.FindAllStringIndex(rawStr, -1)

	// fmt.Println("idx arr:", idxArr)

	if len(idxArr) > 0 {
		var result string

		for idx, matched := range idxArr {
			if idx == 0 {
				result += html.EscapeString(rawStr[0:matched[0]])
			}

			urlStr := rawStr[matched[0]:matched[1]]
			shortenUrl := urlStr
			if len(urlStr) > maxDisplayURLLength {
				shortenUrl = string([]rune(urlStr)[:maxDisplayURLLength]) + "..."
			}

			result += fmt.Sprintf("<a title=\"%s\" href=\"%s\">%s</a>", urlStr, urlStr, shortenUrl)

			if idx < len(idxArr)-1 {
				nextMatch := idxArr[idx+1]
				result += html.EscapeString(rawStr[matched[1]:nextMatch[0]])
			} else {
				result += html.EscapeString(rawStr[matched[1]:])
			}
		}
		return result
	} else {
		return str
	}
}

var urlDomainRegex = regexp.MustCompile(`[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,4}`)

func getDomain(url string) string {
	return urlDomainRegex.FindString(url)
}
