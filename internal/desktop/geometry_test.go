package desktop

import "testing"

func TestResizeEdges(t *testing.T) {
	if got := hitResizeEdge(1, 1, 340, 300); got != edgeLeft|edgeTop {
		t.Fatalf("top-left hit = %v", got)
	}
	if got := hitResizeEdge(339, 299, 340, 300); got != edgeRight|edgeBottom {
		t.Fatalf("bottom-right hit = %v", got)
	}
	if got := hitResizeEdge(100, 50, 340, 300); got != 0 {
		t.Fatalf("content must not be a resize zone: %v", got)
	}
}

func TestResizeBoundsAnchorsAndLimits(t *testing.T) {
	start := bounds{-600, 100, -260, 296}
	work := bounds{-1920, 0, 0, 1080}
	small := resizedBounds(start, work, edgeLeft|edgeTop, 1000, 1000, MinWidth, MinHeight)
	if small != (bounds{-500, 136, -260, 296}) {
		t.Fatalf("minimum size moved the opposite corner: %+v", small)
	}
	large := resizedBounds(start, work, edgeLeft|edgeTop, -200, -300, MinWidth, MinHeight)
	if large != (bounds{-800, 0, -260, 296}) {
		t.Fatalf("top work-area clamp lost the anchor: %+v", large)
	}
	right := resizedBounds(start, work, edgeRight|edgeBottom, 400, 200, MinWidth, MinHeight)
	if right != (bounds{-600, 100, 0, 496}) {
		t.Fatalf("right work-area clamp lost the anchor: %+v", right)
	}
}
