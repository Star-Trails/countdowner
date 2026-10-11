package main

import "github.com/egoist/mygo/ui"

func fullLabelButton(c *ui.Context, label string) ui.Element {
	return ui.Button(c, "").Label(label).Shrink(0).Children(func() {
		ui.Text(c, label).NoWrap().Shrink(0)
	})
}
