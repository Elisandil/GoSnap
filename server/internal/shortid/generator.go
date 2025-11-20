package shortid

import (
	"math"
	"strings"
)

const base62Chars string = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Generator is responsible for generating short IDs using a specified base.
type Generator struct {
	base int
}

// NewGenerator creates a new Generator instance with base62 encoding.
func NewGenerator() *Generator {

	return &Generator{
		base: len(base62Chars),
	}
}

// Encode converts a given integer to a base62 encoded string.
func (g *Generator) Encode(number int64) string {

	if number == 0 {
		return string(base62Chars[0])
	}

	var results strings.Builder
	for number > 0 {
		remainder := number % int64(g.base)
		results.WriteByte(base62Chars[remainder])
		number = number / int64(g.base)
	}

	// Reverse the string since we constructed it backwards
	encoded := results.String()
	runes := []rune(encoded)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j+1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// Decode converts a base62 encoded string back to its integer representation.
func (g *Generator) Decode(encoded string) int64 {
	var number int64
	for i, char := range encoded {
		power := len(encoded) - i - 1
		index := strings.IndexRune(base62Chars, char)
		if index == -1 {
			return -1
		}
		number += int64(index) * int64(math.Pow(float64(g.base), float64(power)))
	}
	return number
}
