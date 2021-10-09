package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
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
}

type helsinkiCrawler struct{}

type mehrspurCrawler struct{}

func (helsinkiCrawler) getConcerts() []Concert {
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
							Location: "Helsinki",
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

func (mehrspurCrawler) getConcerts() []Concert {
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
								//fmt.Println(attr.Val)
								postSection = true
								if currentConcert.Artist != "" {
									concerts = append(concerts, currentConcert)
								}
								currentConcert = Concert{Location: "Mehrspur"}
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
								//fmt.Println(attr.Val)
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
				//fmt.Println(html.UnescapeString(token.String()))
				headerPostSection = false
				currentConcert.Artist = html.UnescapeString(token.String())
			} else if dateSection {
				dateSection = false
				dateString = html.UnescapeString(token.String())
				//dateString = dateString[3:]
				//fmt.Println(dateString)
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
				//fmt.Println(t)
			} else if commentSection {
				//fmt.Println(html.UnescapeString(token.String()))
				commentSection = false
				postSection = false
				currentConcert.Comment = html.UnescapeString(token.String())
			} else if !postSection && previousToken.DataAtom == atom.P {
				re := regexp.MustCompile("^20[0-9]{2}")
				match := re.Match([]byte(token.String()))
				if match {
					yearString = token.String()
					//fmt.Println(yearString)
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

func prettyPrintConcerts(concerts []Concert) {
	for _, concert := range concerts {
		fmt.Printf("Artist: %v\nLocation: %v\nDate: %v\nLink: %v\nComment: %v\n\n",
			concert.Artist, concert.Location, concert.Date, concert.Link, concert.Comment)
	}
}

func main() {
	//writeConcertsToAPI(helsinkiCrawler{})
	writeConcertsToAPI(mehrspurCrawler{})
}
