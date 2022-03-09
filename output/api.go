package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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

func (f *APIWriter) Write(itemsList chan []map[string]interface{}) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	apiURL := f.writerConfig.Uri
	apiUser := f.writerConfig.User
	apiPassword := f.writerConfig.Password
	for items := range itemsList {
		if len(items) > 0 {
			// delete events of this scraper from first date on
			firstDate := items[0]["date"].(time.Time).UTC().Format("2006-01-02 15:04")
			deleteURL := fmt.Sprintf("%s?location=%s&datetime=%s", apiURL, url.QueryEscape(items[0]["location"].(string)), url.QueryEscape(firstDate))
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
			// add new events
			for _, item := range items {
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
					log.Fatalf("something went wrong while adding a new event. Status Code: %d", resp.StatusCode)
				}
				resp.Body.Close()
			}
			log.Printf("wrote %d %s events to api", len(items), items[0]["location"])
		}
	}
}
