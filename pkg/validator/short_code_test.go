package validator

import "testing"

func TestIsValidShortCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid short code",
			input:    "abc123",
			expected: true,
		},
		{
			name:     "short code with special characters",
			input:    "abc$123",
			expected: false,
		},
		{
			name:     "too long short code",
			input:    "abcdefghijklmnopqrstuvwxyz",
			expected: false,
		},
		{
			name:     "exactly minimum length (1 character)",
			input:    "a",
			expected: true,
		},
		{
			name:     "exactly maximum length (10 characters)",
			input:    "abcdefghij",
			expected: true,
		},
		{
			name:     "one character over maximum (11 characters)",
			input:    "abcdefghijk",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "short code with spaces",
			input:    "abc 123",
			expected: false,
		},
		{
			name:     "numeric short code",
			input:    "123456",
			expected: true,
		},
		{
			name:     "alphanumeric short code",
			input:    "a1b2c3",
			expected: true,
		},
		{
			name:     "uppercase short code",
			input:    "ABC123",
			expected: true,
		},
		{
			name:     "mixed case short code",
			input:    "AbC123",
			expected: true,
		},
		{
			name:     "short code with underscore",
			input:    "abc_123",
			expected: false,
		},
		{
			name:     "short code with hyphen",
			input:    "abc-123",
			expected: false,
		},
		{
			name:     "short code with period",
			input:    "abc.123",
			expected: false,
		},
		{
			name:     "short code with unicode characters",
			input:    "abcñ123",
			expected: false,
		},
		{
			name:     "short code with emoji",
			input:    "abc😊123",
			expected: false,
		},
		{
			name:     "short code with leading space",
			input:    " abc123",
			expected: false,
		},
		{
			name:     "short code with trailing space",
			input:    "abc123 ",
			expected: false,
		},
		{
			name:     "short code with newline",
			input:    "abc123\n",
			expected: false,
		},
		{
			name:     "character before '0' (ASCII 47)",
			input:    "abc/123",
			expected: false,
		},
		{
			name:     "character after 'z' (ASCII 123)",
			input:    "abc{123",
			expected: false,
		},
		{
			name:     "character between '9' and 'A' (ASCII 58-64)",
			input:    "abc:123",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidShortCode(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidShortCode(%s) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}
