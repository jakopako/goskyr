// Package types defines shared types used across the application.
package types

import "time"

// Interaction represents a simple user interaction with a webpage
type Interaction struct {
	Type     string `yaml:"type,omitempty"`
	Selector string `yaml:"selector,omitempty"`
	Count    int    `yaml:"count,omitempty"`
	Delay    int    `yaml:"delay,omitempty"`
}

// ScraperStatus represents the status of a scraper run.
type ScraperStatus struct {
	ScraperName     string    `json:"scraperName"`
	NrItems         int       `json:"nrItems"`
	NrErrors        int       `json:"nrErrors"`
	LastScrapeStart time.Time `json:"lastScrapeStart"`
	LastScrapeEnd   time.Time `json:"lastScrapeEnd"`
	ScraperLogs     string    `json:"scraperLogs"` // not yet used
}

const (
	InteractionTypeClick  = "click"
	InteractionTypeScroll = "scroll"
)
