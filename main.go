package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
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

// // TODO: it's ugly to copy paste this from the croncert-api project.
// type Event struct {
// 	Title    string    `bson:"title,omitempty" json:"title,omitempty" validate:"required" example:"ExcitingTitle"`
// 	Location string    `bson:"location,omitempty" json:"location,omitempty" validate:"required" example:"SuperLocation"`
// 	City     string    `bson:"city,omitempty" json:"city,omitempty" validate:"required" example:"SuperCity"`
// 	Date     time.Time `bson:"date,omitempty" json:"date,omitempty" validate:"required" example:"2021-10-31T19:00:00.000Z"`
// 	URL      string    `bson:"url,omitempty" json:"url,omitempty" validate:"required,url" example:"http://link.to/concert/page"`
// 	Comment  string    `bson:"comment,omitempty" json:"comment,omitempty" example:"Super exciting comment."`
// 	Type     EventType `bson:"type,omitempty" json:"type,omitempty" validate:"required" example:"concert"`
// }

type Config struct {
	Crawlers []Crawler `yaml:"crawlers"`
}

type RegexConfig struct {
	Exp   string `yaml:"exp"`
	Index int    `yaml:"index"`
}

type ElementLocation struct {
	Selector     string      `yaml:"selector"`
	NodeIndex    int         `yaml:"node_index"`
	ChildIndex   int         `yaml:"child_index"`
	RegexExtract RegexConfig `yaml:"regex_extract"`
	Attr         string      `yaml:"attr"`
	MaxLength    int         `yaml:"max_length"` // applies to text
}

type CoveredDateParts struct {
	Day   bool `yaml:"day"`
	Month bool `yaml:"month"`
	Year  bool `yaml:"year"`
	Time  bool `yaml:"time"`
}

type DateComponent struct {
	Covers          CoveredDateParts `yaml:"covers"`
	ElementLocation ElementLocation  `yaml:"location"`
	Layout          string           `yaml:"layout"`
	// Selector     string       `yaml:"selector"`
	// NodeIndex    int          `yaml:"node_index"`
	// ChildIndex   int          `yaml:"child_index"`
	// RegexExtract RegexConfig  `yaml:"regex_extract"`
	// Attr         string       `yaml:"attr"`
}

type Field struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"` // can currently be text, url or date
	// If a field can be found on a subpage the following variable has to contain a field name of
	// a field of type 'url' that is located on the main page.
	ElementLocation ElementLocation `yaml:"location"`
	OnSubpage       string          `yaml:"on_subpage"`   // applies to text, url, date
	CanBeEmpty      bool            `yaml:"can_be_empty"` // applies to text, url
	// Selector     string          `yaml:"selector"`      // applies to text, url
	// NodeIndex    int             `yaml:"node_index"`    // applies to text, url
	// ChildIndex   int             `yaml:"child_index"`   // applies to text, url
	// MaxLength    int             `yaml:"max_length"`    // applies to text
	// RegexExtract RegexConfig     `yaml:"regex_extract"` // applies to text
	Components   []DateComponent `yaml:"components"`    // applies to date
	DateLocation string          `yaml:"date_location"` // applies to date
	DateLanguage string          `yaml:"date_language"` // applies to date
	Relative     bool            `yaml:"relative"`      // applies to url
	// Attr         string          `yaml:"attr"`          // applies to url
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
	Fields              []Field  `yaml:"fields"`
	// Fields              struct {
	// 	Title   Field `yaml:"title"`
	// 	Comment Field `yaml:"comment"`
	// 	URL     struct {
	// 		Selector  string   `yaml:"selector"`
	// 		Relative  bool     `yaml:"relative"`
	// 		OnSubpage []string `yaml:"on_subpage"`
	// 		Attr      string   `yaml:"attr"`
	// 	} `yaml:"url"`
	// 	Date struct {
	// 		Day              DateField `yaml:"day"`
	// 		Month            DateField `yaml:"month"`
	// 		Year             DateField `yaml:"year"`
	// 		DayMonth         DateField `yaml:"day_month"`
	// 		DayMonthYear     DateField `yaml:"day_month_year"`
	// 		DayMonthYearTime DateField `yaml:"day_month_year_time"`
	// 		Time             DateField `yaml:"time"`
	// 		Location         string    `yaml:"location"`
	// 		Language         string    `yaml:"language"`
	// 	} `yaml:"date"`
	// } `yaml:"fields"`
	Filters   []Filter `yaml:"filters"`
	Paginator struct {
		Selector  string `yaml:"selector"`
		Relative  bool   `yaml:"relative"`
		MaxPages  int    `yaml:"max_pages"`
		NodeIndex int    `yaml:"node_index"`
	}
}

