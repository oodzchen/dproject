package utils

import (
	"encoding/json"
	"fmt"

	"github.com/oodzchen/dproject/config"
)

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
