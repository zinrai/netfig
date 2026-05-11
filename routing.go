package main

// Edge routing: deciding the waypoints of an orthogonal polyline
// when a straight line between two nodes would pass through a
// non-endpoint node, and detecting that condition. The book's
// stated preference (chapter 3-3, "直線がお薦め") is to keep edges
// straight when possible; this file provides the obstacle test
// (straightLineClear) that gates that choice and the per-scenario
// detour logic for the cases that need it.

const (
	// detourFaceOffset is how far from the node centre the entry/exit
	// face of an orthogonal detour is placed. Offsetting from the
	// centre prevents the detour's stubs from running along any
	// straight same-column edge that happens to share the column
	// (e.g. an iBGP vertical between a route-reflector and a core
	// router), which would otherwise be visually covered by the
	// detour's heavier stroke.
	detourFaceOffset = 20

	// defaultLaneStep is the per-cell-index lateral lane separation
	// applied throughout an orthogonal polyline. When several nodes
	// share a (band, col) cell and all have edges to the same target,
	// each polyline picks a different lane based on its source's
	// index in the cell, so the trunk segments do not pile up on top
	// of each other.
	//
	// laneStepFor scales this down for dense cells so that the
	// rightmost lane never falls outside the source rect.
	defaultLaneStep = 8
)

// maxLaneFromCenter is the largest lane offset (in either direction)
// that keeps a polyline endpoint inside the source rect under
// detourFaceOffset.
const maxLaneFromCenter = rectWidth/2 - detourFaceOffset

// point is one (x, y) waypoint of a polyline.
type point struct{ X, Y int }

// orthogonalRoute returns the waypoints for an orthogonal polyline
// that routes from f to t. It dispatches to one of three per-shape
// helpers; each helper handles its own obstacle-detection and
// detour decisions.
func orthogonalRoute(f, t *placed, all []placed) []point {
	if f.Band == t.Band {
		return routeSameBand(f, t, all)
	}
	if f.Col == t.Col {
		return routeSameCol(f, t, all)
	}
	return routeCrossCol(f, t, all)
}

// routeSameBand handles edges whose endpoints share a band. Without
// an intermediate-column obstacle in the row, a single horizontal
// face-to-face segment suffices. With an obstacle, the route
// detours through the band-gap above (or below, for the topmost
// band).
func routeSameBand(f, t *placed, all []placed) []point {
	if !hasIntermediateColObstacleInBand(f.Band, f.Col, t.Col, all) {
		return faceToFace(f, t)
	}
	return sameBandDetour(f, t)
}

// faceToFace returns the simple 2-point horizontal between the two
// nodes' inside faces.
func faceToFace(f, t *placed) []point {
	if f.Col < t.Col {
		return []point{
			{f.CX + f.HalfW, f.CY},
			{t.CX - t.HalfW, t.CY},
		}
	}
	return []point{
		{f.CX - f.HalfW, f.CY},
		{t.CX + t.HalfW, t.CY},
	}
}

// sameBandDetour returns the 4-point polyline that loops above the
// source band (or below, for band 0) to skirt an intermediate-column
// obstacle in the same row.
func sameBandDetour(f, t *placed) []point {
	detourY := marginY + f.Band*bandHeight
	sourceFace := f.CY - f.HalfH
	targetFace := t.CY - t.HalfH
	if f.Band == 0 {
		detourY = marginY + (f.Band+1)*bandHeight
		sourceFace = f.CY + f.HalfH
		targetFace = t.CY + t.HalfH
	}
	return []point{
		{f.CX, sourceFace},
		{f.CX, detourY},
		{t.CX, detourY},
		{t.CX, targetFace},
	}
}

// routeSameCol handles cross-band edges that share a column. Without
// an intermediate-band obstacle the route is a straight vertical
// (emitted as a 2-point polyline). With an obstacle, the polyline
// U-turns through the column-gap on the right.
func routeSameCol(f, t *placed, all []placed) []point {
	exitY, entryY, sourceGapY, targetGapY := bandGapYs(f, t)

	if !hasIntermediateObstacleInCol(f.Band, t.Band, f.Col, all) {
		return []point{
			{f.CX, exitY},
			{t.CX, entryY},
		}
	}

	lane := laneOffset(f, t)
	exitX := f.CX + detourFaceOffset + lane
	enterX := t.CX + detourFaceOffset + lane
	detourX := f.ColRightX + lane
	return []point{
		{exitX, exitY},
		{exitX, sourceGapY + lane},
		{detourX, sourceGapY + lane},
		{detourX, targetGapY + lane},
		{enterX, targetGapY + lane},
		{enterX, entryY},
	}
}

