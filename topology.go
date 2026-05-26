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

		if l.Pattern != "" {
			if _, ok := cfg.Legend.Patterns[l.Pattern]; !ok {
				return fmt.Errorf("link[%d] uses pattern %q which is not defined in legend.patterns", i, l.Pattern)
			}
		}
	}

	if err := validateLinkPatternGroups(cfg); err != nil {
		return err
	}

	return nil
}

// validateLinkPatternGroups enforces the contract between
// "two links share endpoints and kind" and "those links must
// declare a pattern". Two interpretations of two same-kind links
// between the same endpoints are equally plausible: they form a
// single redundant pair, or they are unrelated and the writer just
// happened to point both at the same node. netfig refuses to pick.
// The writer must say which by setting Pattern (or by removing the
// duplicate).
//
// Two links between the same endpoints but with different kinds
// are not sibling candidates: their kinds already declare distinct
// visual meanings, so they are not parallel runs of the same thing.
//
// Once pattern is set on members of a sibling group, every sibling
// must agree on the pattern name. One (endpoints, kind) pair can
// host at most one pattern at a time; mixing patterns on the same
// (endpoints, kind) is rejected.
func validateLinkPatternGroups(cfg *Config) error {
	type endpointKindKey struct {
		ep   [2]string
		kind string
	}
	type group struct {
		indices []int
		pattern string
	}
	endpoints := map[endpointKindKey]*group{}
	for i, l := range cfg.Links {
		k := endpointKindKey{ep: canonicalPairKey(l.From, l.To), kind: l.Kind}
		g, ok := endpoints[k]
		if !ok {
			endpoints[k] = &group{indices: []int{i}, pattern: l.Pattern}
			continue
		}
		g.indices = append(g.indices, i)
		// Once a second link arrives on these endpoints with the
		// same kind, both this link and every prior member must
		// declare a pattern.
		if l.Pattern == "" {
			return fmt.Errorf("link[%d] (%s ↔ %s, kind=%q) shares endpoints and kind with link[%d]; both must set pattern, otherwise remove the duplicate",
				i, l.From, l.To, l.Kind, g.indices[0])
		}
		// The first member's pattern was recorded; ensure it was
		// non-empty too.
		if g.pattern == "" {
			return fmt.Errorf("link[%d] (%s ↔ %s, kind=%q) shares endpoints and kind with link[%d]; both must set pattern, otherwise remove the duplicate",
				g.indices[0], cfg.Links[g.indices[0]].From, cfg.Links[g.indices[0]].To, l.Kind, i)
		}
		if g.pattern != l.Pattern {
			return fmt.Errorf("link[%d] uses pattern %q but link[%d] on the same endpoints and kind uses pattern %q; all siblings must share one pattern",
				i, l.Pattern, g.indices[0], g.pattern)
		}
	}
	return nil
}
