package utils

import (
	"testing"
)

func TestShortenString(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		expected string
	}{
		{"hello world", 5, "hello..."},
		{"hello", 10, "hello"},
		{"", 3, ""},
		{"abcdef", 0, "abcdef"},
		{"abcdef", 6, "abcdef"},
		{"abcdef", 3, "abc..."},
	}

	for _, tt := range tests {
		result := ShortenString(tt.input, tt.length)
		if result != tt.expected {
			t.Errorf("ShortenString(%q, %d) = %q; want %q", tt.input, tt.length, result, tt.expected)
		}
	}
}

func TestMostOcc_Int(t *testing.T) {
	tests := []struct {
		input    []int
		expected int
	}{
		{[]int{1, 2, 2, 3, 2, 4}, 2},
		{[]int{5, 5, 5, 5}, 5},
		// {[]int{1, 2, 3, 4}, 1}, // all unique, returns first seen; INCORRECT, returns random int
		{[]int{}, 0}, // zero value for int
	}

	for _, tt := range tests {
		result := MostOcc(tt.input)
		if result != tt.expected {
			t.Errorf("MostOcc(%v) = %v; want %v", tt.input, result, tt.expected)
		}
	}
}

func TestMostOcc_String(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{"a", "b", "a", "c", "a"}, "a"},
		// {[]string{"x", "y", "z"}, "x"}, // all unique, returns first seen; INCORRECT, returns random string
		{[]string{"foo", "foo", "bar"}, "foo"},
		{[]string{}, ""}, // zero value for string
	}

	for _, tt := range tests {
		result := MostOcc(tt.input)
		if result != tt.expected {
			t.Errorf("MostOcc(%v) = %v; want %v", tt.input, result, tt.expected)
		}
	}
}

func TestContainsDigits(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abc123", true},
		{"123", true},
		{"abc", false},
		{"", false},
		{"!@#$%", false},
		{"a1b2c3", true},
		{"0", true},
		{"no digits here", false},
		{"space 4", true},
	}

	for _, tt := range tests {
		result := ContainsDigits(tt.input)
		if result != tt.expected {
			t.Errorf("ContainsDigits(%q) = %v; want %v", tt.input, result, tt.expected)
		}
	}
}

func TestOnlyContainsDigits(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123456", true},
		{"", true}, // empty string should return true
		{"0", true},
		{"abc", false},
		{"123abc", false},
		{" 123", false},
		{"123 ", false},
		{"12.34", false},
		{"!@#$%", false},
		{"00123", true},
	}

	for _, tt := range tests {
		result := OnlyContainsDigits(tt.input)
		if result != tt.expected {
			t.Errorf("OnlyContainsDigits(%q) = %v; want %v", tt.input, result, tt.expected)
		}
	}
}

