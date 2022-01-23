package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/goodsign/monday"
	"golang.org/x/net/html"
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

	pageUrl := c.URL
	hasNextPage := true
	currentPage := 0
	for hasNextPage {
		res, err := http.Get(pageUrl)
		if err != nil {
			return events, err
		}

		// defer res.Body.Close() // better not defer in a for loop

		if res.StatusCode != 200 {
			log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
		}

		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			return events, err
		}

		doc.Find(c.Event).Each(func(i int, s *goquery.Selection) {
			for _, exclude_selector := range c.ExcludeWithSelector {
				if s.Find(exclude_selector).Length() > 0 || s.Is(exclude_selector) {
					return
				}
			}

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
						// if there is an error skip this event and log error instead of completely canceling the crawl.
						log.Printf("error while parsing field %s: %v. Skipping event.", f, err)
						return
					}
				}
			}

			if len(c.Fields.URL.OnSubpage) > 0 {
				resSub, err := http.Get(currentEvent.URL)
				if err != nil {
					return
				}

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
						// if there is an error skip this event and log error instead of completely canceling the crawl.
						log.Printf("error while parsing field %s: %v. Skipping event %s.", item, err, currentEvent.Title)
						return
					}
				}
				resSub.Body.Close()
			}

			// check if events should be ignored
			ie, err := c.ignoreEvent(&currentEvent)
			if err != nil {
				log.Fatalf("error while removing events: %v", err)
			}
			if !ie {
				events = append(events, currentEvent)
			}
		})

		hasNextPage = false
		if c.Paginator.Loc != "" {
			currentPage += 1
			if currentPage < c.Paginator.MaxPages || c.Paginator.MaxPages == 0 {
				attr := "href"
				nextUrl, exists := doc.Find(c.Paginator.Loc).Attr(attr)
				if exists {
					if c.Paginator.Relative {
						baseURL := fmt.Sprintf("%s://%s", res.Request.URL.Scheme, res.Request.URL.Host)
						if strings.HasPrefix(nextUrl, "?") {
							pageUrl = baseURL + res.Request.URL.Path + nextUrl
						} else if !strings.HasPrefix(nextUrl, "/") {
							pageUrl = baseURL + "/" + nextUrl
						} else {
							pageUrl = baseURL + nextUrl
						}
					} else {
						pageUrl = nextUrl
					}
					hasNextPage = true
					// log.Printf("next page: %s\n", pageUrl)
				}
			}
		}
		res.Body.Close()
	}
	return events, nil
}

func (c Crawler) ignoreEvent(event *Event) (bool, error) {
	for _, filter := range c.Filters {
		regex, err := regexp.Compile(filter.RegexIgnore)
		if err != nil {
			return false, err
		}
		switch filter.Field {
		case "title":
			if regex.MatchString(event.Title) {
				return true, nil
			}
		case "comment":
			if regex.MatchString(event.Comment) {
				return true, nil
			}
		}
	}
	return false, nil
}

