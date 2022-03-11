package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jakopako/goskyr/scraper"
	"gopkg.in/yaml.v2"
)

// .WriterConfig defines the necessary paramters to make a new writer
// which is responsible for writing the scraped data to a specific output
// eg. stdout.
type WriterConfig struct {
	Type     string `yaml:"type" env:"WRITER_TYPE" env-default:"stdout"`
	Uri      string `yaml:"uri" env:"WRITER_URI"`
	User     string `yaml:"user" env:"WRITER_USER"`
	Password string `yaml:"password" env:"WRITER_PASSWORD"`
}

// Config defines the overall structure of the scraper configuration.
// Values will be taken from a config yml file or environment variables
// or both.
type Config struct {
	Writer   WriterConfig      `yaml:"writer"`
	Scrapers []scraper.Scraper `yaml:"scrapers"`
}

func NewConfig(configPath string) (*Config, error) {
	var config Config

	err := cleanenv.ReadConfig(configPath, &config)
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
	return &config, nil
}
