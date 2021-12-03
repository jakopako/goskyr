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

// func (et EventType) String() string {
// 	return []string{"undefined", "concert"}[et]
// }

// TODO: it's ugly to copy paste this from the croncert-api project.
type Event struct {
	Title    string    `bson:"title,omitempty" json:"title,omitempty" validate:"required" example:"ExcitingTitle"`
	Location string    `bson:"location,omitempty" json:"location,omitempty" validate:"required" example:"SuperLocation"`
	Date     time.Time `bson:"date,omitempty" json:"date,omitempty" validate:"required" example:"2021-10-31T19:00:00.000Z"`
	URL      string    `bson:"url,omitempty" json:"url,omitempty" validate:"required,url" example:"http://link.to/concert/page"`
	Comment  string    `bson:"comment,omitempty" json:"comment,omitempty" example:"Super exciting comment."`
	Type     EventType `bson:"type,omitempty" json:"type,omitempty" validate:"required" example:"concert"`
}

func (c Crawler) getEvents() ([]Event, error) {
	events := []Event{}
	eventType := EventType(c.Type)
	err := eventType.IsValid()
	if err != nil {
		return events, err
	}

	loc, err := time.LoadLocation(c.Fields.Date.Location)
	if err != nil {
		return events, err
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
			Type:     EventType(c.Type),
		}

		// extract the title
		var title string
		for _, titleLoc := range c.Fields.Title {
			titleSelection := s.Find(titleLoc).First()
			title = strings.TrimSuffix(titleSelection.Text(), titleSelection.Children().Text())
			if title != "" {
				break
			}
		}
		if title == "" {
			return
		}

		currentEvent.Title = title

		// extract the url
		url := s.Find(c.Fields.URL.Loc).AttrOr("href", c.URL)
		if c.Fields.URL.Relative {
			url = c.URL + url
		}
		currentEvent.URL = url

		// extract the comment
		var comment string
		for _, commentLoc := range c.Fields.Comment {
			comment = s.Find(commentLoc).Text()
			if comment != "" {
				break
			}
		}
		currentEvent.Comment = comment

		// extract date and time
		day := s.Find(c.Fields.Date.Day.Loc).Text()
		month := s.Find(c.Fields.Date.Month.Loc).Text()
		year := time.Now().Year()
		timeString := "20:00"

		layout := fmt.Sprintf("%s %s 2006 15:04", c.Fields.Date.Day.Layout, c.Fields.Date.Month.Layout)
		dateTimeString := fmt.Sprintf("%s %s %d %s", day, month, year, timeString)
		t, err := monday.ParseInLocation(layout, dateTimeString, loc, monday.Locale(c.Fields.Date.Language))
		if err != nil {
			log.Printf("Couldn't parse date: %s", err)
			return
		}

		// if the date t does not come after the previous event's date we increase the year by 1
		if len(events) > 0 {
			if events[len(events)-1].Date.After(t) {
				t = time.Date(int(year+1), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
			}
		}
		currentEvent.Date = t

		events = append(events, currentEvent)

		// topSelection := s.Find(".agenda .top")
		// if len(topSelection.Nodes) > 0 {
		// 	name := topSelection.Nodes[0].FirstChild.Data
		// 	desc := topSelection.Find(".addition").Nodes[0].FirstChild.Data
		// 	fmt.Printf("Name: %s\n", name)
		// 	fmt.Printf("Description: %s\n\n", desc)
		// }
	})

	return events, nil
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
		fmt.Printf("Title: %v\nLocation: %v\nDate: %v\nURL: %v\nComment: %v\nType: %v\n\n",
			event.Title, event.Location, event.Date, event.URL, event.Comment, event.Type)
	}
}

type Config struct {
	Crawlers []Crawler `yaml:"crawlers"`
}

type Crawler struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
	URL    string `yaml:"url"`
	Event  string `yaml:"event"`
	Fields struct {
		Title []string `yaml:"title"`
		URL   struct {
			Loc      string `yaml:"loc"`
			Relative bool   `yaml:"relative"`
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
