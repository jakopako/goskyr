// Package output provides the interface and configuration and implementation for writers
package output

import (
	"fmt"

	"github.com/jakopako/goskyr/internal/types"
)

// Writer defines the interface for all writers that are responsible
// for writing the scraped data to a specific output.
type Writer interface {
	// If a writer encounters a fatal error it should call log.Fatalf
	// to prevent the crawler from uselessly continuing to run.
	// Should Write return an error instead?
	Write(itemChan <-chan map[string]any)
	// WriteStatus writes the status to an output
	WriteStatus(statusChan <-chan types.ScraperStatus)
}

// WriterConfig defines the necessary paramters to make a new writer
// which is responsible for writing the scraped data to a specific output
// eg. stdout.
type WriterConfig struct {
	Type        WriterType `yaml:"type"`
	Uri         string     `yaml:"uri"`
	User        string     `yaml:"user" env:"WRITER_USER"`         // we want to be able to pass credentials via env vars
	Password    string     `yaml:"password" env:"WRITER_PASSWORD"` // we want to be able to pass credentials via env vars
	FileDir     string     `yaml:"filedir"`
	DryRun      bool       `yaml:"dryrun"`
	UriDryRun   string     `yaml:"uri_dryrun"`
	UriStatus   string     `yaml:"uri_status"`
	WriteStatus bool       `yaml:"write_status"`
	BatchSize   int        `yaml:"batch_size,omitempty"`
}

// WriterType encapsulates the type of a writer
// See below constants for possible types
type WriterType string

const (
	STDOUT_WRITER_TYPE WriterType = "stdout"
	FILE_WRITER_TYPE   WriterType = "file"
	API_WRITER_TYPE    WriterType = "api"
)

// NewWriter returns a new writer depending on the writer type
func NewWriter(wc *WriterConfig) (Writer, error) {
	switch wc.Type {
	case STDOUT_WRITER_TYPE:
		return NewStdoutWriter(wc), nil
	case FILE_WRITER_TYPE:
		return NewFileWriter(wc)
	case API_WRITER_TYPE:
		return NewAPIWriter(wc)
	default:
		return nil, fmt.Errorf("writer of type '%s' not implemented", wc.Type)
	}
}
