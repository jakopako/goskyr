package automate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jakopako/goskyr/scraper"
	"github.com/jakopako/goskyr/utils"
	"golang.org/x/net/html"
)

func getSelector(pathSlice []string) string {
	return strings.Join(pathSlice, " > ")
}

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
	locOcc := map[scraper.ElementLocation]int{}
	nrChildren := map[string]int{}
	nodePath := []string{}
	depth := 0
	inBody := false
parse:
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			break parse
		case html.TextToken:
			if inBody {
				text := string(z.Text())
				p := getSelector(nodePath)
				if strings.TrimSpace(text) != "" {
					l := scraper.ElementLocation{
						Selector:   p,
						ChildIndex: nrChildren[p],
					}
					if nr, found := locOcc[l]; found {
						locOcc[l] = nr + 1
					} else {
						locOcc[l] = 1
					}
				}
				nrChildren[p] += 1
			}
		case html.StartTagToken, html.EndTagToken:
			tn, _ := z.TagName()
			tnString := string(tn)
			if tnString == "body" {
				inBody = !inBody
			}
			if inBody {
				// what type of token is <br /> ? Same as <br> ?
				if tnString == "br" {
					nrChildren[getSelector(nodePath)] += 1
					continue
				}
				if tt == html.StartTagToken {
					nrChildren[getSelector(nodePath)] += 1
					moreAttr := true
					for moreAttr {
						k, v, m := z.TagAttr()
						if string(k) == "class" && string(v) != "" {
							tnString += fmt.Sprintf(".%s", strings.Replace(strings.TrimSpace(string(v)), " ", ".", -1))
						}
						moreAttr = m
					}
					if tnString != "br" {
						nodePath = append(nodePath, tnString)
						nrChildren[getSelector(nodePath)] = 0
						depth++
					}
				} else {
					if strings.Split(nodePath[len(nodePath)-1], ".")[0] == tnString {
						delete(nrChildren, getSelector(nodePath))
						nodePath = nodePath[:len(nodePath)-1]
						depth--
						if tnString == "body" {
							break parse
						}
					}
				}
			}
		}
	}
	for k, v := range locOcc {
		if v > 10 {
			fmt.Println(k, v)
		}
	}
	return nil
}
