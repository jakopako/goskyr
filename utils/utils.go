package utils

import (
	"fmt"
	"math"
	"sort"

	"golang.org/x/exp/constraints"
)

func ShortenString(s string, l int) string {
	if len(s) > l && l != 0 {
		return fmt.Sprintf("%s...", s[:l])
	}
	return s
}

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

func MostOcc[T comparable](predictions []T) T {
	count := map[T]int{}
	for _, pred := range predictions {
		count[pred]++
	}
	var pred T
	maxOcc := 0
	for p, c := range count {
		if c > maxOcc {
			maxOcc = c
			pred = p
		}
	}
	return pred
}

func RuneIsOneOf(r rune, rs []rune) bool {
	for _, ru := range rs {
		if r == ru {
			return true
		}
	}
	return false
}

func ContainsDigits(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func OnlyContainsDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func SortSlice[T constraints.Ordered](s []T) {
	sort.Slice(s, func(i, j int) bool {
		return s[i] < s[j]
	})
}

func IntersectionSlices[T constraints.Ordered](a, b []T) []T {
	SortSlice(a)
	SortSlice(b)
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

func SliceEquals[T constraints.Ordered](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	SortSlice(a)
	SortSlice(b)
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func ReverseSlice[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
