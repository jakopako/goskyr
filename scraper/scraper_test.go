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
	htmlString4 = `
	<div class="text">
		<a href="programm.php?m=4&j=2023&vid=4378">
			<div class="reihe">Treffpunkt</div>
			<div class="titel">Kreativ-Workshop: "My message to the world"
				<span class="supportband">— Творча майстерня: "Моє послання до світу"</span>
			</div>
			<div class="beschreibung"><em>Osterferienprogramm Ukrainehilfe / ПРОГРАМА ПАСХАЛЬНИХ КАНІКУЛ ПІДТРИМКА УКРАЇНЦІВ</em></div>
		</a>
	</div>`
	htmlString5 = `                                        
	<h2>
		<a href="?bli=bla"
			title="Heinz Rudolf Kunze &amp; Verstärkung &#8211; ABGESAGT">
			<span>29.02.</span><span>Heinz Rudolf Kunze &amp; Verstärkung
				&#8211; ABGESAGT</span> </a>
	</h2>`
	htmlString6 = `                                        
	<h2>
		<a href="../site/event/id/165"
			title="Heinz Rudolf Kunze &amp; Verstärkung &#8211; ABGESAGT">
			<span>29.02.</span><span>Heinz Rudolf Kunze &amp; Verstärkung
				&#8211; ABGESAGT</span> </a>
	</h2>`
	htmlString7 = `                                        
	<h2>
		<a href="../site/event/id/165"
			title="Heinz Rudolf Kunze &amp; Verstärkung &#8211; ABGESAGT">
			<span>20.02.</span><span>Heinz Rudolf Kunze &amp; Verstärkung
				&#8211; ABGESAGT</span> </a>
	</h2>`
)

func TestFilterItemMatchTrue(t *testing.T) {
	item := map[string]interface{}{"title": "Jacob Collier - Concert"}
	s := &Scraper{
		Fields: []Field{
			{
				Name: "title",
			},
		},
		Filters: []*Filter{
			{
				Field:      "title",
				Expression: ".*Concert",
				Match:      true,
			},
		},
	}
	err := s.initializeFilters()
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	f := s.filterItem(item)
	if !f {
		t.Fatalf("expected 'true' but got 'false'")
	}
}

func TestFilterItemMatchFalse(t *testing.T) {
	item := map[string]interface{}{"title": "Jacob Collier - Cancelled"}
	s := &Scraper{
		Fields: []Field{
			{
				Name: "title",
			},
		},
		Filters: []*Filter{
			{
				Field:      "title",
				Expression: ".*Cancelled",
				Match:      false,
			},
		},
	}
	err := s.initializeFilters()
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	f := s.filterItem(item)
	if f {
		t.Fatalf("expected 'false' but got 'true'")
	}
}

func TestFilterItemByDateMatchTrue(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	item := map[string]interface{}{"date": time.Date(2023, 10, 20, 19, 1, 0, 0, loc)}
	s := &Scraper{
		Fields: []Field{
			{
				Name: "date",
				Type: "date",
			},
		},
		Filters: []*Filter{
			{
				Field:      "date",
				Expression: "> 2023-10-20T19:00",
				Match:      true,
			},
		},
	}
	err := s.initializeFilters()
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	f := s.filterItem(item)
	if !f {
		t.Fatalf("expected 'true' but got 'false'")
	}
}

func TestFilterItemByDateMatchTrue2(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	item := map[string]interface{}{"date": time.Date(2023, 10, 20, 19, 0, 0, 0, loc)}
	s := &Scraper{
		Fields: []Field{
			{
				Name: "date",
				Type: "date",
			},
		},
		Filters: []*Filter{
			{
				Field:      "date",
				Expression: "> 2023-10-20T19:00",
				Match:      true,
			},
		},
	}
	err := s.initializeFilters()
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	f := s.filterItem(item)
	if f {
		t.Fatalf("expected 'false' but got 'true'")
	}
}

func TestFilterItemByDateMatchFalse(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	item := map[string]interface{}{"date": time.Date(2023, 10, 20, 19, 1, 0, 0, loc)}
	s := &Scraper{
		Fields: []Field{
			{
				Name: "date",
				Type: "date",
			},
		},
		Filters: []*Filter{
			{
				Field:      "date",
				Expression: "> 2023-10-20T19:00",
				Match:      false,
			},
		},
	}
	err := s.initializeFilters()
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	f := s.filterItem(item)
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

func TestExtractFieldUrlFile(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString4))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	f := &Field{
		Name: "url",
		Type: "url",
		ElementLocations: []ElementLocation{
			{
				Selector: "div > a",
			},
		},
	}
	event := map[string]interface{}{}
	err = extractField(f, event, doc.Selection, "https://www.roxy.ulm.de/programm/programm.php")
	if err != nil {
		t.Fatalf("unexpected error while extracting the time field: %v", err)
	}
	if v, ok := event["url"]; !ok {
		t.Fatal("event doesn't contain the expected url field")
	} else {
		expected := "https://www.roxy.ulm.de/programm/programm.php?m=4&j=2023&vid=4378"
		if v != expected {
			t.Fatalf("expected '%s' for url but got '%s'", expected, v)
		}
	}
}

