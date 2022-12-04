package automate

import "testing"

func TestPathToSelector(t *testing.T) {
	path := []string{"body.home", "div.wrapper.schedule", "div.event-list", "a.item"}
	expected := "body.home > div.wrapper.schedule > div.event-list > a.item"
	if s := pathToSelector(path); s != expected {
		t.Fatalf("expected '%s', but got '%s'", expected, s)
	}
}

func TestSelectorToPath(t *testing.T) {
	selector := "body.home > div.wrapper.schedule > div.event-list > a.item"
	expected := []string{"body.home", "div.wrapper.schedule", "div.event-list", "a.item"}
	p := selectorToPath(selector)
	if len(p) != len(expected) {
		t.Fatalf("expected %v but got %v", expected, p)
	}
	for i, e := range expected {
		if e != p[i] {
			t.Fatalf("expected '%v', but got '%v'", expected, p)
		}
	}
}

func TestNodesEqual(t *testing.T) {
	n1 := []string{"div.event-list", "div.text.nano-content"}
	n2 := []string{"div.event-list", "div.nano-content.text"}
	for i, n := range n1 {
		if !nodesEqual(n, n2[i]) {
			t.Fatalf("nodes %s and %s are equal", n, n2[i])
		}
	}
}

func TestNotNodesEqual(t *testing.T) {
	n1 := []string{"div.event-list", "div.text.nano-content"}
	n2 := []string{"div", "div.text.nano"}
	for i, n := range n1 {
		if nodesEqual(n, n2[i]) {
			t.Fatalf("nodes %s and %s are not equal", n, n2[i])
		}
	}
}