// routeCrossCol handles cross-band edges between different columns.
// Without intermediate-band obstacles, a 4-point Z-shape suffices.
// With an obstacle on either endpoint's column, the route traverses
// two band-gaps with the vertical between them at the column-gap.
func routeCrossCol(f, t *placed, all []placed) []point {
	exitY, entryY, sourceGapY, targetGapY := bandGapYs(f, t)
	lane := laneOffset(f, t)
	exitOff, enterOff, colGapX := crossColFaces(f, t, lane)

	fObs := hasIntermediateObstacleInCol(f.Band, t.Band, f.Col, all)
	tObs := hasIntermediateObstacleInCol(f.Band, t.Band, t.Col, all)
	if fObs || tObs {
		return []point{
			{f.CX + exitOff + lane, exitY},
			{f.CX + exitOff + lane, sourceGapY + lane},
			{colGapX, sourceGapY + lane},
			{colGapX, targetGapY + lane},
			{t.CX + enterOff + lane, targetGapY + lane},
			{t.CX + enterOff + lane, entryY},
		}
	}

	gapY := targetGapY + lane
	if f.Band > t.Band {
		gapY = sourceGapY + lane
	}
	return []point{
		{f.CX + exitOff + lane, exitY},
		{f.CX + exitOff + lane, gapY},
		{t.CX + enterOff + lane, gapY},
		{t.CX + enterOff + lane, entryY},
	}
}

// bandGapYs returns the y-coordinates the polyline needs for a
// cross-band edge: the source/target face the segment exits/enters,
// and the mid-y of the gap above/below each band.
func bandGapYs(f, t *placed) (exitY, entryY, sourceGapY, targetGapY int) {
	if f.Band < t.Band {
		exitY = f.CY + f.HalfH
		entryY = t.CY - t.HalfH
		sourceGapY = marginY + (f.Band+1)*bandHeight
		targetGapY = marginY + t.Band*bandHeight
		return
	}
	exitY = f.CY - f.HalfH
	entryY = t.CY + t.HalfH
	sourceGapY = marginY + f.Band*bandHeight
	targetGapY = marginY + (t.Band+1)*bandHeight
	return
}

// crossColFaces picks the exit/entry face offsets and the
// column-gap x for a cross-column polyline. The polyline reads as
// a single sweep in the direction from f to t, so the offsets are
// applied on the side that faces the target.
func crossColFaces(f, t *placed, lane int) (exitOff, enterOff, colGapX int) {
	if f.CX < t.CX {
		return detourFaceOffset, -detourFaceOffset, f.ColRightX + lane
	}
	return -detourFaceOffset, detourFaceOffset, f.ColLeftX + lane
}

// laneOffset returns the lateral lane offset for the polyline from f
// to t. The base offset is taken from whichever endpoint has more
// siblings in its cell — the "fan side". This handles both shapes of
// dense edge bundles symmetrically:
//
//   - many siblings → one target (e.g. several transits to one core
//     router): the source side fans out and the base lane is taken
//     from the source's cell index.
//   - one source → many siblings (e.g. one aggregation switch to a
//     fleet of servers): the target side fans out and the base lane
//     is taken from the target's cell index.
//
// When both endpoints have multiple siblings, the side with more
// siblings dominates; ties pick the source.
//
// On top of the base lane, a per-edge tie-breaker is added, derived
// from the non-fan side's (col, idx) plus the fan-side col. Without
// it, two visually distinct cases collapse onto the same lane:
//
//   - "one source → multiple cell-different targets" (e.g. one
//     Internet attachment feeding two pods, where each pod's border
//     leaf is its cell's index 0) — these share fan-side index but
//     differ in fan-side col.
//   - "one source → multiple targets in different cells with same
//     intra-cell index" — these share fan-side index but differ in
//     non-fan side col.
//
// The final lane is clamped to ±maxLaneFromCenter so even with the
// perturbation the polyline endpoint stays inside the source rect.
// Clamping can collapse adjacent lanes in the most extreme dense-cell
// fan-out cases; that is preferable to letting endpoints drift
// outside the rect boundary.
func laneOffset(f, t *placed) int {
	fan := f
	if t.CellCount > f.CellCount {
		fan = t
	}
	other := t
	if fan == t {
		other = f
	}

	base := 0
	if fan.CellCount >= 2 {
		step := laneStepFor(fan.CellCount)
		base = (2*fan.CellIdx - (fan.CellCount - 1)) * step / 2
	}
	perturbation := other.Col*4 + other.CellIdx*7 + fan.Col*3
	lane := base + perturbation

	if lane > maxLaneFromCenter {
		lane = maxLaneFromCenter
	} else if lane < -maxLaneFromCenter {
		lane = -maxLaneFromCenter
	}
	return lane
}

