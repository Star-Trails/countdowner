package main

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"image/png"
	"sync"

	"github.com/egoist/mygo/ui"
)

//go:embed assets/fonts/*.ttc
var fontFiles embed.FS

var fontOnce sync.Once
var fontError error

func registerFonts() error {
	fontOnce.Do(func() {
		for _, name := range []string{"NotoSansCJK-Regular.ttc", "NotoSansCJK-Bold.ttc"} {
			data, err := fontFiles.ReadFile("assets/fonts/" + name)
			if err == nil {
				err = ui.RegisterFont(data, "")
			}
			if err != nil {
				fontError = err
				return
			}
		}
	})
	return fontError
}

func makeTrayIcon(dark bool) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	ink, halo := color.RGBA{30, 30, 30, 255}, color.RGBA{245, 245, 245, 255}
	if dark {
		ink, halo = halo, ink
	}
	// Opposing outlines remain visible even if the app theme and taskbar
	// theme differ (the app's forced appearance must not hide its tray icon).
	for y := 3; y < 30; y++ {
		for x := 3; x < 30; x++ {
			dx, dy := x-16, y-16
			r := dx*dx + dy*dy
			if r > 144 && r <= 169 {
				img.SetRGBA(x, y, halo)
			}
			if (r >= 100 && r <= 144) || (x >= 15 && x <= 17 && y >= 9 && y <= 17) || (y >= 15 && y <= 17 && x >= 16 && x <= 23) {
				img.SetRGBA(x, y, ink)
			}
		}
	}
	var data bytes.Buffer
	_ = png.Encode(&data, img)
	return data.Bytes()
}
