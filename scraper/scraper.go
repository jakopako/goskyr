package scraper

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/goodsign/monday"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jakopako/goskyr/fetch"
	"github.com/jakopako/goskyr/output"
	"github.com/jakopako/goskyr/utils"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

// GlobalConfig is used for storing global configuration parameters that
// are needed across all scrapers
type GlobalConfig struct {
	UserAgent string `yaml:"user-agent"`
}

// Config defines the overall structure of the scraper configuration.
// Values will be taken from a config yml file or environment variables
// or both.
type Config struct {
	Writer   output.WriterConfig `yaml:"writer,omitempty"`
	Scrapers []Scraper           `yaml:"scrapers,omitempty"`
	Global   GlobalConfig        `yaml:"global,omitempty"`
}

func NewConfig(configPath string) (*Config, error) {
	var config Config
	err := cleanenv.ReadConfig(configPath, &config)
	return &config, err
}

// RegexConfig is used for extracting a substring from a string based on the
// given Exp and Index
type RegexConfig struct {
	Exp   string `yaml:"exp"`
	Index int    `yaml:"index"`
}

// ElementLocation is used to find a specific string in a html document
type ElementLocation struct {
	Selector      string      `yaml:"selector,omitempty"`
	NodeIndex     int         `yaml:"node_index,omitempty"`
	ChildIndex    int         `yaml:"child_index,omitempty"`
	RegexExtract  RegexConfig `yaml:"regex_extract,omitempty"`
	Attr          string      `yaml:"attr,omitempty"`
	MaxLength     int         `yaml:"max_length,omitempty"`
	EntireSubtree bool        `yaml:"entire_subtree,omitempty"`
	AllNodes      bool        `yaml:"all_nodes,omitempty"`
	Separator     string      `yaml:"separator,omitempty"`
}

// CoveredDateParts is used to determine what parts of a date a
// DateComponent covers
type CoveredDateParts struct {
	Day   bool `yaml:"day,omitempty"`
	Month bool `yaml:"month,omitempty"`
	Year  bool `yaml:"year,omitempty"`
	Time  bool `yaml:"time,omitempty"`
}

// TransformConfig is used to replace an existing substring with some other
// kind of string. Processing needs to happen before extracting dates.
type TransformConfig struct {
	TransformType string `yaml:"type,omitempty"`    // only regex-replace for now
	RegexPattern  string `yaml:"regex,omitempty"`   // a container for the pattern
	Replacement   string `yaml:"replace,omitempty"` // a plain string for replacement
}

// A DateComponent is used to find a specific part of a date within
// a html document
type DateComponent struct {
	Covers          CoveredDateParts  `yaml:"covers"`
	ElementLocation ElementLocation   `yaml:"location"`
	Layout          []string          `yaml:"layout"`
	Transform       []TransformConfig `yaml:"transform,omitempty"`
}

// A Field contains all the information necessary to scrape
// a dynamic field from a website, ie a field who's value changes
// for each item
type Field struct {
	Name             string           `yaml:"name"`
	Value            string           `yaml:"value,omitempty"`
	Type             string           `yaml:"type,omitempty"`     // can currently be text, url or date
	ElementLocations ElementLocations `yaml:"location,omitempty"` // elements are string joined using the given Separator
	Separator        string           `yaml:"separator,omitempty"`
	// If a field can be found on a subpage the following variable has to contain a field name of
	// a field of type 'url' that is located on the main page.
	OnSubpage    string          `yaml:"on_subpage,omitempty"`    // applies to text, url, date
	CanBeEmpty   bool            `yaml:"can_be_empty,omitempty"`  // applies to text, url
	Components   []DateComponent `yaml:"components,omitempty"`    // applies to date
	DateLocation string          `yaml:"date_location,omitempty"` // applies to date
	DateLanguage string          `yaml:"date_language,omitempty"` // applies to date
	Hide         bool            `yaml:"hide,omitempty"`          // appliess to text, url, date
}

type ElementLocations []ElementLocation

func (e *ElementLocations) UnmarshalYAML(value *yaml.Node) error {
	var multi []ElementLocation
	err := value.Decode(&multi)
	if err != nil {
		var single ElementLocation
		err := value.Decode(&single)
		if err != nil {
			return err
		}
		*e = []ElementLocation{single}
	} else {
		*e = multi
	}
	return nil
}

