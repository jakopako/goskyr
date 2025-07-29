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

	"github.com/jakopako/goskyr/types"
)

// The APIWriter is meant to write to a custom API and assumes many things.
// So currently, it is better not to use this APIWriter.
type APIWriter struct {
	writerConfig *WriterConfig
	logger       *slog.Logger
}

// NewAPIWriter returns a new APIWriter
func NewAPIWriter(wc *WriterConfig) *APIWriter {
	return &APIWriter{
		writerConfig: wc,
		logger:       slog.With(slog.String("writer", string(API_WRITER_TYPE))),
	}
}

func (f *APIWriter) Write(items <-chan map[string]any) {
	client := &http.Client{
		Timeout: time.Second * 60,
	}

	deletedSources := map[string]bool{}
	nrItemsWritten := 0
	batch := []map[string]any{}

	// This code assumes that within one source, items are ordered
	// by date ascending.
	for item := range items {

		// delete all future items from the given source if not in dry run mode
		if !f.writerConfig.DryRun {
			currentSrc := item["sourceUrl"].(string)
			if _, found := deletedSources[currentSrc]; !found {
				deletedSources[currentSrc] = true
				// delete all items from the given source
				firstDate, ok := item["date"].(time.Time)
				if !ok {
					f.logger.Error(fmt.Sprintf("error while trying to cast the date field of item %v to time.Time", item))
					continue
				}
				firstDateUTCF := firstDate.UTC().Format("2006-01-02 15:04")
				deleteURL := fmt.Sprintf("%s?sourceUrl=%s&datetime=%s", f.writerConfig.Uri, url.QueryEscape(currentSrc), url.QueryEscape(firstDateUTCF))
				req, _ := http.NewRequest("DELETE", deleteURL, nil)
				req.SetBasicAuth(f.writerConfig.User, f.writerConfig.Password)
				resp, err := client.Do(req)
				// the following errors are considered fatal. If they occur we assume that there's something fundamentally wrong.
				if err != nil {
					f.logger.Error(fmt.Sprintf("error while deleting items from the api: %v\n", err))
					os.Exit(1)
				}
				if resp.StatusCode != 200 {
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						f.logger.Error(fmt.Sprintf("%v", err))
					} else {
						f.logger.Error(fmt.Sprintf("error while deleting items. Status Code: %d\nUrl: %s Response: %s\n", resp.StatusCode, deleteURL, body))
					}
					os.Exit(1)
				}
				resp.Body.Close()
			}
		}

		batch = append(batch, item)
		if len(batch) == 100 {
			nrItemsWritten += f.writeBatch(client, batch)
			batch = []map[string]any{}
		}
	}

	nrItemsWritten += f.writeBatch(client, batch)
	if !f.writerConfig.DryRun {
		f.logger.Info(fmt.Sprintf("wrote %d items from %d sources to the api", nrItemsWritten, len(deletedSources)))
	}
}

func (f *APIWriter) WriteStatus(scraperStatusC <-chan types.ScraperStatus) {
	if f.writerConfig.DryRun {
		f.logger.Info("dry run mode enabled, not writing scraper status")
		return
	}
	client := &http.Client{
		Timeout: time.Second * 60,
	}
	for status := range scraperStatusC {
		statusJSON, err := json.Marshal(status)
		if err != nil {
			f.logger.Error(fmt.Sprintf("error while marshaling scraper status: %v", err))
			continue
		}
		req, _ := http.NewRequest("POST", f.writerConfig.UriStatus, bytes.NewBuffer(statusJSON))
		req.Header = map[string][]string{
			"Content-Type": {"application/json"},
		}
		req.SetBasicAuth(f.writerConfig.User, f.writerConfig.Password)
		resp, err := client.Do(req)
		if err != nil {
			f.logger.Error(fmt.Sprintf("error while sending post request for scraper status: %v", err))
			continue
		}
		if resp.StatusCode != 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				f.logger.Error(fmt.Sprintf("error while reading post request response: %v", err))
			} else {
				f.logger.Error(fmt.Sprintf("error while posting scraper status. Status Code: %d Response: %s", resp.StatusCode, body))
			}
			resp.Body.Close()
			continue
		}
		f.logger.Info(fmt.Sprintf("successfully posted scraper status for scraper %s", status.ScraperName))
		resp.Body.Close()
	}
}

func (f *APIWriter) writeBatch(client *http.Client, batch []map[string]any) int {
	if f.writerConfig.DryRun {
		result, err := validateBatch(client, batch, f.writerConfig.UriDryRun)
		if err != nil {
			f.logger.Error(fmt.Sprintf("error while validating batch: %v", err))
		} else {
			f.logger.Info("validation result")
			fmt.Println(result)
		}

		// in dry run mode we do not write anything to the api
		return 0
	} else {
		if err := persistBatch(client, batch, f.writerConfig.Uri, f.writerConfig.User, f.writerConfig.Password); err != nil {
			f.logger.Error(fmt.Sprintf("error while posting batch: %v", err))
			// not correct. With the latest api implementation it can happen that in one request some events are successfully written while others are not.
			return 0
		} else {
			return len(batch)
		}
	}

}

func persistBatch(client *http.Client, batch []map[string]any, apiURL, apiUser, apiPassword string) error {
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
