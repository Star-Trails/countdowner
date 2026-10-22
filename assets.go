package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

func makeTrayIcon(dark bool) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	ink, halo := color.RGBA{30, 30, 30, 255}, color.RGBA{245, 245, 245, 255}
	if dark {
		ink, halo = halo, ink
	}
	// A contrasting outline keeps the icon visible on translucent taskbars.
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