func TestExtractFieldUrlParentDir(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString6))
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
	err = extractField(f, event, doc.Selection, "http://point11.ch/site/home")
	if err != nil {
		t.Fatalf("unexpected error while extracting the time field: %v", err)
	}
	if v, ok := event["url"]; !ok {
		t.Fatal("event doesn't contain the expected url field")
	} else {
		expected := "http://point11.ch/site/event/id/165"
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

func TestExtractFieldDate29Feb(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString5))
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
				},
				ElementLocation: ElementLocation{
					Selector: "h2 > a > span",
				},
				Layout: []string{
					"02.01.",
				},
			},
			{
				Covers: date.CoveredDateParts{
					Time: true,
				},
				ElementLocation: ElementLocation{
					Default: "19:30",
				},
				Layout: []string{
					"15:04",
				},
			},
		},
		DateLocation: "Europe/Berlin",
		GuessYear:    true,
	}
	dt, err := getDates(f, doc.Selection, dateDefaults{year: 2023})
	if err != nil {
		t.Fatalf("unexpected error while extracting the date field: %v", err)
	}
	if dt[0].Year() != 2024 {
		t.Fatalf("expected '2024' as year of date but got '%d'", dt[0].Year())
	}
}

func TestGuessYearSimple(t *testing.T) {
	// items dates span period around change of year
	s := &Scraper{
		Fields: []Field{
			{
				Type:      "date",
				GuessYear: true,
				Name:      "date",
			},
		},
	}
	loc, _ := time.LoadLocation("CET")
	items := []map[string]interface{}{
		{
			"date": time.Date(2023, 12, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 12, 24, 21, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 1, 2, 20, 0, 0, 0, loc),
		},
	}
	expectedItems := []map[string]interface{}{
		{
			"date": time.Date(2023, 12, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 12, 24, 21, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2024, 1, 2, 20, 0, 0, 0, loc),
		},
	}
	s.guessYear(items, time.Date(2023, 11, 30, 20, 30, 0, 0, loc))
	for i, d := range items {
		if d["date"] != expectedItems[i]["date"] {
			t.Fatalf("expected '%v' as year of date but got '%v'", expectedItems[i]["date"], d["date"])
		}
	}
}

func TestGuessYearUnordered(t *testing.T) {
	// items dates are not perfectly ordered and span
	// period around change of year
	s := &Scraper{
		Fields: []Field{
			{
				Type:      "date",
				GuessYear: true,
				Name:      "date",
			},
		},
	}
	loc, _ := time.LoadLocation("CET")
	items := []map[string]interface{}{
		{
			"date": time.Date(2023, 11, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 12, 14, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 12, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 12, 24, 21, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 1, 2, 20, 0, 0, 0, loc),
		},
	}
	expectedItems := []map[string]interface{}{
		{
			"date": time.Date(2023, 11, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 12, 14, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 12, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 12, 24, 21, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2024, 1, 2, 20, 0, 0, 0, loc),
		},
	}
	s.guessYear(items, time.Date(2023, 11, 1, 20, 30, 0, 0, loc))
	for i, d := range items {
		if d["date"] != expectedItems[i]["date"] {
			t.Fatalf("expected '%v' as year of date but got '%v'", expectedItems[i]["date"], d["date"])
		}
	}
}

func TestGuessYear2Years(t *testing.T) {
	// items dates span more than 2 years
	s := &Scraper{
		Fields: []Field{
			{
				Type:      "date",
				GuessYear: true,
				Name:      "date",
			},
		},
	}
	loc, _ := time.LoadLocation("CET")
	items := []map[string]interface{}{
		{
			"date": time.Date(2023, 12, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 1, 14, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 5, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 9, 24, 21, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 2, 2, 20, 0, 0, 0, loc),
		},
	}
	expectedItems := []map[string]interface{}{
		{
			"date": time.Date(2023, 12, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2024, 1, 14, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2024, 5, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2024, 9, 24, 21, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2025, 2, 2, 20, 0, 0, 0, loc),
		},
	}
	s.guessYear(items, time.Date(2023, 11, 1, 20, 30, 0, 0, loc))
	for i, d := range items {
		if d["date"] != expectedItems[i]["date"] {
			t.Fatalf("expected '%v' as year of date but got '%v'", expectedItems[i]["date"], d["date"])
		}
	}
}

