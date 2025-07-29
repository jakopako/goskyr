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
	DUMMY_FETCHER_TYPE   FetcherType = "dummy"
)

type DummyPage struct {
	Url     string `yaml:"url"`
	Content string `yaml:"content"`
}

type FetcherConfig struct {
	Type           FetcherType `yaml:"type"`
	UserAgent      string      `yaml:"user_agent"`
	PageLoadWaitMS int         `yaml:"page_load_wait_ms"`
	DummyPages     []DummyPage `yaml:"dummy_pages"`
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
	case DUMMY_FETCHER_TYPE:
		return NewDummyFetcher(fc), nil
	default:
		return nil, fmt.Errorf("fetcher of type %s not implemented", fc.Type)
	}
}
