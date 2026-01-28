package fetch

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/jakopako/goskyr/internal/log"
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

func (s *StaticFetcher) Fetch(ctx context.Context, url string, opts FetchOpts) (string, error) {
	logger := log.LoggerFromContext(ctx)
	logger.Debug("fetching page", slog.String("fetcher", "static"), slog.String("url", url), slog.String("user-agent", s.UserAgent))
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
	if log.Debug {
		writeHTMLToFile(ctx, url, resString, s.DebugDir)
	}
	return resString, nil
}

func (s *StaticFetcher) Cancel() {}
