package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/jakopako/goskyr/scraper"
)

// WriteItemsToAPI writes the scraped events to an API defined through
// env vars. This method is not really useful because it is tailored to
// one specific API. Might change in the future.
func WriteItemsToAPI(wg *sync.WaitGroup, c scraper.Scraper) {
	// This function is not yet documented in the README because it might soon change and the entire result / output handling
	// might be refactored / improved.
	log.Printf("crawling %s\n", c.Name)
	defer wg.Done()
	apiURL := os.Getenv("EVENT_API")
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	events, err := c.GetItems()

	if err != nil {
		log.Printf("%s ERROR: %s", c.Name, err)
		return
	}

	if len(events) == 0 {
		log.Printf("location %s has no events. Skipping.", c.Name)
		return
	}
	log.Printf("fetched %d %s events\n", len(events), c.Name)

	// delete events of this scraper from first date on

	firstDate := events[0]["date"].(time.Time).UTC().Format("2006-01-02 15:04")
	deleteURL := fmt.Sprintf("%s?location=%s&datetime=%s", apiURL, url.QueryEscape(c.Name), url.QueryEscape(firstDate))
	req, _ := http.NewRequest("DELETE", deleteURL, nil)
	req.SetBasicAuth(os.Getenv("API_USER"), os.Getenv("API_PASSWORD"))
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatalf("Something went wrong while deleting events. Status Code: %d\nUrl: %s Response: %s", resp.StatusCode, deleteURL, body)
	}

	// add new events
	for _, event := range events {
		concertJSON, err := json.Marshal(event)
		if err != nil {
			log.Fatal(err)
		}
		req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(concertJSON))
		req.Header = map[string][]string{
			"Content-Type": {"application/json"},
		}
		req.SetBasicAuth(os.Getenv("API_USER"), os.Getenv("API_PASSWORD"))
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode != 201 {
			log.Fatalf("something went wrong while adding a new event. Status Code: %d", resp.StatusCode)

		}
	}
	log.Printf("done crawling and writing %s data to API.\n", c.Name)
}