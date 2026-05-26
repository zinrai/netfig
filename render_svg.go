package main

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// SVG emission: turning placed nodes and routed edges into the final
// SVG document. netfig is the renderer — there is no external layout
// engine to delegate to. The book's rules give a complete (band,
// location) grid, so coordinates are computed directly from the
// validated layout (placement.go), and edges are drawn as straight
// lines wherever a straight line is clear of non-endpoint nodes —
// the book's stated preference in chapter 3-3 ("直線がお薦め").
// Where a straight line would pass through a non-endpoint node, the
// edge is routed orthogonally (routing.go).

// GenerateSVG converts a validated configuration into SVG source.
func GenerateSVG(cfg *Config, info *LayoutInfo) string {
	patterns := buildPatternIndex(cfg)
	nodes, cols := placeNodes(cfg, info, patterns)
	groups := placeGroups(cfg, info, cols)
	byID := make(map[string]*placed, len(nodes))
	for i := range nodes {
		byID[nodes[i].ID] = &nodes[i]
	}

	height := marginY*2 + len(info.BandNames)*bandHeight

	var b strings.Builder
	emitSVGHeader(&b, cols.totalWidth, height)

	for _, g := range groups {
		emitGroup(&b, g)
	}

	for i, l := range cfg.Links {
		f, ok1 := byID[l.From]
		t, ok2 := byID[l.To]
		if !ok1 || !ok2 {
			continue
		}
		emitEdge(&b, cfg, l, f, t, nodes, patterns.siblingOf(i))
	}
	for _, n := range nodes {
		emitNode(&b, n)
	}

	b.WriteString("</svg>\n")
	return b.String()
}

