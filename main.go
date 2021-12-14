package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/goodsign/monday"
	"gopkg.in/yaml.v2"
)

type EventType string

const (
	Concert EventType = "concert"
)

func (et EventType) IsValid() error {
	switch et {
	case Concert:
		return nil
	}
	errorString := fmt.Sprintf("invalid event type: %s", et)
	return errors.New(errorString)
}

// TODO: it's ugly to copy paste this from the croncert-api project.
type Event struct {
	Title    string    `bson:"title,omitempty" json:"title,omitempty" validate:"required" example:"ExcitingTitle"`
	Location string    `bson:"location,omitempty" json:"location,omitempty" validate:"required" example:"SuperLocation"`
	City     string    `bson:"city,omitempty" json:"city,omitempty" validate:"required" example:"SuperCity"`
	Date     time.Time `bson:"date,omitempty" json:"date,omitempty" validate:"required" example:"2021-10-31T19:00:00.000Z"`
	URL      string    `bson:"url,omitempty" json:"url,omitempty" validate:"required,url" example:"http://link.to/concert/page"`
	Comment  string    `bson:"comment,omitempty" json:"comment,omitempty" example:"Super exciting comment."`
	Type     EventType `bson:"type,omitempty" json:"type,omitempty" validate:"required" example:"concert"`
}

func (c Crawler) getEvents() ([]Event, error) {
	dynamicFields := []string{"title", "comment", "url", "date"}
	events := []Event{}
	eventType := EventType(c.Type)
	err := eventType.IsValid()
	if err != nil {
		return events, err
	}

	// city
	if c.City == "" {
		err := errors.New("city cannot be an empty string")
		return events, err
	}

	// time zone
	loc, err := time.LoadLocation(c.Fields.Date.Location)
	if err != nil {
		return events, err
	}

	// locale (language)
	mLocale := "de_DE"
	if c.Fields.Date.Language != "" {
		mLocale = c.Fields.Date.Language
	}

	res, err := http.Get(c.URL)
	if err != nil {
		return events, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return events, err
	}

	doc.Find(c.Event).Each(func(i int, s *goquery.Selection) {
		currentEvent := Event{
			Location: c.Name,
			City:     c.City,
			Type:     EventType(c.Type),
		}

		for _, f := range dynamicFields {
			fOnSubpage := false
			for _, sf := range c.Fields.URL.OnSubpage {
				if f == sf {
					fOnSubpage = true
				}
			}
			if !fOnSubpage {
				err := extractField(f, s, &c, &currentEvent, events, loc, mLocale, res)
				if err != nil {
					log.Println(err)
				}
			}
		}

		// extract the url
		// var url string
		// if c.Fields.URL.Loc == "" {
		// 	url = s.AttrOr("href", c.URL)
		// } else {
		// 	url = s.Find(c.Fields.URL.Loc).AttrOr("href", c.URL)
		// }

		// if c.Fields.URL.Relative {
		// 	baseURL := fmt.Sprintf("%s://%s", res.Request.URL.Scheme, res.Request.URL.Host)
		// 	if !strings.HasPrefix(url, "/") {
		// 		baseURL = baseURL + "/"
		// 	}
		// 	url = baseURL + url
		// }
		// currentEvent.URL = url

		if len(c.Fields.URL.OnSubpage) > 0 {
			resSub, err := http.Get(currentEvent.URL)
			if err != nil {
				return
			}

			defer resSub.Body.Close()

			if resSub.StatusCode != 200 {
				log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
			}

			docSub, err := goquery.NewDocumentFromReader(resSub.Body)
			if err != nil {
				log.Fatalf("error while reading document: %v", err)
			}
			for _, item := range c.Fields.URL.OnSubpage {
				err := extractField(item, docSub.Selection, &c, &currentEvent, events, loc, mLocale, resSub)
				if err != nil {
					log.Fatalf("error while parsing field %s: %v", item, err)
				}
			}
		}

		// // extract the title
		// var title string
		// for _, titleLoc := range c.Fields.Title {
		// 	titleSelection := s.Find(titleLoc).First()
		// 	title = strings.TrimSuffix(titleSelection.Text(), titleSelection.Children().Text())
		// 	if title != "" {
		// 		break
		// 	}
		// }
		// if title == "" {
		// 	return
		// }

		// currentEvent.Title = title

		// // extract the comment
		// var comment string
		// for _, commentLoc := range c.Fields.Comment {
		// 	comment = s.Find(commentLoc).Text()
		// 	if comment != "" {
		// 		break
		// 	}
		// }
		// currentEvent.Comment = comment

		// // extract date and time
		// year := time.Now().Year()

		// var dateTimeString, layout string
		// if c.Fields.Date.DayMonthYearTime.Loc != "" {
		// 	dateTimeString = s.Find(c.Fields.Date.DayMonthYearTime.Loc).Text()
		// 	layout = c.Fields.Date.DayMonthYearTime.Layout
		// } else {
		// 	var dayMonthString, dayMonthStringLayout string
		// 	if c.Fields.Date.DayMonth.Loc != "" {
		// 		dayMonthString = s.Find(c.Fields.Date.DayMonth.Loc).Text()
		// 		dayMonthStringLayout = c.Fields.Date.DayMonth.Layout
		// 	} else if c.Fields.Date.Day.Loc != "" && c.Fields.Date.Month.Loc != "" {
		// 		dayString := s.Find(c.Fields.Date.Day.Loc).Text()
		// 		monthString := s.Find(c.Fields.Date.Month.Loc).Text()
		// 		dayMonthString = dayString + " " + monthString
		// 		dayMonthStringLayout = c.Fields.Date.Day.Layout + " " + c.Fields.Date.Month.Layout
		// 	}

		// 	var timeString, timeStringLayout string
		// 	if c.Fields.Date.Time.Loc == "" {
		// 		timeString = "20:00"
		// 		timeStringLayout = "15:04"
		// 	} else {
		// 		timeString = s.Find(c.Fields.Date.Time.Loc).Text()
		// 		timeStringLayout = c.Fields.Date.Time.Layout
		// 	}

		// 	layout = fmt.Sprintf("%s 2006 %s", dayMonthStringLayout, timeStringLayout)
		// 	dateTimeString = fmt.Sprintf("%s %d %s", dayMonthString, year, timeString)
		// }

		// t, err := monday.ParseInLocation(layout, dateTimeString, loc, monday.Locale(mLocale))
		// if err != nil {
		// 	log.Printf("Couldn't parse date: %s", err)
		// 	return
		// }

		// // if the date t does not come after the previous event's date we increase the year by 1
		// // actually this is only necessary if we have to guess the date but currently for ease of implementation
		// // this check is done always.
		// if len(events) > 0 {
		// 	if events[len(events)-1].Date.After(t) {
		// 		t = time.Date(int(year+1), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
		// 	}
		// }
		// currentEvent.Date = t

		events = append(events, currentEvent)
	})

	return events, nil
}

