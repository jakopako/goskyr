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

type ScraperStatus struct {
	Name      string
	NrItems   int
	NrErrors  int
	StartTime time.Time
	EndTime   time.Time
}

const (
	InteractionTypeClick  = "click"
	InteractionTypeScroll = "scroll"
)
