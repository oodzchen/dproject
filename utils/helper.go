package utils

import "os"

func GetSiteHost() string {
	domain := os.Getenv("DOMAIN_NAME")
	port := os.Getenv("PORT")

	if port != ":80" && port != "443" {
		return domain + port
	} else {
		return domain
	}
}
