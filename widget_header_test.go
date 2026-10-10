package main

import (
	"testing"

	"countdowner/internal/desktop"
	"countdowner/internal/model"

	"github.com/egoist/mygo/ui"
)

func TestWidgetSettingsGearAndHeaderHitArea(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		a := &application{config: model.Default(), controller: &desktop.Controller{}}
		tester := ui.NewTester(func(c *ui.Context) {
			a.configureTheme(c, scale)
			a.widgetHeader(c, scale)
		}, 510, 90)
		gear, ok := tester.Find("Settings")
		if !ok || tester.HasText("Drag here to move") || gear.W != 24*scale {
			t.Fatalf("missing settings gear: %+v", gear)
		}
		x, y := float64((gear.X+gear.W/2)/scale), float64((gear.Y+gear.H/2)/scale)
		if !a.controller.HeaderInteractive(x, y) || a.controller.HeaderInteractive(20, y) {
			t.Fatal("gear must be excluded from native header dragging, but the rest of the header must remain draggable")
		}
		if err := tester.Click("Settings"); err != nil {
			t.Fatal(err)
		}
		if !tester.Focused("Settings") {
			t.Fatal("gear did not receive the click")
		}
	}
}
