package utils

import (
	"context"
	"fmt"
	"net/http"

	"github.com/chromedp/chromedp"
)

func FetchUrl(url, userAgent string) (*http.Response, error) {
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

func FetchUrlChrome(url, userAgent string) {
	// have some kind of fetcher interface

	// checkout https://github.com/geziyor/geziyor/blob/738852f9321de26c193ae88a9b2fb4d6aebb6540/client/client.go#L169
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	if err := chromedp.Run(ctx, 
		chromedp.Navigate(url),
		chromedp.WaitVisible(`body`),
	)
}
