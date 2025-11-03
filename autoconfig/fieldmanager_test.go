package autoconfig

import (
	"slices"
	"testing"

	"github.com/jakopako/goskyr/utils"
)

func TestNewElementManagerFromHtml(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected fieldManager
	}{
		{
			name: "single element with text",
			html: `<html><body><div class="container">Hello World</div></body></html>`,
			expected: []*fieldProps{
				{
					path: []node{
						{tagName: "body"},
						{tagName: "div", classes: []string{"container"}},
					},
					attr:     "",
					count:    1,
					examples: []string{"Hello World"},
				},
			},
		},
		{
			name: "single element with attributes",
			html: `<html><body><img class="image" src="image.jpg"/></body></html>`,
			expected: []*fieldProps{
				{
					path: []node{
						{tagName: "body"},
						{tagName: "img", classes: []string{"image"}},
					},
					attr:     "src",
					count:    1,
					examples: []string{"image.jpg"},
				},
			},
		},
		{
			name: "child elements",
			html: `<html><body><div class="container">child0<p>foo</p>child2</div></body></html>`,
			expected: []*fieldProps{
				{
					path: []node{
						{tagName: "body"},
						{tagName: "div", classes: []string{"container"}},
					},
					textIndex: 0,
					count:     1,
					examples:  []string{"child0"},
				},
				{
					path: []node{
						{tagName: "body"},
						{tagName: "div", classes: []string{"container"}},
						{tagName: "p"},
					},
					textIndex: 0,
					count:     1,
					examples:  []string{"foo"},
				},
				{
					path: []node{
						{tagName: "body"},
						{tagName: "div", classes: []string{"container"}},
					},
					textIndex: 2,
					count:     1,
					examples:  []string{"child2"},
				},
			},
		},
		{
			name: "multiple nodes same level, tag & classes",
			html: `<html><body><ul class="list"><li class="item">item1</li><li class="item">item2</li><li class="item">item3</li></ul></body></html>`,
			expected: []*fieldProps{
				{
					path: []node{
						{tagName: "body"},
						{tagName: "ul", classes: []string{"list"}},
						{tagName: "li", classes: []string{"item"}},
					},
					count:    1,
					examples: []string{"item1"},
				},
				{
					path: []node{
						{tagName: "body"},
						{tagName: "ul", classes: []string{"list"}},
						{tagName: "li", classes: []string{"item"}, pseudoClasses: []string{"nth-child(2)"}},
					},
					count:    1,
					examples: []string{"item2"},
				},
				{
					path: []node{
						{tagName: "body"},
						{tagName: "ul", classes: []string{"list"}},
						{tagName: "li", classes: []string{"item"}, pseudoClasses: []string{"nth-child(3)"}},
					},
					count:    1,
					examples: []string{"item3"},
				},
			},
		},
		{
			name: "props in non-self-closing tags",
			html: `<html><body><a href="https://example.com" title="Example Link">Click Here</a></body></html>`,
			expected: []*fieldProps{
				{
					path: []node{
						{tagName: "body"},
						{tagName: "a"},
					},
					attr:     "href",
					count:    1,
					examples: []string{"https://example.com"},
				},
				{
					path: []node{
						{tagName: "body"},
						{tagName: "a"},
					},
					attr:     "title",
					count:    1,
					examples: []string{"Example Link"},
				},
				{
					path: []node{
						{tagName: "body"},
						{tagName: "a"},
					},
					count:    1,
					examples: []string{"Click Here"},
				},
			},
		},
		{
			name: "siblings with overlapping classes -> nth-child",
			html: `<html><body><div class="box highlight">Box 1</div><div class="box">Box 2</div></body></html>`,
			expected: []*fieldProps{
				{
					path: []node{
						{tagName: "body"},
						{tagName: "div", classes: []string{"box", "highlight"}},
					},
					count:    1,
					examples: []string{"Box 1"},
				},
				{
					path: []node{
						{tagName: "body"},
						{tagName: "div", classes: []string{"box"}, pseudoClasses: []string{"nth-child(2)"}},
					},
					count:    1,
					examples: []string{"Box 2"},
				},
			},
		},
		{
			name: "child elements with comments",
			html: `<html><body><div><!-- This is a comment -->Visible Text<p>Paragraph Text<!-- Another comment --></p></div></body></html>`,
			expected: []*fieldProps{
				{
					path: []node{
						{tagName: "body"},
						{tagName: "div"},
					},
					count:     1,
					textIndex: 1,
					examples:  []string{"Visible Text"},
				},
				{
					path: []node{
						{tagName: "body"},
						{tagName: "div"},
						{tagName: "p"},
					},
					count:     1,
					textIndex: 0,
					examples:  []string{"Paragraph Text"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := newFieldManagerFromHtml(tt.html)

			if !fm.equals(&tt.expected) {
				t.Fatalf("fieldManager mismatch.\nGot: \n%s\nWant: \n%s", fm.string(), tt.expected.string())
			}
		})
	}
}

func TestCompareFieldProps(t *testing.T) {
	basePath := path{
		{tagName: "html"},
		{tagName: "body"},
		{tagName: "div"},
	}
	makeFP := func(p path, attr string, textIndex, count int, examples []string, name string, iStrip int) *fieldProps {
		return &fieldProps{
			path:      p,
			attr:      attr,
			textIndex: textIndex,
			count:     count,
			examples:  examples,
			name:      name,
			iStrip:    iStrip,
		}
	}

	tests := []struct {
		name string
		a    *fieldProps
		b    *fieldProps
		want int // -1: a<b, 0: a==b, 1: a>b
	}{
		{
			name: "equal props",
			a:    makeFP(basePath, "", 0, 1, []string{"x"}, "", 0),
			b:    makeFP(basePath, "", 0, 1, []string{"x"}, "", 0),
			want: 0,
		},
		{
			name: "path ordering",
			a:    makeFP(path{{tagName: "a"}}, "", 0, 1, []string{"x"}, "", 0),
			b:    makeFP(path{{tagName: "b"}}, "", 0, 1, []string{"x"}, "", 0),
			want: -1,
		},
		{
			name: "attr ordering",
			a:    makeFP(basePath, "a", 0, 1, []string{"x"}, "", 0),
			b:    makeFP(basePath, "b", 0, 1, []string{"x"}, "", 0),
			want: -1,
		},
		{
			name: "textIndex ordering",
			a:    makeFP(basePath, "", 0, 1, []string{"x"}, "", 0),
			b:    makeFP(basePath, "", 1, 1, []string{"x"}, "", 0),
			want: -1,
		},
		{
			name: "count ordering",
			a:    makeFP(basePath, "", 0, 1, []string{"x"}, "", 0),
			b:    makeFP(basePath, "", 0, 2, []string{"x"}, "", 0),
			want: -1,
		},
		{
			name: "examples ordering",
			a:    makeFP(basePath, "", 0, 1, []string{"a"}, "", 0),
			b:    makeFP(basePath, "", 0, 1, []string{"b"}, "", 0),
			want: -1,
		},
		{
			name: "name ordering",
			a:    makeFP(basePath, "", 0, 1, []string{"x"}, "a", 0),
			b:    makeFP(basePath, "", 0, 1, []string{"x"}, "b", 0),
			want: -1,
		},
		{
			name: "iStrip ordering",
			a:    makeFP(basePath, "", 0, 1, []string{"x"}, "", 0),
			b:    makeFP(basePath, "", 0, 1, []string{"x"}, "", 1),
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareFieldProps(tt.a, tt.b)
			switch tt.want {
			case 0:
				if got != 0 {
					t.Fatalf("expected equal (0), got %d", got)
				}
			case -1:
				if got >= 0 {
					t.Fatalf("expected a < b (<0), got %d", got)
				}
			case 1:
				if got <= 0 {
					t.Fatalf("expected a > b (>0), got %d", got)
				}
			}
		})
	}
}

