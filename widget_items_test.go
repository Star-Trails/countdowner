package main

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"countdowner/internal/model"

	"github.com/egoist/mygo/ui"
)

func widgetFixture(t *testing.T) (*application, *time.Time, *ui.Tester) {
	t.Helper()
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	a := &application{config: model.Default(), store: model.Store{Path: filepath.Join(t.TempDir(), "config.json")}}
	a.config.Countdowns = []model.Countdown{
		{ID: "late", Title: "Late", Target: now.Add(45 * model.Day)},
		{ID: "early", Title: "Early", Target: now.Add(5 * model.Day)},
		{ID: "middle", Title: "Middle", Target: now.Add(15 * model.Day)},
	}
	tester := ui.NewTester(func(c *ui.Context) {
		a.configureTheme(c, 1)
		ui.Column(c).Fill().Padding(18).Gap(16).Children(func() {
			a.widgetHeader(c, 1)
			a.widgetItems(c, 1, 1, now)
		})
	}, 340, 420)
	return a, &now, tester
}

func TestWidgetLongPressReordering(t *testing.T) {
	a, now, tester := widgetFixture(t)
	early, _ := tester.Find("Reorder: Early")
	late, _ := tester.Find("Late")
	x, y := early.X+early.W/2, early.Y+early.H/2
	tx, ty := late.X+2, late.Y+late.H/2
	// A quick drag must not reorder an item.
	tester.Press(x, y)
	*now = now.Add(299 * time.Millisecond)
	tester.Frame()
	tester.Move(tx, ty)
	tester.Release(tx, ty)
	if a.config.Sort != "deadline" {
		t.Fatal("short press activated a reorder")
	}
	// The real UI drag runs without sleeping by advancing the supplied time.
	tester.Press(x, y)
	*now = now.Add(reorderHold)
	tester.Frame()
	tester.Move(tx, ty)
	tester.Release(tx, ty)
	loaded, err := a.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, item := range loaded.Ordered() {
		ids = append(ids, item.ID)
	}
	if loaded.Sort != "manual" || !slices.Equal(ids, []string{"middle", "late", "early"}) {
		t.Fatalf("long-press reorder did not persist: %q, mode=%s", ids, loaded.Sort)
	}
	// A previous drag must not leave the next press armed.
	first, _ := tester.Find("Reorder: Middle")
	last, _ := tester.Find("Early")
	tester.Press(first.X+first.W/2, first.Y+first.H/2)
	tester.Move(last.X+2, last.Y+last.H/2)
	tester.Release(last.X+2, last.Y+last.H/2)
	if a.config.Countdowns[0].ID != "middle" {
		t.Fatal("stale drag state bypassed the hold threshold")
	}
}

func TestWidgetHandleLayoutAndExclusiveDragging(t *testing.T) {
	a, now, tester := widgetFixture(t)
	indicator, _ := tester.Find("Countdown indicator: Early")
	trash, _ := tester.Find("Delete: Early")
	handle, _ := tester.Find("Reorder: Early")
	if trash.X < indicator.X+indicator.W || trash.Y != indicator.Y ||
		handle.Y+handle.H/2 != indicator.Y+indicator.H/2 || handle.X != trash.X {
		t.Fatalf("trash top must align with indicator top, and handle must align with indicator center: %+v %+v %+v", indicator, trash, handle)
	}
	if _, shown := tester.Find("Countdown indicator: Late"); shown {
		t.Fatal("45-day event still has a visualization container")
	}
	if !tester.HasText("45 days") || !tester.HasText("Delete: Late") || !tester.HasText("Reorder: Late") {
		t.Fatal("distant event lost its text or actions")
	}
	target, _ := tester.Find("Late")
	for _, label := range []string{"Early", "5 days", "Countdown indicator: Early"} {
		source, _ := tester.Find(label)
		tester.Press(source.X+2, source.Y+source.H/2)
		*now = now.Add(time.Second)
		tester.Frame()
		// Entering the handle after pressing elsewhere must not arm dragging.
		tester.Move(handle.X+handle.W/2, handle.Y+handle.H/2)
		tester.Move(target.X+2, target.Y+target.H/2)
		tester.Release(target.X+2, target.Y+target.H/2)
		if a.config.Sort != "deadline" {
			t.Fatalf("dragging %q activated reordering", label)
		}
	}
}

