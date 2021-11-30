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

		title := s.Find(c.Fields.Title)
		currentEvent.Title = strings.TrimSuffix(title.Text(), title.Children().Text())
		url := s.Find(c.Fields.URL.Loc).AttrOr("href", c.URL)
		if c.Fields.URL.Relative {
			url = c.URL + url
		}
		currentEvent.URL = url
		currentEvent.Comment = s.Find(c.Fields.Comment).Text()

		fmt.Println(s.Find(c.Fields.Date.Day).Text())
		fmt.Println(s.Find(c.Fields.Date.Month).Text())

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
		Title string `yaml:"title"`
		URL   struct {
			Loc      string `yaml:"loc"`
			Relative bool   `yaml:"relative"`
		} `yaml:"url"`
		Date struct {
			Day   string `yaml:"day"`
			Month string `yaml:"month"`
		} `yaml:"date"`
		Comment string `yaml:"comment"`
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
