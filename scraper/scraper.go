// Package scraper provides functionality to scrape data from websites
package scraper

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/jsonquery"
	"github.com/goodsign/monday"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jakopako/goskyr/config"
	"github.com/jakopako/goskyr/date"
	"github.com/jakopako/goskyr/fetch"
	"github.com/jakopako/goskyr/output"
	"github.com/jakopako/goskyr/types"
	"github.com/jakopako/goskyr/utils"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

// GlobalConfig is used for storing global configuration parameters that
// are needed across all scrapers
type GlobalConfig struct {
	UserAgent string `yaml:"user_agent"`
	// WriteScraperStatus bool   `yaml:"write_scraper_status,omitempty"` // if true, the scraper will write the status of each scraper to the writer. TODO: move to writer config?
}

// Config defines the overall structure of the scraper configuration.
// Values will be taken from a config yml file or environment variables
// or both.
type Config struct {
	// We cannot use a pointer for Writer otherwise reading from
	// env vars for the Writer output.WriterConfig fields does
	// not work with cleanenv.ReadConfig because it does not supported
	// nested structs with pointers
	Writer   output.WriterConfig `yaml:"writer,omitempty"`
	Scrapers []Scraper           `yaml:"scrapers,omitempty"`
	Global   *GlobalConfig       `yaml:"global,omitempty"`
}

// NewConfig reads a configuration file from the given path and returns
// a Config struct. If the path is a directory it will read all files
// in that directory and merge them into one Config struct.
func NewConfig(configPath string) (*Config, error) {
	var config Config
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() {
		err := filepath.WalkDir(configPath, func(path string, d fs.DirEntry, err error) error {
			if !d.IsDir() {
				var configTmp Config
				if err := cleanenv.ReadConfig(path, &configTmp); err != nil {
					return err
				}
				config.Scrapers = append(config.Scrapers, configTmp.Scrapers...)

				// writer
				if configTmp.Writer.Type != "" {
					if config.Writer.Type == "" {
						config.Writer = configTmp.Writer
					} else {
						return fmt.Errorf("config files must only contain max. one writer config")
					}
				}

				// global
				if configTmp.Global != nil {
					if config.Global == nil {
						config.Global = configTmp.Global
					} else {
						return fmt.Errorf("config files must only contain max. one global config")
					}
				}
			}
			return nil // skipping everything that is not a file
		})
		if err != nil {
			return nil, err
		}
	} else {
		if err := cleanenv.ReadConfig(configPath, &config); err != nil {
			return nil, err
		}
	}

	if config.Global == nil {
		config.Global = &GlobalConfig{
			UserAgent: "goskyr web scraper (github.com/jakopako/goskyr)",
		}
	}

	// default to stdout writer
	if config.Writer.Type == "" {
		config.Writer.Type = output.STDOUT_WRITER_TYPE
	}

	// log warnings for depricated fields
	hasLogged := map[string]bool{}
	for _, s := range config.Scrapers {
		if s.RenderJs {
			if !hasLogged["render_js"] {
				slog.Warn("'render_js' is deprecated. Please use 'fetcher.type' instead")
				hasLogged["render_js"] = true
			}
		}
		if s.PageLoadWait > 0 {
			if !hasLogged["page_load_wait"] {
				slog.Warn("'page_load_wait' is deprecated. Please use 'fetcher.page_load_wait_ms' instead")
				hasLogged["page_load_wait"] = true
			}
		}
	}

	return &config, nil
}

// RegexConfig is used for extracting a substring from a string based on the
// given RegexPattern and Index
type RegexConfig struct {
	RegexPattern string `yaml:"exp"`
	Index        int    `yaml:"index"`
	IgnoreErrors bool   `yaml:"ignore_errors"`
}

