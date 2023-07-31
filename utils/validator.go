package utils

import "regexp"

// Validate email string
// https://regex101.com/r/RzBwPX/1
func ValidateEmail(email string) bool {
	re := regexp.MustCompile(`^(?P<name>[a-zA-Z0-9.!#$%&'*+/=?^_ \x60{|}~-]+)@(?P<domain>[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*)$`)
	return re.Match([]byte(email))
}
