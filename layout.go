package main

import (
	"fmt"
	"sort"
	"strings"
)

// maxNodesPerCell caps the number of nodes that may share a single
// (band, location) cell. The limit comes from two sides:
//
//   - Implementation: the lane-separation mechanism in render_svg.go
//     keeps polyline endpoints inside the source rect only while the
//     spread (N-1)*step stays within rectWidth - 2*detourFaceOffset.
//     With the default geometry that bound is 60 px, so a cell of 12
//     uses lane step 5 (spread 55).
//   - Readability: the book the netfig rules come from caps a single
//     diagram at roughly 50 elements; 12 nodes on one row already
//     pushes the visual budget for a single (band, location). Beyond
//     that the author should split the band or the location.
//
// netfig follows the same "rules + discipline" stance as the existing
// validations (legend shape vocabulary, three-line-style ceiling) and
// rejects the input rather than silently rendering something that
// will not read.
const maxNodesPerCell = 12

// LayoutInfo carries the validated mapping from roles to bands and
// from locations to columns. The renderer turns these into pixel
// coordinates directly.
type LayoutInfo struct {
	// RoleToBand maps a role name to its band index (0 = top).
	RoleToBand map[string]int
	// LocationToCol maps a location name to its column index (0 = left).
	LocationToCol map[string]int
	// BandNames is the ordered list of band names (top to bottom).
	BandNames []string
}

// ValidateLayout checks that the bands and locations are internally
// consistent and that every used role is assigned to exactly one
// band and every used location is defined.
func ValidateLayout(cfg *Config) (*LayoutInfo, error) {
	roleToBand := make(map[string]int)
	bandNames := make([]string, 0, len(cfg.Layout.Bands))
	for i, b := range cfg.Layout.Bands {
		bandNames = append(bandNames, b.Name)
		for _, r := range b.Roles {
			if _, dup := roleToBand[r]; dup {
				return nil, fmt.Errorf("role %q appears in multiple bands", r)
			}
			roleToBand[r] = i
		}
	}

	for _, n := range cfg.Nodes {
		if _, ok := roleToBand[n.Role]; !ok {
			return nil, fmt.Errorf("node %s: role %q is not assigned to any band", n.ID, n.Role)
		}
	}

	locationToCol := make(map[string]int)
	for loc, axis := range cfg.Layout.Locations {
		col, err := parseColumn(axis)
		if err != nil {
			return nil, fmt.Errorf("location %s: %w", loc, err)
		}
		locationToCol[loc] = col
	}

	for _, n := range cfg.Nodes {
		if n.Location == "" {
			continue
		}
		if _, ok := locationToCol[n.Location]; !ok {
			return nil, fmt.Errorf("node %s: location %q is not defined in layout.locations", n.ID, n.Location)
		}
	}

	if err := validateCellDensity(cfg, roleToBand, locationToCol, bandNames); err != nil {
		return nil, err
	}

	return &LayoutInfo{
		RoleToBand:    roleToBand,
		LocationToCol: locationToCol,
		BandNames:     bandNames,
	}, nil
}

// validateCellDensity rejects configurations that put more than
// maxNodesPerCell nodes into a single (band, col) cell.
func validateCellDensity(cfg *Config, roleToBand, locationToCol map[string]int, bandNames []string) error {
	type cellKey struct{ col, band int }
	count := map[cellKey]int{}
	for _, n := range cfg.Nodes {
		bi, ok := roleToBand[n.Role]
		if !ok {
			continue
		}
		ci, ok := locationToCol[n.Location]
		if !ok {
			continue
		}
		count[cellKey{col: ci, band: bi}]++
	}

	// Inverse map col -> location names (multiple locations may share a
	// column, though the typical case is one-to-one).
	colToLocs := map[int][]string{}
	for loc, c := range locationToCol {
		colToLocs[c] = append(colToLocs[c], loc)
	}
	for c := range colToLocs {
		sort.Strings(colToLocs[c])
	}

	for k, n := range count {
		if n <= maxNodesPerCell {
			continue
		}
		band := bandNames[k.band]
		locs := strings.Join(colToLocs[k.col], "/")
		if locs == "" {
			locs = fmt.Sprintf("col %d", k.col)
		}
		return fmt.Errorf(
			"cell (band %q, location %s) holds %d nodes; limit is %d (split the band or the location)",
			band, locs, n, maxNodesPerCell)
	}
	return nil
}

// parseColumn converts a location axis value to a column index.
// Accepts integers ("0", "1", "2", ...) for now.
func parseColumn(s string) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, fmt.Errorf("expected integer, got %q", s)
	}
	if n < 0 {
		return 0, fmt.Errorf("column must be non-negative, got %d", n)
	}
	return n, nil
}
