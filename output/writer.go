// Package output provides the interface and configuration for writers
package output

import (
	"fmt"

	"github.com/jakopako/goskyr/types"
)

// Writer defines the interface for all writers that are responsible
// for writing the scraped data to a specific output.
type Writer interface {
	// If a writer encounters a fatal error it should call log.Fatalf
	// to prevent the crawler from uselessly continuing to run.
	// Should Write return an error instead?
	Write(itemsList <-chan map[string]any)
	// WriteStatus writes the status to an output
	WriteStatus(scraperStatusC <-chan types.ScraperStatus)
}

// WriterConfig defines the necessary paramters to make a new writer
// which is responsible for writing the scraped data to a specific output
// eg. stdout.
type WriterConfig struct {
	Type      WriterType `yaml:"type" env:"WRITER_TYPE"`
	Uri       string     `yaml:"uri" env:"WRITER_URI"`
	User      string     `yaml:"user" env:"WRITER_USER"`
	Password  string     `yaml:"password" env:"WRITER_PASSWORD"`
	FilePath  string     `yaml:"filepath" env:"WRITER_FILEPATH"`
	DryRun    bool       `yaml:"dryrun" env:"WRITER_DRYRUN"`
	UriDryRun string     `yaml:"uriDryRun" env:"WRITER_URI_DRYRUN"`
	UriStatus string     `yaml:"uriStatus" env:"WRITER_URI_STATUS"`
}

type WriterType string

const (
	STDOUT_WRITER_TYPE WriterType = "stdout"
	FILE_WRITER_TYPE   WriterType = "file"
	API_WRITER_TYPE    WriterType = "api"
)

func NewWriter(wc *WriterConfig) (Writer, error) {
	switch wc.Type {
	case STDOUT_WRITER_TYPE:
		return NewStdoutWriter(wc), nil
	case FILE_WRITER_TYPE:
		return NewFileWriter(wc), nil
	case API_WRITER_TYPE:
		return NewAPIWriter(wc), nil
	default:
		return nil, fmt.Errorf("writer of type %s not implemented", wc.Type)
	}
}
