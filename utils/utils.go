package utils

import (
	"fmt"
)

func ShortenString(s string, l int) string {
	if len(s) > l {
		return fmt.Sprintf("%s...", s[:l-3])
	}
	return s
}
