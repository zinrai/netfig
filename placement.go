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
	bandHeight       = 140 // vertical pixels per band row
	defaultColWidth  = 140 // horizontal pixels per single-node column
	marginX          = 80
	marginY          = 60
	rectWidth        = 100
	rectHeight       = 40
	ellipseRx        = 60
	ellipseRy        = 24
	intraCellGap     = 10 // gap between sibling nodes in the same cell
	intraCellPadding = 20 // empty space inside a column on each side
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
// its (band, col) pair in info. It also returns the total SVG width
// required, since multi-node cells widen their column.
//
// Within a (band, col) cell, nodes are spread side-by-side in YAML
// order. The cell's intra-cell step is sized to fit the widest shape
// in the cell — rect cells keep step at rectWidth+gap; ellipse cells
// (which are wider than rects under the current legend geometry)
// expand to ellipseRx*2+gap so siblings do not overlap. Single-node
// cells keep the default column width.
func placeNodes(cfg *Config, info *LayoutInfo) ([]placed, int) {
	cells, nodeIdx, maxCol := groupNodesByCell(cfg, info)
	cols := computeColumnLayout(cells, maxCol)
	return placeAll(cfg, info, cells, nodeIdx, cols), cols.totalWidth
}

// groupNodesByCell collects nodes into their (band, col) cells, in
// YAML order, and records each node's intra-cell index. It also
// records each cell's max shape width and pre-computes the cell step.
func groupNodesByCell(cfg *Config, info *LayoutInfo) (map[cellKey]*cellMeta, map[string]int, int) {
	cells := map[cellKey]*cellMeta{}
	nodeIdx := map[string]int{}
	maxCol := 0
	for _, n := range cfg.Nodes {
		bi, okB := info.RoleToBand[n.Role]
		if !okB {
			continue
		}
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

// computeColumnLayout sizes each column to fit the widest cell in it
// and computes column centres from the cumulative widths.
func computeColumnLayout(cells map[cellKey]*cellMeta, maxCol int) columnLayout {
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
	}

	return columnLayout{widths: widths, centers: centers, totalWidth: x + marginX}
}

// placeAll builds the placed slice from the grouped cells and column
// layout. Each node's intra-cell index was cached during grouping,
// so no scan is required here.
func placeAll(cfg *Config, info *LayoutInfo, cells map[cellKey]*cellMeta, nodeIdx map[string]int, cols columnLayout) []placed {
	out := make([]placed, 0, len(cfg.Nodes))
	for _, n := range cfg.Nodes {
		bi, okB := info.RoleToBand[n.Role]
		if !okB {
			continue
		}
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