func emitSVGHeader(b *strings.Builder, width, height int) {
	fmt.Fprintln(b, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintf(b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n",
		width, height, width, height)
	fmt.Fprintln(b, `  <style>`)
	fmt.Fprintln(b, `    .node { fill: white; stroke: black; stroke-width: 1; }`)
	fmt.Fprintln(b, `    .node-label { font-family: sans-serif; font-size: 11px; text-anchor: middle; dominant-baseline: central; }`)
	fmt.Fprintln(b, `    .edge { stroke: black; fill: none; }`)
	fmt.Fprintln(b, `    .edge-label { font-family: sans-serif; font-size: 10px; text-anchor: middle; }`)
	fmt.Fprintln(b, `    .group { fill: #f2f2f2; stroke: none; }`)
	fmt.Fprintln(b, `    .group-label { font-family: sans-serif; font-size: 11px; fill: #555; }`)
	fmt.Fprintln(b, `  </style>`)
}

// emitGroup writes the filled rounded rectangle for one group, plus a
// small label anchored at the top-left of the rectangle. The rect is
// drawn without an outline and uses a light grey fill so it sits
// visually under the node and link layer; that combination is what
// keeps the group boundary from competing with link lines.
func emitGroup(b *strings.Builder, g placedGroup) {
	fmt.Fprintf(b, `  <rect class="group" x="%d" y="%d" width="%d" height="%d" rx="%d" ry="%d"/>`+"\n",
		g.X, g.Y, g.W, g.H, groupCornerRadius, groupCornerRadius)
	if g.Label == "" {
		return
	}
	fmt.Fprintf(b, `  <text class="group-label" x="%d" y="%d">%s</text>`+"\n",
		g.X+8, g.Y+14, escapeText(g.Label))
}

func emitNode(b *strings.Builder, p placed) {
	switch p.Shape {
	case "ellipse":
		fmt.Fprintf(b, `  <ellipse class="node" cx="%d" cy="%d" rx="%d" ry="%d"/>`+"\n",
			p.CX, p.CY, ellipseRx, ellipseRy)
	default:
		x := p.CX - rectWidth/2
		y := p.CY - rectHeight/2
		fmt.Fprintf(b, `  <rect class="node" x="%d" y="%d" width="%d" height="%d"/>`+"\n",
			x, y, rectWidth, rectHeight)
	}
	fmt.Fprintf(b, `  <text class="node-label" x="%d" y="%d">%s</text>`+"\n",
		p.CX, p.CY, escapeText(p.Label))
}

// emitEdge writes one link. The default is a straight <line> between
// the two node centres; the rect/ellipse fills overlay the segment so
// only the portion outside the boxes is visible. If the straight line
// would pass through a non-endpoint node, an orthogonal rounded-corner
// <path> is emitted instead.
//
// When the link is part of a sibling group declared by a pattern
// (two or more links sharing the same endpoints and the same pattern
// name), the route is shifted perpendicularly so the siblings render
// as parallel runs rather than overlapping each other. Each link's
// label is rendered on its own run; the writer is responsible for
// choosing labels that fit. The placement layer widens the column
// spacing in proportion to the maximum sibling count so labels have
// room to render without intruding on adjacent columns.
//
// Straight-line emission is bypassed for any sibling because
// offsetting a free diagonal does not produce a clean parallel pair —
// orthogonal routing with a perpendicular lane offset does.
func emitEdge(b *strings.Builder, cfg *Config, l Link, f, t *placed, all []placed, sib siblingInfo) {
	attrs := edgeStyleAttrs(cfg, l)

	if sib.Size == 1 && straightLineClear(f, t, all) {
		emitStraightEdge(b, l, f, t, attrs)
		return
	}
	pairLane := pairLaneOffset(sib.Index)
	emitPolylineEdge(b, l, orthogonalRoute(f, t, all, pairLane), attrs)
}

// emitStraightEdge writes a straight <line> and (if any) a label
// at the segment's midpoint.
func emitStraightEdge(b *strings.Builder, l Link, f, t *placed, attrs string) {
	fmt.Fprintf(b, `  <line class="edge" x1="%d" y1="%d" x2="%d" y2="%d"%s/>`+"\n",
		f.CX, f.CY, t.CX, t.CY, attrs)
	if l.Label == "" {
		return
	}
	mx, my := (f.CX+t.CX)/2, (f.CY+t.CY)/2
	fmt.Fprintf(b, `  <text class="edge-label" x="%d" y="%d">%s</text>`+"\n",
		mx, my-3, escapeText(l.Label))
}

// emitPolylineEdge writes a rounded-corner <path> for the routed
// waypoints and (if any) a label placed on the longest horizontal
// segment so the text reads cleanly. The path uses straight L
// commands between segments and a quadratic Q at each interior
// vertex; the control point of each Q sits at the original vertex,
// so the curve is tangent to the incoming and outgoing segments and
// reads as a softened right angle rather than a free curve. Where a
// segment is shorter than 2*edgeCornerRadius, the effective radius is
// clamped so the rounded corner cannot consume more than half the
// segment.
func emitPolylineEdge(b *strings.Builder, l Link, pts []point, attrs string) {
	fmt.Fprintf(b, `  <path class="edge" d="%s"%s/>`+"\n",
		roundedPath(pts), attrs)
	if l.Label == "" {
		return
	}
	mx, my := longestHorizontalMidpoint(pts)
	fmt.Fprintf(b, `  <text class="edge-label" x="%d" y="%d">%s</text>`+"\n",
		mx, my-3, escapeText(l.Label))
}

// longestHorizontalMidpoint returns the midpoint of the longest
// horizontal segment in pts, falling back to the first point if no
// horizontal segment exists.
func longestHorizontalMidpoint(pts []point) (int, int) {
	mx, my := pts[0].X, pts[0].Y
	bestLen := 0
	for i := 1; i < len(pts); i++ {
		if pts[i].Y != pts[i-1].Y {
			continue
		}
		dl := absInt(pts[i].X - pts[i-1].X)
		if dl <= bestLen {
			continue
		}
		bestLen = dl
		mx, my = (pts[i].X+pts[i-1].X)/2, pts[i].Y
	}
	return mx, my
}

func edgeStyleAttrs(cfg *Config, l Link) string {
	parts := []string{}
	if lk, ok := cfg.Legend.LineKinds[l.Kind]; ok {
		switch lk.Style {
		case "dashed":
			parts = append(parts, `stroke-dasharray="6,4"`)
		case "dotted":
			parts = append(parts, `stroke-dasharray="2,2"`)
		}
		if lk.Width >= 2 {
			parts = append(parts, fmt.Sprintf(`stroke-width="%d"`, lk.Width))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

func escapeText(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
