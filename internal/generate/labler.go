package generate

import (
	"context"
	"fmt"

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
	ApiKey string `yaml:"api_key,omitempty"`
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
	llm llms.Model
}

func newRemoteLLMLabler(lc *LablerConfig) (*remoteLLMLabler, error) {
	gai, err := googleai.New(context.Background(), googleai.WithAPIKey(lc.ApiKey), googleai.WithDefaultModel("gemini-2.5-flash"))
	if err != nil {
		return nil, err
	}

	return &remoteLLMLabler{
		llm: gai,
	}, nil
}

func (r *remoteLLMLabler) labelFields(fm fieldManager) error {
	prompt := "Who was the second person to walk on the moon?"
	answer, err := llms.GenerateFromSinglePrompt(context.Background(), r.llm, prompt)
	if err != nil {
		return err
	}

	fmt.Println(answer)
	return nil
}
