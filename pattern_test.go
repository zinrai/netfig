package main

import (
	"strings"
	"testing"
)

func TestValidateLegend_Accepts_PatternWithMeaning(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Patterns: map[string]Pattern{
				"redundant_pair": {Meaning: "primary/secondary pair on the same endpoints"},
			},
		},
	}
	if err := ValidateLegend(cfg); err != nil {
		t.Errorf("expected legend with one well-formed pattern to validate, got: %v", err)
	}
}

func TestValidateLegend_Rejects_PatternWithEmptyMeaning(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Patterns: map[string]Pattern{
				"redundant_pair": {Meaning: ""},
			},
		},
	}
	err := ValidateLegend(cfg)
	if err == nil || !strings.Contains(err.Error(), "no meaning") {
		t.Errorf("expected error for pattern without meaning, got: %v", err)
	}
}

func TestValidateTopology_Rejects_UndeclaredLinkPattern(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{"r": {Shape: "rect"}},
		},
		Nodes: []Node{
			{ID: "a", Role: "r", Location: "site"},
			{ID: "b", Role: "r", Location: "site"},
		},
		Links: []Link{
			{From: "a", To: "b", Pattern: "ghost_pattern"},
		},
	}
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "not defined in legend.patterns") {
		t.Errorf("expected undefined-pattern error, got: %v", err)
	}
}

func TestValidateTopology_Rejects_SiblingsWithoutPattern(t *testing.T) {
	// Two links between the same endpoints with no pattern set must
	// be rejected: the writer's intent is ambiguous.
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{"r": {Shape: "rect"}},
		},
		Nodes: []Node{
			{ID: "a", Role: "r", Location: "site"},
			{ID: "b", Role: "r", Location: "site"},
		},
		Links: []Link{
			{From: "a", To: "b"},
			{From: "a", To: "b"},
		},
	}
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "pattern") {
		t.Errorf("expected error demanding pattern on duplicate endpoints, got: %v", err)
	}
}

func TestValidateTopology_Rejects_PartialPatternOnSiblings(t *testing.T) {
	// First link has no pattern, second link does: still rejected
	// because the pair as a whole is partially declared.
	cfg := &Config{
		Legend: Legend{
			Symbols:  map[string]Symbol{"r": {Shape: "rect"}},
			Patterns: map[string]Pattern{"p": {Meaning: "x"}},
		},
		Nodes: []Node{
			{ID: "a", Role: "r", Location: "site"},
			{ID: "b", Role: "r", Location: "site"},
		},
		Links: []Link{
			{From: "a", To: "b"},
			{From: "a", To: "b", Pattern: "p"},
		},
	}
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "pattern") {
		t.Errorf("expected error for partially-declared sibling pair, got: %v", err)
	}
}

func TestValidateTopology_Rejects_DifferentPatternsOnSameEndpoints(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{"r": {Shape: "rect"}},
			Patterns: map[string]Pattern{
				"p1": {Meaning: "one"},
				"p2": {Meaning: "two"},
			},
		},
		Nodes: []Node{
			{ID: "a", Role: "r", Location: "site"},
			{ID: "b", Role: "r", Location: "site"},
		},
		Links: []Link{
			{From: "a", To: "b", Pattern: "p1"},
			{From: "a", To: "b", Pattern: "p2"},
		},
	}
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "share one pattern") {
		t.Errorf("expected error for mixed patterns on same endpoints, got: %v", err)
	}
}

func TestValidateTopology_Accepts_ConsistentPatternOnSiblings(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols:  map[string]Symbol{"r": {Shape: "rect"}},
			Patterns: map[string]Pattern{"p": {Meaning: "x"}},
		},
		Nodes: []Node{
			{ID: "a", Role: "r", Location: "site"},
			{ID: "b", Role: "r", Location: "site"},
		},
		Links: []Link{
			{From: "a", To: "b", Pattern: "p"},
			{From: "a", To: "b", Pattern: "p"},
		},
	}
	if err := ValidateTopology(cfg); err != nil {
		t.Errorf("expected consistent pattern on siblings to validate, got: %v", err)
	}
}

// TestValidateTopology_Accepts_SameEndpointsDifferentKinds documents
// that two links between the same two boxes are allowed when their
// kinds differ — the kinds already declare distinct visual meanings,
// so the lines are not parallel runs of the same thing. A common
// real-world case is an intra-site physical link drawn alongside an
// iBGP session that rides over it.
func TestValidateTopology_Accepts_SameEndpointsDifferentKinds(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{"r": {Shape: "rect"}},
			LineKinds: map[string]LineKind{
				"physical": {Style: "solid"},
				"session":  {Style: "dashed", Meaning: "iBGP session"},
			},
		},
		Nodes: []Node{
			{ID: "a", Role: "r", Location: "site"},
			{ID: "b", Role: "r", Location: "site"},
		},
		Links: []Link{
			{From: "a", To: "b", Kind: "physical"},
			{From: "a", To: "b", Kind: "session"},
		},
	}
	if err := ValidateTopology(cfg); err != nil {
		t.Errorf("expected two links of different kinds on same endpoints to validate, got: %v", err)
	}
}

