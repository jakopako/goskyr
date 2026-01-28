package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/jakopako/goskyr/internal/types"
)

// APIWriter represents a writer that writes to a custom API.
// The corresponding API can be found here: https://github.com/jakopako/event-api
type APIWriter struct {
	*WriterConfig
	logger *slog.Logger
}

// NewAPIWriter returns a new APIWriter
func NewAPIWriter(wc *WriterConfig) (*APIWriter, error) {
	if wc.WriteStatus && wc.UriStatus == "" {
		return nil, errors.New("if write_status is true, uri_status needs to be set")
	}
	if wc.BatchSize == 0 {
		wc.BatchSize = 100 // default
	}
	return &APIWriter{
		WriterConfig: wc,
		logger:       slog.With(slog.String("writer", string(API_WRITER_TYPE))),
	}, nil
}

func (w *APIWriter) Write(itemChan <-chan map[string]any) {
	client := &http.Client{
		Timeout: time.Second * 60,
	}

	deletedSources := map[string]bool{}
	nrItemsWritten := 0
	batch := []map[string]any{}

	// This code assumes that within one source, items are ordered
	// by date ascending.
	for item := range itemChan {

		// delete all future items from the given source if not in dry run mode
		if !w.DryRun {
			currentSrcStr, ok := item["sourceUrl"].(string)
			if !ok {
				w.logger.Error(fmt.Sprintf("error while trying to cast the sourceUrl field of item %v to string. The sourceUrl field is mandatory if using the APIWriter.", item))
				continue
			}
			if _, found := deletedSources[currentSrcStr]; !found {
				deletedSources[currentSrcStr] = true
				// delete all items from the given source
				firstDate, ok := item["date"].(time.Time)
				if !ok {
					w.logger.Error(fmt.Sprintf("error while trying to cast the date field of item %v to time.Time. The date field is mandatory if using the APIWriter.", item))
					continue
				}
				firstDateUTCF := firstDate.UTC().Format("2006-01-02 15:04")
				deleteURL := fmt.Sprintf("%s?sourceUrl=%s&datetime=%s", w.Uri, url.QueryEscape(currentSrcStr), url.QueryEscape(firstDateUTCF))
				req, _ := http.NewRequest("DELETE", deleteURL, nil)
				req.SetBasicAuth(w.User, w.Password)
				resp, err := client.Do(req)
				// the following errors are considered fatal. If they occur we assume that there's something fundamentally wrong.
				if err != nil {
					w.logger.Error(fmt.Sprintf("error while deleting items from the api: %v\n", err))
					os.Exit(1)
				}
				if resp.StatusCode != 200 {
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						w.logger.Error(fmt.Sprintf("%v", err))
					} else {
						w.logger.Error(fmt.Sprintf("error while deleting items. Status Code: %d\nUrl: %s Response: %s\n", resp.StatusCode, deleteURL, body))
					}
					os.Exit(1)
				}
				resp.Body.Close()
			}
		}

		batch = append(batch, item)
		if len(batch) == w.BatchSize {
			nrItemsWritten += w.writeBatch(client, batch)
			batch = []map[string]any{}
		}
	}

	nrItemsWritten += w.writeBatch(client, batch)
	if !w.DryRun {
		w.logger.Info(fmt.Sprintf("wrote %d items from %d sources to the api", nrItemsWritten, len(deletedSources)))
	}
}

func (w *APIWriter) WriteStatus(statusChan <-chan types.ScraperStatus) {
	client := &http.Client{
		Timeout: time.Second * 60,
	}
	for status := range statusChan {
		statusJSON, err := json.Marshal(status)
		if err != nil {
			w.logger.Error(fmt.Sprintf("error while marshaling scraper status: %v", err))
			continue
		}
		req, _ := http.NewRequest("POST", w.UriStatus, bytes.NewBuffer(statusJSON))
		req.Header = map[string][]string{
			"Content-Type": {"application/json"},
		}
		req.SetBasicAuth(w.User, w.Password)
		resp, err := client.Do(req)
		if err != nil {
			w.logger.Error(fmt.Sprintf("error while sending post request for scraper status: %v", err))
			continue
		}
		if resp.StatusCode != 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				w.logger.Error(fmt.Sprintf("error while reading post request response: %v", err))
			} else {
				w.logger.Error(fmt.Sprintf("error while posting scraper status. Status Code: %d Response: %s", resp.StatusCode, body))
			}
			resp.Body.Close()
			continue
		}
		w.logger.Info(fmt.Sprintf("successfully posted scraper status for scraper '%s'", status.ScraperName))
		resp.Body.Close()
	}
}

func (w *APIWriter) writeBatch(client *http.Client, batch []map[string]any) int {
	if w.DryRun {
		result, err := validateBatch(client, batch, w.UriDryRun)
		if err != nil {
			w.logger.Error(fmt.Sprintf("error while validating batch: %v", err))
		} else {
			w.logger.Info("validation result")
			fmt.Println(result)
		}

		// in dry run mode we do not write anything to the api
		return 0
	} else {
		if err := w.persistBatch(client, batch, w.Uri, w.User, w.Password); err != nil {
			w.logger.Error(fmt.Sprintf("error while posting batch: %v", err))
			// not correct. With the latest api implementation it can happen that in one request some events are successfully written while others are not.
			return 0
		} else {
			return len(batch)
		}
	}

}

func (w *APIWriter) persistBatch(client *http.Client, batch []map[string]any, apiURL, apiUser, apiPassword string) error {
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
		w.logger.Debug(fmt.Sprintf("post request body %s", concertJSON))
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

func validateBatch(client *http.Client, batch []map[string]any, apiURL string) (string, error) {
	concertJSON, err := json.Marshal(batch)
	if err != nil {
		return "", err
	}
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(concertJSON))
	req.Header = map[string][]string{
		"Content-Type": {"application/json"},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error while sending post request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error while reading post request respone: %v", err)
	}

	// beautify the json response
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		return "", fmt.Errorf("error while indenting json: %v", err)
	}
	return prettyJSON.String(), nil
}
