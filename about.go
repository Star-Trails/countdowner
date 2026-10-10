package main

import (
	_ "embed"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const repositoryURL = "https://github.com/Star-Trails/countdowner"

//go:embed assets/icons/code.svg
var codeSVG []byte

//go:embed assets/icons/browser.svg
var nativeUISVG []byte

//go:embed assets/icons/github.svg
var githubSVG []byte

var codeIcon = ui.MustParseSVG(codeSVG)
var nativeUIIcon = ui.MustParseSVG(nativeUISVG)
var githubIcon = ui.MustParseSVG(githubSVG)

func (a *application) aboutView(c *ui.Context) {
	ink := c.Theme().TextMuted
	ui.Column(c.Key("about")).Gap(6).Children(func() {
		ui.Text(c, a.t("About")).FontSize(12).Bold().TextColor(ink)
		ui.Row(c).Gap(16).Wrap().AlignItems(ui.Center).Children(func() {
			ui.Row(c).Gap(6).Shrink(0).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, codeIcon).Size(16, 16).TextColor(ink)
				ui.Text(c, "Go "+strings.TrimPrefix(runtime.Version(), "go")).FontSize(12).TextColor(ink)
			})
			ui.Row(c).Gap(6).Shrink(0).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, nativeUIIcon).Size(16, 16).TextColor(ink)
				ui.Text(c, "MyGo "+mygo.Version).FontSize(12).TextColor(ink)
			})
			ui.Button(c, "").Label("GitHub").Tooltip(repositoryURL).Shrink(0).Padding(4, 8).
				Cursor(ui.CursorPointer).Children(func() {
				ui.Icon(c, githubIcon).Size(16, 16)
				ui.Text(c, "GitHub").FontSize(12).SingleLine()
			}).OnClick(func() { c.OpenURL(repositoryURL) })
		})
	})
}
