package scraper

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/goodsign/monday"
	"golang.org/x/net/html"
)

// RegexConfig is used for extracting a substring from a string based on the
// given Exp and Index
type RegexConfig struct {
	Exp   string `yaml:"exp"`
	Index int    `yaml:"index"`
}

// ElementLocation is used to find a specific string in a html document
type ElementLocation struct {
	Selector     string      `yaml:"selector"`
	NodeIndex    int         `yaml:"node_index"`
	ChildIndex   int         `yaml:"child_index"`
	RegexExtract RegexConfig `yaml:"regex_extract"`
	Attr         string      `yaml:"attr"`
	MaxLength    int         `yaml:"max_length"`
}

// CoveredDateParts is used to determine what parts of a date a
// DateComponent covers
type CoveredDateParts struct {
	Day   bool `yaml:"day"`
	Month bool `yaml:"month"`
	Year  bool `yaml:"year"`
	Time  bool `yaml:"time"`
}

// A DateComponent is used to find a specific part of a date within
// a html document
type DateComponent struct {
	Covers          CoveredDateParts `yaml:"covers"`
	ElementLocation ElementLocation  `yaml:"location"`
	Layout          []string           `yaml:"layout"`
}

// A StaticField defines a field that has a fixed name and value
// across all scraped items
type StaticField struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// A DynamicField contains all the information necessary to scrape
// a dynamic field from a website, ie a field who's value changes
// for each item
type DynamicField struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"` // can currently be text, url or date
	// If a field can be found on a subpage the following variable has to contain a field name of
	// a field of type 'url' that is located on the main page.
	ElementLocation ElementLocation `yaml:"location"`
	OnSubpage       string          `yaml:"on_subpage"`    // applies to text, url, date
	CanBeEmpty      bool            `yaml:"can_be_empty"`  // applies to text, url
	Components      []DateComponent `yaml:"components"`    // applies to date
	DateLocation    string          `yaml:"date_location"` // applies to date
	DateLanguage    string          `yaml:"date_language"` // applies to date
	Relative        bool            `yaml:"relative"`      // applies to url
	Hide            bool            `yaml:"hide"`          // appliess to text, url, date
}

// A Filter is used to filter certain items from the result list
type Filter struct {
	Field string `yaml:"field"`
	Regex string `yaml:"regex"`
	Match bool   `yaml:"match"`
}

// A Scraper contains all the necessary config parameters and structs needed
// to extract the desired information from a website
type Scraper struct {
	Name                string   `yaml:"name"`
	URL                 string   `yaml:"url"`
	Item                string   `yaml:"item"`
	ExcludeWithSelector []string `yaml:"exclude_with_selector"`
	Fields              struct {
		Static  []StaticField  `yaml:"static"`
		Dynamic []DynamicField `yaml:"dynamic"`
	} `yaml:"fields"`
	Filters   []Filter `yaml:"filters"`
	Paginator struct {
		Selector  string `yaml:"selector"`
		Relative  bool   `yaml:"relative"`
		MaxPages  int    `yaml:"max_pages"`
		NodeIndex int    `yaml:"node_index"`
	}
}

