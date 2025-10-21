package autoconfig

import (
	"errors"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/jakopako/goskyr/fetch"
	"github.com/jakopako/goskyr/scraper"
)

func GetDynamicFieldsConfig(s *scraper.Scraper, minOcc int, removeStaticFields bool, modelName, wordsDir string) error {
	if s.URL == "" {
		return errors.New("URL field cannot be empty")
	}
	s.Name = s.URL

	fetcher, err := fetch.NewFetcher(&s.FetcherConfig)
	if err != nil {
		return fmt.Errorf("error creating fetcher: %v", err)
	}

	res, err := fetcher.Fetch(s.URL, fetch.FetchOpts{})
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
	err = fieldMgr.process(minOcc, removeStaticFields, modelName, wordsDir)
	if err != nil {
		return err
	}

	return fieldMgr.interactiveFieldSelection(s)
}
