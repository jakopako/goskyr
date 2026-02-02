// Package generate provides functionality to generate scraper configurations
// by analyzing web pages and labeling found fields.
package generate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jakopako/goskyr/internal/fetch"
	"github.com/jakopako/goskyr/internal/scraper"
)

type Config struct {
	MinOccurrences int                 `yaml:"min_occurrences"`
	DistinctValues bool                `yaml:"distinct_values"` // if true, only fields with distinct values will be included
	LablerConfig   LablerConfig        `yaml:"labler"`
	FetcherConfig  fetch.FetcherConfig `yaml:"fetcher"`
}

func NewConfigFromFile(path string) (*Config, error) {
	var config Config

	err := cleanenv.ReadConfig(path, &config)
	if err != nil {
		return nil, err
	}

	if config.FetcherConfig.Type == "" {
		config.FetcherConfig.Type = fetch.DefaultFetcherType()
	}

	return &config, err
}

// GenerateConfig generates a scraper configuration for the given scraper s's URL
// by analyzing the HTML structure and allowing the user to select fields interactively.
// minOcc specifies the minimum occurrences a field must have to be included.
// If removeStaticFields is true, fields that have static values will be removed from the configuration.
// modelName and wordsDir are used for text analysis to predict field names.
func GenerateConfig(s *scraper.Scraper, gc *Config, interactive bool) error {
	slog.Info(fmt.Sprintf("analyzing url %s", s.URL))
	if s.URL == "" {
		return errors.New("URL field cannot be empty")
	}
	s.Name = s.URL

	fetcher, err := fetch.NewFetcher(&gc.FetcherConfig)
	if err != nil {
		return fmt.Errorf("error creating fetcher: %v", err)
	}

	// currently the ctx is only used to pass a logger. If it
	// we don't need a custom logger, we can just use context.Background()
	// and anything that gets the logger from the context will use the default logger,
	// IF the log.LoggerFromContext function is used.
	ctx := context.Background()
	res, err := fetcher.Fetch(ctx, s.URL, fetch.FetchOpts{})
	if err != nil {
		return err
	}

	// A bit hacky. But goquery seems to manipulate the html (I only know of goquery adding tbody tags if missing)
	// so we rely on goquery to read the html for both scraping AND figuring out the scraping config.
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(res))
	if err != nil {
		return err
	}

	// Now we have to translate the goquery doc back into a string
	htmlStr, err := goquery.OuterHtml(doc.Children())
	if err != nil {
		return err
	}

	fieldMgr := newFieldManagerFromHtml(htmlStr)
	err = fieldMgr.process(gc)
	if err != nil {
		return err
	}

	return fieldMgr.fieldSelection(s, interactive)
}
