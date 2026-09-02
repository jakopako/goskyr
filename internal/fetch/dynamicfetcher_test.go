package fetch

import (
	"testing"

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
