package main

import (
	"math"

	"github.com/egoist/mygo/ui"
)

func drawTrash(p *ui.Painter, r ui.Rect, ink ui.Color, scale float32) {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	p.Line(cx-6*scale, cy-5*scale, cx+6*scale, cy-5*scale, 1.5*scale, ink)
	var path ui.Path
	path.MoveTo(cx-2*scale, cy-5*scale).LineTo(cx-2*scale, cy-7*scale).
		LineTo(cx+2*scale, cy-7*scale).LineTo(cx+2*scale, cy-5*scale)
	path.MoveTo(cx-4.5*scale, cy-3*scale).LineTo(cx-4*scale, cy+5*scale).
		QuadTo(cx-4*scale, cy+6*scale, cx-3*scale, cy+6*scale).
		LineTo(cx+3*scale, cy+6*scale).
		QuadTo(cx+4*scale, cy+6*scale, cx+4*scale, cy+5*scale).
		LineTo(cx+4.5*scale, cy-3*scale)
	p.StrokePath(&path, 1.5*scale, ink)
	for _, x := range []float32{-1.5, 1.5} {
		p.Line(cx+x*scale, cy-scale, cx+x*scale, cy+3.5*scale, scale, ink)
	}
}

func drawGear(p *ui.Painter, r ui.Rect, ink ui.Color, scale float32) {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	var path ui.Path
	for index := range 32 {
		radius := float32(5.8)
		if index%4 < 2 {
			radius = 7.5
		}
		angle := -math.Pi/2 + float64(index)*2*math.Pi/32
		x, y := cx+radius*scale*float32(math.Cos(angle)), cy+radius*scale*float32(math.Sin(angle))
		if index == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	path.Close()
	p.StrokePath(&path, 1.3*scale, ink)
	var center ui.Path
	center.Circle(cx, cy, 2.5*scale)
	p.StrokePath(&center, 1.3*scale, ink)
}
