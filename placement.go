package main

// Node placement: turning the validated (band, location) grid into
// concrete pixel coordinates. The diagram is laid out as a grid of
// (band, location) cells. A cell may contain more than one node when
// the YAML places several nodes at the same (role's-band, location)
// pair (e.g. two transit providers in the same city, both with role
// `transit`). Multi-node cells spread the nodes side-by-side and
// widen the affected column to fit them; single-node cells keep the
// default column width.

const (
	bandHeight = 140 // vertical pixels per band row
	marginX    = 80
	marginY    = 60

	// rectWidth and ellipseRx are sized to fit the longest labels
	// realistically expected on a network diagram — role identifiers
	// plus a brief qualifier ("site-b-edge1 (transit)", "customer-a
	// AS65100"). A label longer than these widths is a signal that
	// the writer's label is too verbose for the diagram and should
	// be shortened, not that the node should grow further.
	rectWidth  = 160
	rectHeight = 40
	ellipseRx  = 90
	ellipseRy  = 28

	intraCellGap     = 15 // gap between sibling nodes in the same cell
	intraCellPadding = 20 // empty space inside a column on each side

	// defaultColWidth is the width of a column carrying a single
	// node; it leaves intraCellPadding worth of empty space on each
	// side of the widest possible single shape (an ellipse).
	defaultColWidth = ellipseRx*2 + 2*intraCellPadding
)

// placed is a node enriched with computed pixel coordinates.
type placed struct {
	ID        string
	Shape     string // "rect" | "ellipse"
	Label     string
	Band      int
	Col       int
	CX        int
	CY        int
	HalfW     int
	HalfH     int
	ColLeftX  int // x of the left edge of this node's column
	ColRightX int // x of the right edge of this node's column
	CellIdx   int // 0-based position of this node within its (band, col) cell
	CellCount int // total number of nodes in this node's (band, col) cell
}

// cellKey identifies one (band, col) cell.
type cellKey struct{ col, band int }

// cellMeta holds per-cell layout state: which nodes belong to it
// (in YAML order), the max shape width among them, and the
// intra-cell step (= shape width + gap).
type cellMeta struct {
	nodeIDs []string
	shapeW  int
	step    int
}

// columnLayout holds per-column geometry derived from the cells.
type columnLayout struct {
	widths     []int
	centers    []int
	totalWidth int
}

// placeNodes assigns pixel coordinates to every node, derived from
// its (band, col) pair in info. It also returns the columnLayout used
// for the placement, so other renderers (groups, edge labels) can map
// (band, col) cells to pixel coordinates from the same source of
// truth.
//
// The pattern index is consulted so that columns linked by sibling
// (parallel-run) groups get extra horizontal spacing. A pair of
// columns connected by N parallel siblings needs room for N stacked
// labels and N offset paths in the gap between them; widening the
// gap here is what keeps those labels from spilling into the
// neighbouring columns at render time.
//
// Within a (band, col) cell, nodes are spread side-by-side in YAML
// order. The cell's intra-cell step is sized to fit the widest shape
// in the cell — rect cells keep step at rectWidth+gap; ellipse cells
// (which are wider than rects under the current legend geometry)
// expand to ellipseRx*2+gap so siblings do not overlap. Single-node
// cells keep the default column width.
func placeNodes(cfg *Config, info *LayoutInfo, patterns *patternIndex) ([]placed, columnLayout) {
	cells, nodeIdx, maxCol := groupNodesByCell(cfg, info)
	gapDemand := siblingGapDemand(cfg, info, patterns, maxCol)
	cols := computeColumnLayout(cells, maxCol, gapDemand)
	return placeAll(cfg, info, cells, nodeIdx, cols), cols
}

// siblingGapDemand returns, for each inter-column gap, the largest
// sibling-group size whose links cross that gap. gap[i] is the
// horizontal space between column i and column i+1; the last entry
// (gap[maxCol]) is unused.
//
// A pair of columns linked by N parallel siblings needs room in the
// gap between them for N stacked labels and N offset paths. Widening
// the gap rather than the columns themselves is what keeps the gap
// available for labels: a wider column would push its node toward
// its centre, but a wider gap pushes the next column out, freeing
// the inter-node space where the labels actually live.
func siblingGapDemand(cfg *Config, info *LayoutInfo, patterns *patternIndex, maxCol int) []int {
	out := make([]int, maxCol+1)
	for i := range out {
		out[i] = 1
	}
	for i, l := range cfg.Links {
		sib := patterns.siblingOf(i)
		if sib.Size <= 1 {
			continue
		}
		fc, okF := info.LocationToCol[locationOf(cfg, l.From)]
		tc, okT := info.LocationToCol[locationOf(cfg, l.To)]
		if !okF || !okT || fc == tc {
			continue
		}
		lo, hi := fc, tc
		if lo > hi {
			lo, hi = hi, lo
		}
		// A link from fc to tc crosses every gap in between.
		for g := lo; g < hi; g++ {
			if sib.Size > out[g] {
				out[g] = sib.Size
			}
		}
	}
	return out
}

// locationOf returns the location of the node with the given id, or
// "" if not found. Used to map a link's endpoint to a column.
func locationOf(cfg *Config, id string) string {
	for _, n := range cfg.Nodes {
		if n.ID == id {
			return n.Location
		}
	}
	return ""
}

