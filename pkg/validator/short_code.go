package validator

import (
	"regexp"
)

const (
	MinShortCodeLength      = 1
	MaxShortCodeLength      = 10
	StandardShortCodeLength = 6
)

var validShortCodeRegex = regexp.MustCompile(`^[0-9A-Za-z]+$`)

// IsValidShortCode checks if the provided short code is valid.
// A valid short code is between MinShortCodeLength and MaxShortCodeLength characters
// and contains only alphanumeric characters.
func IsValidShortCode(shortCode string) bool {

	if shortCode == "" {
		return false
	}

	length := len(shortCode)
	if length < MinShortCodeLength || length > MaxShortCodeLength {
		return false
	}

	return validShortCodeRegex.MatchString(shortCode)
}
