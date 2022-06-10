package automate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jakopako/goskyr/scraper"
	"github.com/jakopako/goskyr/utils"
	"golang.org/x/net/html"
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
	// body > div.content > div.mainContentContainer > div.mainContent > div.mainContentFloat > h1
	// body > div.content > div.mainContentContainer > div.mainContent > div.mainContentFloat > div.leftContainer > div:nth-child(2) > div.quoteDetails > div.quoteText
	// body > div.content > div.mainContentContainer > div.mainContent > div.mainContentFloat > div.leftContainer > div:nth-child(2) > div.quoteDetails > div.quoteText > span
	z := html.NewTokenizer(res.Body)
	nodePath := []string{}
	depth := 0
	inBody := false
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return z.Err()
		case html.TextToken:
			if inBody {
				text := string(z.Text())
				if strings.TrimSpace(text) != "" {
					fmt.Printf("Text at path %s with depth %d: %s\n", strings.Join(nodePath, " > "), depth, text)
				}
			}
		case html.StartTagToken, html.EndTagToken:
			tn, _ := z.TagName()
			tnString := string(tn)
			if tnString == "br" {
				continue
			}
			if inBody {
				if tt == html.StartTagToken {
					// fmt.Printf("<%s>\n", tnString)
					nodePath = append(nodePath, tnString)
					depth++
				} else {
					if nodePath[len(nodePath)-1] == tnString {
						nodePath = nodePath[:len(nodePath)-1]
					}
					// fmt.Printf("</%s>\n", tnString)
					depth--
				}
			} else {
				if tnString == "body" {
					inBody = !inBody
				}
			}
		}
	}

	return nil
}
