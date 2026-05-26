package main

import (
	"strconv"
	"strings"
	"testing"
)

func TestGenerateSVG_BandOrderingTopToBottom(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{
				"top":    {Shape: "rect"},
				"bottom": {Shape: "rect"},
			},
		},
		Layout: Layout_{
			Bands: []Band{
				{Name: "top", Roles: []string{"top"}},
				{Name: "bottom", Roles: []string{"bottom"}},
			},
			Locations: map[string]string{"only": "0"},
		},
		Nodes: []Node{
			{ID: "a", Role: "top", Location: "only"},
			{ID: "b", Role: "bottom", Location: "only"},
		},
		Links: []Link{{From: "a", To: "b"}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	svg := GenerateSVG(cfg, info)

	yA, okA := nodeCY(svg, "a")
	yB, okB := nodeCY(svg, "b")
	if !okA || !okB {
		t.Fatalf("expected node labels in SVG, got:\n%s", svg)
	}
	if yA >= yB {
		t.Errorf("expected band 'top' (a) above band 'bottom' (b); got yA=%d yB=%d", yA, yB)
	}
}

func TestGenerateSVG_DashedLineKindEmittedAsStrokeDasharray(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols:   map[string]Symbol{"r": {Shape: "rect"}},
			LineKinds: map[string]LineKind{"planned": {Style: "dashed", Meaning: "planned"}},
		},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"l": "0", "r": "1"},
		},
		Nodes: []Node{
			{ID: "a", Role: "r", Location: "l"},
			{ID: "b", Role: "r", Location: "r"},
		},
		Links: []Link{{From: "a", To: "b", Kind: "planned"}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	svg := GenerateSVG(cfg, info)
	if !strings.Contains(svg, `stroke-dasharray="6,4"`) {
		t.Errorf("expected stroke-dasharray for dashed kind, got:\n%s", svg)
	}
}

func TestGenerateSVG_EllipseShapeMappedToEllipseElement(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{"endpoint": {Shape: "ellipse"}},
		},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"endpoint"}}},
			Locations: map[string]string{"only": "0"},
		},
		Nodes: []Node{{ID: "user", Role: "endpoint", Location: "only"}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	svg := GenerateSVG(cfg, info)
	if !strings.Contains(svg, "<ellipse") {
		t.Errorf("expected <ellipse> for ellipse symbol, got:\n%s", svg)
	}
}

func TestGenerateSVG_HeavyLineKindEmittedAsStrokeWidth(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols:   map[string]Symbol{"r": {Shape: "rect"}},
			LineKinds: map[string]LineKind{"fiber": {Style: "solid", Width: 2}},
		},
		Layout: Layout_{
			Bands:     []Band{{Name: "only", Roles: []string{"r"}}},
			Locations: map[string]string{"l": "0", "r": "1"},
		},
		Nodes: []Node{
			{ID: "a", Role: "r", Location: "l"},
			{ID: "b", Role: "r", Location: "r"},
		},
		Links: []Link{{From: "a", To: "b", Kind: "fiber"}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	svg := GenerateSVG(cfg, info)
	if !strings.Contains(svg, `stroke-width="2"`) {
		t.Errorf("expected stroke-width=\"2\" for width-2 line, got:\n%s", svg)
	}
}

func TestGenerateSVG_StraightLineDetourWhenObstructed(t *testing.T) {
	// Three bands, single column, with a node in the middle band that
	// would be pierced by a straight vertical line from top to bottom.
	// The renderer must emit a routed edge (rounded-corner <path>),
	// not a straight <line>.
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{
				"top": {Shape: "rect"},
				"mid": {Shape: "rect"},
				"bot": {Shape: "rect"},
			},
		},
		Layout: Layout_{
			Bands: []Band{
				{Name: "b0", Roles: []string{"top"}},
				{Name: "b1", Roles: []string{"mid"}},
				{Name: "b2", Roles: []string{"bot"}},
			},
			Locations: map[string]string{"only": "0"},
		},
		Nodes: []Node{
			{ID: "n-top", Role: "top", Location: "only"},
			{ID: "n-mid", Role: "mid", Location: "only"},
			{ID: "n-bot", Role: "bot", Location: "only"},
		},
		Links: []Link{{From: "n-top", To: "n-bot"}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	svg := GenerateSVG(cfg, info)
	if !strings.Contains(svg, `<path class="edge"`) {
		t.Errorf("expected <path class=\"edge\"> detour around 'n-mid' obstacle, got:\n%s", svg)
	}
	if !strings.Contains(svg, " Q") {
		t.Errorf("expected at least one quadratic-curve (Q) corner in the detour path, got:\n%s", svg)
	}
}

func TestGenerateSVG_SameBandDetourWhenObstructed(t *testing.T) {
	// Three nodes in the same band across three columns. A link
	// connecting the leftmost to the rightmost would, drawn as a
	// straight horizontal, pass through the middle node. The renderer
	// must emit a routed edge (rounded-corner <path>) that detours
	// through the band-gap above.
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{
				"top": {Shape: "rect"},
				"r":   {Shape: "rect"},
			},
		},
		Layout: Layout_{
			Bands: []Band{
				{Name: "above", Roles: []string{"top"}},
				{Name: "row", Roles: []string{"r"}},
			},
			Locations: map[string]string{
				"left":   "0",
				"middle": "1",
				"right":  "2",
			},
		},
		Nodes: []Node{
			{ID: "anchor", Role: "top", Location: "left"},
			{ID: "a", Role: "r", Location: "left"},
			{ID: "b", Role: "r", Location: "middle"},
			{ID: "c", Role: "r", Location: "right"},
		},
		Links: []Link{{From: "a", To: "c"}},
	}
	info, err := ValidateLayout(cfg)
	if err != nil {
		t.Fatalf("validate layout: %v", err)
	}
	svg := GenerateSVG(cfg, info)
	if !strings.Contains(svg, `<path class="edge"`) {
		t.Errorf("expected <path class=\"edge\"> detour around middle obstacle 'b', got:\n%s", svg)
	}
}

// nodeCY returns the cy of the node label matching the given id (Label
// defaults to ID, so plain id text suffices for these test cases).
func nodeCY(svg, id string) (int, bool) {
	needle := `>` + id + `</text>`
	i := strings.Index(svg, needle)
	if i < 0 {
		return 0, false
	}
	prefix := svg[:i]
	yIdx := strings.LastIndex(prefix, `y="`)
	if yIdx < 0 {
		return 0, false
	}
	rest := prefix[yIdx+len(`y="`):]
	endIdx := strings.Index(rest, `"`)
	if endIdx < 0 {
		return 0, false
	}
	n, err := strconv.Atoi(rest[:endIdx])
	if err != nil {
		return 0, false
	}
	return n, true
}