func TestGuessYearStartBeforeReference(t *testing.T) {
	// items date start before given reference
	s := &Scraper{
		Fields: []Field{
			{
				Type:      "date",
				GuessYear: true,
				Name:      "date",
			},
		},
	}
	loc, _ := time.LoadLocation("CET")
	items := []map[string]interface{}{
		{
			"date": time.Date(2023, 12, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 12, 24, 21, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 1, 2, 20, 0, 0, 0, loc),
		},
	}
	expectedItems := []map[string]interface{}{
		{
			"date": time.Date(2023, 12, 2, 20, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2023, 12, 24, 21, 30, 0, 0, loc),
		},
		{
			"date": time.Date(2024, 1, 2, 20, 0, 0, 0, loc),
		},
	}
	s.guessYear(items, time.Date(2024, 1, 30, 20, 30, 0, 0, loc))
	for i, d := range items {
		if d["date"] != expectedItems[i]["date"] {
			t.Fatalf("expected '%v' as year of date but got '%v'", expectedItems[i]["date"], d["date"])
		}
	}
}

func TestDefaultTextValue(t *testing.T) {
	d := "default text"
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString5))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	l := &ElementLocation{
		Selector: ".non-existent",
		Default:  d,
	}
	v, err := getTextString(l, doc.Selection)
	if err != nil {
		t.Fatalf("unexpected error while extracting the element: %v", err)
	}
	if v != d {
		t.Fatalf("expected '%s' but got '%s'", d, v)
	}
}

func TestDefaultTextValueExistentValue(t *testing.T) {
	d := "default text"
	e := "Treffpunkt"
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString4))
	if err != nil {
		t.Fatalf("unexpected error while reading html string: %v", err)
	}
	l := &ElementLocation{
		Selector: "div > a > div",
		Default:  d,
	}
	v, err := getTextString(l, doc.Selection)
	if err != nil {
		t.Fatalf("unexpected error while extracting the element: %v", err)
	}
	if v != e {
		t.Fatalf("expected '%s' but got '%s'", e, v)
	}
}

func TestDefaultValueDateComponentNonExistent(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString7))
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
				},
				ElementLocation: ElementLocation{
					Selector: "h2 > a > span",
				},
				Layout: []string{
					"02.01.",
				},
			},
			{
				Covers: date.CoveredDateParts{
					Time: true,
				},
				ElementLocation: ElementLocation{
					Selector: ".non-existent",
					Default:  "19:30",
				},
				Layout: []string{
					"15:04",
				},
			},
		},
		DateLocation: "Europe/Berlin",
		GuessYear:    true,
	}
	dt, err := getDates(f, doc.Selection, dateDefaults{})
	if err != nil {
		t.Fatalf("unexpected error while extracting the date field: %v", err)
	}
	if dt[0].Hour() != 19 {
		t.Fatalf("expected hour to be %d but got %d", 19, dt[0].Hour())
	}
	if dt[0].Minute() != 30 {
		t.Fatalf("expected minute to be %d but got %d", 30, dt[0].Minute())
	}
}

func TestDefaultValueDateComponentRegexExtractError(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString7))
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
				},
				ElementLocation: ElementLocation{
					Selector: "h2 > a > span",
					Default:  "1. April",
					RegexExtract: RegexConfig{
						RegexPattern: "[A-Z]{20}", // non-matching regex
						IgnoreErrors: true,        // will make sure the selector returns an empty string in case of an error in which case we default to the given default
					},
				},
				Layout: []string{
					"2. January",
				},
			},
			{
				Covers: date.CoveredDateParts{
					Time: true,
				},
				ElementLocation: ElementLocation{
					Selector: ".non-existent",
					Default:  "19:30",
				},
				Layout: []string{
					"15:04",
				},
			},
		},
		DateLocation: "Europe/Berlin",
		GuessYear:    true,
	}
	dt, err := getDates(f, doc.Selection, dateDefaults{})
	if err != nil {
		t.Fatalf("unexpected error while extracting the date field: %v", err)
	}
	if dt[0].Day() != 1 {
		t.Fatalf("expected day to be %d but got %d", 1, dt[0].Day())
	}
	if dt[0].Month() != 4 {
		t.Fatalf("expected month to be %d but got %d", 4, dt[0].Month())
	}
}
