package regex

import "regexp"

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// IsEmail reports whether s is a valid email address.
func IsEmail(s string) bool {
	return emailRegex.MatchString(s)
}
