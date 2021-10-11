package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/goodsign/monday"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// type Config struct {
// 	DB struct {
// 		User     string `yaml:"user"`
// 		Password string `yaml:"password"`
// 		Cluster  string `yaml:"cluster"`
// 		Database string `yaml:"database"`
// 	} `yaml:"db"`
// }

// TODO: it's ugly to copy paste this from the croncert-api project.
type Concert struct {
	Artist   string    `bson:"artist,omitempty" json:"artist,omitempty" validate:"required" example:"SuperArtist"`
	Location string    `bson:"location,omitempty" json:"location,omitempty" validate:"required" example:"SuperLocation"`
	Date     time.Time `bson:"date,omitempty" json:"date,omitempty" validate:"required" example:"2021-10-31T19:00:00.000Z"`
	Link     string    `bson:"link,omitempty" json:"link,omitempty" validate:"required,url" example:"http://link.to/concert/page"`
	Comment  string    `bson:"comment,omitempty" json:"comment,omitempty" example:"Super exciting comment."`
}

type concertCrawler interface {
	getConcerts() []Concert
	getName() string
}

type helsinkiCrawler struct {
	Name string
}

func NewHelsinkiCrawler() helsinkiCrawler {
	return helsinkiCrawler{
		Name: "Helsinki",
	}
}

type mehrspurCrawler struct {
	Name string
}

func NewMehrspurCrawler() mehrspurCrawler {
	return mehrspurCrawler{
		Name: "Mehrspur",
	}
}

type umboCrawler struct {
	Name string
}

func NewUmboCrawler() umboCrawler {
	return umboCrawler{
		Name: "Umbo",
	}
}

type moodsCrawler struct {
	Name string
}

func NewMoodsCrawler() moodsCrawler {
	return moodsCrawler{
		Name: "Moods",
	}
}

// Next:
//  + Moods (https://www.moods.club/en/)
//  + Bogen F (https://www.bogenf.ch/konzerte/aktuell/)
//  + Kasheme (https://kasheme.com/program/)

func (c helsinkiCrawler) getName() string {
	return c.Name
}