func (c Crawler) getEvents() ([]map[string]string, error) {
	// dynamicFields := []string{"title", "comment", "url", "date"}
	var events []map[string]string
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
			err_msg := fmt.Sprintf("status code error: %d %s", res.StatusCode, res.Status)
			return events, errors.New(err_msg)
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

			currentEvent := map[string]string{"location": c.Name, "city": c.City, "type": c.Type}

			// handle all fields on the main page
			for _, f := range c.Fields {
				if f.OnSubpage == "" {
					err := extractField(&f, currentEvent, s, c.URL, res)
					if err != nil {
						log.Printf("%s ERROR: error while parsing field %s: %v. Skipping event %v.", c.Name, f.Name, err, currentEvent)
						return
					}
				}
			}

			// handle all fields on subpages

			// we store the *http.Response as value and not the *goquery.Selection
			// to still be able to close all the response bodies afterwards
			// UPDATE: we also store the *goquery.Document since apparently resSub.Body
			// can only be read once.
			// UPDATE: the previous statement might be incorrect.
			// UPDATE: seems to be correct after all.
			subpagesResp := make(map[string]*http.Response)
			subpagesBody := make(map[string]*goquery.Document)
			for _, f := range c.Fields {
				if f.OnSubpage != "" {
					// check whether we fetched the page already
					resSub, found := subpagesResp[currentEvent[f.OnSubpage]]
					if !found {
						resSub, err = http.Get(currentEvent[f.OnSubpage])
						if err != nil {
							log.Printf("%s ERROR: %v. Skipping event %v.", c.Name, err, currentEvent)
							return
						}
						if resSub.StatusCode != 200 {
							log.Printf("%s ERROR: status code error: %d %s. Skipping event %v.", c.Name, res.StatusCode, res.Status, currentEvent)
							return
						}
						subpagesResp[currentEvent[f.OnSubpage]] = resSub
						docSub, err := goquery.NewDocumentFromReader(resSub.Body)

						if err != nil {
							log.Printf("%s ERROR: error while reading document: %v. Skipping event %v", c.Name, err, currentEvent)
							return
						}
						subpagesBody[currentEvent[f.OnSubpage]] = docSub
					}
					err = extractField(&f, currentEvent, subpagesBody[currentEvent[f.OnSubpage]].Selection, c.URL, resSub)
					if err != nil {
						log.Printf("%s ERROR: error while parsing field %s: %v. Skipping event %v.", c.Name, f.Name, err, currentEvent)
						return
					}
				}
			}
			//Close all the subpages
			for _, resSub := range subpagesResp {
				resSub.Body.Close()
			}

			// check if event should be ignored
			ie, err := c.ignoreEvent(currentEvent)
			if err != nil {
				log.Fatalf("%s ERROR: error while applying ignore filter: %v. Not ignoring event %v.", c.Name, err, currentEvent)
			}
			if !ie {
				events = append(events, currentEvent)
			}
		})

		hasNextPage = false
		if c.Paginator.Selector != "" {
			currentPage += 1
			if currentPage < c.Paginator.MaxPages || c.Paginator.MaxPages == 0 {
				attr := "href"
				if len(doc.Find(c.Paginator.Selector).Nodes) > c.Paginator.NodeIndex {
					pagNode := doc.Find(c.Paginator.Selector).Get(c.Paginator.NodeIndex)
					for _, a := range pagNode.Attr {
						if a.Key == attr {
							nextUrl := a.Val
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
						}
					}
				}
			}
		}
		res.Body.Close()
	}
	// TODO: check if the dates make sense. Sometimes we have to guess the year since it
	// does not appear on the website. In that case, eg. having a list of events around
	// the end of one year and the beginning of the next year we might want to change the
	// year of some events because our previous guess was rather naiv. We also might want
	// to make this functionality optional.

	return events, nil
}

