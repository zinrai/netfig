package main

import "fmt"

// allowedShapes is the set of shape values supported in this stage.
var allowedShapes = map[string]bool{
	"rect":    true,
	"ellipse": true,
}

// allowedStyles is the set of line style values supported in this stage.
var allowedStyles = map[string]bool{
	"solid":  true,
	"dashed": true,
	"dotted": true,
}

// ValidateLegend checks the legend for completeness and internal
// consistency, before the topology is checked against it.
//
// Every check here returns an error rather than a warning. The tool's
// purpose is to produce diagrams that are not hard to read; emitting
// a diagram in a state that is known to be wrong (unrecognised shape,
// dashed line without a declared meaning, more than three distinct
// line styles) contradicts that purpose.
func ValidateLegend(cfg *Config) error {
	for role, sym := range cfg.Legend.Symbols {
		if sym.Shape == "" {
			return fmt.Errorf("legend.symbols[%s] has empty shape", role)
		}
		if !allowedShapes[sym.Shape] {
			return fmt.Errorf("legend.symbols[%s] uses unsupported shape %q; supported: rect, ellipse", role, sym.Shape)
		}
	}

	for name, lk := range cfg.Legend.LineKinds {
		if lk.Style == "" {
			return fmt.Errorf("legend.line_kinds[%s] has empty style", name)
		}
		if !allowedStyles[lk.Style] {
			return fmt.Errorf("legend.line_kinds[%s] uses unsupported style %q; supported: solid, dashed, dotted", name, lk.Style)
		}
		if lk.Style != "solid" && lk.Meaning == "" {
			return fmt.Errorf("legend.line_kinds[%s] is non-solid but has no meaning; non-solid lines must declare a meaning", name)
		}
	}

	// Distinct line styles in use across the legend. The tool's intent
	// is to produce readable diagrams; line variety harms readability,
	// so configurations declaring more than three distinct styles are
	// rejected. Solid is always counted.
	styles := make(map[string]bool)
	styles["solid"] = true
	for _, lk := range cfg.Legend.LineKinds {
		if lk.Style != "" {
			styles[lk.Style] = true
		}
	}
	if len(styles) > 3 {
		return fmt.Errorf("legend declares %d distinct line styles; limit is three", len(styles))
	}

	return nil
}
