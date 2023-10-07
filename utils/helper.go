package utils

import (
	"encoding/json"
	"fmt"

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