func (c Crawler) ignoreEvent(event map[string]string) (bool, error) {
	for _, filter := range c.Filters {
		regex, err := regexp.Compile(filter.RegexIgnore)
		if err != nil {
			return false, err
		}

		if fieldValue, found := event[filter.Field]; found {
			if regex.MatchString(fieldValue) {
				return true, nil
			}
		}
	}
	return false, nil
}

func extractField(field *Field, event map[string]string, s *goquery.Selection, baseUrl string, res *http.Response) error {
	switch field.Type {
	case "text":
		ts, err := getTextString(&field.ElementLocation, s)
		if err != nil {
			return err
		}
		if !field.CanBeEmpty {
			if ts == "" {
				error_msg := fmt.Sprintf("field %s cannot be empty", field.Name)
				return errors.New(error_msg)
			}
		}
		event[field.Name] = ts
	case "url":
		event[field.Name] = getUrlString(field, s, baseUrl, res)
	case "date":
		ds, err := getDateString(field, s)
		if err != nil {
			return err
		}
		event[field.Name] = ds
	default:
		error_msg := fmt.Sprintf("field type '%s' does not exist", field.Type)
		return errors.New(error_msg)
	}
	return nil
}

// func extractFieldOld(item string, s *goquery.Selection, crawler *Crawler, event *Event, events []Event, loc *time.Location, mLocale string, res *http.Response) error {
// 	switch item {
// 	case "date":
// 		currentYear := time.Now().Year()
// 		yearString := strconv.Itoa(currentYear)
// 		yearLayout := "2006"

// 		if crawler.Fields.Date.Year.Selector != "" {
// 			yearStringTmp, yearLayoutTmp := getDateStringAndLayout(&crawler.Fields.Date.Year, s)
// 			// if the found year string is empty we take the default. This might be incorrect but is preferrable to skipping the event entirely.
// 			if yearStringTmp != "" {
// 				yearString, yearLayout = yearStringTmp, yearLayoutTmp
// 			}
// 		}

// 		timeString, timeLayout := "20:00", "15:04"
// 		if crawler.Fields.Date.Time.Selector != "" {
// 			timeStringTmp, timeLayoutTmp := getDateStringAndLayout(&crawler.Fields.Date.Time, s)
// 			// if the found time string is empty we take the default. This might be incorrect but is preferrable to skipping the event entirely.
// 			if timeStringTmp != "" {
// 				timeString, timeLayout = timeStringTmp, timeLayoutTmp
// 			}
// 		}

// 		var dateTimeString, dateTimeLayout string
// 		if crawler.Fields.Date.DayMonthYearTime.Selector != "" {
// 			dateTimeString, dateTimeLayout = getDateStringAndLayout(&crawler.Fields.Date.DayMonthYearTime, s)
// 		} else if crawler.Fields.Date.DayMonthYear.Selector != "" {
// 			dayMonthYearString, dayMonthYearLayout := getDateStringAndLayout(&crawler.Fields.Date.DayMonthYear, s)
// 			dateTimeString = fmt.Sprintf("%s %s", dayMonthYearString, timeString)
// 			dateTimeLayout = fmt.Sprintf("%s %s", dayMonthYearLayout, timeLayout)
// 		} else {
// 			var dayMonthString, dayMonthLayout string
// 			if crawler.Fields.Date.DayMonth.Selector != "" {
// 				dayMonthString, dayMonthLayout = getDateStringAndLayout(&crawler.Fields.Date.DayMonth, s)
// 			} else if crawler.Fields.Date.Day.Selector != "" && crawler.Fields.Date.Month.Selector != "" {
// 				dayString, dayLayout := getDateStringAndLayout(&crawler.Fields.Date.Day, s)
// 				monthString, monthLayout := getDateStringAndLayout(&crawler.Fields.Date.Month, s)
// 				dayMonthString = dayString + " " + monthString
// 				dayMonthLayout = dayLayout + " " + monthLayout
// 			}

