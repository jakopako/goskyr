package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jakopako/goskyr/internal/ml"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
)

const (
	LABLER_TYPE_BASIC      = "basic"
	LABLER_TYPE_LOCAL_ML   = "local-ml"
	LABLER_TYPE_REMOTE_LLM = "remote-llm"
)

type lablerType string

// LablerConfig holds the configuration for the labler used to predict field names
type LablerConfig struct {
	LablerType lablerType `yaml:"type"`
	// Local ML labler config
	ModelName string `yaml:"model_name,omitempty"`
	WordsDir  string `yaml:"words_dir,omitempty"`
	// Remote LLM labler config
	ApiKey   string   `yaml:"api_key,omitempty"`
	LabelSet []string `yaml:"label_set,omitempty"`
}

// labler is an interface for labeling fields in a fieldManager
type labler interface {
	labelFields(fm fieldManager) error
}

// newLabler creates a new labler based on the provided LablerConfig
func newLabler(lc *LablerConfig) (labler, error) {
	switch lc.LablerType {
	case LABLER_TYPE_BASIC:
		return newBasicLabler(), nil
	case LABLER_TYPE_LOCAL_ML:
		return newLocalMLLabler(lc)
	case LABLER_TYPE_REMOTE_LLM:
		return newRemoteLLMLabler(lc)
	default:
		return nil, fmt.Errorf("labler of type %s not implemented", lc.LablerType)
	}
}

// basicLabler is a simple labler that assigns generic names to fields
type basicLabler struct {
}

func newBasicLabler() *basicLabler {
	return &basicLabler{}
}

func (b *basicLabler) labelFields(fm fieldManager) error {
	for i, e := range fm {
		e.name = fmt.Sprintf("field-%d", i)
	}
	return nil
}

// localMLLabler uses a local ML model to predict field names
type localMLLabler struct {
	mlLabler *ml.Labler
}

func newLocalMLLabler(lc *LablerConfig) (*localMLLabler, error) {
	ll, err := ml.LoadLabler(lc.ModelName, lc.WordsDir)
	if err != nil {
		return nil, err
	}

	return &localMLLabler{
		mlLabler: ll,
	}, nil
}

func (l *localMLLabler) labelFields(fm fieldManager) error {
	for _, e := range fm {
		exampleStrs := []string{}
		for _, ex := range e.examples {
			exampleStrs = append(exampleStrs, ex.example)
		}
		pred, err := l.mlLabler.PredictLabel(exampleStrs...)
		if err != nil {
			return err
		}
		e.name = pred // TODO: if label has occured already, add index (eg text-1, text-2...)
	}
	return nil
}

// remoteLLMLabler uses a remote LLM service to predict field names
type remoteLLMLabler struct {
	llm      llms.Model
	labelSet []string
}

func newRemoteLLMLabler(lc *LablerConfig) (*remoteLLMLabler, error) {
	gai, err := googleai.New(context.Background(), googleai.WithAPIKey(lc.ApiKey), googleai.WithDefaultModel("gemini-2.5-flash"))
	if err != nil {
		return nil, err
	}

	return &remoteLLMLabler{
		llm:      gai,
		labelSet: lc.LabelSet,
	}, nil
}

func (r *remoteLLMLabler) labelFields(fm fieldManager) error {
	promptTemplate := `Given the following examples of field values extracted from a webpage, provide a label for each field.
The labels should always be one of the following: %s.
If a field's values do not match any of the labels, label it as "other".

Here are the field examples:

%s

Provide your answer as a plain JSON string where the keys are "field-0", "field-1", etc., and the values are the predicted labels.
Just return the JSON and nothing else. Don't wrap the JSON in any quotes or code blocks.`

	examplesStrs := []string{}
	for i, e := range fm {
		exStr := fmt.Sprintf("field-%d: [\"%s\"]", i, strings.Join(getExamplesStrings(e.examples), "\", \""))
		examplesStrs = append(examplesStrs, exStr)
	}

	prompt := fmt.Sprintf(promptTemplate, strings.Join(r.labelSet, ", "), strings.Join(examplesStrs, "\n"))
	fmt.Println(prompt)

	answer, err := llms.GenerateFromSinglePrompt(context.Background(), r.llm, prompt)
	if err != nil {
		return err
	}

	// parse json answer as map[string]string
	mapping := map[string]string{}
	err = json.Unmarshal([]byte(answer), &mapping)
	if err != nil {
		return fmt.Errorf("error parsing LLM response: %v", err)
	}

	// assign labels to fields
	for i, e := range fm {
		fieldKey := fmt.Sprintf("field-%d", i)
		if label, ok := mapping[fieldKey]; ok {
			e.name = label
		} else {
			e.name = "other"
		}
	}
	fmt.Println(answer)
	return nil
}

func getExamplesStrings(examples []fieldExample) []string {
	result := []string{}
	for _, ex := range examples {
		result = append(result, ex.exampleString())
	}
	return result
}
