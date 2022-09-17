package output

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"sync"
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
func (fr *FileWriter) Write(items chan map[string]interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	f, err := os.Create(fr.writerConfig.FilePath)
	if err != nil {
		log.Fatalf("FileWriter ERROR while trying to open file: %v", err)
	}
	defer f.Close()
	allItems := []map[string]interface{}{}
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
		log.Printf("FileWriter ERROR while encoding items: %v", err)
		return
	}

	var indentBuffer bytes.Buffer
	if err := json.Indent(&indentBuffer, buffer.Bytes(), "", "  "); err != nil {
		log.Printf("FileWriter ERROR while indenting json: %v", err)
		return
	}
	_, err = f.Write(indentBuffer.Bytes())
	if err != nil {
		log.Printf("FileWriter ERROR while writing json to file: %v", err)
	}
	log.Printf("wrote %d items to file %s", len(allItems), fr.writerConfig.FilePath)
}
