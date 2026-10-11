package main

import (
	"testing"
	"time"

	"countdowner/internal/model"

	"github.com/egoist/mygo/ui"
)

func TestDeleteConfirmationKeepsFullButtonLabels(t *testing.T) {
	a := &application{config: model.Default()}
	a.config.Countdowns = []model.Countdown{{ID: "one", Title: "Deadline", Target: time.Now().Add(48 * time.Hour)}}
	tester := ui.NewTester(func(c *ui.Context) {
		a.configureTheme(c, 1)
		a.countdownsView(c)
	}, 650, 360)
	if err := tester.Click("Delete"); err != nil {
		t.Fatal(err)
	}
	wideCancel, _ := tester.Find("Cancel")
	wideConfirm, _ := tester.Find("Confirm delete")
	tester.SetSize(280, 360)
	cancel, ok := tester.Find("Cancel")
	confirm, _ := tester.Find("Confirm delete")
	edit, _ := tester.Find("Edit")
	if !ok || cancel.W < wideCancel.W || confirm.W < wideConfirm.W ||
		cancel.X+cancel.W > 280 || cancel.Y != confirm.Y || confirm.Y <= edit.Y {
		t.Fatalf("confirmation buttons must wrap together without shrinking: cancel=%+v confirm=%+v edit=%+v", cancel, confirm, edit)
	}
	if err := tester.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if a.pendingDelete != "" || len(a.config.Countdowns) != 1 {
		t.Fatal("cancel must dismiss confirmation without deleting the event")
	}
}
