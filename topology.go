package main

import "fmt"

// ValidateTopology checks the integrity of the configuration against
// the legend.
//
// Every check returns an error. The tool's intent is to produce
// readable diagrams; states like "this link uses an undefined kind"
// or "this role is not in the legend" mean the input does not
// uniquely determine what should be drawn, so the tool refuses
// rather than silently picking a default.
func ValidateTopology(cfg *Config) error {
	nodeIDs := make(map[string]bool)
	for _, n := range cfg.Nodes {
		if n.ID == "" {
			return fmt.Errorf("node has empty id")
		}
		if nodeIDs[n.ID] {
			return fmt.Errorf("duplicate node id: %s", n.ID)
		}
		nodeIDs[n.ID] = true

		if n.Role == "" {
			return fmt.Errorf("node %s has empty role", n.ID)
		}
		if _, ok := cfg.Legend.Symbols[n.Role]; !ok {
			return fmt.Errorf("node %s uses role %q which is not defined in legend", n.ID, n.Role)
		}
	}

	for i, l := range cfg.Links {
		if !nodeIDs[l.From] {
			return fmt.Errorf("link[%d] references unknown node: %s", i, l.From)
		}
		if !nodeIDs[l.To] {
			return fmt.Errorf("link[%d] references unknown node: %s", i, l.To)
		}
		if l.From == l.To {
			return fmt.Errorf("link[%d] connects node %s to itself", i, l.From)
		}

		if l.Kind != "" {
			if _, ok := cfg.Legend.LineKinds[l.Kind]; !ok {
				return fmt.Errorf("link[%d] uses kind %q which is not defined in legend.line_kinds", i, l.Kind)
			}
		}
	}

	return nil
}
