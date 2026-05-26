package main

import (
	"strings"
	"testing"
)

// TestRoundedPath_TwoPointsIsStraight documents that a 2-point input
// becomes a single M ... L command with no curves. There is nothing
// to round on a 2-point line, so the path is straight.
func TestRoundedPath_TwoPointsIsStraight(t *testing.T) {
	pts := []point{{X: 0, Y: 0}, {X: 100, Y: 0}}
	got := roundedPath(pts)
	if !strings.HasPrefix(got, "M") {
		t.Errorf("expected path to start with M, got: %s", got)
	}
	if strings.Contains(got, "Q") {
		t.Errorf("expected no curve commands on a 2-point line, got: %s", got)
	}
}

// TestRoundedPath_InteriorVertexBecomesQuadratic documents the
// rounding rule: an interior vertex of an orthogonal polyline turns
// into a Q quadratic curve whose control point is the original
// vertex. Rounding lets the reader's eye continue past the bend
// without re-acquiring the line on the far side.
func TestRoundedPath_InteriorVertexBecomesQuadratic(t *testing.T) {
	pts := []point{
		{X: 0, Y: 0},
		{X: 100, Y: 0},
		{X: 100, Y: 100},
	}
	got := roundedPath(pts)
	if !strings.Contains(got, "Q") {
		t.Errorf("expected at least one Q curve for a single bend, got: %s", got)
	}
}

// TestCornerRadiusFor_ShortSegmentClampsRadius documents that a
// segment shorter than the standard edge radius does not pull the
// rounding past half its length. Otherwise two adjacent corners would
// consume each other's straight run.
func TestCornerRadiusFor_ShortSegmentClampsRadius(t *testing.T) {
	if r := cornerRadiusFor(point{X: 0, Y: 0}, point{X: 6, Y: 0}); r > 3 {
		t.Errorf("expected corner radius to be clamped to <=3 on a 6-pixel segment, got %d", r)
	}
}

// TestRoundedPath_CornerStaysOnAxis verifies that the entry and exit
// of the quadratic curve are both axis-aligned with their adjacent
// straight runs: when the incoming segment is horizontal, the entry
// shares its Y; when the outgoing segment is vertical, the exit
// shares its X. This is what makes the rounded corner read as a
// softened right angle and not a free curve.
func TestRoundedPath_CornerStaysOnAxis(t *testing.T) {
	pts := []point{
		{X: 0, Y: 0},
		{X: 100, Y: 0},
		{X: 100, Y: 100},
	}
	got := roundedPath(pts)
	// Path looks like: M0,0 L92,0 Q100,0 100,8 L100,100
	// The L before Q must share Y=0 with the incoming horizontal.
	if !strings.Contains(got, "L92,0") {
		t.Errorf("expected entry point to share incoming Y=0, got: %s", got)
	}
	// The point after Q must share X=100 with the outgoing vertical.
	if !strings.Contains(got, "100,8") {
		t.Errorf("expected exit point to share outgoing X=100, got: %s", got)
	}
}
