package scraper

import (
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jakopako/goskyr/date"
)

const (
	htmlString = `
                            <div class="teaser event-teaser teaser-border teaser-hover">
                                <div class="event-teaser-image event-teaser-image--full"><a
                                        href="/events/10-03-2023-krachstock-final-story" class=""><!--[--><img
                                            src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"
                                            class="image image--event_teaser v-lazy-image"><!--]--><!----></a>
                                    <div class="event-tix"><a class="button"
                                            href="https://www.petzi.ch/events/51480/tickets" target="_blank"
                                            rel="nofollow">Tickets</a></div>
                                </div>
                                <div class="event-teaser-info">
                                    <div class="event-teaser-top"><a href="/events/10-03-2023-krachstock-final-story"
                                            class="event-date size-m bold">Fr, 10.03.2023 - 20:00</a></div><a
                                        href="/events/10-03-2023-krachstock-final-story" class="event-teaser-bottom">
                                        <div class="size-xl event-title">Krachstock</div>
                                        <div class="artist-list"><!--[-->
                                            <h3 class="size-xxl"><!--[-->
                                                <div class="artist-teaser">
                                                    <div class="artist-name">Final Story</div>
                                                    <div class="artist-info">Aargau</div>
                                                </div><!----><!--]-->
                                            </h3>
                                            <h3 class="size-xxl"><!--[-->
                                                <div class="artist-teaser">
                                                    <div class="artist-name">Moment Of Madness</div>
                                                    <div class="artist-info">Basel</div>
                                                </div><!----><!--]-->
                                            </h3>
                                            <h3 class="size-xxl"><!--[-->
                                                <div class="artist-teaser">
                                                    <div class="artist-name">Irony of Fate</div>
                                                    <div class="artist-info">Bern</div>
                                                </div><!----><!--]-->
                                            </h3><!--]--><!---->
                                        </div><!---->
                                        <div class="event-teaser-tags"><!--[-->
                                            <div class="tag">Konzert</div><!--]--><!--[-->
                                            <div class="tag">Metal</div>
                                            <div class="tag">Metalcore</div><!--]-->
                                        </div>
                                    </a>
                                </div>
                            </div>`
	htmlString2 = `                                        
	<h2>
		<a href="https://www.eventfabrik-muenchen.de/event/heinz-rudolf-kunze-verstaerkung-2/"
			title="Heinz Rudolf Kunze &amp; Verstärkung &#8211; ABGESAGT">
			<span>Di. | 03.05.2022</span><span>Heinz Rudolf Kunze &amp; Verstärkung
				&#8211; ABGESAGT</span> </a>
	</h2>`
	htmlString3 = `                                        
	<h2>
		<a href="?bli=bla"
			title="Heinz Rudolf Kunze &amp; Verstärkung &#8211; ABGESAGT">
			<span>Di. | 03.05.2022</span><span>Heinz Rudolf Kunze &amp; Verstärkung
				&#8211; ABGESAGT</span> </a>
	</h2>`
)

func TestFilterItemMatchTrue(t *testing.T) {
	item := map[string]interface{}{"title": "Jacob Collier - Concert"}
	s := &Scraper{
		Filters: []Filter{
			{
				Field: "title",
				Regex: ".*Concert",
				Match: true,
			},
		},
	}
	f, err := s.filterItem(item)
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	if !f {
		t.Fatalf("expected 'true' but got 'false'")
	}
}

func TestFilterItemMatchFalse(t *testing.T) {
	item := map[string]interface{}{"title": "Jacob Collier - Cancelled"}
	s := &Scraper{
		Filters: []Filter{
			{
				Field: "title",
				Regex: ".*Cancelled",
				Match: false,
			},
		},
	}
	f, err := s.filterItem(item)
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	if f {
		t.Fatalf("expected 'false' but got 'true'")
	}
}