func TestIntersectionSlices_Int(t *testing.T) {
	tests := []struct {
		a, b     []int
		expected []int
	}{
		{[]int{1, 2, 3}, []int{2, 3, 4}, []int{2, 3}},
		{[]int{1, 1, 2, 3}, []int{1, 1, 3, 5}, []int{1, 1, 3}},
		{[]int{1, 1, 2, 3}, []int{1, 3, 5, 1}, []int{1, 1, 3}},
		{[]int{1, 2, 3}, []int{4, 5, 6}, []int{}},
		{[]int{}, []int{1, 2, 3}, []int{}},
		{[]int{1, 2, 3}, []int{}, []int{}},
		{[]int{}, []int{}, []int{}},
		{[]int{1, 2, 2, 3}, []int{2, 2, 3, 3}, []int{2, 2, 3}},
		{[]int{1, 2, 2, 3}, []int{3, 3, 2, 2}, []int{2, 2, 3}},
	}

	for _, tt := range tests {
		result := IntersectionSlices(tt.a, tt.b)
		if !SliceEquals(result, tt.expected) {
			t.Errorf("IntersectionSlices(%v, %v) = %v; want %v", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestIntersectionSlices_String(t *testing.T) {
	tests := []struct {
		a, b     []string
		expected []string
	}{
		{[]string{"a", "b", "c"}, []string{"b", "c", "d"}, []string{"b", "c"}},
		{[]string{"a", "a", "b"}, []string{"a", "c", "a"}, []string{"a", "a"}},
		{[]string{"x", "y", "z"}, []string{"a", "b", "c"}, []string{}},
		{[]string{}, []string{"a", "b"}, []string{}},
		{[]string{"a", "b"}, []string{}, []string{}},
		{[]string{}, []string{}, []string{}},
	}

	for _, tt := range tests {
		result := IntersectionSlices(tt.a, tt.b)
		if !SliceEquals(result, tt.expected) {
			t.Errorf("IntersectionSlices(%v, %v) = %v; want %v", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestSliceEquals_Int(t *testing.T) {
	tests := []struct {
		a, b     []int
		expected bool
	}{
		{[]int{1, 2, 3}, []int{3, 2, 1}, true},
		{[]int{1, 2, 2, 3}, []int{2, 1, 3, 2}, true},
		{[]int{1, 2, 3}, []int{1, 2, 3}, true},
		{[]int{1, 2, 3}, []int{1, 2}, false},
		{[]int{1, 2, 3}, []int{4, 5, 6}, false},
		{[]int{}, []int{}, true},
		{[]int{1}, []int{}, false},
		{[]int{}, []int{1}, false},
	}

	for _, tt := range tests {
		result := SliceEquals(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("SliceEquals(%v, %v) = %v; want %v", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestSliceEquals_String(t *testing.T) {
	tests := []struct {
		a, b     []string
		expected bool
	}{
		{[]string{"a", "b", "c"}, []string{"c", "b", "a"}, true},
		{[]string{"a", "a", "b"}, []string{"b", "a", "a"}, true},
		{[]string{"a", "b", "c"}, []string{"a", "b"}, false},
		{[]string{"a", "b", "c"}, []string{"x", "y", "z"}, false},
		{[]string{}, []string{}, true},
		{[]string{"a"}, []string{}, false},
		{[]string{}, []string{"a"}, false},
	}

	for _, tt := range tests {
		result := SliceEquals(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("SliceEquals(%v, %v) = %v; want %v", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestReverseSlice_Int(t *testing.T) {
	tests := []struct {
		input    []int
		expected []int
	}{
		{[]int{1, 2, 3, 4}, []int{4, 3, 2, 1}},
		{[]int{1, 2, 3}, []int{3, 2, 1}},
		{[]int{1}, []int{1}},
		{[]int{}, []int{}},
		{[]int{5, 5, 5}, []int{5, 5, 5}},
	}

	for _, tt := range tests {
		inputCopy := make([]int, len(tt.input))
		copy(inputCopy, tt.input)
		ReverseSlice(inputCopy)
		if !SliceEquals(inputCopy, tt.expected) {
			t.Errorf("ReverseSlice(%v) = %v; want %v", tt.input, inputCopy, tt.expected)
		}
	}
}

func TestReverseSlice_String(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{[]string{"a", "b", "c"}, []string{"c", "b", "a"}},
		{[]string{"hello"}, []string{"hello"}},
		{[]string{}, []string{}},
		{[]string{"x", "y", "x"}, []string{"x", "y", "x"}},
	}

	for _, tt := range tests {
		inputCopy := make([]string, len(tt.input))
		copy(inputCopy, tt.input)
		ReverseSlice(inputCopy)
		if !SliceEquals(inputCopy, tt.expected) {
			t.Errorf("ReverseSlice(%v) = %v; want %v", tt.input, inputCopy, tt.expected)
		}
	}
}

func TestRandomString(t *testing.T) {
	base := "testbase"
	result1, err1 := RandomString(base)
	if err1 != nil {
		t.Fatalf("RandomString(%q) returned error: %v", base, err1)
	}
	if len(result1) <= len(base)+1 {
		t.Errorf("RandomString(%q) = %q; expected longer string with random suffix", base, result1)
	}
	if got, want := result1[:len(base)], base; got != want {
		t.Errorf("RandomString(%q) prefix = %q; want %q", base, got, want)
	}
	if result1[len(base)] != '-' {
		t.Errorf("RandomString(%q) missing '-' after base: %q", base, result1)
	}
	// Check that the suffix is 16 hex characters (8 bytes)
	suffix := result1[len(base)+1:]
	if len(suffix) != 16 {
		t.Errorf("RandomString(%q) suffix length = %d; want 16", base, len(suffix))
	}
	// Check that two calls produce different results (very likely)
	result2, err2 := RandomString(base)
	if err2 != nil {
		t.Fatalf("RandomString(%q) returned error: %v", base, err2)
	}
	if result1 == result2 {
		t.Errorf("RandomString(%q) produced duplicate results: %q", base, result1)
	}
}

func TestRandomString_Error(t *testing.T) {
	// It's difficult to force rand.Read to fail, so this test is mostly for coverage.
	// If you want to simulate an error, you would need to refactor RandomString to accept a rand.Reader.
	// For now, just ensure it doesn't return error in normal use.
	_, err := RandomString("base")
	if err != nil {
		t.Errorf("RandomString returned unexpected error: %v", err)
	}
}