func TestCheckOverlapAndUpdate(t *testing.T) {
	makeNode := func(tag string, classes []string, pcls []string) node {
		return node{tagName: tag, classes: classes, pseudoClasses: pcls}
	}
	makeFP := func(p path, attr string, textIndex int, examples []string, count int, iStrip int) *fieldProps {
		return &fieldProps{
			path:      p,
			attr:      attr,
			textIndex: textIndex,
			count:     count,
			examples:  append([]string{}, examples...),
			iStrip:    iStrip,
		}
	}
	equalPath := func(a, b path) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i].tagName != b[i].tagName {
				return false
			}
			if !utils.SliceEquals(a[i].classes, b[i].classes) {
				return false
			}
			if !utils.SliceEquals(a[i].pseudoClasses, b[i].pseudoClasses) {
				return false
			}
		}
		return true
	}

	tests := []struct {
		name          string
		fp            *fieldProps
		other         *fieldProps
		wantUpdated   bool
		wantPathAfter path
		wantCount     int
		wantExamples  []string
	}{
		{
			name:         "different textIndex -> no update",
			fp:           makeFP(path{{tagName: "body"}, {tagName: "div"}}, "", 0, []string{"a"}, 1, 0),
			other:        makeFP(path{{tagName: "body"}, {tagName: "div"}}, "", 1, []string{"b"}, 1, 0),
			wantUpdated:  false,
			wantCount:    1,
			wantExamples: []string{"a"},
		},
		{
			name:         "different attr -> no update",
			fp:           makeFP(path{{tagName: "body"}, {tagName: "img"}}, "src", 0, []string{"a"}, 1, 0),
			other:        makeFP(path{{tagName: "body"}, {tagName: "img"}}, "title", 0, []string{"b"}, 1, 0),
			wantUpdated:  false,
			wantCount:    1,
			wantExamples: []string{"a"},
		},
		{
			name:         "different path length -> no update",
			fp:           makeFP(path{{tagName: "body"}, {tagName: "div"}}, "", 0, []string{"a"}, 1, 0),
			other:        makeFP(path{{tagName: "body"}}, "", 0, []string{"b"}, 1, 0),
			wantUpdated:  false,
			wantCount:    1,
			wantExamples: []string{"a"},
		},
		{
			name:         "tag mismatch -> no update",
			fp:           makeFP(path{{tagName: "body"}, {tagName: "div"}}, "", 0, []string{"a"}, 1, 0),
			other:        makeFP(path{{tagName: "body"}, {tagName: "span"}}, "", 0, []string{"b"}, 1, 0),
			wantUpdated:  false,
			wantCount:    1,
			wantExamples: []string{"a"},
		},
		{
			name: "pseudoClasses differ but i>iStrip so compared -> mismatch -> no update",
			fp: makeFP(path{
				makeNode("body", nil, nil),
				makeNode("li", nil, []string{"nth-child(1)"}),
			}, "", 0, []string{"a"}, 1, 0), // iStrip 0 so at i=1 we compare pseudoClasses
			other: makeFP(path{
				makeNode("body", nil, nil),
				makeNode("li", nil, []string{"nth-child(2)"}),
			}, "", 0, []string{"b"}, 1, 0),
			wantUpdated:  false,
			wantCount:    1,
			wantExamples: []string{"a"},
		},
		{
			name: "both classes empty -> update, classes stay empty",
			fp: makeFP(path{
				makeNode("body", nil, nil),
				makeNode("p", nil, nil),
			}, "", 0, []string{"a"}, 1, 0),
			other: makeFP(path{
				makeNode("body", nil, nil),
				makeNode("p", nil, nil),
			}, "", 0, []string{"b"}, 1, 0),
			wantUpdated:   true,
			wantPathAfter: path{makeNode("body", nil, nil), makeNode("p", nil, nil)},
			wantCount:     2,
			wantExamples:  []string{"a", "b"},
		},
		{
			name: "overlapping classes before iStrip -> accept intersection",
			fp: makeFP(path{
				makeNode("body", nil, nil),
				makeNode("div", []string{"a", "b"}, nil),
			}, "", 0, []string{"a"}, 1, 1), // iStrip=1 so at i=1 we are NOT > iStrip -> treat as before iStrip
			other: makeFP(path{
				makeNode("body", nil, nil),
				makeNode("div", []string{"b", "c"}, nil),
			}, "", 0, []string{"b"}, 1, 1),
			wantUpdated:   true,
			wantPathAfter: path{makeNode("body", nil, nil), makeNode("div", []string{"b"}, nil)},
			wantCount:     2,
			wantExamples:  []string{"a", "b"},
		},
		{
			name: "overlapping classes before iStrip -> accept intersection, real case",
			fp: makeFP(path{
				makeNode("div", []string{"whats-on"}, nil),
				makeNode("div", []string{"grid", "loading"}, nil),
				makeNode("div", []string{"classical", "grid-item"}, nil),
				makeNode("div", []string{"grid-item__inner", "lev"}, nil),
				makeNode("div", []string{"classical", "text"}, []string{"nth-child(3)"}),
				makeNode("h2", []string{"grid-item__title"}, nil),
			}, "", 0, []string{"a"}, 1, 2),
			other: makeFP(path{
				makeNode("div", []string{"whats-on"}, nil),
				makeNode("div", []string{"grid", "loading"}, nil),
				makeNode("div", []string{"grid-item"}, nil),
				makeNode("div", []string{"grid-item__inner", "lev"}, nil),
				makeNode("div", []string{"classical", "text"}, []string{"nth-child(3)"}),
				makeNode("h2", []string{"grid-item__title"}, nil),
			}, "", 0, []string{"b"}, 1, 0),
			wantUpdated: true,
			wantPathAfter: path{
				makeNode("div", []string{"whats-on"}, nil),
				makeNode("div", []string{"grid", "loading"}, nil),
				makeNode("div", []string{"grid-item"}, nil),
				makeNode("div", []string{"grid-item__inner", "lev"}, nil),
				makeNode("div", []string{"classical", "text"}, []string{"nth-child(3)"}),
				makeNode("h2", []string{"grid-item__title"}, nil),
			},
			wantCount:    2,
			wantExamples: []string{"a", "b"},
		},
		{
			name: "overlapping classes before iStrip -> accept intersection, real case reverse",
			fp: makeFP(path{
				makeNode("div", []string{"whats-on"}, nil),
				makeNode("div", []string{"grid", "loading"}, nil),
				makeNode("div", []string{"grid-item"}, nil),
				makeNode("div", []string{"grid-item__inner", "lev"}, nil),
				makeNode("div", []string{"classical", "text"}, []string{"nth-child(3)"}),
				makeNode("h2", []string{"grid-item__title"}, nil),
			}, "", 0, []string{"b"}, 1, 2),
			other: makeFP(path{
				makeNode("div", []string{"whats-on"}, nil),
				makeNode("div", []string{"grid", "loading"}, nil),
				makeNode("div", []string{"classical", "grid-item"}, nil),
				makeNode("div", []string{"grid-item__inner", "lev"}, nil),
				makeNode("div", []string{"classical", "text"}, []string{"nth-child(3)"}),
				makeNode("h2", []string{"grid-item__title"}, nil),
			}, "", 0, []string{"a"}, 1, 0),
			wantUpdated: true,
			wantPathAfter: path{
				makeNode("div", []string{"whats-on"}, nil),
				makeNode("div", []string{"grid", "loading"}, nil),
				makeNode("div", []string{"grid-item"}, nil),
				makeNode("div", []string{"grid-item__inner", "lev"}, nil),
				makeNode("div", []string{"classical", "text"}, []string{"nth-child(3)"}),
				makeNode("h2", []string{"grid-item__title"}, nil),
			},
			wantCount:    2,
			wantExamples: []string{"b", "a"},
		},
		{
			name: "overlapping classes before iStrip, overlapping classes after iStrip, distinct nth-child -> reject",
			fp: makeFP(
				path{
					makeNode("body", nil, nil),
					makeNode("div", []string{"a", "b"}, nil),
					makeNode("div", nil, nil),
					makeNode("div", []string{"c", "d"}, []string{"nth-child(1)"}),
				}, "", 0, []string{"a"}, 1, 1),
			other: makeFP(
				path{
					makeNode("body", nil, nil),
					makeNode("div", []string{"b", "e"}, nil),
					makeNode("div", nil, nil),
					makeNode("div", []string{"d", "f"}, []string{"nth-child(2)"}),
				}, "", 0, []string{"b"}, 1, 0),
			wantUpdated:  false,
			wantCount:    1,
			wantExamples: []string{"a"},
		},
		{
			name: "overlapping classes before iStrip, overlapping classes after iStrip, same nth-child -> accept",
			fp: makeFP(
				path{
					makeNode("body", nil, nil),
					makeNode("div", []string{"a", "b"}, nil),
					makeNode("div", nil, nil),
					makeNode("div", []string{"c", "d"}, []string{"nth-child(2)"}),
				}, "", 0, []string{"a"}, 1, 1),
			other: makeFP(
				path{
					makeNode("body", nil, nil),
					makeNode("div", []string{"b", "e"}, nil),
					makeNode("div", nil, nil),
					makeNode("div", []string{"d", "f"}, []string{"nth-child(2)"}),
				}, "", 0, []string{"b"}, 1, 0),
			wantUpdated: true,
			wantPathAfter: path{
				makeNode("body", nil, nil),
				makeNode("div", []string{"b"}, nil),
				makeNode("div", nil, nil),
				makeNode("div", []string{"d"}, []string{"nth-child(2)"}),
			},
			wantCount:    2,
			wantExamples: []string{"a", "b"},
		},
		// {
		// 	name: "overlapping classes after iStrip requires full match -> reject if partial",
		// 	fp: makeFP(path{
		// 		makeNode("body", nil, nil),
		// 		makeNode("div", []string{"a", "b"}, nil),
		// 	}, "", 0, []string{"a"}, 1, 0), // iStrip=0 so at i=1 we are > iStrip and require full match
		// 	other: makeFP(path{
		// 		makeNode("body", nil, nil),
		// 		makeNode("div", []string{"b", "c"}, nil),
		// 	}, "", 0, []string{"b"}, 1, 0),
		// 	wantUpdated:  false,
		// 	wantCount:    1,
		// 	wantExamples: []string{"a"},
		// },
		// {
		// 	name: "overlapping classes after iStrip but no full match -> reject",
		// 	fp: makeFP(path{
		// 		makeNode("body", nil, nil),
		// 		makeNode("div", []string{"a", "b"}, nil),
		// 	}, "", 0, []string{"a"}, 1, 0), // iStrip=0 so at i=1 we are > iStrip
		// 	other: makeFP(path{
		// 		makeNode("body", nil, nil),
		// 		makeNode("div", []string{"a", "b", "c"}, nil),
		// 	}, "", 0, []string{"b"}, 1, 0),
		// 	wantUpdated: false,
		// 	// wantPathAfter: path{makeNode("body", nil, nil), makeNode("div", []string{"a", "b"}, nil)},
		// 	wantCount:    1,
		// 	wantExamples: []string{"a"},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origCount := tt.fp.count
			origExamples := append([]string{}, tt.fp.examples...)
			updated := tt.fp.checkOverlapAndUpdate(tt.other)
			if updated != tt.wantUpdated {
				t.Fatalf("updated = %v, want %v", updated, tt.wantUpdated)
			}
			if updated {
				// count incremented
				if tt.fp.count != tt.wantCount {
					t.Fatalf("count = %d, want %d", tt.fp.count, tt.wantCount)
				}
				// examples appended
				if len(tt.fp.examples) != len(tt.wantExamples) {
					t.Fatalf("examples = %v, want %v", tt.fp.examples, tt.wantExamples)
				}
				for i := range tt.wantExamples {
					if tt.fp.examples[i] != tt.wantExamples[i] {
						t.Fatalf("examples[%d] = %s, want %s", i, tt.fp.examples[i], tt.wantExamples[i])
					}
				}
				// path check if expected provided
				if tt.wantPathAfter != nil {
					if !equalPath(tt.fp.path, tt.wantPathAfter) {
						t.Fatalf("path = %v, want %v", tt.fp.path, tt.wantPathAfter)
					}
				}
			} else {
				// unchanged
				if tt.fp.count != origCount {
					t.Fatalf("count changed to %d, want %d", tt.fp.count, origCount)
				}
				if !utils.SliceEquals(tt.fp.examples, origExamples) {
					t.Fatalf("examples changed to %v, want %v", tt.fp.examples, origExamples)
				}
			}
		})
	}
}

