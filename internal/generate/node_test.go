package generate

import "testing"

func TestNodeString(t *testing.T) {
	tests := []struct {
		name     string
		n        node
		expected string
	}{
		{
			name:     "tag only",
			n:        node{tagName: "div"},
			expected: "div",
		},
		{
			name:     "classes preserved order",
			n:        node{tagName: "a", classes: []string{"btn", "active"}},
			expected: "a.btn.active",
		},
		{
			name: "escape special characters in class",
			n: node{
				tagName: "div",
				classes: []string{"a:b>[c]/d!%'"},
			},
			// ":" -> "\:", ">" -> "\>", "[" -> "\[", "]" -> "\]",
			// "/" -> "\/", "!" -> "\!", "%" -> "\%"
			expected: `div.a\:b\>\[c\]\/d\!\%\'`,
		},
		{
			name: "class starting with digit gets special escape (includes trailing space)",
			n: node{
				tagName: "button",
				classes: []string{"primary", "1st"},
			},
			// "1st" -> "\3st " (note the trailing space from the implementation)
			expected: `button.primary.\31 st`,
		},
		{
			name: "pseudo classes appended with colons",
			n: node{
				tagName:       "span",
				classes:       []string{"label"},
				pseudoClasses: []string{"hover", "focus"},
			},
			expected: "span.label:hover:focus",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.n.string()
			if got != tc.expected {
				t.Fatalf("unexpected string representation\n got: %q\nwant: %q", got, tc.expected)
			}
		})
	}
}

func TestNodeEquals(t *testing.T) {
	tests := []struct {
		name     string
		n1       node
		n2       node
		expected bool
	}{
		{
			name:     "same tag only",
			n1:       node{tagName: "div"},
			n2:       node{tagName: "div"},
			expected: true,
		},
		{
			name:     "different tag",
			n1:       node{tagName: "div"},
			n2:       node{tagName: "span"},
			expected: false,
		},
		{
			name:     "same classes same order",
			n1:       node{tagName: "a", classes: []string{"btn", "active"}},
			n2:       node{tagName: "a", classes: []string{"btn", "active"}},
			expected: true,
		},
		{
			name:     "same classes different order (order does not matter)",
			n1:       node{tagName: "a", classes: []string{"btn", "active"}},
			n2:       node{tagName: "a", classes: []string{"active", "btn"}},
			expected: true,
		},
		{
			name:     "different classes",
			n1:       node{tagName: "a", classes: []string{"btn"}},
			n2:       node{tagName: "a", classes: []string{"link"}},
			expected: false,
		},
		{
			name:     "same pseudo classes same order",
			n1:       node{tagName: "span", pseudoClasses: []string{"hover", "focus"}},
			n2:       node{tagName: "span", pseudoClasses: []string{"hover", "focus"}},
			expected: true,
		},
		{
			name:     "same pseudo classes different order (order does not matter)",
			n1:       node{tagName: "span", pseudoClasses: []string{"hover", "focus"}},
			n2:       node{tagName: "span", pseudoClasses: []string{"focus", "hover"}},
			expected: true,
		},
		{
			name:     "classes and pseudo classes both equal",
			n1:       node{tagName: "button", classes: []string{"primary"}, pseudoClasses: []string{"active"}},
			n2:       node{tagName: "button", classes: []string{"primary"}, pseudoClasses: []string{"active"}},
			expected: true,
		},
		{
			name:     "one has extra pseudo class",
			n1:       node{tagName: "button", classes: []string{"primary"}, pseudoClasses: []string{"active"}},
			n2:       node{tagName: "button", classes: []string{"primary"}, pseudoClasses: []string{"active", "focus"}},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.n1.equals(tc.n2)
			if got != tc.expected {
				t.Fatalf("equals() returned %v; want %v for n1=%+v n2=%+v", got, tc.expected, tc.n1, tc.n2)
			}
		})
	}
}

func TestPathString(t *testing.T) {
	tests := []struct {
		name     string
		p        path
		expected string
	}{
		{
			name:     "empty path",
			p:        path{},
			expected: "",
		},
		{
			name:     "single node",
			p:        path{node{tagName: "div"}},
			expected: "div",
		},
		{
			name: "multiple nodes joined with separator and class/pseudo handling",
			p: path{
				node{tagName: "a", classes: []string{"btn", "active"}},
				node{tagName: "span", classes: []string{"label"}, pseudoClasses: []string{"hover"}},
			},
			expected: "a.btn.active > span.label:hover",
		},
		{
			name: "escaping special chars and digit-starting class in path",
			p: path{
				node{tagName: "div", classes: []string{"a:b>[c]/d!%"}},
				node{tagName: "button", classes: []string{"primary", "1st"}},
			},
			expected: `div.a\:b\>\[c\]\/d\!\% > button.primary.\31 st`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.p.string()
			if got != tc.expected {
				t.Fatalf("unexpected path string\n got: %q\nwant: %q", got, tc.expected)
			}
		})
	}
}

func TestPathTrimPrefix(t *testing.T) {
	tests := []struct {
		name     string
		p        path
		min      int
		expected string
	}{
		{
			name: "trim to last node when minClasses=1",
			p: path{
				node{tagName: "a", classes: []string{"x"}},
				node{tagName: "b", classes: []string{"y", "z"}},
				node{tagName: "c", classes: []string{"w"}},
			},
			min:      1,
			expected: "c.w",
		},
		{
			name: "trim to last two nodes when minClasses=2",
			p: path{
				node{tagName: "a", classes: []string{"x"}},
				node{tagName: "b", classes: []string{"y", "z"}},
				node{tagName: "c", classes: []string{"w"}},
			},
			min:      2,
			expected: "b.y.z > c.w",
		},
		{
			name: "minClasses exactly total across last two nodes (3) returns last two",
			p: path{
				node{tagName: "a", classes: []string{"x"}},
				node{tagName: "b", classes: []string{"y", "z"}},
				node{tagName: "c", classes: []string{"w"}},
			},
			min:      3,
			expected: "b.y.z > c.w",
		},
		{
			name: "minClasses equals total (4) returns full path",
			p: path{
				node{tagName: "a", classes: []string{"x"}},
				node{tagName: "b", classes: []string{"y", "z"}},
				node{tagName: "c", classes: []string{"w"}},
			},
			min:      4,
			expected: "a.x > b.y.z > c.w",
		},
		{
			name: "minClasses greater than total returns original path",
			p: path{
				node{tagName: "a", classes: []string{"x"}},
				node{tagName: "b", classes: []string{"y", "z"}},
				node{tagName: "c", classes: []string{"w"}},
			},
			min:      5,
			expected: "a.x > b.y.z > c.w",
		},
		{
			name: "nodes with no classes - cannot reach min, return original",
			p: path{
				node{tagName: "div"},
				node{tagName: "span"},
			},
			min:      1,
			expected: "div > span",
		},
		{
			name:     "empty path stays empty",
			p:        path{},
			min:      1,
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.p.trimPrefix(tc.min).string()
			if got != tc.expected {
				t.Fatalf("trimPrefix(%d) returned %q; want %q for path=%+v", tc.min, got, tc.expected, tc.p)
			}
		})
	}
}
