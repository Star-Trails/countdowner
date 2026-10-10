package main

import (
	"image/color"
	"math"
	"testing"
	"time"

	"countdowner/internal/model"

	"github.com/egoist/mygo/ui"
)

func TestIndicatorDrawingAndUniformContainers(t *testing.T) {
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	durations := []time.Duration{25 * model.Day, 15 * model.Day, 5 * model.Day, model.Day, 3 * time.Hour, 45 * model.Day, 0}
	for _, dark := range []bool{false, true} {
		for _, scale := range []float32{1, 1.5} {
			tester := ui.NewTester(func(c *ui.Context) {
				ui.Row(c).Gap(8).Children(func() {
					for index, duration := range durations {
						item := model.Countdown{Target: now.Add(duration)}
						countdownIndicator(c, item, now, scale, string(rune('A'+index)))
					}
				})
			}, 1400, 160)
			tester.SetDark(dark)
			pixel := func(x, y float32) color.RGBA {
				return tester.Image().RGBAAt(int(x), int(y))
			}
			asRGBA := func(c ui.Color) color.RGBA { return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A} }
			for index, duration := range durations {
				r, ok := tester.Find(string(rune('A' + index)))
				state := model.CountdownIndicator(now.Add(duration), now)
				if state.Mode == model.IndicatorEmpty {
					if ok {
						t.Fatal("distant event still has a visualization container")
					}
					continue
				}
				if !ok || r.W != indicatorWidth*scale || r.H != indicatorHeight*scale {
					t.Fatalf("nonuniform container: %+v, dark=%v, scale=%v", r, dark, scale)
				}
				background, neutral, fill := indicatorColors(dark, state.Urgency)
				if state.Mode == model.IndicatorGrid {
					x, y := r.X+(r.W-82*scale)/2, r.Y+(r.H-58*scale)/2
					filled := 0
					for cell := range indicatorCells {
						got := pixel(x+float32(cell%7)*12*scale+5*scale, y+float32(cell/7)*12*scale+5*scale)
						want := neutral
						if cell < state.Days {
							want = fill
							filled++
						}
						if got != asRGBA(want) {
							t.Fatalf("cell %d: got %v; want %v", cell, got, want)
						}
					}
					if filled != state.Days {
						t.Fatalf("filled %d; want %d", filled, state.Days)
					}
					// The final row has only three cells, not four placeholders.
					for cell := 31; cell < 35; cell++ {
						if got := pixel(x+float32(cell%7)*12*scale+5*scale, y+4*12*scale+5*scale); got != asRGBA(background) {
							t.Fatalf("unexpected cell %d: %v", cell, got)
						}
					}
				} else {
					cx, cy := r.X+r.W/2, r.Y+r.H/2
					if pixel(cx, cy) != asRGBA(background) {
						t.Fatal("circle interior is not empty")
					}
					if state.Mode == model.IndicatorCircle {
						radius := float32(25) * scale
						want := neutral
						if duration == model.Day {
							want = fill
						}
						if got := pixel(cx+radius, cy); got != asRGBA(want) {
							t.Fatalf("right ring track: %v; want %v", got, want)
						}
						if duration == 3*time.Hour {
							angle := 247.5 * math.Pi / 180
							got := pixel(cx+radius*float32(math.Cos(angle)), cy+radius*float32(math.Sin(angle)))
							if got != asRGBA(fill) {
								t.Fatalf("remaining 12.5%% arc: %v; want %v", got, fill)
							}
						}
					}
				}
			}
			if len(tester.Texts()) != len(durations)-1 {
				t.Fatalf("indicator rendered unwanted text: %q", tester.Texts())
			}
		}
	}
}