// 			dateTimeLayout = fmt.Sprintf("%s %s %s", dayMonthLayout, yearLayout, timeLayout)
// 			dateTimeString = fmt.Sprintf("%s %s %s", dayMonthString, yearString, timeString)
// 			dateTimeString = strings.Replace(dateTimeString, "Mrz", "Mär", 1) // hack for issue #47
// 		}

// 		if dateTimeString == "" {
// 			return errors.New("empty dateTimeString")
// 		}
// 		t, err := monday.ParseInLocation(dateTimeLayout, dateTimeString, loc, monday.Locale(mLocale))
// 		if err != nil {
// 			return err
// 		}
// 		// if the date t does not come after the previous event's date we increase the year by 1
// 		// actually this is only necessary if we have to guess the date but currently for ease of implementation
// 		// this check is done always.

// 		// We relax this condition slightly as it happens that within a day the events might not be sorted chronologically.
// 		if len(events) > 0 {
// 			correctYear := currentYear
// 			for events[len(events)-1].Date.Round(24 * time.Hour).After(t.Round(24 * time.Hour)) {
// 				correctYear += 1
// 				t = time.Date(int(correctYear), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
// 			}
// 		}
// 		event.Date = t
// 	case "title":
// 		title := getTextString(&crawler.Fields.Title, s)
// 		if title == "" {
// 			return errors.New("empty event title")
// 		}
// 		event.Title = title
// 	case "comment":
// 		event.Comment = getTextString(&crawler.Fields.Comment, s)
// 	case "url":
// 		var url string
// 		attr := "href"
// 		if crawler.Fields.URL.Attr != "" {
// 			attr = crawler.Fields.URL.Attr
// 		}
// 		if crawler.Fields.URL.Selector == "" {
// 			url = s.AttrOr(attr, crawler.URL)
// 		} else {
// 			url = s.Find(crawler.Fields.URL.Selector).AttrOr(attr, crawler.URL)
// 		}

// 		if crawler.Fields.URL.Relative {
// 			baseURL := fmt.Sprintf("%s://%s", res.Request.URL.Scheme, res.Request.URL.Host)
// 			if !strings.HasPrefix(url, "/") {
// 				baseURL = baseURL + "/"
// 			}
// 			url = baseURL + url
// 		}
// 		event.URL = url
// 	}
// 	return nil
// }

type DatePart struct {
	stringPart string
	layoutPart string
}

func getDateString(f *Field, s *goquery.Selection) (string, error) {
	// time zone
	loc, err := time.LoadLocation(f.DateLocation)
	if err != nil {
		return "", err
	}

	// locale (language)
	mLocale := "de_DE"
	if f.DateLanguage != "" {
		mLocale = f.DateLanguage
	}

	// collect all the date parts
	dateParts := []DatePart{}
	combinedParts := CoveredDateParts{}
	for _, c := range f.Components {
		if !hasAllDateParts(combinedParts) {
			if err := checkForDoubleDateParts(c.Covers, combinedParts); err != nil {
				return "", err
			}
			sp, err := getTextString(&c.ElementLocation, s)
			if err != nil {
				return sp, err
			}
			if sp != "" {
				dateParts = append(dateParts, DatePart{
					stringPart: strings.Replace(sp, "p.m.", "pm", 1),
					layoutPart: strings.Replace(c.Layout, "p.m.", "pm", 1),
				})
				combinedParts = mergeDateParts(combinedParts, c.Covers)
			}
		}
	}
	// adding default values where necessary
	if !combinedParts.Year {
		currentYear := time.Now().Year()
		dateParts = append(dateParts, DatePart{
			stringPart: strconv.Itoa(currentYear),
			layoutPart: "2006",
		})
	}
	if !combinedParts.Time {
		dateParts = append(dateParts, DatePart{
			stringPart: "20:00",
			layoutPart: "15:04",
		})
	}
	// currently not all date parts have default values
	if !combinedParts.Day || !combinedParts.Month {
		return "", errors.New("date parsing error: to generate a date at least a day and a month is needed")
	}

	var dateTimeLayout, dateTimeString string
	for _, dp := range dateParts {
		dateTimeLayout += dp.layoutPart + " "
		dateTimeString += dp.stringPart + " "
	}
	dateTimeString = strings.Replace(dateTimeString, "Mrz", "Mär", 1) // hack for issue #47
	t, err := monday.ParseInLocation(dateTimeLayout, dateTimeString, loc, monday.Locale(mLocale))
	if err != nil {
		return "", err
	}
	return t.String(), nil
}

