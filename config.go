package main

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

// Config is the top-level YAML structure.
type Config struct {
	Purpose Purpose `yaml:"purpose"`
	Legend  Legend  `yaml:"legend"`
	Layout  Layout_ `yaml:"layout"`
	Nodes   []Node  `yaml:"nodes"`
	Links   []Link  `yaml:"links"`
}

// Purpose corresponds to step 1 of the book: who is the diagram for.
// Recorded as metadata only; not used for any automated decision.
type Purpose struct {
	Audience string `yaml:"audience"`
	Intent   string `yaml:"intent"`
}

// Legend corresponds to step 3: the visual vocabulary defined up front.
type Legend struct {
	Symbols   map[string]Symbol   `yaml:"symbols"`
	LineKinds map[string]LineKind `yaml:"line_kinds"`
}

// Symbol defines how a role is drawn.
// Shape is one of: rect, ellipse.
type Symbol struct {
	Shape string `yaml:"shape"`
	Label string `yaml:"label"`
}

// LineKind defines a named line style for links.
// Non-solid styles must declare a meaning so the diagram does not use
// dashed or dotted lines without conveying what that variation means.
type LineKind struct {
	Style   string `yaml:"style"`   // solid | dashed | dotted
	Width   int    `yaml:"width"`   // optional, defaults to 1
	Meaning string `yaml:"meaning"` // required for non-solid styles
}

// Layout_ corresponds to step 4: how the diagram is structured.
// Bands order the diagram top-to-bottom; locations order it left-to-right.
// netfig delegates the actual coordinate calculation to Graphviz, so
// no routing or crossing-strategy options exist here.
type Layout_ struct {
	Bands     []Band            `yaml:"bands"`
	Locations map[string]string `yaml:"locations"`
}

// Band is a horizontal layer in the diagram.
// Roles in this band are placed at the band's vertical position.
// Bands are ordered top-to-bottom in the YAML (upstream to downstream).
type Band struct {
	Name  string   `yaml:"name"`
	Roles []string `yaml:"roles"`
}

// Node is a single network element.
type Node struct {
	ID       string `yaml:"id"`
	Role     string `yaml:"role"`
	Location string `yaml:"location"`
	Label    string `yaml:"label"`
	Vendor   string `yaml:"vendor"`
	Model    string `yaml:"model"`
	IP       string `yaml:"ip"`
	VLAN     string `yaml:"vlan"`
}

// Link is a connection between two nodes.
// Kind names a LineKind from the legend. Empty means "default" (solid).
type Link struct {
	From  string `yaml:"from"`
	To    string `yaml:"to"`
	Label string `yaml:"label"`
	Kind  string `yaml:"kind"`
}

// LoadConfig reads and parses a YAML config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml from %s: %w", path, err)
	}

	return &cfg, nil
}