// TestValidateTopology_Rejects_SameKindSiblingsWithoutPattern is the
// counterpart of the above: when two links share both endpoints AND
// kind, the writer's intent IS ambiguous (one redundant pair, or two
// independent same-kind relationships?) and netfig refuses to pick.
func TestValidateTopology_Rejects_SameKindSiblingsWithoutPattern(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols:   map[string]Symbol{"r": {Shape: "rect"}},
			LineKinds: map[string]LineKind{"ebgp": {Style: "solid"}},
		},
		Nodes: []Node{
			{ID: "a", Role: "r", Location: "site"},
			{ID: "b", Role: "r", Location: "site"},
		},
		Links: []Link{
			{From: "a", To: "b", Kind: "ebgp"},
			{From: "a", To: "b", Kind: "ebgp"},
		},
	}
	err := ValidateTopology(cfg)
	if err == nil || !strings.Contains(err.Error(), "pattern") {
		t.Errorf("expected error for same-kind duplicate endpoints without pattern, got: %v", err)
	}
}

// TestBuildPatternIndex_DifferentKindsAreNotSiblings documents that
// when two links between the same endpoints have different kinds,
// they are independent (size=1 each), not siblings (size=2). The
// renderer therefore draws them on their default lanes rather than
// offsetting one of them as a parallel run.
func TestBuildPatternIndex_DifferentKindsAreNotSiblings(t *testing.T) {
	cfg := &Config{
		Links: []Link{
			{From: "a", To: "b", Kind: "physical"},
			{From: "a", To: "b", Kind: "session"},
		},
	}
	idx := buildPatternIndex(cfg)
	if idx.siblingOf(0).Size != 1 || idx.siblingOf(1).Size != 1 {
		t.Errorf("expected different-kind links to be solitary (size=1), got %+v / %+v",
			idx.siblingOf(0), idx.siblingOf(1))
	}
}

func TestBuildPatternIndex_SolitaryLinkSize1(t *testing.T) {
	cfg := &Config{
		Links: []Link{{From: "a", To: "b"}},
	}
	idx := buildPatternIndex(cfg)
	sib := idx.siblingOf(0)
	if sib.Size != 1 || sib.Index != 0 {
		t.Errorf("expected solitary link size=1 index=0, got %+v", sib)
	}
}

func TestBuildPatternIndex_GroupsByPatternAndEndpoints(t *testing.T) {
	// Two links with same pattern + endpoints → sibling group of 2.
	// One link with same endpoints but no pattern would have been
	// rejected at validation; here we test the post-validation case.
	cfg := &Config{
		Links: []Link{
			{From: "a", To: "b", Pattern: "p"},
			{From: "a", To: "b", Pattern: "p"},
		},
	}
	idx := buildPatternIndex(cfg)
	if idx.siblingOf(0).Size != 2 || idx.siblingOf(1).Size != 2 {
		t.Errorf("expected both links in size-2 group, got %+v / %+v",
			idx.siblingOf(0), idx.siblingOf(1))
	}
	if idx.siblingOf(0).Index != 0 || idx.siblingOf(1).Index != 1 {
		t.Errorf("expected indices 0 and 1, got %d and %d",
			idx.siblingOf(0).Index, idx.siblingOf(1).Index)
	}
}

func TestBuildPatternIndex_ReversedEndpointsAreSiblings(t *testing.T) {
	cfg := &Config{
		Links: []Link{
			{From: "a", To: "b", Pattern: "p"},
			{From: "b", To: "a", Pattern: "p"},
		},
	}
	idx := buildPatternIndex(cfg)
	if idx.siblingOf(0).Size != 2 || idx.siblingOf(1).Size != 2 {
		t.Errorf("expected reversed-endpoint links to be siblings, got %+v / %+v",
			idx.siblingOf(0), idx.siblingOf(1))
	}
}

func TestPairLaneOffset_AlternatesAroundCentre(t *testing.T) {
	want := []int{0, pairOffsetStep, -pairOffsetStep, 2 * pairOffsetStep, -2 * pairOffsetStep}
	for i, w := range want {
		got := pairLaneOffset(i)
		if got != w {
			t.Errorf("pairLaneOffset(%d): got %d, want %d", i, got, w)
		}
	}
}

func TestGenerateSVG_SiblingLinksRenderAsParallelPathsWithBothLabels(t *testing.T) {
	// Two pattern-declared links between the same nodes. Each link
	// renders its own label; both labels appear in the SVG. The
	// column gap is widened to accommodate the parallel runs.
	cfg := &Config{
		Legend: Legend{
			Symbols:   map[string]Symbol{"r": {Shape: "rect"}},
			LineKinds: map[string]LineKind{"e": {Style: "solid", Width: 2}},
			Patterns:  map[string]Pattern{"redundant": {Meaning: "primary/secondary pair"}},
		},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"l": "0", "r": "1"},
		},
		Nodes: []Node{
			{ID: "a", Role: "r", Location: "l"},
			{ID: "b", Role: "r", Location: "r"},
		},
		Links: []Link{
			{From: "a", To: "b", Label: "primary", Kind: "e", Pattern: "redundant"},
			{From: "a", To: "b", Label: "secondary", Kind: "e", Pattern: "redundant"},
		},
	}
	if err := ValidateTopology(cfg); err != nil {
		t.Fatalf("validate topology: %v", err)
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	svg := GenerateSVG(cfg, info)

	pathCount := strings.Count(svg, `<path class="edge"`)
	if pathCount < 2 {
		t.Errorf("expected two sibling paths, got %d:\n%s", pathCount, svg)
	}
	if !strings.Contains(svg, `>primary</text>`) {
		t.Errorf("expected primary label as own <text>, got:\n%s", svg)
	}
	if !strings.Contains(svg, `>secondary</text>`) {
		t.Errorf("expected secondary label as own <text>, got:\n%s", svg)
	}
}