func checkForDoubleDateParts(dpOne CoveredDateParts, dpTwo CoveredDateParts) error {
	if dpOne.Day && dpTwo.Day {
		return errors.New("date parsing error: 'day' covered at least twice")
	}
	if dpOne.Month && dpTwo.Month {
		return errors.New("date parsing error: 'month' covered at least twice")
	}
	if dpOne.Year && dpTwo.Year {
		return errors.New("date parsing error: 'year' covered at least twice")
	}
	if dpOne.Time && dpTwo.Time {
		return errors.New("date parsing error: 'time' covered at least twice")
	}
	return nil
}

func mergeDateParts(dpOne CoveredDateParts, dpTwo CoveredDateParts) CoveredDateParts {
	return CoveredDateParts{
		Day:   dpOne.Day || dpTwo.Day,
		Month: dpOne.Month || dpTwo.Month,
		Year:  dpOne.Year || dpTwo.Year,
		Time:  dpOne.Time || dpTwo.Time,
	}
}

func hasAllDateParts(cdp CoveredDateParts) bool {
	return cdp.Day && cdp.Month && cdp.Year && cdp.Time
}

// func getDateStringAndLayout(dl *DateComponent, s *goquery.Selection) (string, string) {
// 	var fieldString, fieldLayout string
// 	fieldStringSelection := s.Find(dl.Selector)
// 	if len(fieldStringSelection.Nodes) > 0 {
// 		if dl.Attr == "" {
// 			currentChildIndex := 0
// 			fieldStringNode := fieldStringSelection.Get(dl.NodeIndex).FirstChild
// 			for fieldStringNode != nil {
// 				// If the cild index is 0 (default value if not explicitly defined) we loop over all the children.
// 				// This makes it easier if there are many children and only one matches the regex. If only one
// 				// matches the regex then the child index can even differ inbetween various events.
// 				// Plus we do not need to change existing crawler configs.
// 				if currentChildIndex == dl.ChildIndex || dl.ChildIndex == 0 {
// 					if fieldStringNode.Type == html.TextNode {
// 						var err error
// 						fieldString, err = extractStringRegex(&dl.RegexExtract, fieldStringNode.Data)
// 						if err == nil {
// 							break
// 						}
// 					}
// 				}
// 				fieldStringNode = fieldStringNode.NextSibling
// 				currentChildIndex += 1
// 			}
// 		} else {
// 			fieldString = fieldStringSelection.AttrOr(dl.Attr, "")
// 		}
// 	}
// 	// 'p.m.' is not treated as part of the time string by the date parsing library
// 	// so we have to replace it with 'pm'
// 	fieldLayout = strings.Replace(dl.Layout, "p.m.", "pm", 1)
// 	fieldString = strings.Replace(fieldString, "p.m.", "pm", 1)
// 	return fieldString, fieldLayout
// }

func getUrlString(f *Field, s *goquery.Selection, crawlerURL string, res *http.Response) string {
	var url string
	attr := "href"
	if f.ElementLocation.Attr != "" {
		attr = f.ElementLocation.Attr
	}
	if f.ElementLocation.Selector == "" {
		url = s.AttrOr(attr, crawlerURL)
	} else {
		url = s.Find(f.ElementLocation.Selector).AttrOr(attr, crawlerURL)
	}

	if f.Relative {
		baseURL := fmt.Sprintf("%s://%s", res.Request.URL.Scheme, res.Request.URL.Host)
		if !strings.HasPrefix(url, "/") {
			baseURL = baseURL + "/"
		}
		url = baseURL + url
	}
	return url
}

