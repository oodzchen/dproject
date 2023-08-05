package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

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
