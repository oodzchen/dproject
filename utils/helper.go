package utils

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"

	"github.com/oodzchen/dproject/config"
)

// Print data as JSON string with prefix
func PrintJSONf(prefix string, data any) {
	str := SprintJSONf(data, "", "  ")
	fmt.Printf("%s%s\n", prefix, str)
}

func SprintJSONf(data any, prefix, indent string) string {
	jsonStr, err := json.MarshalIndent(data, prefix, indent)
	if err != nil {
		fmt.Printf("utils.PrintJSONf error: %v", err)
		return ""
	}
	return string(jsonStr)
}

func GetReplyDepthSize() int {
	return config.ReplyDepthPageSize
}

func IsDebug() bool {
	return config.Config.Debug
}

const maxDisplayURLLength = 100

// https://stackoverflow.com/a/3809435
var urlRegex = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9]{1,63}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`)

func ReplaceLink(str string) string {
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

func NewLine2BR(source string) string {
	return strings.ReplaceAll(source, "\r", "<br>")
}

func GetRealIP(r *http.Request) string {
	realIP := r.Header.Get("CF-Connecting-IP")

	if realIP == "" {
		realIP = r.Header.Get("X-Real-IP")
		// fmt.Println("x-real-ip:", realIP)
	}

	if realIP == "" {
		realIP = r.Header.Get("X-Forwarded-For")
		// fmt.Println("x-Forwarded-for:", realIP)
	}

	if realIP == "" {
		realIP = strings.Split(r.RemoteAddr, ":")[0]
		// ip := "38.59.236.10"
		// fmt.Println("geo db metadata:", geoDB.Metadata())
		// fmt.Println("r.RemoteAddr:", realIP)
	}
	return realIP
}