func TestRemoveHiddenFields(t *testing.T) {
	s := &Scraper{
		Fields: []Field{
			{
				Name: "hidden",
				Hide: true,
			},
			{
				Name: "visible",
				Hide: false,
			},
		},
	}
	item := map[string]interface{}{"hidden": "bli", "visible": "bla"}
	r := s.removeHiddenFields(item)
	if _, ok := r["hidden"]; ok {
		t.Fatal("the field 'hidden' should have been removed from the item")
	}
	if _, ok := r["visible"]; !ok {
		t.Fatal("the field 'visible' should not have been removed from the item")
	}
}

func TestExtractFieldText(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	f := &Field{
		Name: "title",
		ElementLocations: []ElementLocation{
			{
				Selector: ".artist-name",
			},
		},
	}
	event := map[string]interface{}{}
	err = extractField(f, event, doc.Selection, "")
	if err != nil {
		t.Fatalf("unexpected error while extracting the text field: %v", err)
	}
	if v, ok := event["title"]; !ok {
		t.Fatal("event doesn't contain the expected title field")
	} else {
		expected := "Final Story"
		if v != expected {
			t.Fatalf("expected '%s' for title but got '%s'", expected, v)
		}
	}
}

func TestExtractFieldTextEntireSubtree(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	f := &Field{
		Name: "title",
		ElementLocations: []ElementLocation{
			{
				Selector:      ".artist-teaser",
				EntireSubtree: true,
			},
		},
	}
	event := map[string]interface{}{}
	err = extractField(f, event, doc.Selection, "")
	if err != nil {
		t.Fatalf("unexpected error while extracting the text field: %v", err)
	}
	if v, ok := event["title"]; !ok {
		t.Fatal("event doesn't contain the expected title field")
	} else {
		expected := `Final Story
                                                    Aargau`
		if v != expected {
			t.Fatalf("expected '%s' for title but got '%s'", expected, v)
		}
	}
}

func TestExtractFieldTextAllNodes(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	f := &Field{
		Name: "title",
		ElementLocations: []ElementLocation{
			{
				Selector:  ".artist-name",
				AllNodes:  true,
				Separator: ", ",
			},
		},
	}
	event := map[string]interface{}{}
	err = extractField(f, event, doc.Selection, "")
	if err != nil {
		t.Fatalf("unexpected error while extracting the text field: %v", err)
	}
	if v, ok := event["title"]; !ok {
		t.Fatal("event doesn't contain the expected title field")
	} else {
		expected := "Final Story, Moment Of Madness, Irony of Fate"
		if v != expected {
			t.Fatalf("expected '%s' for title but got '%s'", expected, v)
		}
	}
}

func TestExtractFieldTextRegex(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	f := &Field{
		Name: "time",
		ElementLocations: []ElementLocation{
			{
				Selector: "a.event-date",
				RegexExtract: RegexConfig{
					RegexPattern: "[0-9]{2}:[0-9]{2}",
				},
			},
		},
	}
	event := map[string]interface{}{}
	err = extractField(f, event, doc.Selection, "")
	if err != nil {
		t.Fatalf("unexpected error while extracting the time field: %v", err)
	}
	if v, ok := event["time"]; !ok {
		t.Fatal("event doesn't contain the expected time field")
	} else {
		expected := "20:00"
		if v != expected {
			t.Fatalf("expected '%s' for title but got '%s'", expected, v)
		}
	}
}

func TestExtractFieldUrl(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	f := &Field{
		Name: "url",
		Type: "url",
		ElementLocations: []ElementLocation{
			{
				Selector: "a.event-date",
			},
		},
	}
	event := map[string]interface{}{}
	err = extractField(f, event, doc.Selection, "https://www.dachstock.ch/events")
	if err != nil {
		t.Fatalf("unexpected error while extracting the time field: %v", err)
	}
	if v, ok := event["url"]; !ok {
		t.Fatal("event doesn't contain the expected url field")
	} else {
		expected := "https://www.dachstock.ch/events/10-03-2023-krachstock-final-story"
		if v != expected {
			t.Fatalf("expected '%s' for url but got '%s'", expected, v)
		}
	}
}

