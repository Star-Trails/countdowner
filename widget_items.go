package main

import (
	"slices"
	"time"

	"countdowner/internal/model"

	"github.com/egoist/mygo/ui"
)

const reorderHold = 300 * time.Millisecond

type countdownDrag string

type reorderPress struct {
	since  time.Time
	handle bool
}

func (a *application) cancelWidgetDelete() {
	if a.widgetDeleteID != "" {
		a.widgetDeleteID = ""
		a.redraw()
	}
}

func (a *application) refreshCountdowns(c *ui.Context) {
	if delay := a.config.NextRefresh(c.Now()); delay > 0 {
		c.After(delay)
	}
}

func (a *application) widgetItems(c *ui.Context, scale, contentScale float32, now time.Time) {
	items := a.config.Ordered()
	var draggedID, anchorID string
	for index, item := range items {
		row := ui.Row(c.Key(item.ID)).Gap(10 * scale).Radius(4 * scale).AlignItems(ui.Center)
		var handle ui.Element
		row.Children(func() {
			ui.Column(c.Key("text")).Grow(1).MinWidth(0).Gap(3 * scale * contentScale).Children(func() {
				ui.Text(c, item.Title).MinWidth(0).FontSize(13 * scale * contentScale).
					MaxLines(2).TextColor(c.Theme().TextMuted)
				ui.Text(c, a.remaining(item, now)).FontSize(26 * scale * contentScale).
					FontWeight(700).FontFeatures("tnum").MaxLines(3)
				ui.Text(c, item.Target.In(time.Local).Format("2006-01-02 15:04")).
					FontSize(10 * scale * contentScale).SingleLine().TextColor(c.Theme().TextMuted)
			})
			countdownIndicator(c.Key("indicator"), item, now, scale, a.text().CountdownIndicator()+": "+item.Title)
			ui.Column(c.Key("actions")).Size(24*scale, indicatorHeight*scale).Shrink(0).
				Padding(0).Gap(4 * scale).AlignItems(ui.Center).Children(func() {
				button := ui.Button(c, "")
				if a.widgetDeleteID == item.ID && button.PressedOutside() {
					a.widgetDeleteID = ""
				}
				armed := a.widgetDeleteID == item.ID
				label, trashInk := a.text().Delete(), c.Theme().Text
				if armed {
					label, trashInk = a.text().ConfirmDelete(), ui.RGB(255, 255, 255)
				}
				button.Size(24*scale, 24*scale).Shrink(0).Padding(0).Radius(4 * scale).
					Label(label + ": " + item.Title).Tooltip(label).Disabled(a.readOnly).
					Draw(func(p *ui.Painter, r ui.Rect) { drawTrash(p, r, trashInk, scale) })
				if armed {
					red := ui.RGB(190, 45, 50)
					if button.Pressed() {
						red = ui.RGB(146, 30, 35)
					} else if button.Hovered() {
						red = ui.RGB(166, 38, 43)
					}
					button.Background(red).BorderColor(red)
				}
				button.OnClick(func() {
					if a.widgetDeleteID == item.ID {
						a.remove(item.ID)
					} else {
						a.widgetDeleteID = item.ID
						a.redraw()
					}
				})
				handle = ui.Box(c.Key("reorder-handle")).Size(24*scale, 24*scale).Shrink(0).Radius(4 * scale).
					Label(a.text().Reorder() + ": " + item.Title).Disabled(a.readOnly).Cursor(ui.CursorGrab)
				if handle.Hovered() && !a.readOnly {
					handle.Background(c.Theme().SurfaceHover)
				}
				ink := c.Theme().TextMuted
				handle.Draw(func(p *ui.Painter, r ui.Rect) {
					for y := -1; y <= 1; y++ {
						for _, x := range []float32{-3, 3} {
							p.Fill(ui.Rect{
								X: r.X + r.W/2 + (x-1)*scale, Y: r.Y + r.H/2 + (float32(y)*6-1)*scale,
								W: 2 * scale, H: 2 * scale,
							}, ink, scale)
						}
					}
				})
			})
		})
		press := ui.Local(row, "reorder-press", func() reorderPress { return reorderPress{} })
		dx, dy, held := row.Dragged()
		row.Drag(nil)
		if held && !a.readOnly {
			if press.since.IsZero() {
				press.since = now
				x, y, _ := row.PointerPosition()
				r := row.Bounds()
				// Gate the whole-row preview by where the press began, not
				// where the pointer moves later. Buttons take their own input.
				press.handle = handle.Bounds().Contains(r.X+x-dx, r.Y+y-dy)
			}
			if press.handle {
				elapsed := now.Sub(press.since)
				if elapsed >= reorderHold {
					row.Drag(countdownDrag(item.ID)).Background(c.Theme().SurfaceHover)
					handle.Cursor(ui.CursorGrabbing)
				} else {
					c.After(reorderHold - elapsed)
				}
			}
		} else {
			*press = reorderPress{}
		}
		if row.Dragging() {
			row.Opacity(0.45)
		}
		if source, over := ui.DragOver[countdownDrag](row); over && string(source) != item.ID {
			from := slices.IndexFunc(items, func(value model.Countdown) bool { return value.ID == string(source) })
			after := from < index
			ink := c.Theme().TextMuted
			row.DrawOver(func(p *ui.Painter, r ui.Rect) {
				y := r.Y
				if after {
					y += r.H
				}
				p.Line(r.X, y, r.X+r.W, y, 2*scale, ink)
			})
		}
		if source, dropped := ui.Drop[countdownDrag](row); dropped && !a.readOnly {
			draggedID, anchorID = string(source), item.ID
		}
	}
	if draggedID != "" && !a.readOnly {
		if next, changed := a.config.Reordered(draggedID, anchorID); changed {
			a.commit(next)
		}
	}
}
