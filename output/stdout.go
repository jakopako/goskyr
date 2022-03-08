package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/jakopako/goskyr/scraper"
)

// PrettyPrintItems pretty prints the scraped items
func PrettyPrintItems(wg *sync.WaitGroup, c scraper.Scraper) {
	defer wg.Done()
	events, err := c.GetItems()
	if err != nil {
		log.Printf("%s ERROR: %s", c.Name, err)
		return
	}

	// We cannot use the following line of code because it automatically replaces certain html characters
	// with the corresponding Unicode replacement rune.
	// eventsJson, err := json.MarshalIndent(events, "", "  ")
	// if err != nil {
	// 	log.Print(err.Error())
	// }
	// See
	// https://stackoverflow.com/questions/28595664/how-to-stop-json-marshal-from-escaping-and
	// https://developpaper.com/the-solution-of-escaping-special-html-characters-in-golang-json-marshal/
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(events)
	if err != nil {
		log.Printf("%s ERROR: %s", c.Name, err)
		return
	}

	var indentBuffer bytes.Buffer
	err = json.Indent(&indentBuffer, buffer.Bytes(), "", "  ")
	if err != nil {
		log.Printf("%s ERROR: %s", c.Name, err)
		return
	}
	fmt.Print(indentBuffer.String())
}

type StdoutWriter struct{}

func (s *StdoutWriter) Write(itemsList chan []map[string]interface{}) {
	for items := range itemsList {
		for _, item := range items {
			buffer := &bytes.Buffer{}
			encoder := json.NewEncoder(buffer)
			encoder.SetEscapeHTML(false)
			if err := encoder.Encode(item); err != nil {
				log.Printf("StdoutWriter ERROR while writing item %v: %v", item, err)
				continue
			}

			var indentBuffer bytes.Buffer
			if err := json.Indent(&indentBuffer, buffer.Bytes(), "", "  "); err != nil {
				log.Printf("StdoutWriter ERROR while writing item %v: %v", item, err)
				continue
			}
			fmt.Print(indentBuffer.String())
		}
	}
}
