package main

import (
	"strings"
	"testing"
)

// minimalConfig returns a config that passes ValidateTopology.
// Tests build on this and mutate one field to exercise a specific
// rule.
func minimalConfig() *Config {
	return &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{"r": {Shape: "rect"}},
			LineKinds: map[string]LineKind{
				"k": {Style: "solid"},
			},
		},
		Nodes: []Node{
			{ID: "a", Role: "r", Location: "site"},
			{ID: "b", Role: "r", Location: "site"},
		},
		Links: []Link{
			{From: "a", To: "b"},
		},
	}
}

func TestValidateTopology_Accepts_Minimal(t *testing.T) {
	if err := ValidateTopology(minimalConfig()); err != nil {
		t.Fatalf("expected minimal config to pass, got: %v", err)
	}
}

func TestValidateTopology_Rejects_EmptyNodeID(t *testing.T) {
	cfg := minimalConfig()
	cfg.Nodes[0].ID = ""
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "empty id") {
		t.Errorf("expected empty-id error, got: %v", err)
	}
}

func TestValidateTopology_Rejects_DuplicateNodeID(t *testing.T) {
	cfg := minimalConfig()
	cfg.Nodes = append(cfg.Nodes, Node{ID: "a", Role: "r", Location: "site"})
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "duplicate node id") {
		t.Errorf("expected duplicate-id error, got: %v", err)
	}
}

func TestValidateTopology_Rejects_EmptyRole(t *testing.T) {
	cfg := minimalConfig()
	cfg.Nodes[0].Role = ""
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "empty role") {
		t.Errorf("expected empty-role error, got: %v", err)
	}
}

func TestValidateTopology_Rejects_UndefinedRole(t *testing.T) {
	cfg := minimalConfig()
	cfg.Nodes[0].Role = "ghost"
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "not defined in legend") {
		t.Errorf("expected legend-undefined-role error, got: %v", err)
	}
}

func TestValidateTopology_Rejects_LinkFromUnknownNode(t *testing.T) {
	cfg := minimalConfig()
	cfg.Links[0].From = "ghost"
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "unknown node") {
		t.Errorf("expected unknown-from-node error, got: %v", err)
	}
}

func TestValidateTopology_Rejects_LinkToUnknownNode(t *testing.T) {
	cfg := minimalConfig()
	cfg.Links[0].To = "ghost"
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "unknown node") {
		t.Errorf("expected unknown-to-node error, got: %v", err)
	}
}

func TestValidateTopology_Rejects_SelfLink(t *testing.T) {
	cfg := minimalConfig()
	cfg.Links[0].To = cfg.Links[0].From
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "itself") {
		t.Errorf("expected self-link error, got: %v", err)
	}
}

func TestValidateTopology_Rejects_LinkWithUndefinedKind(t *testing.T) {
	cfg := minimalConfig()
	cfg.Links[0].Kind = "ghost-kind"
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "not defined in legend.line_kinds") {
		t.Errorf("expected undefined-kind error, got: %v", err)
	}
}
