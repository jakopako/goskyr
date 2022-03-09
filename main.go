package main

import (
	"flag"
	"log"
	"sync"

	"github.com/jakopako/goskyr/config"
	"github.com/jakopako/goskyr/output"
	"github.com/jakopako/goskyr/scraper"
)

func runScraper(s scraper.Scraper, itemsChannel chan []map[string]interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("crawling %s\n", s.Name)
	items, err := s.GetItems()
	if err != nil {
		log.Printf("%s ERROR: %s", s.Name, err)
		return
	}
	log.Printf("fetched %d %s events\n", len(items), s.Name)
	itemsChannel <- items
}

func main() {
	singleScraper := flag.String("single", "", "The name of the scraper to be run.")
	// storeData := flag.Bool("store", false, "If set to true the scraped data will be written to the API. (NOTE: custom function that is not well documented, so don't use it.")
	configFile := flag.String("config", "./config.yml", "The location of the configuration file.")
	// TODO add flag to only write to stdout despite other config in the config file.

	flag.Parse()

	config, err := config.NewConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	itemsChannel := make(chan []map[string]interface{}, len(config.Scrapers))

	var writer output.Writer
	switch config.Writer.Type {
	case "stdout":
		writer = &output.StdoutWriter{}
	case "api":
		writer = output.NewAPIWriter(&config.Writer)
	default:
		log.Fatalf("writer of type %s not implemented", config.Writer.Type)
	}

	for _, s := range config.Scrapers {
		if *singleScraper == "" || *singleScraper == s.Name {
			wg.Add(1)
			go runScraper(s, itemsChannel, &wg)
		}
	}
	wg.Wait()
	close(itemsChannel)
	writer.Write(itemsChannel)
}