func extractField(item string, s *goquery.Selection, crawler *Crawler, event *Event, events []Event, loc *time.Location, mLocale string, res *http.Response) error {
	switch item {
	case "date":
		year := time.Now().Year()

		var timeString, timeStringLayout string
		if crawler.Fields.Date.Time.Loc == "" {
			timeString = "20:00"
			timeStringLayout = "15:04"
		} else {
			timeString, timeStringLayout = getDateStringAndLayout(&crawler.Fields.Date.Time, s)
		}

		var dateTimeString, dateTimeLayout string
		if crawler.Fields.Date.DayMonthYearTime.Loc != "" {
			dateTimeString, dateTimeLayout = getDateStringAndLayout(&crawler.Fields.Date.DayMonthYearTime, s)
		} else if crawler.Fields.Date.DayMonthYear.Loc != "" {
			dayMonthYearString, dayMonthYearLayout := getDateStringAndLayout(&crawler.Fields.Date.DayMonthYear, s)
			dateTimeString = fmt.Sprintf("%s %s", dayMonthYearString, timeString)
			dateTimeLayout = fmt.Sprintf("%s %s", dayMonthYearLayout, timeStringLayout)
		} else {
			var dayMonthString, dayMonthLayout string
			if crawler.Fields.Date.DayMonth.Loc != "" {
				dayMonthString, dayMonthLayout = getDateStringAndLayout(&crawler.Fields.Date.DayMonth, s)
			} else if crawler.Fields.Date.Day.Loc != "" && crawler.Fields.Date.Month.Loc != "" {
				dayString, dayLayout := getDateStringAndLayout(&crawler.Fields.Date.Day, s)
				monthString, monthLayout := getDateStringAndLayout(&crawler.Fields.Date.Month, s)
				dayMonthString = dayString + " " + monthString
				dayMonthLayout = dayLayout + " " + monthLayout
			}

			dateTimeLayout = fmt.Sprintf("%s 2006 %s", dayMonthLayout, timeStringLayout)
			dateTimeString = fmt.Sprintf("%s %d %s", dayMonthString, year, timeString)
		}

		if dateTimeString == "" {
			return errors.New("empty dateTimeString")
		}
		t, err := monday.ParseInLocation(dateTimeLayout, dateTimeString, loc, monday.Locale(mLocale))
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
		title := getFieldString(&crawler.Fields.Title, s)
		if title == "" {
			return errors.New("empty event title")
		}
		event.Title = title
	case "comment":
		event.Comment = getFieldString(&crawler.Fields.Comment, s)
	case "url":
		var url string
		attr := "href"
		if crawler.Fields.URL.Attr != "" {
			attr = crawler.Fields.URL.Attr
		}
		if crawler.Fields.URL.Loc == "" {
			url = s.AttrOr(attr, crawler.URL)
		} else {
			url = s.Find(crawler.Fields.URL.Loc).AttrOr(attr, crawler.URL)
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

func getDateStringAndLayout(dl *DateField, s *goquery.Selection) (string, string) {
	var fieldString, fieldLayout string
	fieldStringSelection := s.Find(dl.Loc)
	// TODO: Add possibility to apply a regex across s.Find(dl.Loc).Text()
	// A bit hacky..
	if len(fieldStringSelection.Nodes) > 0 {
		if dl.Attr == "" {
			fieldStringNode := fieldStringSelection.Get(dl.NodeIndex).FirstChild
			for fieldStringNode != nil {
				if fieldStringNode.Type == html.TextNode {
					// we 'abuse' the extractStringRegex func to find the correct text element.
					var err error
					fieldString, err = extractStringRegex(&dl.RegexExtract, fieldStringNode.Data)
					if err == nil {
						break
					}
				}
				fieldStringNode = fieldStringNode.NextSibling
			}
		} else {
			fieldString = fieldStringSelection.AttrOr(dl.Attr, "")
		}
	}
	fieldLayout = dl.Layout
	return fieldString, fieldLayout
}

func getFieldString(f *Field, s *goquery.Selection) string {
	var fieldString string
	fieldSelection := s.Find(f.Loc)
	if len(fieldSelection.Nodes) > f.NodeIndex {
		fieldNode := fieldSelection.Get(f.NodeIndex).FirstChild
		if fieldNode != nil {
			if fieldNode.Type == html.TextNode {
				fieldString = strings.TrimSpace(fieldSelection.Get(f.NodeIndex).FirstChild.Data)
				if f.MaxLength > 0 && f.MaxLength < len(fieldString) {
					fieldString = fieldString[:f.MaxLength] + "..."
				}
			}
		}
	}
	fieldString, err := extractStringRegex(&f.RegexExtract, fieldString)
	if err != nil {
		log.Fatal(err)
	}
	return fieldString
}

func extractStringRegex(rc *RegexConfig, s string) (string, error) {
	extractedString := s
	if rc.Exp != "" {
		regex, err := regexp.Compile(rc.Exp)
		if err != nil {
			return "", err
		}
		matchingStrings := regex.FindAllString(s, -1)
		if len(matchingStrings) == 0 {
			msg := fmt.Sprintf("no matching strings found for regex: %s", rc.Exp)
			return "", errors.New(msg)
		}
		if rc.Index == -1 {
			extractedString = matchingStrings[len(matchingStrings)-1]
		} else {
			if rc.Index >= len(matchingStrings) {
				msg := fmt.Sprintf("regex index out of bounds. regex '%s' gave only %d matches", rc.Exp, len(matchingStrings))
				return "", errors.New(msg)
			}
			extractedString = matchingStrings[rc.Index]
		}
	}
	return extractedString, nil
}

func writeEventsToAPI(wg *sync.WaitGroup, c Crawler) {
	log.Printf("crawling %s\n", c.Name)
	defer wg.Done()
	apiUrl := os.Getenv("EVENT_API")
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	events, err := c.getEvents()

	if err != nil {
		log.Fatal(err)
	}

	if len(events) == 0 {
		log.Printf("Location %s has no events. Skipping.", c.Name)
		return
	}
	log.Printf("fetched %d %s events\n", len(events), c.Name)
	// sort events by date asc
	sort.Slice(events, func(i, j int) bool {
		return events[i].Date.Before(events[j].Date)
	})

	// delete events of this crawler from first date on
	firstDate := events[0].Date.UTC().Format("2006-01-02 15:04")
	deleteUrl := fmt.Sprintf("%s?location=%s&datetime=%s", apiUrl, url.QueryEscape(c.Name), url.QueryEscape(firstDate))
	req, _ := http.NewRequest("DELETE", deleteUrl, nil)
	req.SetBasicAuth(os.Getenv("API_USER"), os.Getenv("API_PASSWORD"))
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("Something went wrong while deleting events. Status Code: %d\nUrl: %s", resp.StatusCode, deleteUrl)
	}

	// add new events
	for _, event := range events {
		concertJSON, err := json.Marshal(event)
		if err != nil {
			log.Fatal(err)
		}
		req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(concertJSON))
		req.Header = map[string][]string{
			"Content-Type": {"application/json"},
		}
		req.SetBasicAuth(os.Getenv("API_USER"), os.Getenv("API_PASSWORD"))
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode != 201 {
			log.Fatalf("Something went wrong while adding a new event. Status Code: %d", resp.StatusCode)

		}
	}
	log.Printf("Done crawling and writing %s data to API.\n", c.Name)
}

func prettyPrintEvents(wg *sync.WaitGroup, c Crawler) {
	defer wg.Done()
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

type RegexConfig struct {
	Exp   string `yaml:"exp"`
	Index int    `yaml:"index"`
}

type DateField struct {
	Loc          string      `yaml:"loc"`
	Layout       string      `yaml:"layout"`
	NodeIndex    int         `yaml:"node_index"`
	RegexExtract RegexConfig `yaml:"regex_extract"`
	Attr         string      `yaml:"attr"`
}

type Field struct {
	Loc          string      `yaml:"loc"`
	NodeIndex    int         `yaml:"node_index"`
	MaxLength    int         `yaml:"max_length"`
	RegexExtract RegexConfig `yaml:"regex_extract"`
	RegexIgnore  string      `yaml:"regex_ignore"`
}

type Filter struct {
	Field       string `yaml:"field"`
	RegexIgnore string `yaml:"regex_ignore"`
}

type Crawler struct {
	Name                string   `yaml:"name"`
	Type                string   `yaml:"type"`
	URL                 string   `yaml:"url"`
	City                string   `yaml:"city"`
	Event               string   `yaml:"event"`
	ExcludeWithSelector []string `yaml:"exclude_with_selector"`
	Fields              struct {
		Title   Field `yaml:"title"`
		Comment Field `yaml:"comment"`
		URL     struct {
			Loc       string   `yaml:"loc"`
			Relative  bool     `yaml:"relative"`
			OnSubpage []string `yaml:"on_subpage"`
			Attr      string   `yaml:"attr"`
		} `yaml:"url"`
		Date struct {
			Day              DateField `yaml:"day"`
			Month            DateField `yaml:"month"`
			DayMonth         DateField `yaml:"day_month"`
			DayMonthYear     DateField `yaml:"day_month_year"`
			DayMonthYearTime DateField `yaml:"day_month_year_time"`
			Time             DateField `yaml:"time"`
			Location         string    `yaml:"location"`
			Language         string    `yaml:"language"`
		} `yaml:"date"`
	} `yaml:"fields"`
	Filters   []Filter `yaml:"filters"`
	Paginator struct {
		Loc      string `yaml:"loc"`
		Relative bool   `yaml:"relative"`
		MaxPages int    `yaml:"max_pages"`
	}
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

	var wg sync.WaitGroup

	for _, c := range config.Crawlers {
		if *singleCrawler != "" {
			if *singleCrawler == c.Name {
				wg.Add(1)
				if *storeData {
					writeEventsToAPI(&wg, c)
				} else {
					prettyPrintEvents(&wg, c)
				}
				break
			}
		} else {
			wg.Add(1)
			if *storeData {
				go writeEventsToAPI(&wg, c)
			} else {
				go prettyPrintEvents(&wg, c)
			}
		}
	}
	wg.Wait()
}
