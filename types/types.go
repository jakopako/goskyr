package types

// shared types

// Interaction represents a simple user interaction with a webpage
type Interaction struct {
	Type     string `yaml:"type,omitempty"`
	Selector string `yaml:"selector,omitempty"`
	Count    int    `yaml:"count,omitempty"`
}

const (
	InteractionTypeClick  = "click"
	InteractionTypeScroll = "scroll"
)
