package fetch

import (
	"errors"
)

type DummyFetcher struct {
	*FetcherConfig
	pagesMap map[string]string
}

func NewDummyFetcher(fc *FetcherConfig) *DummyFetcher {
	df := &DummyFetcher{
		FetcherConfig: fc,
		pagesMap:      map[string]string{},
	}
	for _, p := range fc.DummyPages {
		df.pagesMap[p.Url] = p.Content
	}
	return df
}

func (d *DummyFetcher) Fetch(urlStr string, opts FetchOpts) (string, error) {
	if p, ok := d.pagesMap[urlStr]; ok {
		return p, nil
	}
	return "", errors.New("page not found")
}

// To comply with the Fetcher interface
func (df *DummyFetcher) Cancel() {}
