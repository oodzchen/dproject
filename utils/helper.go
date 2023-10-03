package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/oodzchen/dproject/config"
)

const ReUsernameStr = `^[a-zA-Z0-9][a-zA-Z0-9._-]+[a-zA-Z0-9]$`
const ReUsernameMiddleStr = `^[a-zA-Z0-9._-]+$`
const ReUsernameEdgeStr = `^[a-zA-Z0-9]+$`

// func GetSiteHost() string {
// 	// domain := os.Getenv("DOMAIN_NAME")
// 	domain := config.Config.DomainName
// 	// port := os.Getenv("PORT")
// 	port := config.Config.AppPort

// 	if port != 80 && port != 443 {
// 		// return domain + ":" + port
// 		return fmt.Sprintf("%s:%d", domain, port)
// 	} else {
// 		return domain
// 	}
// }

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

func ExtractNameFromEmail(email string) string {
	name := strings.Split(email, "@")[0]
	reUsername := regexp.MustCompile(ReUsernameStr)
	reUsernameMiddle := regexp.MustCompile(ReUsernameMiddleStr)
	reUsernameEdge := regexp.MustCompile(ReUsernameEdgeStr)

	var res []string
	if reUsername.Match([]byte(name)) {
		return name
	} else {
		if !reUsernameEdge.Match([]byte(name[:1])) {
			name = name[1:]
			if len(name) < 1 {
				return ""
			}
		}

		if !reUsernameEdge.Match([]byte(name[len(name)-1:])) {
			name = name[:len(name)-1]
			if len(name) < 1 {
				return ""
			}
		}

		for _, rune := range name {
			if reUsernameMiddle.Match([]byte(string(rune))) {
				res = append(res, string(rune))
			} else {
				res = append(res, ".")
			}
		}
	}

	return strings.Join(res, "")
}
