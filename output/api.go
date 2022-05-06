package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/jakopako/goskyr/config"
)

// The APIWriter is meant to write to a custom API and assumes many things.
// So currently, it is better not to use this APIWriter.
type APIWriter struct {
	writerConfig *config.WriterConfig
}

// NewAPIWriter returns a new APIWriter
func NewAPIWriter(wc *config.WriterConfig) *APIWriter {
	return &APIWriter{
		writerConfig: wc,
	}
}

func (f *APIWriter) Write(items chan map[string]interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	apiURL := f.writerConfig.Uri
	apiUser := f.writerConfig.User
	apiPassword := f.writerConfig.Password

	deletedSources := map[string]bool{}
	nrItems := 0

	// This code assumes that within one source, items are ordered
	// by date ascending.
	for item := range items {
		nrItems++
		currentSrc := item["sourceUrl"].(string)
		if _, found := deletedSources[currentSrc]; !found {
			deletedSources[currentSrc] = true
			// delete all events from the given source
			firstDate := item["date"].(time.Time).UTC().Format("2006-01-02 15:04")
			deleteURL := fmt.Sprintf("%s?sourceUrl=%s&datetime=%s", apiURL, url.QueryEscape(currentSrc), url.QueryEscape(firstDate))
			req, _ := http.NewRequest("DELETE", deleteURL, nil)
			req.SetBasicAuth(apiUser, apiPassword)
			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			if resp.StatusCode != 200 {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Fatal(err)
				}
				log.Fatalf("something went wrong while deleting events. Status Code: %d\nUrl: %s Response: %s", resp.StatusCode, deleteURL, body)
			}
			resp.Body.Close()
		}
		concertJSON, err := json.Marshal(item)
		if err != nil {
			log.Fatal(err)
		}
		req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(concertJSON))
		req.Header = map[string][]string{
			"Content-Type": {"application/json"},
		}
		req.SetBasicAuth(apiUser, apiPassword)
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode != 201 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			log.Fatalf("something went wrong while adding a new event. Status Code: %d Response: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	}
	log.Printf("wrote %d events from %d sources to the api", nrItems, len(deletedSources))
}
