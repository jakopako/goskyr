package automate

import (
	"errors"
	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/jakopako/goskyr/scraper"
	"github.com/jakopako/goskyr/utils"
)

func GetDynamicFieldsConfig(s *scraper.Scraper, g *scraper.GlobalConfig) error {
	if s.URL == "" {
		return errors.New("URL field cannot be empty")
	}
	res, err := utils.FetchUrl(s.URL, g.UserAgent)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return err
	}
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		fmt.Println(s.Text())
	})

	return nil
}