func TestStripNthChild_ClearWhenNthChildGE_MinOcc(t *testing.T) {
	// path indices: 0,1,2
	fp := &fieldProps{
		path: []node{
			{tagName: "body", pseudoClasses: []string{"nth-child(1)"}},
			{tagName: "ul", pseudoClasses: []string{"nth-child(2)"}},
			{tagName: "li", pseudoClasses: []string{"nth-child(6)"}},
		},
		iStrip: 0,
	}

	fp.stripNthChild(6)

	// nth-child(6) at index 2 >= minOcc -> should be stripped and iStrip set to 2
	if fp.iStrip != 2 {
		t.Fatalf("iStrip = %d, want %d", fp.iStrip, 2)
	}
	// all indices < iStrip should have been cleared
	for i := range fp.path {
		if len(fp.path[i].pseudoClasses) != 0 {
			t.Fatalf("path[%d].pseudoClasses = %v, want empty", i, fp.path[i].pseudoClasses)
		}
	}
}

func TestStripNthChild_NoClearWhenNthChildLT_MinOcc(t *testing.T) {
	fp := &fieldProps{
		path: []node{
			{tagName: "body", pseudoClasses: []string{"nth-child(1)"}},
			{tagName: "div", pseudoClasses: []string{"nth-child(2)"}},
			{tagName: "span", pseudoClasses: []string{"nth-child(5)"}},
		},
		iStrip: 0,
	}

	fp.stripNthChild(6)

	// none should be stripped because all nth-child < minOcc
	if fp.iStrip != 0 {
		t.Fatalf("iStrip = %d, want %d", fp.iStrip, 0)
	}
	for i := range fp.path {
		if len(fp.path[i].pseudoClasses) == 0 {
			t.Fatalf("path[%d].pseudoClasses was cleared unexpectedly", i)
		}
	}
}

