package main

// Patterns: a vocabulary for link relationships that cannot be
// expressed by line_kind alone. line_kind tells the reader what a
// single link means (an eBGP session, an OSPF adjacency). A pattern
// tells the reader what a *group* of links means together: a
// redundant pair, an ECMP bundle, an active/standby pair. The
// declared pattern name carries the meaning; the rendered output
// (parallel runs) is the visual cue that signals "this is the
// pattern".
//
// A "sibling group" is defined by the unordered endpoint pair
// {from, to} together with the link's kind. Two links between the
// same boxes but with different kinds are not siblings: their
// kinds already declare distinct visual meanings (one is solid, the
// other dashed, and so on), so they are not parallel runs of the
// same thing. They are two separate lines representing two
// separate relationships. Only links that share both endpoints and
// kind are candidates for parallel-run rendering, and even then
// only when the writer declares the pattern explicitly.
//
// The reason patterns exist as a separate concept rather than being
// inferred from "two links between the same endpoints with the same
// kind":
//
//   - Inference would silently turn any same-kind duplicate from/to
//     pair into a parallel-run pattern, regardless of writer
//     intent. Two links that happen to share endpoints and kind but
//     represent independent relationships would be wrongly fused.
//     The diagram must mean what the writer said it means.
//
//   - The reader should be able to read the diagram by looking it
//     up in the legend. A parallel-run rendering is only meaningful
//     if the legend declares what the pattern is. Without that
//     declaration, parallel runs are an unexplained visual
//     convention.
//
// Validation enforces the contract:
//
//   - Every link's pattern (if set) must reference legend.patterns.
//   - When two or more links share endpoints AND kind, every one of
//     them must carry a pattern declaration. Sharing endpoints and
//     kind without a declared pattern is rejected as ambiguous input.
//   - Within a sibling group (links sharing endpoints and kind under
//     one pattern), every link must use the same pattern name.

// Pattern declares a visual convention for a group of related
// links. Its only payload is a human-readable meaning recorded for
// the legend; the rendering rules attached to "parallel runs" are
// fixed by netfig, not configured per pattern.
type Pattern struct {
	Meaning string `yaml:"meaning"`
}

// siblingInfo is what the renderer needs to know about a link's
// place in its sibling group. The first sibling has Index 0; Size 1
// indicates a solitary link with no siblings.
type siblingInfo struct {
	Index int
	Size  int
}

// patternIndex is the per-link sibling assignment, computed once per
// render. It maps each link's slice index in cfg.Links to a
// siblingInfo.
type patternIndex struct {
	sib []siblingInfo
}

// siblingOf returns the siblingInfo for the link at the given index.
func (p *patternIndex) siblingOf(i int) siblingInfo {
	if i < 0 || i >= len(p.sib) {
		return siblingInfo{Index: 0, Size: 1}
	}
	return p.sib[i]
}

// buildPatternIndex groups links by (unordered endpoint pair, kind,
// pattern name) and produces the per-link sibling assignment.
// Validation has already ensured every link in a multi-link
// (endpoints, kind) group has a pattern set and that all members of
// a group share the same pattern name.
func buildPatternIndex(cfg *Config) *patternIndex {
	type key struct {
		ep      [2]string
		kind    string
		pattern string
	}

	groups := map[key][]int{}
	for i, l := range cfg.Links {
		k := key{
			ep:      canonicalPairKey(l.From, l.To),
			kind:    l.Kind,
			pattern: l.Pattern,
		}
		groups[k] = append(groups[k], i)
	}

	out := &patternIndex{sib: make([]siblingInfo, len(cfg.Links))}
	for _, members := range groups {
		size := len(members)
		for idx, linkIdx := range members {
			out.sib[linkIdx] = siblingInfo{Index: idx, Size: size}
		}
	}
	return out
}

// canonicalPairKey returns the unordered endpoint pair as a sorted
// 2-tuple, so {from:A, to:B} and {from:B, to:A} hash to the same
// group.
func canonicalPairKey(from, to string) [2]string {
	if from <= to {
		return [2]string{from, to}
	}
	return [2]string{to, from}
}