// ElementLocation is used to find a specific string in a html document
type ElementLocation struct {
	Selector      string      `yaml:"selector,omitempty"`
	JsonSelector  string      `yaml:"json_selector,omitempty"`
	ChildIndex    int         `yaml:"child_index,omitempty"`
	RegexExtract  RegexConfig `yaml:"regex_extract,omitempty"`
	Attr          string      `yaml:"attr,omitempty"`
	MaxLength     int         `yaml:"max_length,omitempty"`
	EntireSubtree bool        `yaml:"entire_subtree,omitempty"`
	AllNodes      bool        `yaml:"all_nodes,omitempty"`
	Separator     string      `yaml:"separator,omitempty"`
	Default       string      `yaml:"default,omitempty"`
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
	Covers          date.CoveredDateParts `yaml:"covers"`
	ElementLocation ElementLocation       `yaml:"location"`
	Layout          []string              `yaml:"layout"`
	Transform       []TransformConfig     `yaml:"transform,omitempty"`
}

// A Field contains all the information necessary to scrape
// a dynamic field from a website, ie a field who's value changes
// for each item
type Field struct {
	Name             string           `yaml:"name"`
	Value            string           `yaml:"value,omitempty"`
	Type             string           `yaml:"type,omitempty"`     // can currently be text, url or date
	ElementLocations ElementLocations `yaml:"location,omitempty"` // elements are extracted strings joined using the given Separator
	Separator        string           `yaml:"separator,omitempty"`
	// If a field can be found on a subpage the following variable has to contain a field name of
	// a field of type 'url' that is located on the main page.
	OnSubpage    string            `yaml:"on_subpage,omitempty"`    // applies to text, url, date
	CanBeEmpty   bool              `yaml:"can_be_empty,omitempty"`  // applies to text, url
	Components   []DateComponent   `yaml:"components,omitempty"`    // applies to date
	DateLocation string            `yaml:"date_location,omitempty"` // applies to date
	DateLanguage string            `yaml:"date_language,omitempty"` // applies to date
	Hide         bool              `yaml:"hide,omitempty"`          // applies to text, url, date
	GuessYear    bool              `yaml:"guess_year,omitempty"`    // applies to date
	Transform    []TransformConfig `yaml:"transform,omitempty"`     // applies to text
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
	Field      string `yaml:"field"`
	Type       string
	Expression string `yaml:"exp"` // changed from 'regex' to 'exp' in version 0.5.7
	RegexComp  *regexp.Regexp
	DateComp   time.Time
	DateOp     string
	Match      bool `yaml:"match"`
}

func (f *Filter) FilterMatch(value any) bool {
	switch f.Type {
	case "regex":
		return f.RegexComp.MatchString(fmt.Sprint(value))
	case "date":
		d, _ := value.(time.Time)
		if f.DateOp == ">" {
			return d.After(f.DateComp)
		} else {
			return d.Before(f.DateComp)
		}
	default:
		return false
	}
}

func (f *Filter) Initialize(fieldType string) error {
	if fieldType == "date" {
		f.Type = "date"
	} else {
		f.Type = "regex" // default for everything except date fields
	}
	switch f.Type {
	case "regex":
		regex, err := regexp.Compile(f.Expression)
		if err != nil {
			return err
		}
		f.RegexComp = regex
		return nil
	case "date":
		initErr := fmt.Errorf("the expression for filtering by date should be of the following format: '<|> now|YYYY-MM-ddTHH:mm'")
		tokens := strings.Split(f.Expression, " ")
		if len(tokens) != 2 {
			return initErr
		}
		if tokens[0] != ">" && tokens[0] != "<" {
			return initErr
		}
		f.DateOp = tokens[0]
		// parse date, return error
		if tokens[1] != "now" {
			t, err := time.Parse("2006-01-02T15:04", tokens[1])
			if err != nil {
				return initErr
			}
			f.DateComp = t
		} else {
			f.DateComp = time.Now().UTC()
		}
		return nil
	default:
		// we won't ever get here because we set the filter type above
		return fmt.Errorf("type '%s' does not exist for filters", f.Type)
	}
}

// A Paginator is used to paginate through a website
type Paginator struct {
	Location ElementLocation `yaml:"location,omitempty"`
	MaxPages int             `yaml:"max_pages,omitempty"`
}

