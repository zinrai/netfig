package main

// Group rendering: turning the validated group declarations into pixel
// bounding boxes. Groups span a rectangular region of the (band,
// location) grid and are drawn as filled rectangles under every node
// and link. An outline-style boundary would compete with link lines
// for the reader's attention; the filled option does not, so netfig
// adopts the filled option and gives the rectangle rounded corners to
// reinforce the perception of a single bounded region.

import (
	"fmt"
	"sort"
)

const (
	// groupCornerRadius is the corner radius of the rounded rectangle
	// emitted for a group. Rounded corners reinforce the perception
	// of the rectangle as a single bounded region. The numeric value
	// is small relative to bandHeight so the corner remains a clear
	// cue without dominating the geometry.
	groupCornerRadius = 12

	// groupPadX is the horizontal padding added around the cells the
	// group covers. The group rectangle reaches groupPadX pixels
	// outside the bounding columns' edges so nodes at the column
	// boundaries are not flush with the group's edge.
	groupPadX = 12

	// groupPadY is the vertical padding added above and below the
	// cells the group covers. The padding is taken from the band's
	// gap, so the rectangle does not overlap an adjacent band's
	// node row.
	groupPadY = 24
)

// placedGroup is a group enriched with computed pixel coordinates.
type placedGroup struct {
	Name  string
	Label string
	X, Y  int
	W, H  int
}

// ValidateGroups checks that every group references locations and
// bands that exist in the layout. Like every other check in netfig,
// each error returns rather than warns: a group that references an
// unknown cell is an ambiguous input and the tool refuses to guess
// what cells it meant.
func ValidateGroups(cfg *Config, info *LayoutInfo) error {
	for i, g := range cfg.Groups {
		if g.Name == "" {
			return fmt.Errorf("groups[%d] has empty name", i)
		}
		if len(g.Locations) == 0 {
			return fmt.Errorf("group %q has empty locations; at least one location is required", g.Name)
		}
		for _, loc := range g.Locations {
			if _, ok := info.LocationToCol[loc]; !ok {
				return fmt.Errorf("group %q references unknown location %q", g.Name, loc)
			}
		}
		for _, band := range g.Bands {
			if !bandExists(info.BandNames, band) {
				return fmt.Errorf("group %q references unknown band %q", g.Name, band)
			}
		}
	}
	return nil
}

// bandExists reports whether name appears in bandNames.
func bandExists(bandNames []string, name string) bool {
	for _, b := range bandNames {
		if b == name {
			return true
		}
	}
	return false
}

// placeGroups computes pixel bounding boxes for every group from the
// validated layout and the column widths derived from node placement.
// Groups are returned in YAML order so the render layer can emit them
// in that order; with no outline and a uniform fill colour, ordering
// only matters when groups overlap, which the validator accepts (no
// explicit overlap rule) but the author should avoid.
func placeGroups(cfg *Config, info *LayoutInfo, cols columnLayout) []placedGroup {
	out := make([]placedGroup, 0, len(cfg.Groups))
	for _, g := range cfg.Groups {
		bands := groupBands(g, info)
		cols2 := groupCols(g, info)
		if len(bands) == 0 || len(cols2) == 0 {
			continue
		}
		topBand, botBand := bands[0], bands[len(bands)-1]
		leftCol, rightCol := cols2[0], cols2[len(cols2)-1]

		top := marginY + topBand*bandHeight + (bandHeight-rectHeight)/2 - groupPadY
		bot := marginY + botBand*bandHeight + (bandHeight+rectHeight)/2 + groupPadY
		left := cols.centers[leftCol] - cols.widths[leftCol]/2 + groupPadX
		right := cols.centers[rightCol] + cols.widths[rightCol]/2 - groupPadX

		out = append(out, placedGroup{
			Name:  g.Name,
			Label: groupDisplayLabel(g),
			X:     left,
			Y:     top,
			W:     right - left,
			H:     bot - top,
		})
	}
	return out
}

// groupBands returns the band indices the group covers, ascending. An
// empty Bands list means the group covers every band.
func groupBands(g Group, info *LayoutInfo) []int {
	if len(g.Bands) == 0 {
		all := make([]int, len(info.BandNames))
		for i := range info.BandNames {
			all[i] = i
		}
		return all
	}
	idx := make([]int, 0, len(g.Bands))
	for i, name := range info.BandNames {
		for _, b := range g.Bands {
			if name == b {
				idx = append(idx, i)
				break
			}
		}
	}
	sort.Ints(idx)
	return idx
}

// groupCols returns the column indices the group covers, ascending.
func groupCols(g Group, info *LayoutInfo) []int {
	idx := make([]int, 0, len(g.Locations))
	for _, loc := range g.Locations {
		if c, ok := info.LocationToCol[loc]; ok {
			idx = append(idx, c)
		}
	}
	sort.Ints(idx)
	return idx
}

// groupDisplayLabel returns the label to render for a group, falling
// back to the group's name when no explicit label is set.
func groupDisplayLabel(g Group) string {
	if g.Label != "" {
		return g.Label
	}
	return g.Name
}
