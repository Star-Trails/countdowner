package main

import (
	"path/filepath"
	"testing"
	"time"

	"countdowner/internal/model"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

func TestDateOnlyPicker(t *testing.T) {
	a := &application{
		config: model.Default(), store: model.Store{Path: filepath.Join(t.TempDir(), "config.json")},
		draftTime: time.Date(2026, 12, 1, 13, 28, 0, 0, time.Local),
	}
	tester := ui.NewTester(a.draftView, 650, 500)
	click := func(label string) {
		t.Helper()
		if err := tester.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	click("Tomorrow")
	if a.draftTime.Format("2006-01-02") != time.Now().AddDate(0, 0, 1).Format("2006-01-02") || a.draftTime.Hour() != 13 || a.draftTime.Minute() != 28 {
		t.Fatalf("Tomorrow did not preserve the chosen clock time: %v", a.draftTime)
	}
	click("Date only (00:00)")
	if tester.HasText("Hour") || tester.HasText("Minute") {
		t.Fatal("date-only mode should hide the clock controls")
	}
	// Switching modes shouldn't discard the user's previous time selection.
	click("Date only (00:00)")
	if !tester.HasText("Hour") || !tester.HasText("Minute") || a.draftTime.Hour() != 13 || a.draftTime.Minute() != 28 {
		t.Fatal("time selection lost when leaving date-only mode")
	}
	click("Date only (00:00)")
	click("Title")
	tester.Type("Date only deadline")
	click("Add countdown")
	loaded, err := a.store.Load()
	if err != nil || len(loaded.Countdowns) != 1 {
		t.Fatalf("date-only deadline not persisted: %v", err)
	}
	target := loaded.Countdowns[0].Target.In(time.Local)
	if target.Hour() != 0 || target.Minute() != 0 || target.Second() != 0 {
		t.Fatalf("date-only target is not local midnight: %v", target)
	}
	a.edit(loaded.Countdowns[0])
	tester.Frame()
	if !a.draftDateOnly || !a.draftTime.Equal(target) {
		t.Fatal("editing did not restore the saved midnight date")
	}
}

func TestDateSelectionSurvivesLanguageAndRegionChange(t *testing.T) {
	a := &application{
		config: model.Default(), store: model.Store{Path: filepath.Join(t.TempDir(), "config.json")},
		draftTime: time.Date(2026, 12, 1, 13, 28, 0, 0, time.Local),
	}
	tester := ui.NewTester(a.draftView, 650, 500)
	tester.SetLocale("en")
	tester.SetRegion(i18n.Region{Locale: "en-US", FirstWeekday: time.Sunday, HourCycle: 12})
	if err := tester.Click(a.text().Date()); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyRight)
	tester.Key(0, ui.KeyRight)
	// A language/region change rebuilds an open calendar. Keep its pending
	// selection and the independently chosen clock time across that rebuild.
	a.config.Language = "zh-Hans"
	tester.SetLocale(a.config.Language)
	tester.SetRegion(i18n.Region{Locale: "zh-CN", FirstWeekday: time.Monday, HourCycle: 24})
	tester.Key(0, ui.KeyEnter)
	if err := tester.Click(a.text().Title()); err != nil {
		t.Fatal(err)
	}
	tester.Type("Localized deadline")
	if err := tester.Click(a.text().AddCountdown()); err != nil {
		t.Fatal(err)
	}
	loaded, err := a.store.Load()
	if err != nil || len(loaded.Countdowns) != 1 {
		t.Fatalf("localized deadline not saved: %v", err)
	}
	want := time.Date(2026, 12, 3, 13, 28, 0, 0, time.Local)
	if !loaded.Countdowns[0].Target.Equal(want) {
		t.Fatalf("language/region change lost the pending date or time: got %v, want %v", loaded.Countdowns[0].Target, want)
	}
}
