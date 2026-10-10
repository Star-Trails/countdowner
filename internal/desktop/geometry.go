package desktop

import "math"

const (
	MinWidth     = 240
	MinHeight    = 160
	resizeBorder = 6
)

type resizeEdge uint8

const (
	edgeLeft resizeEdge = 1 << iota
	edgeRight
	edgeTop
	edgeBottom
)

type bounds struct {
	left, top, right, bottom float64
}

func hitResizeEdge(x, y, width, height float64) resizeEdge {
	if x < 0 || y < 0 || x >= width || y >= height {
		return 0
	}
	var edge resizeEdge
	if x < resizeBorder {
		edge |= edgeLeft
	} else if x >= width-resizeBorder {
		edge |= edgeRight
	}
	if y < resizeBorder {
		edge |= edgeTop
	} else if y >= height-resizeBorder {
		edge |= edgeBottom
	}
	return edge
}

// Only the grabbed edges move. The opposite edges stay anchored even at the
// minimum size or at a monitor's work-area boundary.
func resizedBounds(start, work bounds, edge resizeEdge, dx, dy, minWidth, minHeight float64) bounds {
	next := start
	if edge&edgeLeft != 0 {
		next.left = math.Max(work.left, math.Min(start.left+dx, start.right-minWidth))
	}
	if edge&edgeRight != 0 {
		next.right = math.Min(work.right, math.Max(start.right+dx, start.left+minWidth))
	}
	if edge&edgeTop != 0 {
		next.top = math.Max(work.top, math.Min(start.top+dy, start.bottom-minHeight))
	}
	if edge&edgeBottom != 0 {
		next.bottom = math.Min(work.bottom, math.Max(start.bottom+dy, start.top+minHeight))
	}
	return next
}
