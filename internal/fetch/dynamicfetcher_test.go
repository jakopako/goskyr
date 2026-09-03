package fetch

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
)

func TestOuterHTMLParamsIncludesShadowDOM(t *testing.T) {
	params := outerHTMLParams(cdp.NodeID(42))

	if params.NodeID != cdp.NodeID(42) {
		t.Fatalf("expected node ID 42, got %d", params.NodeID)
	}

	if !params.IncludeShadowDOM {
		t.Fatal("expected shadow DOM to be included")
	}
}

func TestFlattenShadowDOM(t *testing.T) {
	body, err := flattenShadowDOM(`<article class="event"><template shadowrootmode="open"><div class="location">Berlin</div><template shadowroot="open"><span class="venue">Venue</span></template></template></article>`)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	item := doc.Find(".event")
	if got := item.Find(".location").Text(); got != "Berlin" {
		t.Fatalf("expected location from shadow DOM, got %q", got)
	}
	if got := item.Find(".venue").Text(); got != "Venue" {
		t.Fatalf("expected nested shadow DOM content, got %q", got)
	}
	if got := item.Find("template").Length(); got != 0 {
		t.Fatalf("expected shadow DOM templates to be flattened, found %d", got)
	}
}