func TestExtractFieldUrlFull(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString2))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	f := &Field{
		Name: "url",
		Type: "url",
		ElementLocations: []ElementLocation{
			{
				Selector: "h2 > a",
			},
		},
	}
	event := map[string]interface{}{}
	err = extractField(f, event, doc.Selection, "https://www.eventfabrik-muenchen.de/events?s=&tribe_events_cat=konzert&tribe_events_venue=&tribe_events_month=")
	if err != nil {
		t.Fatalf("unexpected error while extracting the time field: %v", err)
	}
	if v, ok := event["url"]; !ok {
		t.Fatal("event doesn't contain the expected url field")
	} else {
		expected := "https://www.eventfabrik-muenchen.de/event/heinz-rudolf-kunze-verstaerkung-2/"
		if v != expected {
			t.Fatalf("expected '%s' for url but got '%s'", expected, v)
		}
	}
}

func TestExtractFieldUrlQuery(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString3))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	f := &Field{
		Name: "url",
		Type: "url",
		ElementLocations: []ElementLocation{
			{
				Selector: "h2 > a",
			},
		},
	}
	event := map[string]interface{}{}
	err = extractField(f, event, doc.Selection, "https://www.eventfabrik-muenchen.de/events?s=&tribe_events_cat=konzert&tribe_events_venue=&tribe_events_month=")
	if err != nil {
		t.Fatalf("unexpected error while extracting the time field: %v", err)
	}
	if v, ok := event["url"]; !ok {
		t.Fatal("event doesn't contain the expected url field")
	} else {
		expected := "https://www.eventfabrik-muenchen.de/events?bli=bla"
		if v != expected {
			t.Fatalf("expected '%s' for url but got '%s'", expected, v)
		}
	}
}

func TestExtractFieldDate(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	f := &Field{
		Name: "date",
		Type: "date",
		Components: []DateComponent{
			{
				Covers: date.CoveredDateParts{
					Day:   true,
					Month: true,
					Year:  true,
					Time:  true,
				},
				ElementLocation: ElementLocation{
					Selector: "a.event-date",
				},
				Layout: []string{
					"Mon, 02.01.2006 - 15:04",
				},
			},
		},
		DateLocation: "Europe/Berlin",
	}
	event := map[string]interface{}{}
	err = extractField(f, event, doc.Selection, "")
	if err != nil {
		t.Fatalf("unexpected error while extracting the date field: %v", err)
	}
	if v, ok := event["date"]; !ok {
		t.Fatal("event doesn't contain the expected date field")
	} else {
		loc, _ := time.LoadLocation(f.DateLocation)
		expected := time.Date(2023, 3, 10, 20, 0, 0, 0, loc)
		vTime, ok := v.(time.Time)
		if !ok {
			t.Fatalf("%v is not of type time.Time", err)
		}
		if !vTime.Equal(expected) {
			t.Fatalf("expected '%s' for date but got '%s'", expected, vTime)
		}
	}
}

func TestExtractFieldDateTransform(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	f := &Field{
		Name: "date",
		Type: "date",
		Components: []DateComponent{
			{
				Covers: date.CoveredDateParts{
					Day:   true,
					Month: true,
					Year:  true,
					Time:  true,
				},
				ElementLocation: ElementLocation{
					Selector: "a.event-date",
				},
				Transform: []TransformConfig{
					{
						TransformType: "regex-replace",
						RegexPattern:  "\\.",
						Replacement:   "/",
					},
				},
				Layout: []string{
					"Mon, 02/01/2006 - 15:04",
				},
			},
		},
		DateLocation: "Europe/Berlin",
	}
	event := map[string]interface{}{}
	err = extractField(f, event, doc.Selection, "")
	if err != nil {
		t.Fatalf("unexpected error while extracting the date field: %v", err)
	}
	if v, ok := event["date"]; !ok {
		t.Fatal("event doesn't contain the expected date field")
	} else {
		loc, _ := time.LoadLocation(f.DateLocation)
		expected := time.Date(2023, 3, 10, 20, 0, 0, 0, loc)
		vTime, ok := v.(time.Time)
		if !ok {
			t.Fatalf("%v is not of type time.Time", err)
		}
		if !vTime.Equal(expected) {
			t.Fatalf("expected '%s' for date but got '%s'", expected, vTime)
		}
	}
}