func TestStripNthChild_SubTwoBehavior_MinOccSmall(t *testing.T) {
	// minOcc < 6 -> sub becomes 2; ensure loop starts higher up and clearing behaves
	fp := &fieldProps{
		path: []node{
			{tagName: "html", pseudoClasses: []string{"nth-child(1)"}},
			{tagName: "body", pseudoClasses: []string{"nth-child(3)"}},
			{tagName: "section", pseudoClasses: []string{"nth-child(4)"}},
		},
		iStrip: 0,
	}

	fp.stripNthChild(3) // sub=2, loop starts at len-2 = 1

	// nth-child(3) at index 1 >= minOcc => should be stripped and iStrip set to 1
	if fp.iStrip != 1 {
		t.Fatalf("iStrip = %d, want %d", fp.iStrip, 1)
	}
	// index 1 should be cleared
	if len(fp.path[1].pseudoClasses) != 0 {
		t.Fatalf("path[1].pseudoClasses = %v, want empty", fp.path[1].pseudoClasses)
	}
	// indices < iStrip (i.e., index 0) should also be cleared
	if len(fp.path[0].pseudoClasses) != 0 {
		t.Fatalf("path[0].pseudoClasses = %v, want empty", fp.path[0].pseudoClasses)
	}
	// index 2 was not visited (loop started at 1) so should remain
	if len(fp.path[2].pseudoClasses) == 0 {
		t.Fatalf("path[2].pseudoClasses was cleared unexpectedly")
	}
}

