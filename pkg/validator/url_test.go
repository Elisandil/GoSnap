package validator

import "testing"

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "valid https url",
			url:      "https://www.google.com",
			expected: true,
		},
		{
			name:     "valid http url",
			url:      "http://example.com",
			expected: true,
		},
		{
			name:     "valid url with path",
			url:      "https://github.com/Elisandil/GoSnap",
			expected: true,
		},
		{
			name:     "valid url with query",
			url:      "http://example.com/search?q=golang",
			expected: true,
		},
		{
			name:     "empty string",
			url:      "",
			expected: false,
		},
		{
			name:     "no scheme url",
			url:      "www.google.com",
			expected: false,
		},
		{
			name:     "invalid scheme format",
			url:      "ftp://example.com",
			expected: false,
		},
		{
			name:     "no host",
			url:      "https://",
			expected: false,
		},
		{
			name:     "malformed url",
			url:      "ht!!p://example.com",
			expected: false,
		},
		{
			name:     "localhost url",
			url:      "http://localhost:8080",
			expected: true,
		},
		{
			name:     "ip address url",
			url:      "http://192.168.1.1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidURL(tt.url)
			if result != tt.expected {
				t.Errorf("IsValidURL(%s) = %v; want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already has https",
			input:    "https://www.google.com",
			expected: "https://www.google.com",
		},
		{
			name:     "already has http",
			input:    "http://example.com",
			expected: "http://example.com",
		},
		{
			name:     "no scheme",
			input:    "www.marca.com",
			expected: "https://www.marca.com",
		},
		{
			name:     "with whitespaces",
			input:    "   www.vandal.net   ",
			expected: "https://www.vandal.net",
		},
		{
			name:     "exmpty string",
			input:    "",
			expected: "https://",
		},
		{
			name:     "with path and no scheme",
			input:    "https://example.com/path/to/resource",
			expected: "https://example.com/path/to/resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeURL(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}
