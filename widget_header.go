package main

import "github.com/egoist/mygo/ui"

func (a *application) widgetHeader(c *ui.Context, scale float32) {
	ui.Row(c.Key("header")).Height(30 * scale).Children(func() {
		ui.Text(c, "Countdowner").FontSize(14 * scale).Grow(1)
		ink := c.Theme().Text
		ui.Button(c, "").Size(24*scale, 24*scale).Shrink(0).Padding(0).
			Bind(&a.settingsButton).Label(a.text().Settings()).Tooltip(a.text().Settings()).
			Draw(func(p *ui.Painter, r ui.Rect) { drawGear(p, r, ink, scale) }).
			OnClick(a.showSettings)
	})
	if a.controller != nil {
		// The native drag hook receives target-monitor DIPs; the handle's
		// committed bounds use the UI surface's compensated layout units.
		a.controller.HeaderInteractive = func(x, y float64) bool {
			return a.settingsButton.Bounds(c).Contains(float32(x)*scale, float32(y)*scale)
		}
	}
}
