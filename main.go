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

func NewConfig(configPath string) (*scraper.Config, error) {
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
	singleCrawler := flag.String("single", "", "The name of the crawler to be run.")
	storeData := flag.Bool("store", false, "If set to true the crawled data will be written to the API.")
	configFile := flag.String("config", "./config.yml", "The location of the configuration file.")

	flag.Parse()

	config, err := NewConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

	for _, s := range config.Scrapers {
		if *singleCrawler != "" {
			if *singleCrawler == s.Name {
				wg.Add(1)
				if *storeData {
					output.WriteEventsToAPI(&wg, s)
				} else {
					output.PrettyPrintEvents(&wg, s)
				}
				break
			}
		} else {
			wg.Add(1)
			if *storeData {
				go output.WriteEventsToAPI(&wg, s)
			} else {
				go output.PrettyPrintEvents(&wg, s)
			}
		}
	}
	wg.Wait()
}
