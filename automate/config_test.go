package automate

// func TestPathToSelector(t *testing.T) {
// 	path := []string{"body.home", "div.wrapper.schedule", "div.event-list", "a.item"}
// 	expected := "body.home > div.wrapper.schedule > div.event-list > a.item"
// 	if s := pathToSelector(path); s != expected {
// 		t.Fatalf("expected '%s', but got '%s'", expected, s)
// 	}
// }

// func TestSelectorToPath(t *testing.T) {
// 	selector := "body.home > div.wrapper.schedule > div.event-list > a.item"
// 	expected := []string{"body.home", "div.wrapper.schedule", "div.event-list", "a.item"}
// 	p := selectorToPath(selector)
// 	if len(p) != len(expected) {
// 		t.Fatalf("expected %v but got %v", expected, p)
// 	}
// 	for i, e := range expected {
// 		if e != p[i] {
// 			t.Fatalf("expected '%v', but got '%v'", expected, p)
// 		}
// 	}
// }

// func TestNodesEqual(t *testing.T) {
// 	n1 := []string{"div.event-list", "div.text.nano-content"}
// 	n2 := []string{"div.event-list", "div.nano-content.text"}
// 	for i, n := range n1 {
// 		if !nodesEqual(n, n2[i]) {
// 			t.Fatalf("nodes %s and %s are equal", n, n2[i])
// 		}
// 	}
// }

// func TestNotNodesEqual(t *testing.T) {
// 	n1 := []string{"div.event-list", "div.text.nano-content"}
// 	n2 := []string{"div", "div.text.nano"}
// 	for i, n := range n1 {
// 		if nodesEqual(n, n2[i]) {
// 			t.Fatalf("nodes %s and %s are not equal", n, n2[i])
// 		}
// 	}
// }

// func TestFilter(t *testing.T) {
// 	l := []*locationProps{
// 		{
// 			loc:   scraper.ElementLocation{},
// 			count: 4,
// 		},
// 		{
// 			loc:   scraper.ElementLocation{},
// 			count: 8,
// 		},
// 	}
// 	nl := filter(l, 8, false)
// 	if len(nl) != 1 {
// 		t.Fatalf("expected only 1 element to remain after filtering but got %+v", nl)
// 	}
// 	if nl[0].count != 8 {
// 		t.Fatal("wrong element filtered.")
// 	}
// }

// func TestNoStaticFieldsFilter(t *testing.T) {
// 	l := []*locationProps{
// 		{
// 			loc:      scraper.ElementLocation{},
// 			count:    4,
// 			examples: []string{"a", "a"},
// 		},
// 		{
// 			loc:      scraper.ElementLocation{},
// 			count:    8,
// 			examples: []string{"a", "b"},
// 		},
// 	}
// 	nl := filter(l, 4, true)
// 	if len(nl) != 1 {
// 		t.Fatalf("expected only 1 element to remain after filtering but got %+v", nl)
// 	}
// 	if nl[0].count != 8 { // the remaining element should have count 8
// 		t.Fatal("wrong element filtered.")
// 	}
// }
