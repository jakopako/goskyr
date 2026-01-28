package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/jakopako/goskyr/internal/types"
)

const (
	itemsFilename  = "items.json"
	statusFilename = "status.json"
)

// FileWriter represents a writer that writes to a file
type FileWriter struct {
	*WriterConfig
	logger *slog.Logger
}

// NewFileWriter returns a new FileWriter
func NewFileWriter(wc *WriterConfig) (*FileWriter, error) {
	if wc.FileDir == "" {
		return nil, errors.New("filedir needs to be specified for the FileWriter")
	}

	if err := os.MkdirAll(wc.FileDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", wc.FileDir, err)
	}

	return &FileWriter{
		WriterConfig: wc,
		logger:       slog.With(slog.String("writer", string(FILE_WRITER_TYPE))),
	}, nil
}

func (w *FileWriter) Write(itemChan <-chan map[string]any) {
	filepath := path.Join(w.FileDir, itemsFilename)
	f, err := os.Create(filepath)
	if err != nil {
		w.logger.Error(fmt.Sprintf("error while trying to open file: %v", err))
		os.Exit(1)
	}
	defer f.Close()
	allItems := []map[string]any{}
	for item := range itemChan {
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
		w.logger.Error(fmt.Sprintf("error while encoding items: %v", err))
		return
	}

	var indentBuffer bytes.Buffer
	if err := json.Indent(&indentBuffer, buffer.Bytes(), "", "  "); err != nil {
		w.logger.Error(fmt.Sprintf("error while indenting json: %v", err))
		return
	}
	if _, err = f.Write(indentBuffer.Bytes()); err != nil {
		w.logger.Error(fmt.Sprintf("error while writing items json to file: %v", err))
	} else {
		w.logger.Info(fmt.Sprintf("wrote %d items to file %s", len(allItems), filepath))
	}
}

func (w *FileWriter) WriteStatus(statusChan <-chan types.ScraperStatus) {
	filepath := path.Join(w.FileDir, statusFilename)
	f, err := os.Create(filepath)
	if err != nil {
		w.logger.Error(fmt.Sprintf("error while trying to open file: %v", err))
		os.Exit(1)
	}
	defer f.Close()
	allStatus := []types.ScraperStatus{}
	for status := range statusChan {
		allStatus = append(allStatus, status)
	}

	statusJson, err := json.MarshalIndent(allStatus, "", "  ")
	if err != nil {
		w.logger.Error(fmt.Sprintf("error while marshalling status json: %v", err))
	}

	if _, err = f.Write(statusJson); err != nil {
		w.logger.Error(fmt.Sprintf("error while writing status json to file: %v", err))
	} else {
		w.logger.Info(fmt.Sprintf("wrote status to file %s", filepath))
	}
}
