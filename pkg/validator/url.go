package validator

import (
	"net/url"
	"strings"
)

// IsValidURL checks if the provided string is a valid URL.
// It verifies that the URL starts with "http://" or "https://"
// and that it can be parsed correctly.
func IsValidURL(rawURL string) bool {

	if rawURL == "" {
		return false
	}
	if !strings.HasPrefix(rawURL, "http://") &&
		!strings.HasPrefix(rawURL, "https://") {

		return false
	}

	finalURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if finalURL.Host == "" {
		return false
	}

	return true
}

// NormalizeURL ensures that the provided URL string starts with "https://".
// If it doesn't, the function prepends "https://" to the URL.
// It also trims any leading or trailing whitespace from the URL.
func NormalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)

	if !strings.HasPrefix(rawURL, "http://") &&
		!strings.HasPrefix(rawURL, "https://") {

		rawURL = "https://" + rawURL
	}

	return rawURL
}
