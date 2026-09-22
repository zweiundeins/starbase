// Package pixelart builds the site's 8-bit illustrations as SVG.
//
// Art is authored as small character grids (sprites) or generated
// procedurally (planets, smoke), composed into layers and emitted as SVG
// <rect> runs with crisp edges. Illustrations are pixel-perfect at any size
// and cost no image requests beyond one cached SVG each.
package pixelart

import (
	"fmt"
	"math"
	"strings"
)

// Grid is a raster of CSS colours; "" is transparent.
type Grid struct {
	W, H int
	Px   []string
}

func NewGrid(w, h int) Grid { return Grid{W: w, H: h, Px: make([]string, w*h)} }

func (g Grid) At(x, y int) string {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return ""
	}
	return g.Px[y*g.W+x]
}

func (g Grid) Set(x, y int, c string) {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return
	}
	g.Px[y*g.W+x] = c
}

// Sprite parses rows of palette characters. '.' and ' ' are transparent.
func Sprite(palette map[rune]string, rows ...string) Grid {
	w := 0
	for _, r := range rows {
		w = max(w, len([]rune(r)))
	}
	g := NewGrid(w, len(rows))
	for y, r := range rows {
		for x, ch := range []rune(r) {
			if c, ok := palette[ch]; ok {
				g.Set(x, y, c)
			}
		}
	}
	return g
}

// Scale returns g enlarged by an integer factor.
func (g Grid) Scale(n int) Grid {
	out := NewGrid(g.W*n, g.H*n)
	for y := 0; y < out.H; y++ {
		for x := 0; x < out.W; x++ {
			out.Set(x, y, g.At(x/n, y/n))
		}
	}
	return out
}

// Rotate returns g rotated by deg (clockwise) with nearest-neighbour
// sampling, which keeps the hard pixel look.
func (g Grid) Rotate(deg float64) Grid {
	rad := deg * math.Pi / 180
	sin, cos := math.Sin(rad), math.Cos(rad)
	w := int(math.Ceil(math.Abs(float64(g.W)*cos) + math.Abs(float64(g.H)*sin)))
	h := int(math.Ceil(math.Abs(float64(g.W)*sin) + math.Abs(float64(g.H)*cos)))
	out := NewGrid(w, h)
	cx, cy := float64(g.W)/2, float64(g.H)/2
	ox, oy := float64(w)/2, float64(h)/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x)+0.5-ox, float64(y)+0.5-oy
			sx := cos*dx + sin*dy + cx
			sy := -sin*dx + cos*dy + cy
			out.Set(x, y, g.At(int(math.Floor(sx)), int(math.Floor(sy))))
		}
	}
	return out
}

// Layer places a grid on the canvas, optionally inside a CSS class group.
type Layer struct {
	Grid  Grid
	X, Y  int
	Class string
}

// SVG renders layers into a standalone SVG document. css is embedded in a
// <style> element (animations work even when the SVG is used via <img>).
func SVG(w, h int, css string, layers ...Layer) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" class="px-art" viewBox="0 0 %d %d" width="%d" height="%d" shape-rendering="crispEdges">`, w, h, w*4, h*4)
	if css != "" {
		b.WriteString("<style>" + css + "</style>")
	}
	for _, l := range layers {
		if l.Class != "" {
			fmt.Fprintf(&b, `<g class="%s">`, l.Class)
		}
		writeRuns(&b, l.Grid, l.X, l.Y)
		if l.Class != "" {
			b.WriteString("</g>")
		}
	}
	b.WriteString("</svg>")
	return b.String()
}

// writeRuns emits one <rect> per horizontal run of equal colour, grouped
// by colour to keep the markup small.
func writeRuns(b *strings.Builder, g Grid, ox, oy int) {
	type run struct{ x, y, w int }
	byColor := map[string][]run{}
	var order []string
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; {
			c := g.At(x, y)
			if c == "" {
				x++
				continue
			}
			start := x
			for x < g.W && g.At(x, y) == c {
				x++
			}
			if _, seen := byColor[c]; !seen {
				order = append(order, c)
			}
			byColor[c] = append(byColor[c], run{start, y, x - start})
		}
	}
	for _, c := range order {
		if strings.HasPrefix(c, "var(") {
			fmt.Fprintf(b, `<g style="fill:%s">`, c) // a theme token (see art.go)
		} else {
			fmt.Fprintf(b, `<g fill="%s">`, c)
		}
		for _, r := range byColor[c] {
			fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="1"/>`, r.x+ox, r.y+oy, r.w)
		}
		b.WriteString("</g>")
	}
}

// noise is deterministic 2D value noise in [0,1).
func noise(x, y float64, seed uint32) float64 {
	xi, yi := math.Floor(x), math.Floor(y)
	xf, yf := x-xi, y-yi
	h := func(i, j float64) float64 {
		n := uint32(int32(i))*374761393 + uint32(int32(j))*668265263 + seed*2246822519
		n = (n ^ (n >> 13)) * 1274126177
		return float64(n^(n>>16)) / float64(1<<32)
	}
	s := func(t float64) float64 { return t * t * (3 - 2*t) }
	a, b := h(xi, yi), h(xi+1, yi)
	c, d := h(xi, yi+1), h(xi+1, yi+1)
	u, v := s(xf), s(yf)
	return a + (b-a)*u + (c-a)*v + (a-b-c+d)*u*v
}

// Planet describes a procedurally shaded disc.
type Planet struct {
	R         int    // radius in pixels
	Outline   string // optional rim colour
	Light     string // lit side
	Base      string
	Shade     string // terminator side
	Feature   string // continents / craters (lit)
	FeatureDk string // features on the shaded side
	Scale     float64
	Threshold float64 // noise above this becomes a feature
	Seed      uint32
	Highlight string // tiny specular pixels, optional
}

// Grid renders the planet lit from the upper left.
func (p Planet) Grid() Grid {
	d := p.R*2 + 1
	g := NewGrid(d, d)
	r := float64(p.R) + 0.35
	for y := 0; y < d; y++ {
		for x := 0; x < d; x++ {
			dx, dy := float64(x-p.R), float64(y-p.R)
			dist := math.Hypot(dx, dy)
			if dist > r {
				continue
			}
			if p.Outline != "" && dist > r-1 {
				g.Set(x, y, p.Outline)
				continue
			}
			// Light direction: upper-left. lit in [-1,1].
			lit := (-dx*0.6 - dy*0.8) / r
			c := p.Base
			switch {
			case lit > 0.45:
				c = p.Light
			case lit < -0.35:
				c = p.Shade
			}
			if p.Feature != "" && noise(float64(x)/p.Scale, float64(y)/p.Scale, p.Seed) > p.Threshold {
				c = p.Feature
				if lit < -0.35 && p.FeatureDk != "" {
					c = p.FeatureDk
				}
			}
			g.Set(x, y, c)
		}
	}
	if p.Highlight != "" {
		hx, hy := p.R-p.R/2, p.R-p.R/2
		g.Set(hx, hy, p.Highlight)
		g.Set(hx+1, hy, p.Highlight)
		g.Set(hx, hy+1, p.Highlight)
	}
	return g
}