func (c helsinkiCrawler) getConcerts() []Concert {
	log.Println("Fetching Helsinki concerts.")
	url := "https://www.helsinkiklub.ch/"
	concerts := []Concert{}
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
	z := html.NewTokenizer(res.Body)
	var currentConcert Concert
	var previousToken, token html.Token
	token = html.Token{}
	var day, month string
	parse := true
	for parse {
		tokenType := z.Next()
		previousToken = token
		token = z.Token()
		if tokenType == html.ErrorToken {
			break
		}
		if tokenType == html.StartTagToken {
			if token.DataAtom == atom.Div {
				for _, attr := range token.Attr {
					if attr.Key == "class" && attr.Val == "event" {
						if currentConcert.Artist != "" {
							// Occasionally, the year of the concert is wrong even though we try
							// to parse it from the context, e.g. because there is simply no year.
							// Therefore we apply the following check.
							currentTime := time.Now()
							if currentTime.After(currentConcert.Date) {
								d := currentConcert.Date
								year := currentTime.Year() + 1
								currentConcert.Date = time.Date(int(year), d.Month(), d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
							}
							concerts = append(concerts, currentConcert)
						}
						currentConcert = Concert{
							Location: c.Name,
							Link:     url}
					}
					if attr.Key == "id" && attr.Val == "col2" {
						parse = false
						break
					}
				}
			}
		}
		if tokenType == html.TextToken {
			for _, attr := range previousToken.Attr {
				if attr.Key == "class" {
					switch attr.Val {
					case "top":
						currentConcert.Artist = html.UnescapeString(token.String())
					case "day":
						day = token.String()
					case "month":
						month = token.String()
						year := time.Now().Year()
						loc, _ := time.LoadLocation("Europe/Berlin")
						layout := "2 January 2006 15:04"
						d := fmt.Sprintf("%s %s %d 20:00", day, month, year)
						t, err := monday.ParseInLocation(layout, d, loc, monday.LocaleDeDE)
						if err != nil {
							log.Fatalf("Couldn't parse date %s: %v", d, err)
						}
						currentConcert.Date = t
					case "addition":
						currentConcert.Comment = html.UnescapeString(token.String())
						// sometimes the year of a concert can be found in the comment.
						re := regexp.MustCompile("20[0-9]{2}")
						match := re.FindString(currentConcert.Comment)
						if len(match) > 0 {
							d := currentConcert.Date
							year, _ := strconv.Atoi(match) // we ignore the error because the regex ensures that it's an int.
							currentConcert.Date = time.Date(int(year), d.Month(), d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
						}
					}
				}
			}
		}
	}
	concerts = append(concerts, currentConcert)
	return concerts
}

func (c mehrspurCrawler) getName() string {
	return c.Name
}

func (c mehrspurCrawler) getConcerts() []Concert {
	log.Println("Fetching Mehrspur concerts.")
	url := "https://www.mehrspur.ch/veranstaltungen"
	concerts := []Concert{}
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
	z := html.NewTokenizer(res.Body)
	var currentConcert Concert
	var token, previousToken html.Token
	token = html.Token{}
	// var day, month string
	postSection, headerPostSection, dateSection, timeSection, commentSection := false, false, false, false, false
	var dateString string
	var yearString string
	for {
		tokenType := z.Next()
		previousToken = token
		token = z.Token()
		if tokenType == html.ErrorToken {
			break
		}
		if tokenType == html.StartTagToken {
			if !postSection {
				if token.DataAtom == atom.Div {
					for _, attr := range token.Attr {
						if attr.Key == "id" {
							re := regexp.MustCompile("^post-[0-9]{5}$")
							match := re.Match([]byte(attr.Val))
							if match {
								postSection = true
								if currentConcert.Artist != "" {
									concerts = append(concerts, currentConcert)
								}
								currentConcert = Concert{Location: c.Name}
							}
						}
					}
				}
			} else {
				if token.DataAtom == atom.H3 {
					for _, attr := range token.Attr {
						if attr.Key == "class" && attr.Val == "block_under_title" {
							headerPostSection = true
						}
					}
				} else if headerPostSection {
					if token.DataAtom == atom.A {
						for _, attr := range token.Attr {
							if attr.Key == "href" {
								currentConcert.Link = attr.Val
							}
						}
					}
				} else if token.DataAtom == atom.Li {
					for _, attr := range token.Attr {
						if attr.Key == "class" {
							if attr.Val == "event-date" {
								dateSection = true
							} else if attr.Val == "event-time" {
								timeSection = true
							}
						}
					}
				} else if token.DataAtom == atom.Div {
					for _, attr := range token.Attr {
						if attr.Key == "class" && attr.Val == "event-excerpt-fluid" {
							commentSection = true
						}
					}
				}
			}
		} else if tokenType == html.TextToken {
			if headerPostSection {
				headerPostSection = false
				currentConcert.Artist = html.UnescapeString(token.String())
			} else if dateSection {
				dateSection = false
				dateString = html.UnescapeString(token.String())
			} else if timeSection {
				timeSection = false
				loc, _ := time.LoadLocation("Europe/Berlin")
				layout := "Mon 2.Jan. 2006 15:04"
				d := fmt.Sprintf("%s %s %s", dateString, yearString, token.String())
				t, err := monday.ParseInLocation(layout, d, loc, monday.LocaleDeDE)
				if err != nil {
					log.Fatalf("Couldn't parse date %s: %v", d, err)
				}
				currentConcert.Date = t
			} else if commentSection {
				commentSection = false
				postSection = false
				currentConcert.Comment = html.UnescapeString(token.String())
			} else if !postSection && previousToken.DataAtom == atom.P {
				re := regexp.MustCompile("^20[0-9]{2}")
				match := re.Match([]byte(token.String()))
				if match {
					yearString = token.String()
				}
			}
		}
	}
	concerts = append(concerts, currentConcert)
	return concerts

}

func (c umboCrawler) getName() string {
	return c.Name
}

func (c umboCrawler) getConcerts() []Concert {
	log.Println("Fetching Umbo concerts.")
	url := "https://www.umbo.wtf/"
	concerts := []Concert{}
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
	z := html.NewTokenizer(res.Body)
	var currentConcert Concert
	var token, previousToken html.Token
	token = html.Token{}
	for {
		tokenType := z.Next()
		previousToken = token
		token = z.Token()
		if tokenType == html.ErrorToken {
			break
		}
		if tokenType == html.TextToken {
			for _, attr := range previousToken.Attr {
				if attr.Key == "class" {
					switch attr.Val {
					case "text-block-26":
						if currentConcert.Artist != "" {
							concerts = append(concerts, currentConcert)
						}
						loc, _ := time.LoadLocation("Europe/Berlin")
						layout := "2.1.2006 15:04"
						// d := fmt.Sprintf("%s %s %s", dateString, yearString, token.String())
						t, err := monday.ParseInLocation(layout, token.String(), loc, monday.LocaleDeDE)
						if err != nil {
							log.Fatalf("Couldn't parse date %s: %v", token.String(), err)
						}
						currentConcert = Concert{
							Location: c.Name,
							Date:     t,
						}
						//fmt.Println(t)
					case "text-block-21":
						//fmt.Println(token.String())
						currentConcert.Artist = html.UnescapeString(token.String())
					case "text-block-28":
						//fmt.Println(token.String())
						currentConcert.Comment = html.UnescapeString(token.String())
					}
				}
			}
			if token.String() == "mehr erfahren" {
				for _, attr := range previousToken.Attr {
					if attr.Key == "href" {
						currentConcert.Link = fmt.Sprintf("%s%s", strings.TrimRight(url, "/"), attr.Val)
					}
				}
			}
		}
	}
	concerts = append(concerts, currentConcert)
	return concerts
}

func (c moodsCrawler) getName() string {
	return c.Name
}

func (c moodsCrawler) getConcerts() []Concert {
	log.Println("Fetching Moods concerts.")
	url := "https://www.moods.club/en/?a=1"
	baseUrl := "https://www.moods.club"
	concerts := []Concert{}
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
	z := html.NewTokenizer(res.Body)
	var currentConcert Concert
	var token, previousToken html.Token
	var day, month string
	year := time.Now().Year()
	token = html.Token{}
	for {
		tokenType := z.Next()
		previousToken = token
		token = z.Token()
		if tokenType == html.ErrorToken {
			break
		}
		if tokenType == html.StartTagToken {
			if token.DataAtom == atom.Div {
				for _, attr := range token.Attr {
					if attr.Key == "class" && strings.HasPrefix(attr.Val, "event") {
						if currentConcert.Artist != "" {
							concerts = append(concerts, currentConcert)
						}
						currentConcert = Concert{
							Location: c.Name,
						}
					}
				}
			} else if token.DataAtom == atom.A {
				if currentConcert.Link == "" {
					for _, attr := range token.Attr {
						if attr.Key == "href" {
							// TODO: get all relevant information from those subpages.
							// Should be way easier to extract.
							currentConcert.Link = fmt.Sprintf("%s%s", baseUrl, attr.Val)
						}
					}
				}
			}
		} else if tokenType == html.TextToken {
			if previousToken.Type == html.StartTagToken {
				if previousToken.DataAtom == atom.Span {
					for _, attr := range previousToken.Attr {
						if attr.Key == "class" {
							switch attr.Val {
							case "day":
								day = token.String()
								fmt.Println(day)
							case "month_name":
								month = token.String()
								fmt.Println(month)
								fmt.Println(year)
							}
						}
					}
				} else if previousToken.DataAtom == atom.H2 {
					currentConcert.Artist = html.UnescapeString(token.String())
				} else if previousToken.DataAtom == atom.Div {
					for _, attr := range previousToken.Attr {
						if attr.Key == "class" && attr.Val == "content" && currentConcert.Comment == "" {
							currentConcert.Comment = html.UnescapeString(token.String())
						}
					}
				}
			}
		}
	}
	concerts = append(concerts, currentConcert)
	return concerts
}

func writeConcertsToAPI(c concertCrawler) {
	apiUrl := os.Getenv("CRONCERT_API")
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	for _, concert := range c.getConcerts() {
		concertJSON, err := json.Marshal(concert)
		if err != nil {
			log.Fatal(err)
		}
		req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(concertJSON))
		req.Header = map[string][]string{
			"Content-Type": {"application/json"},
		}
		req.SetBasicAuth(os.Getenv("API_POST_USER"), os.Getenv("API_POST_PASSWORD"))
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode != 201 {
			log.Fatalf("Something went wrong while adding a new concert. Status Code: %d", resp.StatusCode)

		}
	}
}

func prettyPrintConcerts(c concertCrawler) {
	for _, concert := range c.getConcerts() {
		fmt.Printf("Artist: %v\nLocation: %v\nDate: %v\nLink: %v\nComment: %v\n\n",
			concert.Artist, concert.Location, concert.Date, concert.Link, concert.Comment)
	}
}

func main() {
	everyCrawler := flag.Bool("all", false, "Use this flag to indicate that all crawlers should be run.")
	singleCrawler := flag.String("single", "", "The name of the crawler to be run.")
	storeData := flag.Bool("store", false, "If set to true the crawled data will be written to the API.")

	flag.Parse()

	allCrawlers := []concertCrawler{
		NewHelsinkiCrawler(),
		NewMehrspurCrawler(),
		NewUmboCrawler(),
		NewMoodsCrawler(),
	}

	var todoCrawlers []concertCrawler
	if *everyCrawler {
		todoCrawlers = allCrawlers
	} else {
		for _, c := range allCrawlers {
			if c.getName() == *singleCrawler {
				todoCrawlers = append(todoCrawlers, c)
			}
		}
	}

	for _, c := range todoCrawlers {
		if *storeData {
			writeConcertsToAPI(c)
		} else {
			prettyPrintConcerts(c)
		}
	}
}
