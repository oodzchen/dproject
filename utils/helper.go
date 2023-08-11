package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

const DEFAULT_REPLY_DEPTH_SIZE int = 10

func GetSiteHost() string {
	domain := os.Getenv("DOMAIN_NAME")
	port := os.Getenv("PORT")

	if port != ":80" && port != "443" {
		return domain + port
	} else {
		return domain
	}
}

// Print data as JSON string with prefix
func PrintJSONf(prefix string, data any) {
	jsonStr, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("utils.PrintJSONf error: %v", err)
	}
	fmt.Printf("%s%s\n", prefix, jsonStr)
}

func GetReplyDepthSize() int {
	size, _ := strconv.Atoi(os.Getenv("REPLY_DEPTH_PAGE_SIZE"))
	if size == 0 {
		return DEFAULT_REPLY_DEPTH_SIZE
	}
	return size
}

func IsDebug() bool {
	return os.Getenv("DEBUG") == "1"
}
