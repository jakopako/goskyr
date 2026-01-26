package generate

const (
	LABLER_TYPE_LOCAL_ML   = "local-ml"
	LABLER_TYPE_REMOTE_LLM = "remote-llm"
)

type LablerType string

type LablerConfig struct {
	LablerType LablerType
	// Local ML labler config
	ModelName string
	WordsDir  string
}
