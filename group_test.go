package main

import (
	"strings"
	"testing"
)

func TestValidateGroups_Accepts_WellFormed(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{"r": {Shape: "rect"}},
		},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"site-a": "0", "site-b": "1"},
		},
		Nodes:  []Node{{ID: "n", Role: "r", Location: "site-a"}},
		Groups: []Group{{Name: "g", Locations: []string{"site-a"}}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	if err := ValidateGroups(cfg, info); err != nil {
		t.Errorf("expected well-formed group to validate, got: %v", err)
	}
}

func TestValidateGroups_Rejects_UnknownLocation(t *testing.T) {
	cfg := &Config{
		Legend: Legend{Symbols: map[string]Symbol{"r": {Shape: "rect"}}},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"site-a": "0"},
		},
		Nodes:  []Node{{ID: "n", Role: "r", Location: "site-a"}},
		Groups: []Group{{Name: "g", Locations: []string{"ghost"}}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	err = ValidateGroups(cfg, info)
	if err == nil || !strings.Contains(err.Error(), "unknown location") {
		t.Errorf("expected unknown-location error, got: %v", err)
	}
}

func TestValidateGroups_Rejects_UnknownBand(t *testing.T) {
	cfg := &Config{
		Legend: Legend{Symbols: map[string]Symbol{"r": {Shape: "rect"}}},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"site-a": "0"},
		},
		Nodes:  []Node{{ID: "n", Role: "r", Location: "site-a"}},
		Groups: []Group{{Name: "g", Locations: []string{"site-a"}, Bands: []string{"ghost"}}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	err = ValidateGroups(cfg, info)
	if err == nil || !strings.Contains(err.Error(), "unknown band") {
		t.Errorf("expected unknown-band error, got: %v", err)
	}
}

func TestValidateGroups_Rejects_EmptyName(t *testing.T) {
	cfg := &Config{
		Legend: Legend{Symbols: map[string]Symbol{"r": {Shape: "rect"}}},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"site-a": "0"},
		},
		Nodes:  []Node{{ID: "n", Role: "r", Location: "site-a"}},
		Groups: []Group{{Name: "", Locations: []string{"site-a"}}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	err = ValidateGroups(cfg, info)
	if err == nil || !strings.Contains(err.Error(), "empty name") {
		t.Errorf("expected empty-name error, got: %v", err)
	}
}

func TestValidateGroups_Rejects_EmptyLocations(t *testing.T) {
	cfg := &Config{
		Legend: Legend{Symbols: map[string]Symbol{"r": {Shape: "rect"}}},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"site-a": "0"},
		},
		Nodes:  []Node{{ID: "n", Role: "r", Location: "site-a"}},
		Groups: []Group{{Name: "g", Locations: []string{}}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	err = ValidateGroups(cfg, info)
	if err == nil || !strings.Contains(err.Error(), "empty locations") {
		t.Errorf("expected empty-locations error, got: %v", err)
	}
}

// TestGenerateSVG_GroupEmittedBeforeNodes guards the painter order: a
// group is meant to sit visually under every node and link, which in
// SVG requires it to be emitted earlier in document order. The filled
// background only avoids competing with link lines if it is drawn
// behind them.
func TestGenerateSVG_GroupEmittedBeforeNodes(t *testing.T) {
	cfg := &Config{
		Legend: Legend{Symbols: map[string]Symbol{"r": {Shape: "rect"}}},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"site-a": "0"},
		},
		Nodes:  []Node{{ID: "n1", Role: "r", Location: "site-a"}},
		Groups: []Group{{Name: "g", Locations: []string{"site-a"}}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	if err := ValidateGroups(cfg, info); err != nil {
		t.Fatalf("validate groups: %v", err)
	}
	svg := GenerateSVG(cfg, info)
	groupIdx := strings.Index(svg, `class="group"`)
	nodeIdx := strings.Index(svg, `class="node"`)
	if groupIdx < 0 {
		t.Fatalf("expected a group rect in the SVG, got:\n%s", svg)
	}
	if nodeIdx < 0 {
		t.Fatalf("expected a node rect in the SVG, got:\n%s", svg)
	}
	if groupIdx > nodeIdx {
		t.Errorf("expected group to be emitted before nodes; group at %d, node at %d", groupIdx, nodeIdx)
	}
}

// TestGenerateSVG_GroupHasNoStroke documents that the group fill has
// no outline, which is the rendering choice that keeps the group
// boundary from competing visually with link lines.
func TestGenerateSVG_GroupHasNoStroke(t *testing.T) {
	cfg := &Config{
		Legend: Legend{Symbols: map[string]Symbol{"r": {Shape: "rect"}}},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"site-a": "0"},
		},
		Nodes:  []Node{{ID: "n", Role: "r", Location: "site-a"}},
		Groups: []Group{{Name: "g", Locations: []string{"site-a"}}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	if err := ValidateGroups(cfg, info); err != nil {
		t.Fatalf("validate groups: %v", err)
	}
	svg := GenerateSVG(cfg, info)
	if !strings.Contains(svg, ".group { fill: ") {
		t.Errorf("expected a group fill style declaration, got:\n%s", svg)
	}
	if !strings.Contains(svg, "stroke: none") {
		t.Errorf("expected the group class to use stroke: none, got:\n%s", svg)
	}
}
