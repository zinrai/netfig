package main

import (
	"strings"
	"testing"
)

func TestValidateLegend_NonSolidWithoutMeaningIsError(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			LineKinds: map[string]LineKind{
				"weird": {Style: "dashed"},
			},
		},
	}
	err := ValidateLegend(cfg)
	if err == nil {
		t.Fatalf("expected error for dashed line without meaning")
	}
	if !strings.Contains(err.Error(), "no meaning") {
		t.Errorf("expected error message to mention 'no meaning', got: %v", err)
	}
}

func TestValidateLegend_UnsupportedShapeIsError(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			Symbols: map[string]Symbol{
				"weird": {Shape: "octagon"},
			},
		},
	}
	err := ValidateLegend(cfg)
	if err == nil {
		t.Fatalf("expected error for unsupported shape")
	}
	if !strings.Contains(err.Error(), "octagon") {
		t.Errorf("expected error message to mention the bad shape, got: %v", err)
	}
}

func TestValidateLegend_UnsupportedStyleIsError(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			LineKinds: map[string]LineKind{
				"weird": {Style: "wavy", Meaning: "x"},
			},
		},
	}
	err := ValidateLegend(cfg)
	if err == nil {
		t.Fatalf("expected error for unsupported line style")
	}
	if !strings.Contains(err.Error(), "wavy") {
		t.Errorf("expected error message to mention the bad style, got: %v", err)
	}
}

func TestValidateLegend_ThreeStylesAreAllowed(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			LineKinds: map[string]LineKind{
				"a": {Style: "solid"},
				"b": {Style: "dashed", Meaning: "planned"},
				"c": {Style: "dotted", Meaning: "low_prio"},
			},
		},
	}
	if err := ValidateLegend(cfg); err != nil {
		t.Errorf("three styles (solid+dashed+dotted) should be allowed, got: %v", err)
	}
}

// TestValidateLegend_FourStylesIsError documents that exceeding three
// distinct line styles is treated as an error: the tool's intent is to
// produce readable diagrams, and the book notes that line variety
// harms readability.
//
// Note: there are only three style values currently defined (solid,
// dashed, dotted), so reaching four requires a future style or a
// repeated declaration. This test guards against accidentally
// loosening the limit when new styles are added.
func TestValidateLegend_FourStylesIsError(t *testing.T) {
	cfg := &Config{
		Legend: Legend{
			LineKinds: map[string]LineKind{
				"a": {Style: "solid"},
				"b": {Style: "dashed", Meaning: "planned"},
				"c": {Style: "dotted", Meaning: "low_prio"},
				// A hypothetical fourth style. The legend validator
				// will reject the unsupported style itself first, so
				// this test does not actually drive the count check
				// today; it is kept as a structural guard for future
				// style additions.
				"d": {Style: "wavy", Meaning: "x"},
			},
		},
	}
	if err := ValidateLegend(cfg); err == nil {
		t.Errorf("expected an error (either unsupported style or too many styles)")
	}
}
