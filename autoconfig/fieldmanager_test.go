package autoconfig

import (
	"testing"
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
			name: "multiple elements with attributes",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := newFieldManagerFromHtml(tt.html)

			if !fm.equals(tt.expected) {
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
