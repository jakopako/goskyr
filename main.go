package main

import (
	"flag"
	"fmt"
	"log"
	"sync"

	automate "github.com/jakopako/goskyr/generate"
	"github.com/jakopako/goskyr/output"
	"github.com/jakopako/goskyr/scraper"
	"gopkg.in/yaml.v3"
)

var version = "dev"

func runScraper(s scraper.Scraper, itemsChannel chan map[string]interface{}, globalConfig *scraper.GlobalConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("crawling %s\n", s.Name)
	// This could probably be improved. We could pass the channel to
	// GetItems instead of waiting for the scraper to finish.
	items, err := s.GetItems(globalConfig)
	if err != nil {
		log.Printf("%s ERROR: %s", s.Name, err)
		return
	}
	log.Printf("fetched %d %s events\n", len(items), s.Name)
	for _, item := range items {
		itemsChannel <- item
	}
}

func main() {
	singleScraper := flag.String("single", "", "The name of the scraper to be run.")
	toStdout := flag.Bool("stdout", false, "If set to true the scraped data will be written to stdout despite any other existing writer configurations.")
	configFile := flag.String("config", "./config.yml", "The location of the configuration file.")
	printVersion := flag.Bool("version", false, "The version of goskyr.")
	// add flag to pass min nr of items for the generate flag.
	generateConfig := flag.String("generate", "", "Needs an additional argument of the url whose config needs to be generated.")
	m := flag.Int("min", 20, "The minimum number of events on a page. This is needed to filter out noise.")
	d := flag.Bool("details", false, "Show details when presenting the different fields found with the generate flag.")

	flag.Parse()

	if *printVersion {
		fmt.Println(version)
		return
	}

	if *generateConfig != "" {
		s := &scraper.Scraper{URL: *generateConfig}
		err := automate.GetDynamicFieldsConfig(s, *m, *d)
		if err != nil {
			log.Fatal(err)
		}
		c := scraper.Config{
			Scrapers: []scraper.Scraper{
				*s,
			},
		}
		yamlData, err := yaml.Marshal(&c)
		if err != nil {
			log.Fatalf("Error while Marshaling. %v", err)
		}

		fmt.Println(string(yamlData))
		return
	}

	config, err := scraper.NewConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	var scraperWg sync.WaitGroup
	var writerWg sync.WaitGroup
	itemsChannel := make(chan map[string]interface{}, len(config.Scrapers))

	var writer output.Writer
	if *toStdout {
		writer = &output.StdoutWriter{}
	} else {
		switch config.Writer.Type {
		case "stdout":
			writer = &output.StdoutWriter{}
		case "api":
			writer = output.NewAPIWriter(&config.Writer)
		case "file":
			writer = output.NewFileWriter(&config.Writer)
		default:
			log.Fatalf("writer of type %s not implemented", config.Writer.Type)
		}
	}

	for _, s := range config.Scrapers {
		if *singleScraper == "" || *singleScraper == s.Name {
			scraperWg.Add(1)
			go runScraper(s, itemsChannel, &config.Global, &scraperWg)
		}
	}
	writerWg.Add(1)
	go writer.Write(itemsChannel, &writerWg)
	scraperWg.Wait()
	close(itemsChannel)
	writerWg.Wait()
}