func TestWidgetTrashClickAwayCancelsConfirmation(t *testing.T) {
	for _, label := range []string{"Early", "Countdown indicator: Early", "Reorder: Early", "Settings", ""} {
		t.Run(label, func(t *testing.T) {
			a, _, tester := widgetFixture(t)
			if err := tester.Click("Delete: Early"); err != nil {
				t.Fatal(err)
			}
			x, y := float32(5), float32(5) // Widget padding.
			if label != "" {
				r, ok := tester.Find(label)
				if !ok {
					t.Fatalf("missing click-away target %q", label)
				}
				x, y = r.X+r.W/2, r.Y+r.H/2
			}
			tester.Press(x, y)
			tester.Release(x, y)
			if a.widgetDeleteID != "" || len(a.config.Countdowns) != 3 ||
				!tester.HasText("Delete: Early") || tester.HasText("Confirm delete: Early") {
				t.Fatal("clicking away must restore the monochrome button without deleting")
			}
			if err := tester.Click("Delete: Early"); err != nil {
				t.Fatal(err)
			}
			if a.widgetDeleteID != "early" || len(a.config.Countdowns) != 3 {
				t.Fatal("first click after cancellation must arm again, not delete")
			}
			if err := tester.Click("Confirm delete: Early"); err != nil {
				t.Fatal(err)
			}
			if len(a.config.Countdowns) != 2 {
				t.Fatal("second click on the armed button must still delete")
			}
		})
	}
}

func TestWidgetTrashConfirmationAndAutomaticSort(t *testing.T) {
	a, _, tester := widgetFixture(t)
	if err := tester.Click("Delete: Early"); err != nil {
		t.Fatal(err)
	}
	if a.widgetDeleteID != "early" || len(a.config.Countdowns) != 3 || a.pendingDelete != "" || a.config.Sort != "deadline" {
		t.Fatal("first click must only arm the widget button, without opening settings or reordering")
	}
	red, ok := tester.Find("Confirm delete: Early")
	if !ok {
		t.Fatal("confirmation button did not appear")
	}
	pixel := tester.Image().RGBAAt(int(red.X+4), int(red.Y+4))
	if int(pixel.R) < 2*int(pixel.G) || int(pixel.R) < 2*int(pixel.B) {
		t.Fatalf("armed trash button is not red: %+v", pixel)
	}
	if err := tester.Click("Confirm delete: Early"); err != nil {
		t.Fatal(err)
	}
	if len(a.config.Countdowns) != 2 || a.widgetDeleteID != "" || tester.HasText("Early") {
		t.Fatal("second click did not remove the event and clear confirmation")
	}
	a.readOnly = true
	tester.Frame()
	if err := tester.Click("Delete: Middle"); err != nil {
		t.Fatal(err)
	}
	if a.widgetDeleteID != "" || len(a.config.Countdowns) != 2 {
		t.Fatal("read-only widget must not arm deletion")
	}
	a.readOnly = false
	a.config.Sort = "manual"
	management := ui.NewTester(a.countdownsView, 650, 500)
	if err := management.Click("Manual order"); err != nil {
		t.Fatal(err)
	}
	if err := management.Click("Deadline, earliest first"); err != nil {
		t.Fatal(err)
	}
	loaded, err := a.store.Load()
	if err != nil || loaded.Sort != "deadline" || loaded.Ordered()[0].ID != "middle" {
		t.Fatalf("settings did not restore automatic sorting: %+v, %v", loaded, err)
	}
}
