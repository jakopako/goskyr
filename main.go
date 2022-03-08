package main

import (
	"flag"
	"log"
	"os"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jakopako/goskyr/output"
	"github.com/jakopako/goskyr/scraper"
	"gopkg.in/yaml.v2"
)

// Config defines the overall structure of the scraper configuration.
// Values will be taken from a config yml file or environment variables
// or both.
type Config struct {
	// Config.Writer defines the necessary paramters to make a new writer
	// which is responsible for writing the scraped data to a specific output
	// eg. stdout.
	Writer struct {
		Type     string `yaml:"output" env:"WRITER_TYPE" env-default:"stdout"`
		Uri      string `yaml:"uri" env:"WRITER_URI"`
		User     string `yaml:"user" env:"WRITER_USER"`
		Password string `yaml:"password" env:"WRITER_PASSWORD"`
	} `yaml:"writer"`
	Scrapers []scraper.Scraper `yaml:"scrapers"`
}

func newConfig(configPath string) (*Config, error) {
	var config *Config

	err := cleanenv.ReadConfig(configPath, config)
	if err != nil {
		log.Fatal(err)
	}

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

func runScraper(s scraper.Scraper, itemsChannel chan []map[string]interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	// logging?
	items, err := s.GetItems()
	if err != nil {
		log.Printf("%s ERROR: %s", s.Name, err)
		return
	}
	itemsChannel <- items
}

func main() {
	singleScraper := flag.String("single", "", "The name of the scraper to be run.")
	// storeData := flag.Bool("store", false, "If set to true the scraped data will be written to the API. (NOTE: custom function that is not well documented, so don't use it.")
	configFile := flag.String("config", "./config.yml", "The location of the configuration file.")

	flag.Parse()

	config, err := newConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	itemsChannel := make(chan []map[string]interface{}, len(config.Scrapers))

	var writer output.Writer
	switch config.Writer.Type {
	case "stdout":
		writer = &output.StdoutWriter{}
	// case "mongodb":
	// 	writer, err = output.NewMongoDBWriter(config.Writer)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
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
