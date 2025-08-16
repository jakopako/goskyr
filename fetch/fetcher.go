// Package fetch provides functionality to fetch web pages, both static and dynamic.
package fetch

import (
	"fmt"

	"github.com/jakopako/goskyr/types"
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
	Fetch(url string, opts FetchOpts) (string, error)
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
