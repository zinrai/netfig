package main

import "testing"

func TestLaneOffset_DistinctForAllSiblingsAtCellCap(t *testing.T) {
	// At the per-cell cap, every sibling on the fan side must get a
	// distinct lane offset so their orthogonal polylines don't pile
	// up. This case fans on the source side (single-target).
	target := &placed{CellIdx: 0, CellCount: 1}
	seen := map[int]int{}
	for i := 0; i < maxNodesPerCell; i++ {
		f := &placed{CellIdx: i, CellCount: maxNodesPerCell}
		off := laneOffset(f, target)
		if prev, dup := seen[off]; dup {
			t.Errorf("idx %d collides with idx %d at offset %d", i, prev, off)
		}
		seen[off] = i
	}
	if len(seen) != maxNodesPerCell {
		t.Errorf("expected %d distinct lane offsets, got %d", maxNodesPerCell, len(seen))
	}
}

func TestLaneOffset_TargetSideFanOut(t *testing.T) {
	// One source → many siblings: the lane offset must come from the
	// target's cell index so each polyline gets a distinct lane.
	source := &placed{CellIdx: 0, CellCount: 1}
	seen := map[int]int{}
	for i := 0; i < maxNodesPerCell; i++ {
		t2 := &placed{CellIdx: i, CellCount: maxNodesPerCell}
		off := laneOffset(source, t2)
		if prev, dup := seen[off]; dup {
			t.Errorf("idx %d collides with idx %d at offset %d", i, prev, off)
		}
		seen[off] = i
	}
	if len(seen) != maxNodesPerCell {
		t.Errorf("expected %d distinct lane offsets, got %d", maxNodesPerCell, len(seen))
	}
}

func TestLaneStepFor_KeepsOutermostLaneInsideRect(t *testing.T) {
	// For every cell size up to the cap, the outermost source's
	// polyline endpoint (= cx + detourFaceOffset + lane) must stay
	// inside its rect.
	target := &placed{CellIdx: 0, CellCount: 1}
	for n := 2; n <= maxNodesPerCell; n++ {
		f := &placed{CellIdx: n - 1, CellCount: n}
		off := laneOffset(f, target)
		if detourFaceOffset+off > rectWidth/2 {
			t.Errorf("cellCount=%d: detourFaceOffset(%d)+lane(%d)=%d exceeds rectWidth/2=%d",
				n, detourFaceOffset, off, detourFaceOffset+off, rectWidth/2)
		}
	}
}
