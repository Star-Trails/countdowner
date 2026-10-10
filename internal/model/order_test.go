package model

import (
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestManualReorderingAndRestoringAutomaticSort(t *testing.T) {
	now := time.Now()
	config := Default()
	if config.Sort != "deadline" {
		t.Fatal("new configurations must default to deadline sorting")
	}
	config.Countdowns = []Countdown{
		{ID: "late", Title: "Late", Target: now.Add(3 * Day)},
		{ID: "early", Title: "Early", Target: now.Add(Day)},
		{ID: "middle", Title: "Middle", Target: now.Add(2 * Day)},
	}
	ids := func(c Config) []string {
		var result []string
		for _, item := range c.Ordered() {
			result = append(result, item.ID)
		}
		return result
	}
	if !slices.Equal(ids(config), []string{"early", "middle", "late"}) {
		t.Fatal("automatic deadline order incorrect")
	}
	next, changed := config.Reordered("early", "late")
	if !changed || next.Sort != "manual" || !slices.Equal(ids(next), []string{"middle", "late", "early"}) {
		t.Fatalf("downward reorder failed: %+v", next)
	}
	if !slices.Equal(ids(config), []string{"early", "middle", "late"}) {
		t.Fatal("reordering mutated the original configuration")
	}
	next, changed = next.Reordered("early", "middle")
	if !changed || !slices.Equal(ids(next), []string{"early", "middle", "late"}) {
		t.Fatal("upward reorder failed")
	}
	next, _ = next.Reordered("late", "early")
	store := Store{Path: filepath.Join(t.TempDir(), "config.json")}
	if err := store.Save(next); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil || !slices.Equal(ids(loaded), []string{"late", "early", "middle"}) {
		t.Fatalf("custom order was not persisted: %+v, %v", loaded, err)
	}
	loaded.Countdowns[0].Target = now.Add(-Day)
	if !slices.Equal(ids(loaded), []string{"late", "early", "middle"}) {
		t.Fatal("changing deadlines reset manual order")
	}
	loaded.Countdowns = append(loaded.Countdowns, Countdown{ID: "new", Title: "New", Target: now})
	if !slices.Equal(ids(loaded), []string{"late", "early", "middle", "new"}) {
		t.Fatal("new event did not append in manual mode")
	}
	loaded.Sort = "deadline"
	if !slices.Equal(ids(loaded), []string{"late", "new", "early", "middle"}) {
		t.Fatal("automatic sorting was not restored")
	}
	if _, changed := loaded.Reordered("missing", "early"); changed {
		t.Fatal("missing event changed the order")
	}
}
