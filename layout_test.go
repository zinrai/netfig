package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateLayout_RejectsOverDenseCell(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{"r": {Shape: "rect"}},
		},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"site": "0"},
		},
	}
	for i := 1; i <= maxNodesPerCell+1; i++ {
		cfg.Nodes = append(cfg.Nodes, Node{
			ID: fmt.Sprintf("n%d", i), Role: "r", Location: "site",
		})
	}
	_, err := ValidateLayout(cfg)
	if err == nil {
		t.Fatalf("expected error for %d nodes in one cell", maxNodesPerCell+1)
	}
	if !strings.Contains(err.Error(), "limit is") {
		t.Errorf("expected error to mention the limit, got: %v", err)
	}
	if !strings.Contains(err.Error(), "site") {
		t.Errorf("expected error to mention the location 'site', got: %v", err)
	}
}

func TestValidateLayout_AcceptsCellAtCap(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{"r": {Shape: "rect"}},
		},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"site": "0"},
		},
	}
	for i := 1; i <= maxNodesPerCell; i++ {
		cfg.Nodes = append(cfg.Nodes, Node{
			ID: fmt.Sprintf("n%d", i), Role: "r", Location: "site",
		})
	}
	if _, err := ValidateLayout(cfg); err != nil {
		t.Fatalf("expected %d nodes in one cell to be accepted, got: %v", maxNodesPerCell, err)
	}
}
