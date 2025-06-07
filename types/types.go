// Package types defines shared types used across the application.
package types

// Interaction represents a simple user interaction with a webpage
type Interaction struct {
	Type     string `yaml:"type,omitempty"`
	Selector string `yaml:"selector,omitempty"`
	Count    int    `yaml:"count,omitempty"`
	Delay    int    `yaml:"delay,omitempty"`
}

const (
	InteractionTypeClick  = "click"
	InteractionTypeScroll = "scroll"
)