func TestSquash_MergeIdentical(t *testing.T) {
	makeNode := func(tag string, classes []string, pcls []string) node {
		return node{tagName: tag, classes: classes, pseudoClasses: pcls}
	}
	makeFP := func(p path, attr string, textIndex, count int, examples []string, iStrip int) *fieldProps {
		return &fieldProps{
			path:      p,
			attr:      attr,
			textIndex: textIndex,
			count:     count,
			examples:  append([]string{}, examples...),
			iStrip:    iStrip,
		}
	}

	fp1 := makeFP(path{
		makeNode("body", nil, nil),
		makeNode("div", []string{"container"}, nil),
	}, "", 0, 1, []string{"one"}, 0)

	fp2 := makeFP(path{
		makeNode("body", nil, nil),
		makeNode("div", []string{"container"}, nil),
	}, "", 0, 1, []string{"two"}, 0)

	fm := &fieldManager{fp1, fp2}
	expected := fieldManager{
		makeFP(path{
			makeNode("body", nil, nil),
			makeNode("div", []string{"container"}, nil),
		}, "", 0, 2, []string{"one", "two"}, 0),
	}

	fm.squash(1)

	if !fm.equals(&expected) {
		t.Fatalf("squash did not merge identical entries.\nGot: \n%s\nWant: \n%s", fm.string(), expected.string())
	}
}

