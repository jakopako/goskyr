// Package fetch provides functionality to fetch web pages, both static and dynamic.
package fetch

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/jakopako/goskyr/log"
	"github.com/jakopako/goskyr/types"
	"github.com/jakopako/goskyr/utils"
)

type FetcherType string

const (
	STATIC_FETCHER_TYPE  FetcherType = "static"
	DYNAMIC_FETCHER_TYPE FetcherType = "dynamic"
	MOCK_FETCHER_TYPE    FetcherType = "mock"
)

type MockPage struct {
	Url     string `yaml:"url"`
	Content string `yaml:"content"`
}

type FetcherConfig struct {
	Type           FetcherType `yaml:"type"`
	UserAgent      string      `yaml:"user_agent,omitempty"`
	PageLoadWaitMS int         `yaml:"page_load_wait_ms,omitempty"`
	MockPages      []MockPage  `yaml:"mock_pages,omitempty"`
}

type FetchOpts struct {
	Interaction []*types.Interaction
}

// A Fetcher allows to fetch the content of a web page
type Fetcher interface {
	// Fetch retrieves the content of the page at the given URL
	// according to the provided options. The context is used for logging.
	Fetch(ctx context.Context, url string, opts FetchOpts) (string, error)
	Cancel() // only needed for the dynamic fetcher
}

func NewFetcher(fc *FetcherConfig) (Fetcher, error) {
	switch fc.Type {
	case STATIC_FETCHER_TYPE:
		return NewStaticFetcher(fc), nil
	case DYNAMIC_FETCHER_TYPE:
		return NewDynamicFetcher(fc), nil
	case MOCK_FETCHER_TYPE:
		return NewMockFetcher(fc), nil
	default:
		return nil, fmt.Errorf("fetcher of type %s not implemented", fc.Type)
	}
}

// writeHTMLToFile writes the given HTML content to a file for debugging purposes.
func writeHTMLToFile(ctx context.Context, urlStr, htmlStr string) error {
	logger := log.LoggerFromContext(ctx)
	u, _ := url.Parse(urlStr)
	r, err := utils.RandomString(u.Host)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.html", r)
	logger.Debug(fmt.Sprintf("writing html to file %s", filename), slog.String("url", urlStr))
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to write html file: %v", err)
	}
	defer f.Close()

	_, err = f.WriteString(htmlStr)
	if err != nil {
		return fmt.Errorf("failed to write html file: %v", err)
	}
	return nil
}
