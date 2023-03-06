package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/jakopako/goskyr/automate"
	"github.com/jakopako/goskyr/ml"
	"github.com/jakopako/goskyr/output"
	"github.com/jakopako/goskyr/scraper"
	"gopkg.in/yaml.v3"
)

var version = "dev"

func runScraper(s scraper.Scraper, itemsChannel chan map[string]interface{}, globalConfig *scraper.GlobalConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("scraping %s\n", s.Name)
	// This could probably be improved. We could pass the channel to
	// GetItems instead of waiting for the scraper to finish.
	items, err := s.GetItems(globalConfig, false)
	if err != nil {
		log.Printf("%s ERROR: %s", s.Name, err)
		return
	}
	log.Printf("fetched %d %s items\n", len(items), s.Name)
	for _, item := range items {
		itemsChannel <- item
	}
}

func main() {
	singleScraper := flag.String("s", "", "The name of the scraper to be run.")
	toStdout := flag.Bool("stdout", false, "If set to true the scraped data will be written to stdout despite any other existing writer configurations. In combination with the -generate flag the newly generated config will be written to stdout instead of to a file.")
	configFile := flag.String("c", "./config.yml", "The location of the configuration file.")
	printVersion := flag.Bool("v", false, "The version of goskyr.")
	generateConfig := flag.String("g", "", "Automatically generate a config file for the given url.")
	m := flag.Int("m", 20, "The minimum number of items on a page. This is needed to filter out noise. Works in combination with the -g flag.")
	f := flag.Bool("f", false, "Only show fields that have varying values across the list of items. Works in combination with the -g flag.")
	d := flag.Bool("d", false, "Render JS before generating a configuration file. Works in combination with the -g flag.")
	extractFeatures := flag.String("e", "", "Extract ML features based on the given configuration file (-c) and write them to the given file in csv format.")
	wordsDir := flag.String("w", "word-lists", "The directory that contains a number of files containing words of different languages. This is needed for the ML part (use with -e or -b).")
	buildModel := flag.String("t", "", "Train a ML model based on the given csv features file. This will generate 2 files, goskyr.model and goskyr.class")

	flag.Parse()

	if *printVersion {
		fmt.Println(version)
		return
	}

	if *generateConfig != "" {
		s := &scraper.Scraper{URL: *generateConfig}
		if *d {
			s.RenderJs = true
		}
		err := automate.GetDynamicFieldsConfig(s, *m, *f)
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

		if *toStdout {
			fmt.Println(string(yamlData))
		} else {
			f, err := os.Create(*configFile)
			if err != nil {
				log.Fatalf("ERROR while trying to open file: %v", err)
			}
			defer f.Close()
			_, err = f.Write(yamlData)
			if err != nil {
				log.Fatalf("ERROR while trying to write to file: %v", err)
			}
			log.Printf("successfully wrote config to file %s", *configFile)
		}
		return
	}

	if *buildModel != "" {
		if err := ml.TrainModel(*buildModel); err != nil {
			log.Fatal(err)
		}
		return
	}

	config, err := scraper.NewConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	if *extractFeatures != "" {
		if err := ml.ExtractFeatures(config, *extractFeatures, *wordsDir); err != nil {
			log.Fatal(err)
		}
		return
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

	if config.Global.UserAgent == "" {
		config.Global.UserAgent = "goskyr web scraper (github.com/jakopako/goskyr)"
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