func extractField(item string, s *goquery.Selection, crawler *Crawler, event *Event, events []Event, loc *time.Location, mLocale string, res *http.Response) error {
	switch item {
	case "date":
		// extract date and time
		year := time.Now().Year()

		var dateTimeString, layout string
		if crawler.Fields.Date.DayMonthYearTime.Loc != "" {
			dateTimeString = s.Find(crawler.Fields.Date.DayMonthYearTime.Loc).Text()
			layout = crawler.Fields.Date.DayMonthYearTime.Layout
		} else {
			var dayMonthString, dayMonthStringLayout string
			if crawler.Fields.Date.DayMonth.Loc != "" {
				dayMonthString = s.Find(crawler.Fields.Date.DayMonth.Loc).Text()
				dayMonthStringLayout = crawler.Fields.Date.DayMonth.Layout
			} else if crawler.Fields.Date.Day.Loc != "" && crawler.Fields.Date.Month.Loc != "" {
				dayString := s.Find(crawler.Fields.Date.Day.Loc).Text()
				monthString := s.Find(crawler.Fields.Date.Month.Loc).Text()
				dayMonthString = dayString + " " + monthString
				dayMonthStringLayout = crawler.Fields.Date.Day.Layout + " " + crawler.Fields.Date.Month.Layout
			}

			var timeString, timeStringLayout string
			if crawler.Fields.Date.Time.Loc == "" {
				timeString = "20:00"
				timeStringLayout = "15:04"
			} else {
				// Normally, there should be only one occurence of the time. However, on the
				// Moods website there are two of which we only want the last. Let's see how
				// well this works for other websites.
				timeString = s.Find(crawler.Fields.Date.Time.Loc).Last().Text()
				timeStringLayout = crawler.Fields.Date.Time.Layout
			}

			layout = fmt.Sprintf("%s 2006 %s", dayMonthStringLayout, timeStringLayout)
			dateTimeString = fmt.Sprintf("%s %d %s", dayMonthString, year, timeString)
		}

		t, err := monday.ParseInLocation(layout, dateTimeString, loc, monday.Locale(mLocale))
		if err != nil {
			return err
		}
		// if the date t does not come after the previous event's date we increase the year by 1
		// actually this is only necessary if we have to guess the date but currently for ease of implementation
		// this check is done always.
		if len(events) > 0 {
			if events[len(events)-1].Date.After(t) {
				t = time.Date(int(year+1), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
			}
		}
		event.Date = t
	case "title":
		var title string
		for _, titleLoc := range crawler.Fields.Title {
			titleSelection := s.Find(titleLoc).First()
			title = strings.TrimSuffix(titleSelection.Text(), titleSelection.Children().Text())
			if title != "" {
				break
			}
		}
		if title == "" {
			return errors.New("empty event title")
		}
		event.Title = title
	case "comment":
		var comment string
		for _, commentLoc := range crawler.Fields.Comment {
			comment = strings.TrimSpace(s.Find(commentLoc).First().Text())
			if comment != "" {
				break
			}
		}
		event.Comment = comment
	case "url":
		var url string
		if crawler.Fields.URL.Loc == "" {
			url = s.AttrOr("href", crawler.URL)
		} else {
			url = s.Find(crawler.Fields.URL.Loc).AttrOr("href", crawler.URL)
		}

		if crawler.Fields.URL.Relative {
			baseURL := fmt.Sprintf("%s://%s", res.Request.URL.Scheme, res.Request.URL.Host)
			if !strings.HasPrefix(url, "/") {
				baseURL = baseURL + "/"
			}
			url = baseURL + url
		}
		event.URL = url
	}
	return nil
}

func writeEventsToAPI(c Crawler) {
	apiUrl := os.Getenv("CRONCERT_API")
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	events, err := c.getEvents()

	if err != nil {
		log.Fatal(err)
	}

	for _, concert := range events {
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

func prettyPrintEvents(c Crawler) {
	events, err := c.getEvents()
	if err != nil {
		log.Fatal(err)
	}

	for _, event := range events {
		fmt.Printf("Title: %v\nLocation: %v\nCity: %v\nDate: %v\nURL: %v\nComment: %v\nType: %v\n\n",
			event.Title, event.Location, event.City, event.Date, event.URL, event.Comment, event.Type)
	}
}

type Config struct {
	Crawlers []Crawler `yaml:"crawlers"`
}

type Crawler struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
	URL    string `yaml:"url"`
	City   string `yaml:"city"`
	Event  string `yaml:"event"`
	Fields struct {
		Title []string `yaml:"title"`
		URL   struct {
			Loc       string   `yaml:"loc"`
			Relative  bool     `yaml:"relative"`
			OnSubpage []string `yaml:"on_subpage"`
		} `yaml:"url"`
		Date struct {
			Day struct {
				Loc    string `yaml:"loc"`
				Layout string `yaml:"layout"`
			} `yaml:"day"`
			Month struct {
				Loc    string `yaml:"loc"`
				Layout string `yaml:"layout"`
			} `yaml:"month"`
			DayMonth struct {
				Loc    string `yaml:"loc"`
				Layout string `yaml:"layout"`
			} `yaml:"day_month"`
			DayMonthYearTime struct {
				Loc    string `yaml:"loc"`
				Layout string `yaml:"layout"`
			} `yaml:"day_month_year_time"`
			Time struct {
				Loc    string `yaml:"loc"`
				Layout string `yaml:"layout"`
			} `yaml:"time"`
			Location string `yaml:"location"`
			Language string `yaml:"language"`
		} `yaml:"date"`
		Comment []string `yaml:"comment"`
	} `yaml:"fields"`
}

func NewConfig(configPath string) (*Config, error) {
	config := &Config{}
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	d := yaml.NewDecoder(file)
	if err := d.Decode(&config); err != nil {
		return nil, err
	}
	return config, nil
}

func main() {
	//everyCrawler := flag.Bool("all", false, "Use this flag to indicate that all crawlers should be run.")
	singleCrawler := flag.String("single", "", "The name of the crawler to be run.")
	storeData := flag.Bool("store", false, "If set to true the crawled data will be written to the API.")
	configFile := flag.String("config", "./config.yml", "The location of the configuration file.")

	flag.Parse()

	config, err := NewConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	for _, c := range config.Crawlers {
		if *singleCrawler != "" {
			if *singleCrawler == c.Name {
				if *storeData {
					writeEventsToAPI(c)
				} else {
					prettyPrintEvents(c)
				}
				break
			}
		} else {
			if *storeData {
				writeEventsToAPI(c)
			} else {
				prettyPrintEvents(c)
			}
		}
	}
}
