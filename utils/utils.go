// Package utils provides various utility functions for string manipulation, color conversion, and slice operations.
package utils

import (
	"crypto/rand"
	"fmt"
	"math"

	"slices"

	"golang.org/x/exp/constraints"
)

// ShortenString shortens a string to a given length and appends "..." if it exceeds that length.
func ShortenString(s string, l int) string {
	if len(s) > l && l != 0 {
		return fmt.Sprintf("%s...", s[:l])
	}
	return s
}

// HSVToRGB converts HSV color values to RGB color values.
func HSVToRGB(h, s, v float64) (int32, int32, int32) {
	// from https://go.dev/play/p/9q5yBNDh3W
	var r, g, b float64
	h = h * 6
	i := math.Floor(h)
	v1 := v * (1 - s)
	v2 := v * (1 - s*(h-i))
	v3 := v * (1 - s*(1-(h-i)))

	if i == 0 {
		r = v
		g = v3
		b = v1
	} else if i == 1 {
		r = v2
		g = v
		b = v1
	} else if i == 2 {
		r = v1
		g = v
		b = v3
	} else if i == 3 {
		r = v1
		g = v2
		b = v
	} else if i == 4 {
		r = v3
		g = v1
		b = v
	} else {
		r = v
		g = v1
		b = v2
	}

	r = r * 255 //RGB results from 0 to 255
	g = g * 255
	b = b * 255
	return int32(r), int32(g), int32(b)
}

// MostOcc returns the most occurring element in a slice of comparable elements.
func MostOcc[T comparable](items []T) T {
	count := map[T]int{}
	for _, item := range items {
		count[item]++
	}
	var mostOcc T
	maxOcc := 0
	for item, c := range count {
		if c > maxOcc {
			maxOcc = c
			mostOcc = item
		}
	}
	return mostOcc
}

// ContainsDigits checks if a string contains any digit characters.
func ContainsDigits(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// OnlyContainsDigits checks if a string contains only digit characters.
func OnlyContainsDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// IntersectionSlices returns the intersection of two slices.
func IntersectionSlices[T constraints.Ordered](a, b []T) []T {
	slices.Sort(a)
	slices.Sort(b)
	result := []T{}
	for j, k := 0, 0; j < len(a) && k < len(b); {
		if a[j] == b[k] {
			result = append(result, a[j])
			j++
			k++
		} else if a[j] > b[k] {
			k++
		} else {
			j++
		}
	}
	return result
}

// SliceEquals checks if two slices are equal, ignoring order.
func SliceEquals[T constraints.Ordered](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	slices.Sort(a)
	slices.Sort(b)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ReverseSlice reverses a slice in place.
func ReverseSlice[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// RandomString generates a random string prepended with a base string.
func RandomString(base string) (string, error) {
	bs := make([]byte, 8)
	_, err := rand.Read(bs)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %v", err)
	}
	return fmt.Sprintf("%s-%x", base, bs[:8]), nil
}