// A Scraper contains all the necessary config parameters and structs needed
// to extract the desired information from a website
type Scraper struct {
	Name          string               `yaml:"name"`
	URL           string               `yaml:"url"`
	Item          string               `yaml:"item"`
	Fields        []Field              `yaml:"fields,omitempty"`
	Filters       []*Filter            `yaml:"filters,omitempty"`
	Paginator     Paginator            `yaml:"paginator,omitempty"`
	RenderJs      bool                 `yaml:"render_js,omitempty"`      // DEPRECATED
	PageLoadWait  int                  `yaml:"page_load_wait,omitempty"` // DEPRECATED (milliseconds. Only taken into account when render_js = true)
	Interaction   []*types.Interaction `yaml:"interaction,omitempty"`
	FetcherConfig fetch.FetcherConfig  `yaml:"fetcher"`
	fetcher       fetch.Fetcher
}

// ScraperResult contains the results of a scrape. It consists of the scraped items
// and the scraper status
type ScraperResult struct {
	Items []map[string]any
	Stats *types.ScraperStatus
}

// Scrape fetches and returns all items from a website according to the
// Scraper's paramaters. When rawDyn is set to true the items returned are
// not processed according to their type but instead the raw values based
// only on the location are returned (ignore regex_extract??). And only those
// of dynamic fields, ie fields that don't have a predefined value and that are
// present on the main page (not subpages). This is used by the ML feature generation.
func (c Scraper) Scrape(globalConfig *GlobalConfig, rawDyn bool) (*ScraperResult, error) {
	scrLogger := slog.With(slog.String("name", c.Name))

	// initialize fetcher
	// default fetcher type is static
	if c.FetcherConfig.Type == "" {
		c.FetcherConfig.Type = fetch.STATIC_FETCHER_TYPE
	}

	// if local UserAgent empty, set global one
	if c.FetcherConfig.UserAgent == "" {
		c.FetcherConfig.UserAgent = globalConfig.UserAgent
	}

	fetcher, err := fetch.NewFetcher(&c.FetcherConfig)
	if err != nil {
		return nil, err
	}
	defer fetcher.Cancel()

	scrLogger.Info(fmt.Sprintf("using %s fetcher", c.FetcherConfig.Type))
	c.fetcher = fetcher

	result := &ScraperResult{
		Items: []map[string]any{},
		Stats: &types.ScraperStatus{
			ScraperName:     c.Name,
			LastScrapeStart: time.Now().UTC(),
		},
	}

	scrLogger.Debug("initializing filters")
	if err := c.initializeFilters(); err != nil {
		return result, err
	}

	hasNextPage := true
	currentPage := 0
	var doc *goquery.Document

	hasNextPage, pageURL, doc, err := c.fetchPage(nil, currentPage, c.URL, c.Interaction)
	if err != nil {
		return result, err
	}

	for hasNextPage {
		baseUrl := getBaseURL(pageURL, doc)
		doc.Find(c.Item).Each(func(i int, s *goquery.Selection) {
			currentItem := make(map[string]any)
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
							err = extractRawField(&f, currentItem, s)
						} else {
							err = extractField(&f, currentItem, s, baseUrl)
						}
						if err != nil {
							scrLogger.Error(fmt.Sprintf("error while parsing field %s: %v. Skipping item %v.", f.Name, err, currentItem))
							result.Stats.NrErrors++
							return
						}
					}
					// to speed things up we check the filter after each field.
					// Like that we safe time if we already know for sure that
					// we want to filter out a certain item. Especially, if
					// certain elements would need to be fetched from subpages.
					// filter fast!
					if !c.filterItem(currentItem) {
						return
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
							subDoc, err := c.fetchToDoc(subpageURL, fetch.FetchOpts{})
							if err != nil {
								scrLogger.Error(fmt.Sprintf("%v. Skipping item %v.", err, currentItem))
								result.Stats.NrErrors++
								return
							}
							subDocs[subpageURL] = subDoc
						}
						baseURLSubpage := getBaseURL(subpageURL, subDocs[subpageURL])
						err = extractField(&f, currentItem, subDocs[subpageURL].Selection, baseURLSubpage)
						if err != nil {
							scrLogger.Error(fmt.Sprintf("error while parsing field %s: %v. Skipping item %v.", f.Name, err, currentItem))
							result.Stats.NrErrors++
							return
						}
						// filter fast!
						if !c.filterItem(currentItem) {
							return
						}
					}
				}
			}

			// check if item should be filtered
			filter := c.filterItem(currentItem)
			if filter {
				currentItem = c.removeHiddenFields(currentItem)
				result.Items = append(result.Items, currentItem)
				result.Stats.NrItems++
			}
		})

		currentPage++
		hasNextPage, pageURL, doc, err = c.fetchPage(doc, currentPage, pageURL, nil)
		if err != nil {
			return result, err
		}
	}

	c.guessYear(result.Items, time.Now())

	result.Stats.LastScrapeEnd = time.Now().UTC()

	return result, nil
}

