package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/jakopako/goskyr/types"
)

type FileWriter struct {
	writerConfig *WriterConfig
}

// NewFileWriter returns a new FileWriter
func NewFileWriter(wc *WriterConfig) *FileWriter {
	return &FileWriter{
		writerConfig: wc,
	}
}
func (fr *FileWriter) Write(items chan map[string]any) {
	logger := slog.With(slog.String("writer", FILE_WRITER_TYPE))
	f, err := os.Create(fr.writerConfig.FilePath)
	if err != nil {
		logger.Error(fmt.Sprintf("error while trying to open file: %v", err))
		os.Exit(1)
	}
	defer f.Close()
	allItems := []map[string]any{}
	for item := range items {
		allItems = append(allItems, item)
	}

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
	if err := encoder.Encode(allItems); err != nil {
		logger.Error(fmt.Sprintf("error while encoding items: %v", err))
		return
	}

	var indentBuffer bytes.Buffer
	if err := json.Indent(&indentBuffer, buffer.Bytes(), "", "  "); err != nil {
		logger.Error(fmt.Sprintf("error while indenting json: %v", err))
		return
	}
	if _, err = f.Write(indentBuffer.Bytes()); err != nil {
		logger.Error(fmt.Sprintf("error while writing json to file: %v", err))
	} else {
		logger.Info(fmt.Sprintf("wrote %d items to file %s", len(allItems), fr.writerConfig.FilePath))
	}
}

func (fr *FileWriter) WriteStatus(scraperStatus types.ScraperStatus) {
	// TODO implement WriteStatus for FileWriter
}