// GetItems fetches and returns all items from a website according to the
// Scraper's paramaters
func (c Scraper) GetItems() ([]map[string]interface{}, error) {

	var items []map[string]interface{}

	pageURL := c.URL
	hasNextPage := true
	currentPage := 0
	for hasNextPage {
		res, err := http.Get(pageURL)
		if err != nil {
			return items, err
		}

		// defer res.Body.Close() // better not defer in a for loop

		if res.StatusCode != 200 {
			return items, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
		}

		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			return items, err
		}

		doc.Find(c.Item).Each(func(i int, s *goquery.Selection) {
			for _, excludeSelector := range c.ExcludeWithSelector {
				if s.Find(excludeSelector).Length() > 0 || s.Is(excludeSelector) {
					return
				}
			}

			// add static fields
			currentItem := make(map[string]interface{})
			for _, sf := range c.Fields.Static {
				currentItem[sf.Name] = sf.Value
			}

			// handle all fields on the main page
			for _, f := range c.Fields.Dynamic {
				if f.OnSubpage == "" {
					err := extractField(&f, currentItem, s, c.URL, res)
					if err != nil {
						log.Printf("%s ERROR: error while parsing field %s: %v. Skipping item %v.", c.Name, f.Name, err, currentItem)
						return
					}
				}
			}

			// handle all fields on subpages

			// we store the *http.Response as value and not the *goquery.Selection
			// to still be able to close all the response bodies afterwards
			// UPDATE: we also store the *goquery.Document since apparently resSub.Body
			// can only be read once.
			subpagesResp := make(map[string]*http.Response)
			subpagesBody := make(map[string]*goquery.Document)
			for _, f := range c.Fields.Dynamic {
				if f.OnSubpage != "" {
					// check whether we fetched the page already
					subpageURL := fmt.Sprint(currentItem[f.OnSubpage])
					resSub, found := subpagesResp[subpageURL]
					if !found {
						resSub, err = http.Get(subpageURL)
						if err != nil {
							log.Printf("%s ERROR: %v. Skipping item %v.", c.Name, err, currentItem)
							return
						}
						if resSub.StatusCode != 200 {
							log.Printf("%s ERROR: status code error: %d %s. Skipping item %v.", c.Name, res.StatusCode, res.Status, currentItem)
							return
						}
						subpagesResp[subpageURL] = resSub
						docSub, err := goquery.NewDocumentFromReader(resSub.Body)

						if err != nil {
							log.Printf("%s ERROR: error while reading document: %v. Skipping item %v", c.Name, err, currentItem)
							return
						}
						subpagesBody[subpageURL] = docSub
					}
					err = extractField(&f, currentItem, subpagesBody[subpageURL].Selection, c.URL, resSub)
					if err != nil {
						log.Printf("%s ERROR: error while parsing field %s: %v. Skipping item %v.", c.Name, f.Name, err, currentItem)
						return
					}
				}
			}
			// close all the subpages
			for _, resSub := range subpagesResp {
				resSub.Body.Close()
			}

			// check if item should be filtered
			filter, err := c.filterItem(currentItem)
			if err != nil {
				log.Fatalf("%s ERROR: error while applying filter: %v.", c.Name, err)
			}
			if filter {
				currentItem = c.removeHiddenFields(currentItem)
				items = append(items, currentItem)
			}
		})

		hasNextPage = false
		if c.Paginator.Selector != "" {
			currentPage++
			if currentPage < c.Paginator.MaxPages || c.Paginator.MaxPages == 0 {
				attr := "href"
				if len(doc.Find(c.Paginator.Selector).Nodes) > c.Paginator.NodeIndex {
					pagNode := doc.Find(c.Paginator.Selector).Get(c.Paginator.NodeIndex)
					for _, a := range pagNode.Attr {
						if a.Key == attr {
							nextURL := a.Val
							if c.Paginator.Relative {
								baseURL := fmt.Sprintf("%s://%s", res.Request.URL.Scheme, res.Request.URL.Host)
								if strings.HasPrefix(nextURL, "?") {
									pageURL = baseURL + res.Request.URL.Path + nextURL
								} else if !strings.HasPrefix(nextURL, "/") {
									pageURL = baseURL + "/" + nextURL
								} else {
									pageURL = baseURL + nextURL
								}
							} else {
								pageURL = nextURL
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
	// to make this functionality optional. See issue #68

	return items, nil
}

func (c *Scraper) filterItem(item map[string]interface{}) (bool, error) {
	nrMatchTrue := 0
	filterMatchTrue := false
	filterMatchFalse := true
	for _, filter := range c.Filters {
		regex, err := regexp.Compile(filter.Regex)
		if err != nil {
			return false, err
		}
		if fieldValue, found := item[filter.Field]; found {
			if filter.Match {
				nrMatchTrue++
				if regex.MatchString(fmt.Sprint(fieldValue)) {
					filterMatchTrue = true
				}
			} else {
				if regex.MatchString(fmt.Sprint(fieldValue)) {
					filterMatchFalse = false
				}
			}
		}
	}
	if nrMatchTrue == 0 {
		filterMatchTrue = true
	}
	return filterMatchTrue && filterMatchFalse, nil
}

func (c *Scraper) removeHiddenFields(item map[string]interface{}) map[string]interface{} {
	for _, f := range c.Fields.Dynamic {
		if f.Hide {
			delete(item, f.Name)
		}
	}
	return item
}

func extractField(field *DynamicField, event map[string]interface{}, s *goquery.Selection, baseURL string, res *http.Response) error {
	switch field.Type {
	case "text", "": // the default, ie when type is not configured, is 'text'
		ts, err := getTextString(&field.ElementLocation, s)
		if err != nil {
			return err
		}
		if !field.CanBeEmpty {
			if ts == "" {
				return fmt.Errorf("field %s cannot be empty", field.Name)
			}
		}
		event[field.Name] = ts
	case "url":
		event[field.Name] = getURLString(field, s, baseURL, res)
	case "date":
		d, err := getDate(field, s)
		if err != nil {
			return err
		}
		event[field.Name] = d
	default:
		return fmt.Errorf("field type '%s' does not exist", field.Type)
	}
	return nil
}

type datePart struct {
	stringPart string
	layoutParts []string
}

func getDate(f *DynamicField, s *goquery.Selection) (time.Time, error) {
	// time zone
	var t time.Time
	loc, err := time.LoadLocation(f.DateLocation)
	if err != nil {
		return t, err
	}

	// locale (language)
	mLocale := "de_DE"
	if f.DateLanguage != "" {
		mLocale = f.DateLanguage
	}

	// collect all the date parts
	dateParts := []datePart{}
	combinedParts := CoveredDateParts{}
	for _, c := range f.Components {
		if !hasAllDateParts(combinedParts) {
			if err := checkForDoubleDateParts(c.Covers, combinedParts); err != nil {
				return t, err
			}
			sp, err := getTextString(&c.ElementLocation, s)
			if err != nil {
				return t, err
			}
			if sp != "" {
				var lp []string
				for _, l := range c.Layout {
					lp = append(lp, strings.Replace(l, "p.m.", "pm", 1))
				}
				dateParts = append(dateParts, datePart{
					stringPart: strings.Replace(sp, "p.m.", "pm", 1),
					layoutParts: lp,
				})
				combinedParts = mergeDateParts(combinedParts, c.Covers)
			}
		}
	}
	// adding default values where necessary
	if !combinedParts.Year {
		currentYear := time.Now().Year()
		dateParts = append(dateParts, datePart{
			stringPart: strconv.Itoa(currentYear),
			layoutParts: []string{"2006"},
		})
	}
	if !combinedParts.Time {
		dateParts = append(dateParts, datePart{
			stringPart: "20:00",
			layoutParts: []string{"15:04"},
		})
	}
	// currently not all date parts have default values
	if !combinedParts.Day || !combinedParts.Month {
		return t, errors.New("date parsing error: to generate a date at least a day and a month is needed")
	}

	var dateTimeString string
	dateTimeLayouts := []string{""}
	for _, dp := range dateParts {
		tmpDateTimeLayouts := dateTimeLayouts
		dateTimeLayouts = []string{}
		for _, tlp := range tmpDateTimeLayouts {
			for _, lp := range dp.layoutParts {
				dateTimeLayouts = append(dateTimeLayouts, tlp + lp + " ")
			}
		}
		dateTimeString += dp.stringPart + " "
	}
	dateTimeString = strings.Replace(dateTimeString, "Mrz", "Mär", 1) // hack for issue #47
	for _, dateTimeLayout := range dateTimeLayouts {
		t, err = monday.ParseInLocation(dateTimeLayout, dateTimeString, loc, monday.Locale(mLocale))
		if err == nil {
			return t, nil
		}
	}
	return t, err
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

func getURLString(f *DynamicField, s *goquery.Selection, scraperURL string, res *http.Response) string {
	var url string
	attr := "href"
	if f.ElementLocation.Attr != "" {
		attr = f.ElementLocation.Attr
	}
	if f.ElementLocation.Selector == "" {
		url = s.AttrOr(attr, scraperURL)
	} else {
		url = s.Find(f.ElementLocation.Selector).AttrOr(attr, scraperURL)
	}

	if f.Relative {
		baseURL := fmt.Sprintf("%s://%s", res.Request.URL.Scheme, res.Request.URL.Host)
		if !strings.HasPrefix(url, "/") {
			baseURL = baseURL + "/"
		}
		url = baseURL + url
	}
	url = strings.TrimSpace(url)
	return url
}

func getTextString(t *ElementLocation, s *goquery.Selection) (string, error) {
	var fieldString string
	var err error
	fieldSelection := s.Find(t.Selector)
	if len(fieldSelection.Nodes) > t.NodeIndex {
		if t.Attr == "" {
			fieldNode := fieldSelection.Get(t.NodeIndex).FirstChild
			currentChildIndex := 0
			for fieldNode != nil {
				// for the case where we want to find the correct string
				// by regex (checking all the children and taking the first one that matches the regex)
				// the ChildIndex has to be set to -1 to
				// distinguish from the default case 0. So when we explicitly set ChildIndex to -1 it means
				// check _all_ of the children.
				if currentChildIndex == t.ChildIndex || t.ChildIndex == -1 {
					if fieldNode.Type == html.TextNode {
						fieldString, err = extractStringRegex(&t.RegexExtract, fieldNode.Data)
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
				currentChildIndex++
			}
		} else {
			fieldString = fieldSelection.AttrOr(t.Attr, "")
			fieldString, err = extractStringRegex(&t.RegexExtract, fieldString)
			if err != nil {
				return fieldString, err
			}
		}
	}
	// automitcally trimming whitespaces might be confusing in some cases...
	fieldString = strings.TrimSpace(fieldString)
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
