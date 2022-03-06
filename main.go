package main

import (
	"flag"
	"log"
	"os"
	"sync"

	"github.com/jakopako/goskyr/output"
	"github.com/jakopako/goskyr/scraper"
	"gopkg.in/yaml.v2"
)

func newConfig(configPath string) (*scraper.Config, error) {
	config := &scraper.Config{}
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	d := yaml.NewDecoder(file)
	if err := d.Decode(&config); err != nil {
		return nil, err
	}
	return config, nil
}

func main() {
	singleScraper := flag.String("single", "", "The name of the scraper to be run.")
	storeData := flag.Bool("store", false, "If set to true the scraped data will be written to the API. (NOTE: custom function that is not well documented, so don't use it.")
	configFile := flag.String("config", "./config.yml", "The location of the configuration file.")

	flag.Parse()

	config, err := newConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

	for _, s := range config.Scrapers {
		if *singleScraper != "" {
			if *singleScraper == s.Name {
				wg.Add(1)
				if *storeData {
					output.WriteItemsToAPI(&wg, s)
				} else {
					output.PrettyPrintItems(&wg, s)
				}
				break
			}
		} else {
			wg.Add(1)
			if *storeData {
				go output.WriteItemsToAPI(&wg, s)
			} else {
				go output.PrettyPrintItems(&wg, s)
			}
		}
	}
	wg.Wait()
}
