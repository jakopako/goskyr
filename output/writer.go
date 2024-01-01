package output

type Writer interface {
	// if a writer encounters a fatal error it should call log.Fatalf
	// to prevent the crawler from uselessly continuing to run.
	Write(itemsList chan map[string]interface{})
}

// .WriterConfig defines the necessary paramters to make a new writer
// which is responsible for writing the scraped data to a specific output
// eg. stdout.
type WriterConfig struct {
	Type     string `yaml:"type" env:"WRITER_TYPE"`
	Uri      string `yaml:"uri" env:"WRITER_URI"`
	User     string `yaml:"user" env:"WRITER_USER"`
	Password string `yaml:"password" env:"WRITER_PASSWORD"`
	FilePath string `yaml:"filepath" env:"WRITER_FILEPATH"`
}

const (
	STDOUT_WRITER_TYPE = "stdout"
	FILE_WRITER_TYPE   = "file"
	API_WRITER_TYPE    = "api"
)
