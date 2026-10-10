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

func TestVisualCountdownPreviews(t *testing.T) {
	dir := os.Getenv("COUNTDOWNER_PREVIEW_DIR")
	if dir == "" {
		t.Skip("optional headless preview")
	}
	if err := registerFonts(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	a := &application{config: model.Default()}
	a.config.Sort = "manual"
	for _, sample := range []struct {
		title    string
		duration time.Duration
	}{
		{"Project launch", 25 * model.Day},
		{"Design review", 15 * model.Day},
		{"Submission deadline", 5 * model.Day},
		{"Exactly 24 hours", model.Day},
		{"Final check", 3 * time.Hour},
		{"Future milestone", 45 * model.Day},
		{"Past milestone", 0},
	} {
		a.config.Countdowns = append(a.config.Countdowns, model.Countdown{
			ID: sample.title, Title: sample.title, Target: now.Add(sample.duration),
		})
	}
	for _, dark := range []bool{false, true} {
		tester := ui.NewTester(func(c *ui.Context) {
			theme := a.configureTheme(c, 1)
			c.Root().Background(theme.Background)
			ui.Column(c).Fill().Padding(18).Gap(16).Children(func() {
				a.widgetHeader(c, 1)
				ui.Divider(c)
				a.widgetItems(c, 1, 1, now)
			})
		}, 380, 770)
		tester.SetDark(dark)
		name := "visual-countdowns-light.png"
		if dark {
			name = "visual-countdowns-dark.png"
		}
		file, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		encodeErr := png.Encode(file, tester.Image())
		closeErr := file.Close()
		if encodeErr != nil || closeErr != nil {
			t.Fatalf("save visual preview: %v; %v", encodeErr, closeErr)
		}
	}
}