// groupNodesByCell collects nodes into their (band, col) cells, in
// YAML order, and records each node's intra-cell index. It also
// records each cell's max shape width and pre-computes the cell step.
func groupNodesByCell(cfg *Config, info *LayoutInfo) (map[cellKey]*cellMeta, map[string]int, int) {
	cells := map[cellKey]*cellMeta{}
	nodeIdx := map[string]int{}
	maxCol := 0
	for _, n := range cfg.Nodes {
		bi := info.RoleToBand[n.Role]
		ci := info.LocationToCol[n.Location]
		k := cellKey{col: ci, band: bi}
		c := cells[k]
		if c == nil {
			c = &cellMeta{shapeW: rectWidth}
			cells[k] = c
		}
		nodeIdx[n.ID] = len(c.nodeIDs)
		c.nodeIDs = append(c.nodeIDs, n.ID)
		if sw := shapeWidthOf(cfg, n.Role); sw > c.shapeW {
			c.shapeW = sw
		}
		if ci > maxCol {
			maxCol = ci
		}
	}
	for _, c := range cells {
		c.step = c.shapeW + intraCellGap
	}
	return cells, nodeIdx, maxCol
}

// shapeWidthOf returns the rendered width of the symbol assigned to
// the given role.
func shapeWidthOf(cfg *Config, role string) int {
	if cfg.Legend.Symbols[role].Shape == "ellipse" {
		return ellipseRx * 2
	}
	return rectWidth
}

// siblingExtraPerStep is the additional horizontal padding added to
// a column for each parallel-run sibling beyond the first. The value
// reserves enough space inside the column for a sibling-group's
// stacked labels and parallel lanes when one of its links exits or
// enters this column. A column with no parallel runs adds nothing.
const siblingExtraPerStep = 48

// computeColumnLayout sizes each column to fit the widest cell in it
// and computes column centres from the cumulative widths. Where
// sibling links cross the gap between two columns, extra horizontal
// space is inserted after the source column so the parallel runs
// and their labels have room in the gap.
func computeColumnLayout(cells map[cellKey]*cellMeta, maxCol int, gapDemand []int) columnLayout {
	colCount := maxCol + 1

	maxContent := make([]int, colCount)
	for c := range maxContent {
		maxContent[c] = rectWidth
	}
	for k, cell := range cells {
		n := len(cell.nodeIDs)
		content := n*cell.shapeW + (n-1)*intraCellGap
		if content > maxContent[k.col] {
			maxContent[k.col] = content
		}
	}

	widths := make([]int, colCount)
	for c, m := range maxContent {
		w := m + 2*intraCellPadding
		if w < defaultColWidth {
			w = defaultColWidth
		}
		widths[c] = w
	}

	centers := make([]int, colCount)
	x := marginX
	for c, w := range widths {
		centers[c] = x + w/2
		x += w
		// Insert extra horizontal space after this column for any
		// sibling-group crossing the gap to the next column. A solo
		// link in this gap (demand 1) gets no extra; a pair gets one
		// step; a triplet gets two steps.
		if c < len(gapDemand) && c < colCount-1 && gapDemand[c] > 1 {
			x += (gapDemand[c] - 1) * siblingExtraPerStep
		}
	}

	return columnLayout{widths: widths, centers: centers, totalWidth: x + marginX}
}

// placeAll builds the placed slice from the grouped cells and column
// layout. Each node's intra-cell index was cached during grouping,
// so no scan is required here.
func placeAll(cfg *Config, info *LayoutInfo, cells map[cellKey]*cellMeta, nodeIdx map[string]int, cols columnLayout) []placed {
	out := make([]placed, 0, len(cfg.Nodes))
	for _, n := range cfg.Nodes {
		bi := info.RoleToBand[n.Role]
		ci := info.LocationToCol[n.Location]
		cell := cells[cellKey{col: ci, band: bi}]
		idx := nodeIdx[n.ID]
		nCount := len(cell.nodeIDs)
		offset := (2*idx - (nCount - 1)) * cell.step / 2
		out = append(out, makePlaced(cfg, n, bi, ci, idx, nCount, offset, cols.widths[ci], cols.centers[ci]))
	}
	return out
}

// makePlaced builds a placed struct from the resolved layout
// inputs. The half-extents are set from the symbol's shape.
func makePlaced(cfg *Config, n Node, band, col, idx, nCount, offset, colWidth, colCenter int) placed {
	shape := cfg.Legend.Symbols[n.Role].Shape
	label := n.Label
	if label == "" {
		label = n.ID
	}
	p := placed{
		ID:        n.ID,
		Shape:     shape,
		Label:     label,
		Band:      band,
		Col:       col,
		CX:        colCenter + offset,
		CY:        marginY + band*bandHeight + bandHeight/2,
		ColLeftX:  colCenter - colWidth/2,
		ColRightX: colCenter + colWidth/2,
		CellIdx:   idx,
		CellCount: nCount,
	}
	if shape == "ellipse" {
		p.HalfW, p.HalfH = ellipseRx, ellipseRy
	} else {
		p.HalfW, p.HalfH = rectWidth/2, rectHeight/2
	}
	return p
}
