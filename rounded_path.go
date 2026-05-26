package main

// SVG path encoding: turning an orthogonal waypoint list produced by
// routing.go into the "d" attribute of an SVG <path>, with rounded
// corners at every interior vertex. The conversion is purely
// mechanical — input is []point, output is a string. It is kept
// separate from render_svg.go (which composes the SVG document and
// knows about Config, Link, placed) so the SVG-syntax machinery does
// not mix with the document-composition layer.

import (
	"fmt"
	"strings"
)

// edgeCornerRadius is the corner radius applied to bends in an
// orthogonal edge path. A sharp right angle interrupts the reader's
// tracking of the line at every bend; rounding the corner lets the
// eye continue past it without re-acquiring the line on the far
// side. The numeric value is small relative to bandHeight so the
// corner is a clear cue without softening the path's overall
// structure.
const edgeCornerRadius = 8

// roundedPath converts an orthogonal waypoint list into an SVG path
// "d" attribute with rounded corners at every interior vertex. A
// 2-point input becomes a single straight line; longer inputs get
// quadratic curves at each bend.
func roundedPath(pts []point) string {
	if len(pts) < 2 {
		return ""
	}
	if len(pts) == 2 {
		return fmt.Sprintf("M%d,%d L%d,%d", pts[0].X, pts[0].Y, pts[1].X, pts[1].Y)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "M%d,%d", pts[0].X, pts[0].Y)
	for i := 1; i < len(pts)-1; i++ {
		prev, cur, next := pts[i-1], pts[i], pts[i+1]
		r1 := cornerRadiusFor(prev, cur)
		r2 := cornerRadiusFor(cur, next)
		// Entry point: along (prev -> cur), r1 short of cur.
		ex := cur.X - sign(cur.X-prev.X)*r1
		ey := cur.Y - sign(cur.Y-prev.Y)*r1
		// Exit point: along (cur -> next), r2 past cur.
		xx := cur.X + sign(next.X-cur.X)*r2
		xy := cur.Y + sign(next.Y-cur.Y)*r2
		fmt.Fprintf(&sb, " L%d,%d Q%d,%d %d,%d", ex, ey, cur.X, cur.Y, xx, xy)
	}
	last := pts[len(pts)-1]
	fmt.Fprintf(&sb, " L%d,%d", last.X, last.Y)
	return sb.String()
}

// cornerRadiusFor returns the effective rounding radius for the
// corner shared by the segment a->b. The rounded corner is meant to
// remain a clear local cue: the radius is clamped to half the segment
// length so a short segment does not get a corner consuming the whole
// run.
func cornerRadiusFor(a, b point) int {
	d := absInt(a.X-b.X) + absInt(a.Y-b.Y) // orthogonal, so one is zero
	r := edgeCornerRadius
	if r*2 > d {
		r = d / 2
	}
	return r
}

// sign returns -1, 0, or 1 reflecting the sign of n.
func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}
