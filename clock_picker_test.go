package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestClockPickerFitsSmallWindowAndScrollsSelection(t *testing.T) {
	selected := "28"
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(12).Children(func() {
			clockPicker(c, &selected, minuteChoices, "Minute", false).Label("Choose minute")
		})
	}, 320, 240)
	if err := tester.Click("Choose minute"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyEnd)
	last, ok := tester.Find("59")
	if !ok || last.Y < 0 || last.Y+last.H > 240 {
		t.Fatalf("keyboard selection must scroll the last minute into the small window: %+v", last)
	}
	tester.Key(0, ui.KeyEnter)
	if selected != "59" || tester.HasText("58") {
		t.Fatal("Enter must choose the highlighted minute and close the popup")
	}
	if err := tester.Click("Choose minute"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyHome)
	first, ok := tester.Find("00")
	if !ok || first.Y < 0 || first.Y+first.H > 240 {
		t.Fatalf("Home must scroll the first minute into view: %+v", first)
	}
	x, y := first.X+first.W/2, first.Y+first.H/2
	tester.Move(x, y)
	tester.Scroll(x, y, 0, 96)
	tester.Frame()
	scrolled, _ := tester.Find("00")
	if scrolled.Y >= first.Y {
		t.Fatal("wheel scrolling must not snap back to the highlighted minute")
	}
	tester.Key(0, ui.KeyEscape)
	if selected != "59" || tester.HasText("00") {
		t.Fatal("Escape must close the popup without changing the chosen minute")
	}
}
