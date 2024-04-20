package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
)

// The APIWriter is meant to write to a custom API and assumes many things.
// So currently, it is better not to use this APIWriter.
type APIWriter struct {
	writerConfig *WriterConfig
}

// NewAPIWriter returns a new APIWriter
func NewAPIWriter(wc *WriterConfig) *APIWriter {
	return &APIWriter{
		writerConfig: wc,
	}
}

func (f *APIWriter) Write(items chan map[string]interface{}) {
	logger := slog.With(slog.String("writer", API_WRITER_TYPE))
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	apiURL := f.writerConfig.Uri
	apiUser := f.writerConfig.User
	apiPassword := f.writerConfig.Password

	deletedSources := map[string]bool{}
	nrItemsWritten := 0
	batch := []map[string]interface{}{}

	// This code assumes that within one source, items are ordered
	// by date ascending.
	for item := range items {
		currentSrc := item["sourceUrl"].(string)
		if _, found := deletedSources[currentSrc]; !found {
			deletedSources[currentSrc] = true
			// delete all items from the given source
			firstDate, ok := item["date"].(time.Time)
			if !ok {
				logger.Error(fmt.Sprintf("error while trying to cast the date field of item %v to time.Time", item))
				continue
			}
			firstDateUTCF := firstDate.UTC().Format("2006-01-02 15:04")
			deleteURL := fmt.Sprintf("%s?sourceUrl=%s&datetime=%s", apiURL, url.QueryEscape(currentSrc), url.QueryEscape(firstDateUTCF))
			req, _ := http.NewRequest("DELETE", deleteURL, nil)
			req.SetBasicAuth(apiUser, apiPassword)
			resp, err := client.Do(req)
			if err != nil {
				logger.Error(fmt.Sprintf("error while deleting items from the api: %v\n", err))
				continue
			}
			if resp.StatusCode != 200 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Error(fmt.Sprintf("%v", err))
				}
				logger.Error(fmt.Sprintf("error while deleting items. Status Code: %d\nUrl: %s Response: %s\n", resp.StatusCode, deleteURL, body))
				os.Exit(1)
			}
			resp.Body.Close()
		}
		batch = append(batch, item)
		if len(batch) == 100 {
			if err := postBatch(client, batch, apiURL, apiUser, apiPassword); err != nil {
				fmt.Printf("%v\n", err)
			} else {
				nrItemsWritten = nrItemsWritten + 100
			}
			batch = []map[string]interface{}{}
		}
	}
	if err := postBatch(client, batch, apiURL, apiUser, apiPassword); err != nil {
		fmt.Printf("%v\n", err)
	} else {
		nrItemsWritten = nrItemsWritten + len(batch)
	}

	logger.Info(fmt.Sprintf("wrote %d items from %d sources to the api", nrItemsWritten, len(deletedSources)))
}

func postBatch(client *http.Client, batch []map[string]interface{}, apiURL, apiUser, apiPassword string) error {
	concertJSON, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(concertJSON))
	req.Header = map[string][]string{
		"Content-Type": {"application/json"},
	}
	req.SetBasicAuth(apiUser, apiPassword)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error while sending post request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error while reading post request respones: %v", err)
		} else {
			return fmt.Errorf("error while adding new events. Status Code: %d Response: %s", resp.StatusCode, body)
		}
	}
	return nil
}
