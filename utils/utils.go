package utils

import (
	"fmt"
	"math"
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
