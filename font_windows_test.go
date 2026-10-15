//go:build windows

package main

import (
	"bytes"
	"testing"

	"countdowner/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestLatinAndDigitWeightsStayStableAcrossLanguages(t *testing.T) {
	var baseline []byte
	for _, language := range languageCodes {
		a := &application{config: model.Default()}
		a.config.Language = language
		tester := ui.NewTester(func(c *ui.Context) {
			a.configureTheme(c, 1)
			ui.Column(c).Padding(8).Children(func() {
				ui.Text(c, "Countdowner 1234567890").FontSize(18)
				ui.Text(c, "Countdowner 1234567890").FontSize(26).FontWeight(700)
			})
		}, 500, 130)
		tester.SetLocale(language)
		pixels := tester.Image().Pix
		if baseline == nil {
			baseline = bytes.Clone(pixels)
		} else if !bytes.Equal(baseline, pixels) {
			t.Errorf("switching to %s changed the regular/bold Latin and digit rendering", language)
		}
	}
}