func TestSquash_MergeOverlappingClassesBeforeIStrip(t *testing.T) {
	makeNode := func(tag string, classes []string, pcls []string) node {
		return node{tagName: tag, classes: classes, pseudoClasses: pcls}
	}
	makeFP := func(p path, attr string, textIndex, count int, examples []string, iStrip int) *fieldProps {
		return &fieldProps{
			path:      p,
			attr:      attr,
			textIndex: textIndex,
			count:     count,
			examples:  append([]string{}, examples...),
			iStrip:    iStrip,
		}
	}

	// two entries with overlapping classes at index 1; fp.iStrip = 1 means index 1 is "not past iStrip"
	// so partial intersection should be accepted and kept in result.
	fpA := makeFP(path{
		makeNode("body", nil, nil),
		makeNode("div", []string{"a", "b"}, nil),
	}, "", 0, 1, []string{"A"}, 1)

	fpB := makeFP(path{
		makeNode("body", nil, nil),
		makeNode("div", []string{"b", "c"}, nil),
	}, "", 0, 1, []string{"B"}, 1)

	fm := fieldManager{fpA, fpB}
	expected := fieldManager{
		makeFP(path{
			makeNode("body", nil, nil),
			makeNode("div", []string{"b"}, nil),
		}, "", 0, 2, []string{"A", "B"}, 1),
	}

	(&fm).squash(1)

	if !fm.equals(&expected) {
		t.Fatalf("squash did not merge overlapping classes before iStrip.\nGot: \n%s\nWant: \n%s", fm.string(), expected.string())
	}
}

