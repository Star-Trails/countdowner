package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"countdowner/internal/model"

	"github.com/egoist/mygo/ui"
)

func TestNativeSettingsAddEditDelete(t *testing.T) {
	a := &application{
		config: model.Default(), store: model.Store{Path: filepath.Join(t.TempDir(), "config.json")},
		draftTime: time.Date(2026, 12, 1, 9, 0, 0, 0, time.Local),
	}
	tester := ui.NewTester(a.settingsView, 760, 820)
	click := func(label string) {
		t.Helper()
		if err := tester.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	click("Title")
	tester.Type("Project 專案 项目 日本語 한국어")
	click("Date")
	for range 14 {
		tester.Key(0, ui.KeyRight)
	}
	tester.Key(0, ui.KeyEnter)
	click("Hour input")
	tester.Key(ui.Ctrl, ui.KeyA)
	tester.Type("13")
	click("Minute input")
	tester.Key(ui.Ctrl, ui.KeyA)
	tester.Type("28")
	click("Add countdown")
	if len(a.config.Countdowns) != 1 || a.statusError {
		t.Fatalf("add failed: %s, %#v", a.status, a.config.Countdowns)
	}
	want := time.Date(2026, 12, 15, 13, 28, 0, 0, time.Local)
	if !a.config.Countdowns[0].Target.Equal(want) {
		t.Fatalf("picker saved %v; want %v", a.config.Countdowns[0].Target, want)
	}
	id := a.config.Countdowns[0].ID
	click("Edit")
	a.draftTitle = "Changed title"
	tester.Frame()
	click("Save changes")
	loaded, err := a.store.Load()
	if err != nil || loaded.Countdowns[0].Title != "Changed title" || loaded.Countdowns[0].ID != id || !loaded.Countdowns[0].Target.Equal(want) {
		t.Fatalf("edit did not persist: %v; %#v", err, loaded)
	}
	click("Delete")
	click("Confirm delete")
	if len(a.config.Countdowns) != 0 {
		t.Fatal("delete did not update the model")
	}
}

func TestLanguageOrderAndNativeViews(t *testing.T) {
	a := &application{config: model.Default()}
	now := time.Now()
	for index, title := range []string{"Conference", "專案交付", "毕业答辩", "日本語の締切", "한국어 일정"} {
		item, err := model.NewCountdown(title, now.Add(time.Duration(index+1)*48*time.Hour).Format("2006-01-02 15:04"), time.Local)
		if err != nil {
			t.Fatal(err)
		}
		a.config.Countdowns = append(a.config.Countdowns, item)
	}
	for index, language := range languageCodes {
		a.config.Language = language
		settings := ui.NewTester(a.settingsView, 760, 820)
		if !settings.HasText(a.text().Language()) || !settings.HasText(languageNames[index]) {
			t.Errorf("language %s not reflected in the native settings", language)
		}
		widget := ui.NewTester(a.widgetView, 340, widgetHeight(5))
		if !widget.HasText("Conference") || !widget.HasText("日本語の締切") {
			t.Errorf("multilingual widget content missing for %s", language)
		}
	}
	// Optional headless layout previews, not evidence of Windows embedding.
	if dir := os.Getenv("COUNTDOWNER_PREVIEW_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		a.config.Language = "zh-Hans"
		for _, view := range []struct {
			name          string
			build         func(*ui.Context)
			width, height int
		}{
			{"settings", a.settingsView, 760, 820},
			{"widget", a.widgetView, 340, widgetHeight(5)},
		} {
			for _, dark := range []bool{false, true} {
				tester := ui.NewTester(view.build, view.width, view.height)
				tester.SetDark(dark)
				name := view.name + "-light.png"
				if dark {
					name = view.name + "-dark.png"
				}
				file, err := os.Create(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				encodeErr := png.Encode(file, tester.Image())
				closeErr := file.Close()
				if encodeErr != nil || closeErr != nil {
					t.Fatalf("save preview: %v; %v", encodeErr, closeErr)
				}
			}
		}
	}
}
