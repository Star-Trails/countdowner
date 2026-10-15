package model

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestRemaining(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		duration time.Duration
		want     RemainingTime
	}{
		{25 * time.Hour, RemainingTime{Days: 2}},
		{24 * time.Hour, RemainingTime{Days: 1}},
		{24*time.Hour - time.Second, RemainingTime{Hours: 24}},
		{90 * time.Minute, RemainingTime{Hours: 1, Minutes: 30}},
		{59*time.Minute + time.Second, RemainingTime{Hours: 1}},
		{30 * time.Second, RemainingTime{Minutes: 1}},
		{0, RemainingTime{Expired: true}},
		{-time.Minute, RemainingTime{Expired: true}},
	} {
		if got := Remaining(now.Add(tc.duration), now); got != tc.want {
			t.Errorf("Remaining(%s) = %+v; want %+v", tc.duration, got, tc.want)
		}
	}
}

func TestParseLocalTarget(t *testing.T) {
	loc := time.FixedZone("Test +09", 9*3600)
	target, err := ParseTarget("2026-12-01", loc)
	if err != nil || target.Hour() != 0 || target.In(time.UTC).Hour() != 15 {
		t.Fatalf("date-only target = %v, %v", target, err)
	}
	target, err = ParseTarget("2026-12-01 14:30", loc)
	if err != nil || target.Hour() != 14 || target.Minute() != 30 {
		t.Fatalf("local target = %v, %v", target, err)
	}
	if _, err := ParseTarget("2026-02-30 14:30", loc); err == nil {
		t.Fatal("invalid calendar date accepted")
	}
}

func TestStoreAndOrdering(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "config.json")}
	config, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	first, _ := NewCountdown("專案交付 日本語 한국어", "2026-12-02 12:00", time.UTC)
	second, _ := NewCountdown("English 简体中文", "2026-12-01 12:00", time.UTC)
	config.Countdowns = []Countdown{first, second}
	config.Sort, config.Language, config.Theme = "deadline", "zh-Hant", "dark"
	config.Position = Position{Monitor: "monitor-id", X: 80.5, Y: 40.25}
	config.WidgetSize = Size{Width: 480, Height: 360}
	config.Autostart = true
	if err := store.Save(config); err != nil {
		t.Fatal(err)
	}
	// Exercise replacement of an EXISTING file, not just first creation.
	config.Position.X = 120
	if err := store.Save(config); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil || !reflect.DeepEqual(loaded, config) {
		t.Fatalf("round trip mismatch: %v; %#v", err, loaded)
	}
	if loaded.Ordered()[0].ID != second.ID || loaded.Countdowns[0].ID != first.ID {
		t.Fatal("deadline order did not preserve the manual order")
	}
	if err := os.WriteFile(store.Path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); err == nil {
		t.Fatal("damaged config must report an error")
	}
	data, _ := os.ReadFile(store.Path)
	if string(data) != "{broken" {
		t.Fatal("load overwrote damaged user data")
	}
}
