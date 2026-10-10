package main

import (
	"math"
	"time"

	"countdowner/internal/model"

	"github.com/egoist/mygo/ui"
)

const (
	indicatorWidth    = 104
	indicatorHeight   = 80
	indicatorPadding  = 10
	indicatorRadius   = 6
	indicatorCell     = 10
	indicatorGap      = 2
	indicatorColumns  = 7
	indicatorCells    = 31
	indicatorDiameter = 54
	indicatorStroke   = 4
)

func indicatorColors(dark bool, urgency model.Urgency) (background, neutral, fill ui.Color) {
	background, neutral = ui.RGB(233, 236, 239), ui.RGB(205, 210, 215)
	colors := [...]uint32{0x238636, 0x9A7400, 0xC33236, 0xB55B12, 0xC24820}
	if dark {
		background, neutral = ui.RGB(48, 52, 59), ui.RGB(80, 88, 98)
		colors = [...]uint32{0x62B879, 0xD9B44A, 0xF07878, 0xF2A44D, 0xFA8458}
	}
	hex := colors[urgency]
	return background, neutral, ui.RGB(uint8(hex>>16), uint8(hex>>8), uint8(hex))
}

func countdownIndicator(c *ui.Context, item model.Countdown, now time.Time, scale float32, label string) ui.Element {
	state := model.CountdownIndicator(item.Target, now)
	if state.Mode == model.IndicatorEmpty {
		return ui.Element{}
	}
	background, neutral, fill := indicatorColors(c.Theme().Dark, state.Urgency)
	return ui.Box(c).Size(indicatorWidth*scale, indicatorHeight*scale).Shrink(0).
		Padding(indicatorPadding * scale).Radius(indicatorRadius * scale).Background(background).
		Label(label).Draw(func(p *ui.Painter, r ui.Rect) {
		switch state.Mode {
		case model.IndicatorGrid:
			gridWidth := float32(indicatorColumns*indicatorCell+(indicatorColumns-1)*indicatorGap) * scale
			gridHeight := float32(5*indicatorCell+4*indicatorGap) * scale
			x, y := r.X+(r.W-gridWidth)/2, r.Y+(r.H-gridHeight)/2
			for index := range indicatorCells {
				color := neutral
				if index < state.Days {
					color = fill
				}
				cell := ui.Rect{
					X: x + float32(index%indicatorColumns)*(indicatorCell+indicatorGap)*scale,
					Y: y + float32(index/indicatorColumns)*(indicatorCell+indicatorGap)*scale,
					W: indicatorCell * scale, H: indicatorCell * scale,
				}
				p.Fill(cell, color, 2*scale)
			}
		case model.IndicatorCircle, model.IndicatorExpired:
			cx, cy := r.X+r.W/2, r.Y+r.H/2
			radius := float32(indicatorDiameter-indicatorStroke) / 2 * scale
			var track ui.Path
			track.Circle(cx, cy, radius)
			p.StrokePath(&track, indicatorStroke*scale, neutral)
			if state.Mode == model.IndicatorCircle {
				arc := remainingArc(cx, cy, radius, state.Fraction)
				p.StrokePath(&arc, indicatorStroke*scale, fill)
			}
		}
	})
}

// The end remains at twelve o'clock while the start advances clockwise,
// so the remaining arc disappears clockwise as time runs out.
func remainingArc(cx, cy, radius float32, fraction float64) ui.Path {
	var path ui.Path
	if fraction >= 1 {
		path.Circle(cx, cy, radius)
		return path
	}
	sweep := 2 * math.Pi * fraction
	segments := int(math.Ceil(sweep / (math.Pi / 2)))
	if segments == 0 {
		return path
	}
	start := -math.Pi/2 + 2*math.Pi*(1-fraction)
	step := sweep / float64(segments)
	x, y := float64(cx)+float64(radius)*math.Cos(start), float64(cy)+float64(radius)*math.Sin(start)
	path.MoveTo(float32(x), float32(y))
	for index := range segments {
		a, b := start+float64(index)*step, start+float64(index+1)*step
		k := 4.0 / 3 * math.Tan(step/4) * float64(radius)
		nx, ny := float64(cx)+float64(radius)*math.Cos(b), float64(cy)+float64(radius)*math.Sin(b)
		path.CubeTo(float32(x-k*math.Sin(a)), float32(y+k*math.Cos(a)),
			float32(nx+k*math.Sin(b)), float32(ny-k*math.Cos(b)), float32(nx), float32(ny))
		x, y = nx, ny
	}
	return path
}
