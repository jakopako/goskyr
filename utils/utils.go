package utils

import (
	"fmt"
	"net/http"
)

func FetchUrl(url string, userAgent string) (*http.Response, error) {
	// NOTE: body has to be closed by caller
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if userAgent == "" {
		req.Header.Set("User-Agent", "goskyr web scraper (github.com/jakopako/goskyr)")
	} else {
		req.Header.Set("User-Agent", userAgent)
	}
	req.Header.Set("Accept", "*/*")
	return client.Do(req)
}

func ShortenString(s string, l int) string {
	if len(s) > l {
		return fmt.Sprintf("%s...", s[:l-3])
	}
	return s
}