func (c *Scraper) guessYear(items []map[string]any, ref time.Time) {
	// get date field names where we need to adapt the year
	dateFieldsGuessYear := map[string]bool{}
	for _, f := range c.Fields {
		if f.Type == "date" {
			if f.GuessYear {
				dateFieldsGuessYear[f.Name] = true
			}
		}
	}

	// main use case:
	// event websites mostly contain a list of events ordered by date. Sometimes the date does
	// not contain the year. In that case we could simply set the year to the current year but
	// it might happen that the list of events spans across more than one year into the next
	// year. In that case we still want to set the correct year which would be current year + n.
	// Moreover, the list might not be ordered at all. In that case we also want to try to set
	// the correct year.
	if len(dateFieldsGuessYear) > 0 {
		for i, item := range items {
			for name, val := range item {
				if dateFieldsGuessYear[name] {
					if t, ok := val.(time.Time); ok {

						// for the first item we compare this item's date with 'now' and try
						// to find the most suitable year, ie the year that brings this item's
						// date closest to now.
						// for the remaining items we do the same as with the first item except
						// that we compare this item's date to the previous item's date instead
						// of 'now'.
						// Obviously, this is still far from perfect and it only works in a certain
						// number of cases.
						if i > 0 {
							ref, _ = items[i-1][name].(time.Time)
						}
						diff := time.Since(time.Unix(0, 0))
						newDate := t
						for y := ref.Year() - 1; y <= ref.Year()+1; y++ {
							tmpT := time.Date(y, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
							if newDiff := tmpT.Sub(ref).Abs(); newDiff < diff {
								diff = newDiff
								newDate = time.Date(y, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
							}
						}
						item[name] = newDate
					}
				}
			}
		}
	}
}

func (c *Scraper) initializeFilters() error {
	// build temporary map field name -> field type
	fieldTypes := map[string]string{}
	for _, field := range c.Fields {
		fieldTypes[field.Name] = field.Type
	}
	for _, f := range c.Filters {
		if fieldType, ok := fieldTypes[f.Field]; ok {
			if err := f.Initialize(fieldType); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("filter error. There is no field with the name '%s'", f.Field)
		}
	}
	return nil
}

func (c *Scraper) filterItem(item map[string]any) bool {
	nrMatchTrue := 0
	foundFields := 0
	filterMatchTrue := false
	filterMatchFalse := true
	for _, f := range c.Filters {
		if fieldValue, found := item[f.Field]; found {
			foundFields++
			if f.Match {
				nrMatchTrue++
				if f.FilterMatch(fieldValue) {
					filterMatchTrue = true
				}
			} else {
				if f.FilterMatch(fieldValue) {
					filterMatchFalse = false
				}
			}
		}
	}
	if nrMatchTrue == 0 {
		filterMatchTrue = true
	}
	// if foundFields < len(c.Filters) that means that we did not get the full item
	// with _all_ fields as input to this function and hence we only want
	// to return false if we are _sure_ that this item should not be part
	// of the final results. And we are only sure if we have a 'match: false'
	// match, i.e. filterMatchFalse has been set to false
	if foundFields < len(c.Filters) {
		return filterMatchFalse
	}
	return filterMatchTrue && filterMatchFalse
}

func (c *Scraper) removeHiddenFields(item map[string]any) map[string]any {
	for _, f := range c.Fields {
		if f.Hide {
			delete(item, f.Name)
		}
	}
	return item
}

func (c *Scraper) fetchPage(doc *goquery.Document, nextPageI int, currentPageUrl string, i []*types.Interaction) (bool, string, *goquery.Document, error) {
	if nextPageI == 0 {
		newDoc, err := c.fetchToDoc(currentPageUrl, fetch.FetchOpts{Interaction: i})
		if err != nil {
			return false, "", nil, err
		}
		return true, currentPageUrl, newDoc, nil
	} else {
		if c.Paginator.Location.Selector != "" {
			if c.FetcherConfig.Type == fetch.DYNAMIC_FETCHER_TYPE {
				// check if node c.Paginator.Location.Selector is present in doc
				pagSelector := doc.Find(c.Paginator.Location.Selector)
				if len(pagSelector.Nodes) > 0 {
					if nextPageI < c.Paginator.MaxPages || c.Paginator.MaxPages == 0 {
						ia := []*types.Interaction{
							{
								Selector: c.Paginator.Location.Selector,
								Type:     types.InteractionTypeClick,
								Count:    nextPageI, // we always need to 'restart' the clicks because we always re-fetch the page
							},
						}
						nextPageDoc, err := c.fetchToDoc(currentPageUrl, fetch.FetchOpts{Interaction: ia})
						if err != nil {
							return false, "", nil, err
						}
						return true, currentPageUrl, nextPageDoc, nil
					}
				}
			} else {
				baseUrl := getBaseURL(currentPageUrl, doc)
				nextPageUrl, err := getURLString(&c.Paginator.Location, doc.Selection, baseUrl)
				if err != nil {
					return false, "", nil, err
				}
				if nextPageUrl != "" {
					nextPageDoc, err := c.fetchToDoc(nextPageUrl, fetch.FetchOpts{})
					if err != nil {
						return false, "", nil, err
					}
					if nextPageI < c.Paginator.MaxPages || c.Paginator.MaxPages == 0 {
						return true, nextPageUrl, nextPageDoc, nil
					}
				}
			}
		}
		return false, "", nil, nil
	}
}

func (c *Scraper) fetchToDoc(urlStr string, opts fetch.FetchOpts) (*goquery.Document, error) {
	res, err := c.fetcher.Fetch(urlStr, opts)
	if err != nil {
		return nil, err
	}
	// fmt.Println(res)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(res))
	if err != nil {
		return nil, err
	}

	// in debug mode we want to write all the html's to files
	if config.Debug {
		u, _ := url.Parse(urlStr)
		r, err := utils.RandomString(u.Host)
		if err != nil {
			return nil, err
		}
		filename := fmt.Sprintf("%s.html", r)
		slog.Debug(fmt.Sprintf("writing html to file %s", filename), slog.String("url", urlStr))
		htmlStr, err := goquery.OuterHtml(doc.Children())
		if err != nil {
			return nil, fmt.Errorf("failed to write html file: %v", err)
		}

		f, err := os.Create(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to write html file: %v", err)
		}
		defer f.Close()
		_, err = f.WriteString(htmlStr)
		if err != nil {
			return nil, fmt.Errorf("failed to write html file: %v", err)
		}
	}
	return doc, nil
}

func extractField(field *Field, event map[string]any, s *goquery.Selection, baseURL string) error {
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

		// transform the string if required
		for _, tr := range field.Transform {
			var err error
			t, err = transformString(&tr, t)
			if err != nil {
				return err
			}
		}

		event[field.Name] = t
	case "url":
		if len(field.ElementLocations) != 1 {
			return fmt.Errorf("a field of type 'url' must exactly have one location")
		}
		url, err := getURLString(&field.ElementLocations[0], s, baseURL)
		if err != nil {
			return err
		}
		if !field.CanBeEmpty && url == "" {
			return fmt.Errorf("field %s cannot be empty", field.Name)
		}

		// transform the url if required
		for _, tr := range field.Transform {
			var err error
			url, err = transformString(&tr, url)
			if err != nil {
				return err
			}
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

func extractRawField(field *Field, event map[string]any, s *goquery.Selection) error {
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
	combinedParts := date.CoveredDateParts{}
	for _, c := range f.Components {
		if !date.HasAllDateParts(combinedParts) {
			if err := date.CheckForDoubleDateParts(c.Covers, combinedParts); err != nil {
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
				if c.Layout == nil {
					// layout needs to be set
					return t, fmt.Errorf("date parsing error: a date component of field '%s' has no layout set", f.Name)
				}
				dateParts = append(dateParts, datePart{
					stringPart:  sp,
					layoutParts: c.Layout,
				})
				combinedParts = date.MergeDateParts(combinedParts, c.Covers)
			}
		}
	}

	// currently not all date parts have default values
	if !combinedParts.Day || !combinedParts.Month || !combinedParts.Time {
		return t, errors.New("date parsing error: to generate a date at least a time, a day and a month are needed")
	}

	// year is special in the sense that it defaults to the current year
	// if nothing is specified
	currentYear := time.Now().Year()

	if !combinedParts.Year {
		dateParts = append(dateParts, datePart{
			stringPart:  strconv.Itoa(currentYear),
			layoutParts: []string{"2006"},
		})
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
		} else if !combinedParts.Year && f.GuessYear {
			// edge case, parsing time "29.02. 20:00 2023 ": day out of range
			// We set the year to the current year but it should actually be 2024
			// We only update the year string in case guess_year is set to true
			// to not confuse the user too much
			if strings.HasSuffix(err.Error(), "day out of range") && strings.Contains(err.Error(), "29") {
				for i := 1; i < 4; i++ {
					dateTimeString = strings.Replace(dateTimeString, strconv.Itoa(currentYear+i-1), strconv.Itoa(currentYear+i), 1)
					t, err = monday.ParseInLocation(dateTimeLayout, dateTimeString, loc, monday.Locale(mLocale))
					if err == nil {
						return t, nil
					}
				}
			}
		}
	}
	return t, err
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

func getURLString(e *ElementLocation, s *goquery.Selection, baseURL string) (string, error) {
	var urlVal, urlRes string
	u, _ := url.Parse(baseURL)
	if e.Attr == "" {
		// set attr to the default if not set
		e.Attr = "href"
	}

	urlVal, err := getTextString(e, s)
	if err != nil {
		return "", err
	}

	urlVal = strings.TrimSpace(urlVal)
	if urlVal == "" {
		return "", nil
	} else if strings.HasPrefix(urlVal, "http") {
		urlRes = urlVal
	} else if strings.HasPrefix(urlVal, "?") || strings.HasPrefix(urlVal, ".?") {
		urlVal = strings.TrimLeft(urlVal, ".")
		urlRes = fmt.Sprintf("%s://%s%s%s", u.Scheme, u.Host, u.Path, urlVal)
	} else if strings.HasPrefix(urlVal, "/") {
		baseURL := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		urlRes = fmt.Sprintf("%s%s", baseURL, urlVal)
	} else if strings.HasPrefix(urlVal, "..") {
		partsUrlVal := strings.Split(urlVal, "/")
		partsPath := strings.Split(u.Path, "/")
		i := 0
		for ; i < len(partsUrlVal); i++ {
			if partsUrlVal[i] != ".." {
				break
			}
		}
		urlRes = fmt.Sprintf("%s://%s%s/%s", u.Scheme, u.Host, strings.Join(partsPath[:len(partsPath)-i-1], "/"), strings.Join(partsUrlVal[i:], "/"))
	} else {
		idx := strings.LastIndex(u.Path, "/")
		if idx > 0 {
			path := u.Path[:idx]
			urlRes = fmt.Sprintf("%s://%s%s/%s", u.Scheme, u.Host, path, urlVal)
		} else {
			urlRes = fmt.Sprintf("%s://%s/%s", u.Scheme, u.Host, urlVal)
		}
	}

	urlRes = strings.TrimSpace(urlRes)
	return urlRes, nil
}

func getTextString(e *ElementLocation, s *goquery.Selection) (string, error) {
	var fieldStrings []string
	var fieldSelection *goquery.Selection
	if e.Selector == "" {
		fieldSelection = s
	} else {
		fieldSelection = s.Find(e.Selector)
	}
	if len(fieldSelection.Nodes) > 0 {
		if e.Attr == "" {
			if e.EntireSubtree {
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
				if e.AllNodes {
					for _, node := range fieldSelection.Nodes {
						f(node)
						fieldStrings = append(fieldStrings, buf.String())
						buf.Reset()
					}
				} else {
					f(fieldSelection.Get(0))
					fieldStrings = append(fieldStrings, buf.String())
				}
			} else {

				var fieldNodes []*html.Node
				if e.AllNodes {
					for _, node := range fieldSelection.Nodes {
						fieldNode := node.FirstChild
						if fieldNode != nil {
							fieldNodes = append(fieldNodes, fieldNode)
						}
					}
				} else {
					fieldNode := fieldSelection.Get(0).FirstChild
					if fieldNode != nil {
						fieldNodes = append(fieldNodes, fieldNode)
					}
				}
				for _, fieldNode := range fieldNodes {
					currentChildIndex := 0
					for fieldNode != nil {
						if currentChildIndex == e.ChildIndex {
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
			fieldStrings = append(fieldStrings, fieldSelection.AttrOr(e.Attr, ""))
		}
	}
	// do json lookup if we have a json_selector
	for i, f := range fieldStrings {
		fieldString, err := extractJsonField(e.JsonSelector, f)
		if err != nil {
			return "", err
		}
		fieldStrings[i] = fieldString
	}
	// regex extract
	for i, f := range fieldStrings {
		fieldString, err := extractStringRegex(&e.RegexExtract, f)
		if err != nil {
			return "", err
		}
		fieldStrings[i] = fieldString
	}
	// automatically trimming whitespaces might be confusing in some cases...
	// TODO make this configurable
	for i, f := range fieldStrings {
		fieldStrings[i] = strings.TrimSpace(f)
	}
	// shortening
	for i, f := range fieldStrings {
		fieldStrings[i] = utils.ShortenString(f, e.MaxLength)
	}
	finalString := strings.Join(fieldStrings, e.Separator)

	if finalString == "" && e.Default != "" {
		return e.Default, nil
	}
	return finalString, nil
}

func extractStringRegex(rc *RegexConfig, s string) (string, error) {
	extractedString := s
	if rc.RegexPattern != "" {
		regex, err := regexp.Compile(rc.RegexPattern)
		if err != nil {
			return "", err
		}
		matchingStrings := regex.FindAllString(s, -1)
		errMsg := ""
		if len(matchingStrings) == 0 {
			errMsg = fmt.Sprintf("no matching strings found for regex: %s", rc.RegexPattern)
		} else if rc.Index < 0 {
			if len(matchingStrings)+rc.Index < 0 {
				errMsg = fmt.Sprintf("regex index out of bounds. regex '%s' gave only %d matches", rc.RegexPattern, len(matchingStrings))
			} else {
				extractedString = matchingStrings[len(matchingStrings)+rc.Index]
			}
		} else {
			if rc.Index >= len(matchingStrings) {
				errMsg = fmt.Sprintf("regex index out of bounds. regex '%s' gave only %d matches", rc.RegexPattern, len(matchingStrings))
			} else {
				extractedString = matchingStrings[rc.Index]
			}
		}
		if errMsg != "" {
			errMsg += fmt.Sprintf(", string was: %s", s)
			if rc.IgnoreErrors {
				slog.Info(fmt.Sprintf("ignoring regex error: %s", errMsg))
				return "", nil
			} else {
				return "", errors.New(errMsg)
			}
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

func getBaseURL(pageUrl string, doc *goquery.Document) string {
	// relevant info: https://www.w3.org/TR/WD-html40-970917/htmlweb.html#relative-urls
	// currently this function does not fully implement the standard
	baseURL := doc.Find("base").AttrOr("href", "")
	if baseURL == "" {
		baseURL = pageUrl
	}
	return baseURL
}

func extractJsonField(p string, s string) (string, error) {
	extractedString := s
	if p != "" {
		// HACK: json has some instances of meaningfull whitespace
		// (\n in values). Let's get rid of them
		spacecleaner := regexp.MustCompile(`\s+`)
		s = spacecleaner.ReplaceAllString(s, " ")
		// HACK: a dangling comma is another common json mistake
		regex := regexp.MustCompile(`,\s*}`)
		s = regex.ReplaceAllString(s, " }")
		doc, err := jsonquery.Parse(strings.NewReader(s))
		if err != nil {
			return "", fmt.Errorf("JSON: %+v : %s", err, s)
		}
		node := jsonquery.FindOne(doc, p)
		extractedString = fmt.Sprintf("%v", node.Value())
	}
	return extractedString, nil
}
