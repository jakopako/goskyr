package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jakopako/goskyr/types"
)

type StdoutWriter struct{}

func NewStdoutWriter(wc *WriterConfig) *StdoutWriter {
	return &StdoutWriter{}
}

func (w *StdoutWriter) Write(itemChan <-chan map[string]any) {
	logger := slog.With(slog.String("writer", string(STDOUT_WRITER_TYPE)))
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
			logger.Error(fmt.Sprintf("error while writing item %v: %v", item, err))
			continue
		}

		var indentBuffer bytes.Buffer
		if err := json.Indent(&indentBuffer, buffer.Bytes(), "", "  "); err != nil {
			logger.Error(fmt.Sprintf("error while writing item %v: %v", item, err))
			continue
		}
		fmt.Print(indentBuffer.String())
	}
}

func (w *StdoutWriter) WriteStatus(statusChan <-chan types.ScraperStatus) {
	// TODO implement WriteStatus for StdoutWriter
}
