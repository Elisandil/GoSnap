package shortid

import "testing"

func TestGenerator_Encode(t *testing.T) {
	g := NewGenerator()

	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{
			name:     "zero",
			input:    0,
			expected: "0",
		},
		{
			name:     "single digit",
			input:    5,
			expected: "5",
		},
		{
			name:     "double digit",
			input:    62,
			expected: "10",
		},
		{
			name:     "large number",
			input:    1000,
			expected: "G8",
		},
		{
			name:     "very large number",
			input:    123456789,
			expected: "8M0kX",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.Encode(tt.input)
			if result != tt.expected {
				t.Errorf("Encode(%d) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerator_Decode(t *testing.T) {
	g := NewGenerator()

	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "zero",
			input:    "0",
			expected: 0,
		},
		{
			name:     "single digit",
			input:    "5",
			expected: 5,
		},
		{
			name:     "double digit",
			input:    "10",
			expected: 62,
		},
		{
			name:     "large number",
			input:    "G8",
			expected: 1000,
		},
		{
			name:     "very large number",
			input:    "8M0kX",
			expected: 123456789,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.Decode(tt.input)
			if result != tt.expected {
				t.Errorf("Decode(%s) = %d; want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerator_EncodeDecodeRoundTrip(t *testing.T) {
	g := NewGenerator()

	testNumbers := []int64{0, 1, 10, 100, 1000, 10000, 100000, 1000000, 123456789}

	for _, num := range testNumbers {
		t.Run("round_trip", func(t *testing.T) {
			encoded := g.Encode(num)
			decoded := g.Decode(encoded)

			if decoded != num {
				t.Errorf("Round trip failed: %d -> %s -> %d", num, encoded, decoded)
			}
		})
	}
}

func TestGenerator_Decode_InvalidCharacter(t *testing.T) {
	g := NewGenerator()

	invalidInputs := []string{"@", "#", "!", "a@b", "123$"}

	for _, input := range invalidInputs {
		t.Run("invalid_"+input, func(t *testing.T) {
			result := g.Decode(input)
			if result != -1 {
				t.Errorf("Decode(%s) should return -1 for invalid input, got %d", input, result)
			}
		})
	}
}

func TestGenerator_UniqueEncoding(t *testing.T) {
	g := NewGenerator()

	seen := make(map[string]bool)

	for i := int64(0); i < 10000; i++ {
		encoded := g.Encode(i)
		if seen[encoded] {
			t.Errorf("Duplicate encoding found: %d and another number both encode to %s", i, encoded)
		}
		seen[encoded] = true
	}
}

func BenchmarkGenerator_Encode(b *testing.B) {
	g := NewGenerator()
	for i := 0; i < b.N; i++ {
		g.Encode(int64(i))
	}
}

func BenchmarkGenerator_Decode(b *testing.B) {
	g := NewGenerator()
	encoded := g.Encode(123456789)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Decode(encoded)
	}
}
