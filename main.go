package main

import (
	"bytes"
	"encoding/json"
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

// TODO: it's ugly to copy paste this from the croncert-api project.
type Concert struct {
	Artist   string    `bson:"artist,omitempty" json:"artist,omitempty" validate:"required" example:"SuperArtist"`
	Location string    `bson:"location,omitempty" json:"location,omitempty" validate:"required" example:"SuperLocation"`
	Date     time.Time `bson:"date,omitempty" json:"date,omitempty" validate:"required" example:"2021-10-31T19:00:00.000Z"`
	URL      string    `bson:"url,omitempty" json:"url,omitempty" validate:"required,url" example:"http://link.to/concert/page"`
	Comment  string    `bson:"comment,omitempty" json:"comment,omitempty" example:"Super exciting comment."`
}

func (c Crawler) getConcerts() []Concert {
	concerts := []Concert{}
	res, err := http.Get(c.URL)

	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)

	if err != nil {
		log.Fatal(err)
	}

	doc.Find(c.Event).Each(func(i int, s *goquery.Selection) {
		currentConcert := Concert{
			Location: c.Name,
		}

		artist := s.Find(c.Fields.Artist)
		currentConcert.Artist = strings.TrimSuffix(artist.Text(), artist.Children().Text())
		currentConcert.URL = s.Find(c.Fields.URL).AttrOr("href", c.URL)
		currentConcert.Comment = s.Find(c.Fields.Comment).Text()

		concerts = append(concerts, currentConcert)

		// topSelection := s.Find(".agenda .top")
		// if len(topSelection.Nodes) > 0 {
		// 	name := topSelection.Nodes[0].FirstChild.Data
		// 	desc := topSelection.Find(".addition").Nodes[0].FirstChild.Data
		// 	fmt.Printf("Name: %s\n", name)
		// 	fmt.Printf("Description: %s\n\n", desc)
		// }
	})

	return concerts
}

func writeConcertsToAPI(c Crawler) {
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

func prettyPrintConcerts(c Crawler) {
	for _, concert := range c.getConcerts() {
		fmt.Printf("Artist: %v\nLocation: %v\nDate: %v\nURL: %v\nComment: %v\n\n",
			concert.Artist, concert.Location, concert.Date, concert.URL, concert.Comment)
	}
}

type Config struct {
	Crawlers []Crawler `yaml:"crawlers"`
}

type Crawler struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Event  string `yaml:"event"`
	Fields struct {
		Artist  string `yaml:"artist"`
		URL     string `yaml:"url"`
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
					writeConcertsToAPI(c)
				} else {
					prettyPrintConcerts(c)
				}
				break
			}
		} else {
			if *storeData {
				writeConcertsToAPI(c)
			} else {
				prettyPrintConcerts(c)
			}
		}
	}
}
