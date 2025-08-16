// Package output provides the interface and configuration and implementation for writers
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
	Write(itemChan <-chan map[string]any)
	// WriteStatus writes the status to an output
	WriteStatus(statusChan <-chan types.ScraperStatus)
}

// WriterConfig defines the necessary paramters to make a new writer
// which is responsible for writing the scraped data to a specific output
// eg. stdout.
type WriterConfig struct {
	Type        WriterType `yaml:"type" env:"WRITER_TYPE"`
	Uri         string     `yaml:"uri" env:"WRITER_URI"`
	User        string     `yaml:"user" env:"WRITER_USER"`
	Password    string     `yaml:"password" env:"WRITER_PASSWORD"`
	FileDir     string     `yaml:"filedir" env:"WRITER_FILEDIR"`
	DryRun      bool       `yaml:"dryrun" env:"WRITER_DRYRUN"`
	UriDryRun   string     `yaml:"uri_dryrun" env:"WRITER_URI_DRYRUN"`
	UriStatus   string     `yaml:"uri_status" env:"WRITER_URI_STATUS"`
	WriteStatus bool       `yaml:"write_status" env:"WRITER_WRITE_STATUS"`
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
		return nil, fmt.Errorf("writer of type %s not implemented", wc.Type)
	}
}
