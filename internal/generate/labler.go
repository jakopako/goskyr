package generate

const (
	LABLER_TYPE_LOCAL_ML   = "local-ml"
	LABLER_TYPE_REMOTE_LLM = "remote-llm"
)

type LablerType string

type LablerConfig struct {
	LablerType LablerType `yaml:"type"`
	// Local ML labler config
	ModelName string `yaml:"model_name,omitempty"`
	WordsDir  string `yaml:"words_dir,omitempty"`
}
