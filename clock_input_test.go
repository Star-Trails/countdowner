package main

import (
	"path/filepath"
	"testing"
	"time"

	"countdowner/internal/model"

	"github.com/egoist/mygo/ui"
)

func TestTypedClockInput(t *testing.T) {
	a := &application{
		config: model.Default(), store: model.Store{Path: filepath.Join(t.TempDir(), "config.json")},
		draftTime: time.Date(2026, 12, 15, 9, 7, 0, 0, time.Local),
	}
	tester := ui.NewTester(a.draftView, 650, 500)
	click := func(label string) {
		t.Helper()
		if err := tester.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	click("Hour input")
	tester.Key(ui.Ctrl, ui.KeyA)
	tester.Type("13")
	click("Minute input")
	tester.Key(ui.Ctrl, ui.KeyA)
	tester.Type("28")
	click("Title")
	tester.Type("Typed deadline")
	click("Add countdown")
	loaded, err := a.store.Load()
	if err != nil || len(loaded.Countdowns) != 1 {
		t.Fatalf("typed deadline not saved: %v", err)
	}
	want := time.Date(2026, 12, 15, 13, 28, 0, 0, time.Local)
	if !loaded.Countdowns[0].Target.Equal(want) {
		t.Fatalf("typed target = %v; want %v", loaded.Countdowns[0].Target, want)
	}
}