// laneStepFor returns a lane spacing for a cell of the given size.
// The constraint comes from keeping the outermost polyline endpoint
// inside the source rect under detourFaceOffset:
//
//	detourFaceOffset + (N-1)*step/2 <= rectWidth/2
//	=> step <= (rectWidth - 2*detourFaceOffset) / (N-1)
//
// Smaller cells use defaultLaneStep; dense cells get a step
// computed from this bound so the polylines remain well-formed.
func laneStepFor(cellCount int) int {
	if cellCount < 2 {
		return 0
	}
	safe := (rectWidth - 2*detourFaceOffset) / (cellCount - 1)
	if safe > defaultLaneStep {
		return defaultLaneStep
	}
	if safe < 1 {
		return 1
	}
	return safe
}

// hasIntermediateObstacleInCol reports whether any node sits in the
// given column, in a band strictly between fBand and tBand.
func hasIntermediateObstacleInCol(fBand, tBand, col int, all []placed) bool {
	lo, hi := fBand, tBand
	if lo > hi {
		lo, hi = hi, lo
	}
	for i := range all {
		p := &all[i]
		if p.Col == col && p.Band > lo && p.Band < hi {
			return true
		}
	}
	return false
}

// hasIntermediateColObstacleInBand reports whether any node sits in
// the given band, at a column strictly between fCol and tCol. This
// is the same-band counterpart to hasIntermediateObstacleInCol: it
// flags a same-band edge whose horizontal segment would cross a
// sibling's bounding box (e.g. a backbone-ring closure that crosses
// the backbone routers between its endpoints).
func hasIntermediateColObstacleInBand(band, fCol, tCol int, all []placed) bool {
	lo, hi := fCol, tCol
	if lo > hi {
		lo, hi = hi, lo
	}
	for i := range all {
		p := &all[i]
		if p.Band == band && p.Col > lo && p.Col < hi {
			return true
		}
	}
	return false
}

// straightLineClear reports whether the straight line connecting the
// centres of f and t is free of non-endpoint nodes.
func straightLineClear(f, t *placed, all []placed) bool {
	for i := range all {
		p := &all[i]
		if p.ID == f.ID || p.ID == t.ID {
			continue
		}
		bx, by := p.CX-p.HalfW, p.CY-p.HalfH
		bw, bh := p.HalfW*2, p.HalfH*2
		if segIntersectsRect(f.CX, f.CY, t.CX, t.CY, bx, by, bw, bh) {
			return false
		}
	}
	return true
}

// segIntersectsRect reports whether segment (x1,y1)-(x2,y2) crosses
// the interior of rectangle [bx, bx+bw] x [by, by+bh] (open-segment
// test: endpoint-only touches do not count).
func segIntersectsRect(x1, y1, x2, y2, bx, by, bw, bh int) bool {
	corners := [4][2]int{
		{bx, by}, {bx + bw, by}, {bx + bw, by + bh}, {bx, by + bh},
	}
	for i := 0; i < 4; i++ {
		a, b := corners[i], corners[(i+1)%4]
		if segCrosses(x1, y1, x2, y2, a[0], a[1], b[0], b[1]) {
			return true
		}
	}
	return false
}

// segCrosses reports whether the open segments (a,b) and (c,d) cross.
func segCrosses(ax, ay, bx, by, cx, cy, dx, dy int) bool {
	d := (bx-ax)*(dy-cy) - (by-ay)*(dx-cx)
	if d == 0 {
		return false
	}
	t := float64((cx-ax)*(dy-cy)-(cy-ay)*(dx-cx)) / float64(d)
	u := float64((cx-ax)*(by-ay)-(cy-ay)*(bx-ax)) / float64(d)
	return t > 1e-9 && t < 1-1e-9 && u > 1e-9 && u < 1-1e-9
}
