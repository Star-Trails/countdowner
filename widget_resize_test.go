package main

import (
	"path/filepath"
	"testing"
	"time"

	"countdowner/internal/model"

	"github.com/egoist/mygo/ui"
)

func TestWidgetResizeLayoutAndPersistence(t *testing.T) {
	if err := registerFonts(); err != nil {
		t.Fatal(err)
	}
	a := &application{config: model.Default(), store: model.Store{Path: filepath.Join(t.TempDir(), "config.json")}}
	item, err := model.NewCountdown("First deadline", time.Now().Add(72*time.Hour).Format("2006-01-02 15:04"), time.Local)
	if err != nil {
		t.Fatal(err)
	}
	a.config.Countdowns = []model.Countdown{item}
	tester := ui.NewTester(a.widgetView, 240, 160)
	small, ok := tester.Find("3 days")
	if !ok {
		t.Fatal("small widget lost its countdown")
	}
	tester.SetSize(640, 480)
	large, ok := tester.Find("3 days")
	if !ok || large.H <= small.H {
		t.Fatalf("countdown text did not grow with the viewport: small=%+v large=%+v", small, large)
	}
	position := model.Position{Monitor: "left-monitor", X: 128.5, Y: 76.25}
	size := model.Size{Width: 640, Height: 480}
	a.saveWidgetBounds(position, size)
	loaded, err := a.store.Load()
	if err != nil || loaded.Position != position || loaded.WidgetSize != size {
		t.Fatalf("resized bounds did not persist: %v; %+v", err, loaded)
	}
	// Adding another deadline must not reset a user-chosen size.
	next := loaded
	second := item
	second.ID += "-second"
	next.Countdowns = append(next.Countdowns, second)
	if !a.commit(next) || a.widgetSize() != size {
		t.Fatal("adding a countdown reset the custom size")
	}
}
