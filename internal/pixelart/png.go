package pixelart

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"regexp"
	"strconv"
)

// Raster images of the illustrations, for places that can't take SVG or
// CSS variables: social previews (Open Graph), touch icons, favicon.ico.
// Theme tokens render as their fallback colours (the Deep Space look).

var hexRe = regexp.MustCompile(`#([0-9A-Fa-f]{6})`)

// rgba reads "#RRGGBB" or "var(--token, #RRGGBB)" (its fallback).
func rgba(c string) (color.RGBA, bool) {
	m := hexRe.FindStringSubmatch(c)
	if m == nil {
		return color.RGBA{}, false
	}
	v, _ := strconv.ParseUint(m[1], 16, 32)
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}, true
}

// PNG draws layers (w×h pixels) at scale on a canvas of cw×ch filled with
// bg ("#RRGGBB"): centred, or standing on the bottom edge.
func PNG(cw, ch, scale int, bottom bool, bg string, w, h int, layers ...Layer) []byte {
	img := image.NewRGBA(image.Rect(0, 0, cw, ch))
	if c, ok := rgba(bg); ok {
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
		}
	}
	ox, oy := (cw-w*scale)/2, (ch-h*scale)/2
	if bottom {
		oy = ch - h*scale
	}
	for _, l := range layers {
		for y := 0; y < l.Grid.H; y++ {
			for x := 0; x < l.Grid.W; x++ {
				c, ok := rgba(l.Grid.At(x, y))
				if !ok {
					continue
				}
				px, py := ox+(l.X+x)*scale, oy+(l.Y+y)*scale
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.SetRGBA(px+dx, py+dy, c)
					}
				}
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// SiteBG is the Deep Space page background, for raster images.
const SiteBG = "#080D1D"

// SocialImage is the Open Graph preview: the hero scene, 1200×630.
func SocialImage() []byte {
	w, h, layers := heroLayers()
	return PNG(1200, 630, 6, true, SiteBG, w, h, layers...)
}

// Icon is the rocket logo on the site background, size×size.
func Icon(size int) []byte {
	w, h, layers := logoLayers()
	return PNG(size, size, max(1, size*3/4/h), false, SiteBG, w, h, layers...)
}
