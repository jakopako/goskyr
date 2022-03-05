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

	"github.com/jakopako/go-crawler/scraper"
)

func WriteEventsToAPI(wg *sync.WaitGroup, c scraper.Scraper) {
	// This function is not yet documented in the README because it might soon change and the entire result / output handling
	// might be refactored / improved.
	log.Printf("crawling %s\n", c.Name)
	defer wg.Done()
	apiUrl := os.Getenv("EVENT_API")
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	events, err := c.GetEvents()

	if err != nil {
		log.Printf("%s ERROR: %s", c.Name, err)
		return
	}

	if len(events) == 0 {
		log.Printf("location %s has no events. Skipping.", c.Name)
		return
	}
	log.Printf("fetched %d %s events\n", len(events), c.Name)

	// delete events of this crawler from first date on

	firstDate := events[0]["date"].(time.Time).UTC().Format("2006-01-02 15:04")
	deleteUrl := fmt.Sprintf("%s?location=%s&datetime=%s", apiUrl, url.QueryEscape(c.Name), url.QueryEscape(firstDate))
	req, _ := http.NewRequest("DELETE", deleteUrl, nil)
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
		log.Fatalf("Something went wrong while deleting events. Status Code: %d\nUrl: %s Response: %s", resp.StatusCode, deleteUrl, body)
	}

	// add new events
	for _, event := range events {
		concertJSON, err := json.Marshal(event)
		if err != nil {
			log.Fatal(err)
		}
		req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(concertJSON))
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
