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

func TestSettingsFooterVersionsAndGitHubLink(t *testing.T) {
	for _, size := range []struct{ width, height int }{{520, 420}, {760, 820}} {
		a := &application{config: model.Default()}
		a.config.Language = "zh-Hans"
		tester := ui.NewTester(a.settingsView, size.width, size.height)
		tester.Scroll(float32(size.width)/2, float32(size.height)/2, 0, 10000)
		for _, label := range []string{a.t("About"), "Go " + strings.TrimPrefix(runtime.Version(), "go"), "MyGo " + mygo.Version, "GitHub"} {
			r, ok := tester.Find(label)
			if !ok || r.W <= 0 || r.H <= 0 || r.Y+r.H > float32(size.height) {
				t.Fatalf("footer %q is missing or clipped at %dx%d: %+v", label, size.width, size.height, r)
			}
		}
		if err := tester.Click("GitHub"); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(tester.OpenedURLs(), []string{repositoryURL}) {
			t.Fatalf("GitHub button opened the wrong link: %v", tester.OpenedURLs())
		}
	}
}
