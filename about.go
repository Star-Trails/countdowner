package main

import (
	_ "embed"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const repositoryURL = "https://github.com/Star-Trails/countdowner"
const goURL = "https://go.dev"
const mygoURL = "https://mygo.egoist.dev"

//go:embed assets/icons/go.svg
var goLogoSVG []byte

//go:embed assets/icons/mygo.svg
var mygoLogoSVG []byte

//go:embed assets/icons/github.svg
var githubSVG []byte

var goLogo = ui.MustParseSVG(goLogoSVG)
var mygoLogo = ui.MustParseSVG(mygoLogoSVG)
var githubIcon = ui.MustParseSVG(githubSVG)

func (a *application) aboutView(c *ui.Context) {
	ui.Column(c.Key("about")).Gap(6).Children(func() {
		ui.Text(c, a.t("About")).FontSize(12).Bold().TextColor(c.Theme().TextMuted).AlignSelf(ui.Center)
		ui.Row(c).Gap(16).Wrap().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			componentLink(c, "Go", goURL, func() {
				ui.Image(c, goLogo).Size(36, 16).Fit(ui.Contain).Shrink(0)
				ui.Text(c, "Go "+strings.TrimPrefix(runtime.Version(), "go")).FontSize(12).NoWrap().Shrink(0)
			})
			componentLink(c, "MyGo", mygoURL, func() {
				ui.Image(c, mygoLogo).Size(16, 16).Fit(ui.Contain).Shrink(0)
				ui.Text(c, "MyGo "+mygo.Version).FontSize(12).NoWrap().Shrink(0)
			})
			componentLink(c, "GitHub", repositoryURL, func() {
				ui.Icon(c, githubIcon).Size(16, 16).Shrink(0)
				ui.Text(c, "v"+mygo.App.Version()).FontSize(12).NoWrap().Shrink(0)
			})
		})
	})
}

func componentLink(c *ui.Context, name, url string, content func()) {
	link := ui.ButtonBase(c).Label(name).Tooltip(url).Shrink(0).Padding(4, 0).Gap(6).
		AlignItems(ui.Center).Radius(4).Cursor(ui.CursorPointer).TextColor(c.Theme().TextMuted)
	if link.Hovered() {
		link.Background(c.Theme().SurfaceHover)
	}
	link.Children(content).OnClick(func() { c.OpenURL(url) })
}