// also have a marshal func for the config generation? so that if the ElementLocations list
// is of length one we output the value in the yaml as ElementLocation and not list of ElementLocations

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
	ExcludeWithSelector []string `yaml:"exclude_with_selector,omitempty"`
	Fields              []Field  `yaml:"fields,omitempty"`
	Filters             []Filter `yaml:"filters,omitempty"`
	Paginator           struct {
		Location ElementLocation `yaml:"location,omitempty"`
		MaxPages int             `yaml:"max_pages,omitempty"`
	} `yaml:"paginator,omitempty"`
	RenderJs bool `yaml:"renderJs,omitempty"`
}

// GetItems fetches and returns all items from a website according to the
// Scraper's paramaters. When rawDyn is set to true the items returned are
// not processed according to their type but instead the raw values based
// only on the location are returned (ignore regex_extract??). And only those
// of dynamic fields, ie fields that don't have a predefined value and that are
// present on the main page (not subpages). This is used by the ML feature generation.
func (c Scraper) GetItems(globalConfig *GlobalConfig, rawDyn bool) ([]map[string]interface{}, error) {

	var items []map[string]interface{}

	pageURL := c.URL
	hasNextPage := true
	currentPage := 0
	var fetcher fetch.Fetcher
	if c.RenderJs {
		fetcher = &fetch.DynamicFetcher{
			UserAgent: globalConfig.UserAgent,
		}
	} else {
		fetcher = &fetch.StaticFetcher{
			UserAgent: globalConfig.UserAgent,
		}
	}
	for hasNextPage {
		res, err := fetcher.Fetch(pageURL)
		if err != nil {
			return items, err
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(res))
		if err != nil {
			return items, err
		}

		doc.Find(c.Item).Each(func(i int, s *goquery.Selection) {
			for _, excludeSelector := range c.ExcludeWithSelector {
				if s.Find(excludeSelector).Length() > 0 || s.Is(excludeSelector) {
					return
				}
			}

			currentItem := make(map[string]interface{})
			for _, f := range c.Fields {
				if f.Value != "" {
					if !rawDyn {
						// add static fields
						currentItem[f.Name] = f.Value
					}
				} else {
					// handle all dynamic fields on the main page
					if f.OnSubpage == "" {
						var err error
						if rawDyn {
							err = extractRawField(&f, currentItem, s, pageURL)
						} else {
							err = extractField(&f, currentItem, s, pageURL)
						}
						if err != nil {
							log.Printf("%s ERROR: error while parsing field %s: %v. Skipping item %v.", c.Name, f.Name, err, currentItem)
							return
						}
					}
				}
			}

			// handle all fields on subpages
			if !rawDyn {
				subDocs := make(map[string]*goquery.Document)
				for _, f := range c.Fields {
					if f.OnSubpage != "" && f.Value == "" {
						// check whether we fetched the page already
						subpageURL := fmt.Sprint(currentItem[f.OnSubpage])
						_, found := subDocs[subpageURL]
						if !found {
							subRes, err := fetcher.Fetch(subpageURL)
							if err != nil {
								log.Printf("%s ERROR: %v. Skipping item %v.", c.Name, err, currentItem)
								return
							}
							subDoc, err := goquery.NewDocumentFromReader(strings.NewReader(subRes))
							if err != nil {
								log.Printf("%s ERROR: error while reading document: %v. Skipping item %v", c.Name, err, currentItem)
								return
							}
							subDocs[subpageURL] = subDoc
						}
						err = extractField(&f, currentItem, subDocs[subpageURL].Selection, c.URL)
						if err != nil {
							log.Printf("%s ERROR: error while parsing field %s: %v. Skipping item %v.", c.Name, f.Name, err, currentItem)
							return
						}
					}
				}
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
		pageURL = getURLString(&c.Paginator.Location, doc.Selection, pageURL)
		if pageURL != "" {
			currentPage++
			if currentPage < c.Paginator.MaxPages || c.Paginator.MaxPages == 0 {
				hasNextPage = true
			}
		}
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
	for _, f := range c.Fields {
		if f.Hide {
			delete(item, f.Name)
		}
	}
	return item
}

func extractField(field *Field, event map[string]interface{}, s *goquery.Selection, baseURL string) error {
	switch field.Type {
	case "text", "": // the default, ie when type is not configured, is 'text'
		parts := []string{}
		for _, p := range field.ElementLocations {
			ts, err := getTextString(&p, s)
			if err != nil {
				return err
			}
			if ts != "" {
				parts = append(parts, ts)
			}
		}
		t := strings.Join(parts, field.Separator)
		if !field.CanBeEmpty && t == "" {
			return fmt.Errorf("field %s cannot be empty", field.Name)
		}
		event[field.Name] = t
	case "url":
		if len(field.ElementLocations) != 1 {
			return fmt.Errorf("a field of type 'url' must exactly have one location")
		}
		url := getURLString(&field.ElementLocations[0], s, baseURL)
		if url == "" {
			url = baseURL
		}
		event[field.Name] = url
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

func extractRawField(field *Field, event map[string]interface{}, s *goquery.Selection, baseURL string) error {
	switch field.Type {
	case "text", "":
		parts := []string{}
		for _, p := range field.ElementLocations {
			ts, err := getTextString(&p, s)
			if err != nil {
				return err
			}
			if ts != "" {
				parts = append(parts, ts)
			}
		}
		t := strings.Join(parts, field.Separator)
		if !field.CanBeEmpty && t == "" {
			return fmt.Errorf("field %s cannot be empty", field.Name)
		}
		event[field.Name] = t
	case "url":
		if len(field.ElementLocations) != 1 {
			return fmt.Errorf("a field of type 'url' must exactly have one location")
		}
		if field.ElementLocations[0].Attr == "" {
			// normally we'd set the default in getUrlString
			// but we're not using this function for the raw extraction
			// because we don't want the url to be auto expanded
			field.ElementLocations[0].Attr = "href"
		}
		ts, err := getTextString(&field.ElementLocations[0], s)
		if err != nil {
			return err
		}
		if !field.CanBeEmpty && ts == "" {
			return fmt.Errorf("field %s cannot be empty", field.Name)
		}
		event[field.Name] = ts
	case "date":
		cs, err := getRawDateComponents(field, s)
		if err != nil {
			return err
		}
		for k, v := range cs {
			event[k] = v
		}
	}
	return nil
}

type datePart struct {
	stringPart  string
	layoutParts []string
}

func getDate(f *Field, s *goquery.Selection) (time.Time, error) {
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
			for _, tr := range c.Transform {
				sp, err = transformString(&tr, sp)
				// we have to return the error here instead of after the loop
				// otherwise errors might be overwritten and hence ignored.
				if err != nil {
					return t, err
				}
			}
			if sp != "" {
				dateParts = append(dateParts, datePart{
					stringPart:  sp,
					layoutParts: c.Layout,
				})
				combinedParts = mergeDateParts(combinedParts, c.Covers)
			}
		}
	}
	// adding default values where necessary
	if !combinedParts.Year {
		currentYear := time.Now().Year()
		dateParts = append(dateParts, datePart{
			stringPart:  strconv.Itoa(currentYear),
			layoutParts: []string{"2006"},
		})
	}
	if !combinedParts.Time {
		dateParts = append(dateParts, datePart{
			stringPart:  "20:00",
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
				dateTimeLayouts = append(dateTimeLayouts, tlp+lp+" ")
			}
		}
		dateTimeString += dp.stringPart + " "
	}
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

func getRawDateComponents(f *Field, s *goquery.Selection) (map[string]string, error) {
	rawComponents := map[string]string{}
	for _, c := range f.Components {
		ts, err := getTextString(&c.ElementLocation, s)
		if err != nil {
			return rawComponents, err
		}
		fName := "date-component"
		if c.Covers.Day {
			fName += "-day"
		}
		if c.Covers.Month {
			fName += "-month"
		}
		if c.Covers.Year {
			fName += "-year"
		}
		if c.Covers.Time {
			fName += "-time"
		}
		rawComponents[fName] = ts
	}
	return rawComponents, nil
}

func getURLString(e *ElementLocation, s *goquery.Selection, baseURL string) string {
	var urlVal, urlRes string
	u, _ := url.Parse(baseURL)
	if e.Attr == "" {
		// set attr to the default if not set
		e.Attr = "href"
	}
	if e.Selector == "" {
		urlVal = s.AttrOr(e.Attr, "")
	} else {
		fieldSelection := s.Find(e.Selector)
		if len(fieldSelection.Nodes) > e.NodeIndex {
			fieldNode := fieldSelection.Get(e.NodeIndex)
			for _, a := range fieldNode.Attr {
				if a.Key == e.Attr {
					urlVal = a.Val
					break
				}
			}
		}
	}

	if urlVal == "" {
		return ""
	} else if strings.HasPrefix(urlVal, "http") {
		urlRes = urlVal
	} else if strings.HasPrefix(urlVal, "?") || strings.HasPrefix(urlVal, ".?") {
		urlVal = strings.TrimLeft(urlVal, ".")
		urlRes = fmt.Sprintf("%s://%s%s%s", u.Scheme, u.Host, u.Path, urlVal)
	} else {
		baseURL := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		if !strings.HasPrefix(urlVal, "/") {
			baseURL = baseURL + "/"
		}
		urlRes = fmt.Sprintf("%s%s", baseURL, urlVal)
	}

	urlRes = strings.TrimSpace(urlRes)
	return urlRes
}

func getTextString(t *ElementLocation, s *goquery.Selection) (string, error) {
	var fieldStrings []string
	var fieldSelection *goquery.Selection
	if t.Selector == "" {
		fieldSelection = s
	} else {
		fieldSelection = s.Find(t.Selector)
	}
	if len(fieldSelection.Nodes) > t.NodeIndex {
		if t.Attr == "" {
			if t.EntireSubtree {
				// copied from https://github.com/PuerkitoBio/goquery/blob/v1.8.0/property.go#L62
				var buf bytes.Buffer
				var f func(*html.Node)
				f = func(n *html.Node) {
					if n.Type == html.TextNode {
						// Keep newlines and spaces, like jQuery
						buf.WriteString(n.Data)
					}
					if n.FirstChild != nil {
						for c := n.FirstChild; c != nil; c = c.NextSibling {
							f(c)
						}
					}
				}
				if t.AllNodes {
					for _, node := range fieldSelection.Nodes {
						f(node)
						fieldStrings = append(fieldStrings, buf.String())
						buf.Reset()
					}
				} else {
					f(fieldSelection.Get(t.NodeIndex))
					fieldStrings = append(fieldStrings, buf.String())
				}
			} else {

				var fieldNodes []*html.Node
				if t.AllNodes {
					for _, node := range fieldSelection.Nodes {
						fieldNode := node.FirstChild
						if fieldNode != nil {
							fieldNodes = append(fieldNodes, fieldNode)
						}
					}
				} else {
					fieldNode := fieldSelection.Get(t.NodeIndex).FirstChild
					if fieldNode != nil {
						fieldNodes = append(fieldNodes, fieldNode)
					}
				}
				for _, fieldNode := range fieldNodes {
					currentChildIndex := 0
					for fieldNode != nil {
						if currentChildIndex == t.ChildIndex {
							if fieldNode.Type == html.TextNode {
								fieldStrings = append(fieldStrings, fieldNode.Data)
								break
							}
						}
						fieldNode = fieldNode.NextSibling
						currentChildIndex++
					}
				}
			}
		} else {
			// WRONG
			// It could be the case that there are multiple nodes that match the selector
			// and we don't want the attr of the first node...
			fieldStrings = append(fieldStrings, fieldSelection.AttrOr(t.Attr, ""))
		}
	}
	// automatically trimming whitespaces might be confusing in some cases...
	for i, f := range fieldStrings {
		fieldStrings[i] = strings.TrimSpace(f)
	}
	// regex extract
	for i, f := range fieldStrings {
		fieldString, err := extractStringRegex(&t.RegexExtract, f)
		if err != nil {
			return "", err
		}
		fieldStrings[i] = fieldString
	}
	// shortening
	for i, f := range fieldStrings {
		fieldStrings[i] = utils.ShortenString(f, t.MaxLength)
	}
	return strings.Join(fieldStrings, t.Separator), nil
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

func transformString(t *TransformConfig, s string) (string, error) {
	extractedString := s
	switch t.TransformType {
	case "regex-replace":
		if t.RegexPattern != "" {
			regex, err := regexp.Compile(t.RegexPattern)
			if err != nil {
				return "", err
			}
			extractedString = regex.ReplaceAllString(s, t.Replacement)
		}
	case "":
		// do nothing
	default:
		return "", fmt.Errorf("transform type '%s' does not exist", t.TransformType)
	}
	return extractedString, nil
}
