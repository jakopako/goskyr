package fetch

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// The StaticFetcher fetches static page content
type StaticFetcher struct {
	*FetcherConfig
}

func NewStaticFetcher(fc *FetcherConfig) *StaticFetcher {
	return &StaticFetcher{
		FetcherConfig: fc,
	}
}

func (s *StaticFetcher) Fetch(url string, opts FetchOpts) (string, error) {
	slog.Debug("fetching page", slog.String("fetcher", "static"), slog.String("url", url), slog.String("user-agent", s.UserAgent))
	var resString string
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return resString, err
	}
	req.Header.Set("User-Agent", s.UserAgent)
	req.Header.Set("Accept", "*/*")
	res, err := client.Do(req)
	if err != nil {
		return resString, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return resString, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return resString, err
	}
	resString = string(bytes)
	return resString, nil
}

func (s *StaticFetcher) Cancel() {}
