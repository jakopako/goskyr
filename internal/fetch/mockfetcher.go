package fetch

import (
	"context"
	"errors"

	"github.com/jakopako/goskyr/internal/log"
)

type MockFetcher struct {
	*FetcherConfig
	pagesMap map[string]string
}

func NewMockFetcher(fc *FetcherConfig) *MockFetcher {
	df := &MockFetcher{
		FetcherConfig: fc,
		pagesMap:      map[string]string{},
	}
	for _, p := range fc.MockPages {
		df.pagesMap[p.Url] = p.Content
	}
	return df
}

func (d *MockFetcher) Fetch(ctx context.Context, urlStr string, opts FetchOpts) (string, error) {
	if p, ok := d.pagesMap[urlStr]; ok {
		if log.Debug {
			writeHTMLToFile(ctx, urlStr, p, d.DebugDir)
		}
		return p, nil
	}

	return "", errors.New("page not found")
}

// To comply with the Fetcher interface
func (df *MockFetcher) Cancel() {}
