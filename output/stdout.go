package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jakopako/goskyr/types"
)

// StdoutWriter represents a writer that writes to stdout
type StdoutWriter struct {
	logger *slog.Logger
}

// NewStdoutWriter returns a new StdoutWriter
func NewStdoutWriter(wc *WriterConfig) *StdoutWriter {
	return &StdoutWriter{
		logger: slog.With(slog.String("writer", string(STDOUT_WRITER_TYPE))),
	}
}

func (w *StdoutWriter) Write(itemChan <-chan map[string]any) {
	for item := range itemChan {
		// We cannot use the following line of code because it automatically replaces certain html characters
		// with the corresponding Unicode replacement rune.
		// itemsJson, err := json.MarshalIndent(items, "", "  ")
		// if err != nil {
		// 	log.Print(err.Error())
		// }
		// See
		// https://stackoverflow.com/questions/28595664/how-to-stop-json-marshal-from-escaping-and
		// https://developpaper.com/the-solution-of-escaping-special-html-characters-in-golang-json-marshal/
		buffer := &bytes.Buffer{}
		encoder := json.NewEncoder(buffer)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(item); err != nil {
			w.logger.Error(fmt.Sprintf("error while writing item %v: %v", item, err))
			continue
		}

		var indentBuffer bytes.Buffer
		if err := json.Indent(&indentBuffer, buffer.Bytes(), "", "  "); err != nil {
			w.logger.Error(fmt.Sprintf("error while writing item %v: %v", item, err))
			continue
		}
		fmt.Print(indentBuffer.String())
	}
}

func (w *StdoutWriter) WriteStatus(statusChan <-chan types.ScraperStatus) {
	for status := range statusChan {
		statusJson, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			w.logger.Error(fmt.Sprintf("error while marshalling status json: %v", err))
		}
		w.logger.Info(fmt.Sprintf("printing scraper status for scraper '%s'", status.ScraperName))
		fmt.Println(string(statusJson))
	}
}