func getTextString(t *ElementLocation, s *goquery.Selection) (string, error) {
	var fieldString string
	fieldSelection := s.Find(t.Selector)
	if len(fieldSelection.Nodes) > t.NodeIndex {
		if t.Attr == "" {
			fieldNode := fieldSelection.Get(t.NodeIndex).FirstChild
			currentChildIndex := 0
			// fieldStringNode := fieldStringSelection.Get(dl.NodeIndex).FirstChild
			for fieldNode != nil {
				// If the cild index is 0 (default value if not explicitly defined) we loop over all the children.
				// This makes it easier if there are many children and only one matches the regex. If only one
				// matches the regex then the child index can even differ inbetween various events.
				// Plus we do not need to change existing crawler configs.
				//
				// we change the index setting for the case where we want to find the correct string
				// by regex (checking all the children and taking the first one that matches the regex) to -1 to
				// distinguish from the default case 0. So when we explicitly set ChildIndex to -1 it means
				// check _all_ of the children.
				if currentChildIndex == t.ChildIndex || t.ChildIndex == -1 {
					if fieldNode.Type == html.TextNode {
						// trimming whitespaces might be confusing in some cases...
						fieldString = strings.TrimSpace(fieldNode.Data)
						fieldString, err := extractStringRegex(&t.RegexExtract, fieldString)
						if err == nil {
							if t.MaxLength > 0 && t.MaxLength < len(fieldString) {
								fieldString = fieldString[:t.MaxLength] + "..."
							}
							break
						} else if t.ChildIndex != -1 {
							// only in case we do not (ab)use the regex to search across all children
							// we want to return the err. Also, we still return the fieldString as
							// this might be useful for narrowing down the reason for the error.
							return fieldString, err
						}
					}
				}
				fieldNode = fieldNode.NextSibling
				currentChildIndex += 1
			}
		} else {
			fieldString = fieldSelection.AttrOr(t.Attr, "")
		}
	}
	return fieldString, nil
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
	return
	// log.Printf("crawling %s\n", c.Name)
	// defer wg.Done()
	// apiUrl := os.Getenv("EVENT_API")
	// client := &http.Client{
	// 	Timeout: time.Second * 10,
	// }
	// events, err := c.getEvents()

	// if err != nil {
	// 	log.Printf("%s ERROR: %s", c.Name, err)
	// 	return
	// }

	// if len(events) == 0 {
	// 	log.Printf("location %s has no events. Skipping.", c.Name)
	// 	return
	// }
	// log.Printf("fetched %d %s events\n", len(events), c.Name)
	// // sort events by date asc
	// sort.Slice(events, func(i, j int) bool {
	// 	return events[i].Date.Before(events[j].Date)
	// })

	// // delete events of this crawler from first date on
	// firstDate := events[0].Date.UTC().Format("2006-01-02 15:04")
	// deleteUrl := fmt.Sprintf("%s?location=%s&datetime=%s", apiUrl, url.QueryEscape(c.Name), url.QueryEscape(firstDate))
	// req, _ := http.NewRequest("DELETE", deleteUrl, nil)
	// req.SetBasicAuth(os.Getenv("API_USER"), os.Getenv("API_PASSWORD"))
	// resp, err := client.Do(req)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// if resp.StatusCode != 200 {
	// 	log.Fatalf("Something went wrong while deleting events. Status Code: %d\nUrl: %s", resp.StatusCode, deleteUrl)
	// }

	// // add new events
	// for _, event := range events {
	// 	concertJSON, err := json.Marshal(event)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(concertJSON))
	// 	req.Header = map[string][]string{
	// 		"Content-Type": {"application/json"},
	// 	}
	// 	req.SetBasicAuth(os.Getenv("API_USER"), os.Getenv("API_PASSWORD"))
	// 	resp, err := client.Do(req)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	if resp.StatusCode != 201 {
	// 		log.Fatalf("something went wrong while adding a new event. Status Code: %d", resp.StatusCode)

	// 	}
	// }
	// log.Printf("done crawling and writing %s data to API.\n", c.Name)
}

func prettyPrintEvents(wg *sync.WaitGroup, c Crawler) {
	defer wg.Done()
	events, err := c.getEvents()
	if err != nil {
		log.Printf("%s ERROR: %s", c.Name, err)
		return
	}

	for _, event := range events {
		eventJson, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			log.Print(err.Error())
		}
		fmt.Println(string(eventJson))
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
