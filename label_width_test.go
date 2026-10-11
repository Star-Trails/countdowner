package main

import (
	"runtime"
	"strings"
	"testing"

	"countdowner/internal/model"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestAboutLinksWrapWithFullTextWidths(t *testing.T) {
	previousVersion := mygo.App.Version()
	mygo.App.SetVersion("0.1.5")
	defer mygo.App.SetVersion(previousVersion)
	a := &application{config: model.Default()}
	labels := []string{"Go " + strings.TrimPrefix(runtime.Version(), "go"), "MyGo " + mygo.Version, "v" + mygo.App.Version()}
	reference := ui.NewTester(func(c *ui.Context) {
		a.configureTheme(c, 1)
		ui.Column(c).AlignItems(ui.Start).Children(func() {
			for _, label := range labels {
				ui.Text(c, label).FontSize(12).NoWrap().Shrink(0)
			}
		})
	}, 600, 160)
	tester := ui.NewTester(func(c *ui.Context) {
		a.configureTheme(c, 1)
		a.aboutView(c)
	}, 200, 160)
	tester.SetScale(1.5)
	for _, label := range labels {
		want, _ := reference.Find(label)
		got, ok := tester.Find(label)
		if !ok || got.W+0.01 < want.W || got.X < 0 || got.X+got.W > 200 || got.Y+got.H > 160 {
			t.Fatalf("%q must retain its full text width: got=%+v full=%+v", label, got, want)
		}
	}
	first, _ := tester.Find("Go")
	last, _ := tester.Find("GitHub")
	if first.Y == last.Y {
		t.Fatal("component links must wrap when the row cannot fit")
	}
}

func TestFullLabelButtonKeepsTextWidth(t *testing.T) {
	a := &application{config: model.Default()}
	labels := []string{"Confirm delete", "Cancel"}
	reference := ui.NewTester(func(c *ui.Context) {
		a.configureTheme(c, 1)
		ui.Row(c).Children(func() {
			for _, label := range labels {
				ui.Text(c, label).NoWrap().Shrink(0)
			}
		})
	}, 600, 80)
	tester := ui.NewTester(func(c *ui.Context) {
		a.configureTheme(c, 1)
		ui.Row(c).Gap(6).Wrap().Children(func() {
			for _, label := range labels {
				fullLabelButton(c, label).Label("button " + label)
			}
		})
	}, 160, 120)
	tester.SetScale(1.5)
	for _, label := range labels {
		want, _ := reference.Find(label)
		got, ok := tester.Find(label)
		if !ok || got.W+0.01 < want.W || got.X < 0 || got.X+got.W > 160 {
			t.Fatalf("%q must retain its full text width: got=%+v full=%+v", label, got, want)
		}
	}
}
