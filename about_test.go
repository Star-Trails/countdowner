package main

import (
	"runtime"
	"slices"
	"strings"
	"testing"

	"countdowner/internal/model"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestSettingsFooterVersionsAndComponentLinks(t *testing.T) {
	previousVersion := mygo.App.Version()
	mygo.App.SetVersion("0.0.0-test")
	defer mygo.App.SetVersion(previousVersion)
	for _, size := range []struct{ width, height int }{{520, 420}, {760, 820}} {
		a := &application{config: model.Default()}
		a.config.Language = "zh-Hans"
		tester := ui.NewTester(a.settingsView, size.width, size.height)
		tester.SetDark(size.width == 760)
		tester.Scroll(float32(size.width)/2, float32(size.height)/2, 0, 10000)
		for _, label := range []string{a.text().About(), "Go " + strings.TrimPrefix(runtime.Version(), "go"), "MyGo " + mygo.Version, "v" + mygo.App.Version()} {
			r, ok := tester.Find(label)
			if !ok || r.W <= 0 || r.H <= 0 || r.Y+r.H > float32(size.height) {
				t.Fatalf("footer %q is missing or clipped at %dx%d: %+v", label, size.width, size.height, r)
			}
		}
		first, _ := tester.Find("Go")
		last, _ := tester.Find("GitHub")
		if center := (first.X + last.X + last.W) / 2; center < float32(size.width)/2-1 || center > float32(size.width)/2+1 {
			t.Fatalf("component links are not centered at %dx%d: center=%v", size.width, size.height, center)
		}
		for _, name := range []string{"Go", "MyGo", "GitHub"} {
			if err := tester.Click(name); err != nil {
				t.Fatal(err)
			}
		}
		if !slices.Equal(tester.OpenedURLs(), []string{goURL, mygoURL, repositoryURL}) {
			t.Fatalf("component links opened the wrong URLs: %v", tester.OpenedURLs())
		}
	}
}