func TestFilter_MinCountAndTruncation(t *testing.T) {
	fm := &fieldManager{
		&fieldProps{
			path:     nil,
			attr:     "",
			count:    1,
			examples: []string{"a", "b"},
		},
		&fieldProps{
			path:     nil,
			attr:     "",
			count:    2,
			examples: []string{"one", "two", "three"},
		},
		&fieldProps{
			path:     nil,
			attr:     "",
			count:    3,
			examples: []string{"same", "same", "same"},
		},
	}

	fm.filter(2, false)

	if len(*fm) != 2 {
		t.Fatalf("expected 2 entries after filter, got %d", len(*fm))
	}

	// first surviving entry should be the original second element
	fp0 := (*fm)[0]
	if fp0.count != 2 {
		t.Fatalf("fp0.count = %d, want %d", fp0.count, 2)
	}
	if !slices.Equal(fp0.examples, []string{"three", "two"}) {
		t.Fatalf("fp0.examples = %v, want %v", fp0.examples, []string{"three", "two"})
	}

	// second surviving entry should be the original third element
	fp1 := (*fm)[1]
	if fp1.count != 3 {
		t.Fatalf("fp1.count = %d, want %d", fp1.count, 3)
	}
	if !slices.Equal(fp1.examples, []string{"same", "same"}) {
		t.Fatalf("fp1.examples = %v, want %v", fp1.examples, []string{"same", "same"})
	}
}

func TestFilter_RemoveStaticFieldsTrue(t *testing.T) {
	fm := &fieldManager{
		&fieldProps{
			path:     nil,
			attr:     "",
			count:    2,
			examples: []string{"x", "x"},
		},
		&fieldProps{
			path:     nil,
			attr:     "",
			count:    2,
			examples: []string{"y", "z"},
		},
		&fieldProps{
			path:     nil,
			attr:     "",
			count:    1,
			examples: []string{"ignored"},
		},
	}

	fm.filter(2, true)

	if len(*fm) != 1 {
		t.Fatalf("expected 1 entry after filter, got %d", len(*fm))
	}

	fp := (*fm)[0]
	if fp.count != 2 {
		t.Fatalf("fp.count = %d, want %d", fp.count, 2)
	}
	// examples reversed and truncated to minCount=2 => ["z","y"]
	if !slices.Equal(fp.examples, []string{"z", "y"}) {
		t.Fatalf("fp.examples = %v, want %v", fp.examples, []string{"z", "y"})
	}
}

func TestFilter_ExamplesTruncationOrder(t *testing.T) {
	fp := &fieldProps{
		path:     nil,
		attr:     "",
		count:    3,
		examples: []string{"first", "second", "third"},
	}
	fm := &fieldManager{fp}

	fm.filter(2, false)

	if len(*fm) != 1 {
		t.Fatalf("expected 1 entry after filter, got %d", len(*fm))
	}
	got := (*fm)[0].examples
	if !slices.Equal(got, []string{"third", "second"}) {
		t.Fatalf("examples = %v, want %v", got, []string{"third", "second"})
	}
}
