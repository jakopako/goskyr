package generate

import (
	"fmt"
	"slices"
	"testing"
)

func TestGetExamplesStrings(t *testing.T) {
	tests := []struct {
		name             string
		examples         []fieldExample
		maxNrExamples    int
		maxExampleStrLen int
		expectedLen      int
		expectedContains []string
	}{
		{
			name:             "empty examples",
			examples:         []fieldExample{},
			maxNrExamples:    5,
			maxExampleStrLen: 100,
			expectedLen:      0,
		},
		{
			name: "all examples within limits",
			examples: []fieldExample{
				{example: "short"},
				{example: "text"},
			},
			maxNrExamples:    5,
			maxExampleStrLen: 100,
			expectedLen:      2,
			expectedContains: []string{"short", "text"},
		},
		{
			name: "exceed max number of examples",
			examples: []fieldExample{
				{example: "first"},
				{example: "second"},
				{example: "third"},
			},
			maxNrExamples:    2,
			maxExampleStrLen: 100,
			expectedLen:      2,
		},
		{
			name: "truncate long example strings",
			examples: []fieldExample{
				{example: "this is a very long string that exceeds the limit"},
			},
			maxNrExamples:    5,
			maxExampleStrLen: 10,
			expectedLen:      1,
		},
		{
			name: "mixed: some long, some short, limited count",
			examples: []fieldExample{
				{example: "short"},
				{example: "this is a very long example string"},
				{example: "another"},
				{example: "one more very long example string here"},
			},
			maxNrExamples:    2,
			maxExampleStrLen: 15,
			expectedLen:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getExamplesStrings(tt.examples, tt.maxNrExamples, tt.maxExampleStrLen)

			if len(result) != tt.expectedLen {
				t.Errorf("expected length %d, got %d", tt.expectedLen, len(result))
			}

			for _, expected := range tt.expectedContains {
				found := slices.Contains(result, expected)
				if !found {
					t.Errorf("expected to contain %q, but not found in result", expected)
				}
			}

			for _, r := range result {
				if len(r) > tt.maxExampleStrLen {
					t.Errorf("result string %q exceeds max length %d", r, tt.maxExampleStrLen)
				}
			}
		})
	}
}

func TestBasicLablerLabelFields(t *testing.T) {
	tests := []struct {
		name     string
		fm       fieldManager
		expected []string
	}{
		{
			name:     "empty field manager",
			fm:       fieldManager{},
			expected: []string{},
		},
		{
			name: "single field",
			fm: fieldManager{
				{name: "", examples: []fieldExample{}},
			},
			expected: []string{"field-0"},
		},
		{
			name: "multiple fields",
			fm: fieldManager{
				{name: "", examples: []fieldExample{}},
				{name: "", examples: []fieldExample{}},
				{name: "", examples: []fieldExample{}},
			},
			expected: []string{"field-0", "field-1", "field-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bl := newBasicLabler()
			err := bl.labelFields(tt.fm)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(tt.fm) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d", len(tt.expected), len(tt.fm))
			}

			for i, e := range tt.fm {
				if e.name != tt.expected[i] {
					t.Errorf("field %d: expected name %q, got %q", i, tt.expected[i], e.name)
				}
			}
		})
	}
}

func TestNewLabler(t *testing.T) {
	tests := []struct {
		name        string
		config      *LablerConfig
		expectError bool
		expectType  string
	}{
		{
			name: "basic labler",
			config: &LablerConfig{
				LablerType: lablerType(LABLER_TYPE_BASIC),
			},
			expectError: false,
			expectType:  "*generate.basicLabler",
		},
		{
			name: "local ml labler - invalid model",
			config: &LablerConfig{
				LablerType: lablerType(LABLER_TYPE_LOCAL_ML),
				ModelName:  "nonexistent-model",
				WordsDir:   "",
			},
			expectError: true,
		},
		{
			name: "remote llm labler - unsupported provider",
			config: &LablerConfig{
				LablerType:  lablerType(LABLER_TYPE_REMOTE_LLM),
				LLMProvider: "unsupported-provider",
			},
			expectError: true,
		},
		{
			name: "unknown labler type",
			config: &LablerConfig{
				LablerType: "unknown-type",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := newLabler(tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.expectType != "" {
				actualType := fmt.Sprintf("%T", l)
				if actualType != tt.expectType {
					t.Errorf("expected type %q, got %q", tt.expectType, actualType)
				}
			}
		})
	}
}
