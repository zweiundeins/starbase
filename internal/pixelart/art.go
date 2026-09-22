package pixelart

import (
	"math"
	"strings"
)

// Palette. Most illustration colours are fixed: they are art, not theme.
// The ones that follow the theme are CSS variables (--sb-art-*, set per theme
// in theme.css and showcase.css) with the original colour as fallback. They
// take effect where the SVG is inlined into the page (see ui.Art); as a
// standalone file (favicon, other sites) it shows the fallbacks.
const (
	outline   = "#1B1F3B"
	white     = "#F3F4FA"
	silver    = "#C9CFEA"
	steel     = "#8E97C4"
	red       = "#E5484D"
	redDark   = "#A52A3A"
	violet    = "var(--sb-art-brand, #8C6BFF)"
	violetLt  = "var(--sb-art-brand-light, #B09AFF)"
	cyan      = "#65BFFF"
	navy      = "#2B4C9A"
	flameY    = "#FFE066"
	flameO    = "#FF9F43"
	flameR    = "#FF5E3A"
	starGold  = "#FFD84D"
	starWhite = "var(--sb-art-star, #F3F4FA)"
	starDim   = "var(--sb-art-star-dim, #7785A8)"
	smokeLt   = "var(--sb-art-smoke-light, #F3F4FA)"
	smoke     = "var(--sb-art-smoke, #DADDF6)"
	smokeDk   = "var(--sb-art-smoke-shade, #A3A8DA)"
	surfLight = "var(--sb-art-planet-light, #8A7CF5)"
	surfBase  = "var(--sb-art-planet, #5B4FD6)"
	surfBand  = "var(--sb-art-planet-band, #6D60E6)"
	surfLip   = "var(--sb-art-planet-lip, #7466EE)"
	surfShade = "var(--sb-art-planet-shade, #3F36A8)"
	surfCrate = "var(--sb-art-planet-crater, #4A40B8)"
	surfDeep  = "var(--sb-art-planet-deep, #2E2780)"
)

var rocketPalette = map[rune]string{
	'D': outline, 'W': white, 'G': silver, 'S': steel, 'R': red, 'r': redDark,
	'P': violet, 'B': cyan, 'b': navy, 'Y': flameY, 'O': flameO, 'F': flameR,
}

// rocketRows is the upright rocket (14 wide); flameRows sit below it.
var rocketRows = []string{
	"......DD......",
	".....DWGD.....",
	"....DWWWGD....",
	"....DWWWGD....",
	"...DWWWWWGD...",
	"...DPPPPPPD...",
	"...DWWWWWGD...",
	"...DWDDDDGD...",
	"...DWDBbDGD...",
	"...DWDbbDGD...",
	"...DWDDDDGD...",
	"...DWWWWWGD...",
	"..DDWWWWWGDD..",
	".DRDWWWWWGDRD.",
	"DRRDWPPPPGDRRD",
	"DRRDWWWWWGDRRD",
	"DRrDDDDDDDDrRD",
	"DrD.DSSSSD.DrD",
	"DD...DDDD...DD",
}

var flameRows = []string{
	".....YYYY.....",
	".....OYYO.....",
	"......OO......",
	"......FF......",
	".......F......",
}

// RocketRows returns the upright rocket sprite plus its flame, one string
// per row, using single-letter colour keys (see rocketPalette).
func RocketRows() []string {
	return append(append([]string{}, rocketRows...), flameRows...)
}

// rocketParts returns the rocket and its flame on identical canvases, so
// both can be rotated and placed at the same offset but animated separately.
func rocketParts() (body, flame Grid) {
	blank := strings.Repeat(".", 14)
	var bodyRows, flameOnly []string
	for _, r := range rocketRows {
		bodyRows = append(bodyRows, r)
		flameOnly = append(flameOnly, blank)
	}
	for _, r := range flameRows {
		bodyRows = append(bodyRows, blank)
		flameOnly = append(flameOnly, r)
	}
	return Sprite(rocketPalette, bodyRows...), Sprite(rocketPalette, flameOnly...)
}

func sparkle(c string) Grid {
	return Sprite(map[rune]string{'#': c, '+': white},
		"..#..",
		"..#..",
		"##+##",
		"..#..",
		"..#..",
	)
}

func dot(c string) Grid { return Sprite(map[rune]string{'#': c}, "#") }

func smokePuff(r int, seed uint32) Grid {
	return Planet{R: r, Light: smokeLt, Base: smoke, Shade: smokeDk, Seed: seed}.Grid()
}

// Crop copies a window of g.
func (g Grid) Crop(x0, y0, w, h int) Grid {
	out := NewGrid(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.Set(x, y, g.At(x0+x, y0+y))
		}
	}
	return out
}

// surface draws the top of a huge planet: a lit rim, banded shading and
// noisy craters, for the horizon of the hero scene.
func surface(w, h, r int) Grid {
	g := NewGrid(w, h)
	cx, cy := float64(w)/2, float64(r)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			if d > float64(r) {
				continue
			}
			depth := float64(r) - d
			c := surfBase
			switch {
			case depth < 1:
				c = surfLight
			case depth < 2.2:
				c = surfBand
			case depth > 11:
				c = surfDeep
			case depth > 7:
				c = surfShade
			}
			if depth > 1.5 {
				n := noise(float64(x)/3.2, float64(y)/1.6, 7)
				if n > 0.74 {
					c = surfCrate
					if depth > 7 {
						c = surfDeep
					}
				}
				if n > 0.74 && noise(float64(x)/3.2, float64(y-1)/1.6, 7) <= 0.74 {
					c = surfLip // crater lip catches the light
				}
			}
			g.Set(x, y, c)
		}
	}
	return g
}

const sceneCSS = `
.px-art .twinkle{animation:px-tw 3.2s steps(2,jump-none) infinite}
.px-art .twinkle.b{animation-delay:-1.3s;animation-duration:4.4s}
.px-art .twinkle.c{animation-delay:-2.6s;animation-duration:5.6s}
.px-art .float{animation:px-fl 4s steps(4,jump-none) infinite alternate}
.px-art .flame{animation:px-fk .4s steps(2,jump-none) infinite}
@keyframes px-tw{0%,70%,100%{opacity:1}80%{opacity:.2}}
@keyframes px-fl{from{transform:translate(0,0)}to{transform:translate(1px,-2px)}}
@keyframes px-fk{from{opacity:1}to{opacity:.55}}
@media (prefers-reduced-motion:reduce){.px-art *{animation:none!important}}
`

// Hero is the launch scene: starfield, moon, earth, a rocket lifting off a
// violet planet in a cloud of exhaust.
func Hero() string {
	const W, H = 180, 84
	var layers []Layer

	// Starfield: deterministic scatter, three twinkle phases.
	stars := [][3]int{
		{12, 8, 0}, {28, 22, 1}, {44, 4, 2}, {60, 30, 0}, {70, 12, 1}, {86, 3, 2},
		{112, 14, 0}, {124, 36, 1}, {132, 6, 2}, {150, 30, 0}, {166, 42, 1}, {176, 20, 2},
		{20, 40, 2}, {52, 48, 1}, {100, 44, 0}, {140, 50, 2}, {6, 26, 1}, {80, 52, 2},
		{158, 58, 0}, {36, 34, 0}, {118, 2, 1}, {172, 64, 2},
	}
	phase := []string{"twinkle", "twinkle b", "twinkle c"}
	for i, s := range stars {
		c := starWhite
		if i%3 == 1 {
			c = starDim
		}
		if i%5 == 2 {
			c = violetLt
		}
		layers = append(layers, Layer{Grid: dot(c), X: s[0], Y: s[1], Class: phase[s[2]]})
	}
	layers = append(layers,
		Layer{Grid: sparkle(starGold), X: 56, Y: 24, Class: "twinkle b"},
		Layer{Grid: sparkle(cyan), X: 18, Y: 30, Class: "twinkle"},
		Layer{Grid: sparkle(cyan), X: 112, Y: 50, Class: "twinkle c"},
		Layer{Grid: sparkle(starGold), X: 170, Y: 32, Class: "twinkle"},
		Layer{Grid: sparkle(starWhite), X: 38, Y: 58, Class: "twinkle c"},
	)

	moon := Planet{R: 7, Outline: "#3A3E7A", Light: "#B4B8EA", Base: "#8C8FD0", Shade: "#5E62A8",
		Feature: "#6E72B8", FeatureDk: "#4E5296", Scale: 2.2, Threshold: 0.62, Seed: 3}.Grid()
	layers = append(layers, Layer{Grid: moon, X: 30, Y: 8})

	earth := Planet{R: 12, Outline: "#123070", Light: "#5AAEFF", Base: "#2F7BE0", Shade: "#1E4FA8",
		Feature: "#4CC46A", FeatureDk: "#2E8A4B", Scale: 3.4, Threshold: 0.56, Seed: 11, Highlight: "#D4ECFF"}.Grid()
	layers = append(layers, Layer{Grid: earth, X: 150, Y: 2})

	// Horizon.
	layers = append(layers, Layer{Grid: surface(W, 26, 150), X: 0, Y: H - 26})

	// Exhaust: a trail of puffs from the nozzle down to a cloud on the ground.
	puffs := [][3]int{
		{81, 53, 2}, {78, 56, 3}, {74, 58, 3},
		{68, 60, 4}, {60, 60, 5}, {51, 61, 4}, {43, 62, 3}, {86, 61, 3}, {92, 62, 2}, {36, 63, 2},
	}
	for i, p := range puffs {
		g := smokePuff(p[2], uint32(i))
		layers = append(layers, Layer{Grid: g, X: p[0] - p[2], Y: p[1] - p[2]})
	}

	body, flame := rocketParts()
	// Upscaling before rotating keeps the diagonal edges clean.
	const tilt = 28
	rb, rf := body.Scale(2).Rotate(tilt), flame.Scale(2).Rotate(tilt)
	layers = append(layers,
		Layer{Grid: rf, X: 70, Y: 2, Class: "float flame"},
		Layer{Grid: rb, X: 70, Y: 2, Class: "float"},
	)
	return SVG(W, H, sceneCSS, layers...)
}

// Logo is the upright rocket with its flame, for the header.
func Logo() string {
	body, flame := rocketParts()
	return SVG(body.W, body.H, sceneCSS,
		Layer{Grid: flame, Class: "flame"},
		Layer{Grid: body},
	)
}

// Saturn is a small ringed planet (sidebar, empty states).
func Saturn() string {
	const W, H = 26, 18
	g := NewGrid(W, H)
	planet := Planet{R: 6, Outline: "#3A2E8F", Light: violetLt, Base: violet, Shade: "#5A41C8",
		Feature: "#7B5CF0", Scale: 1.5, Threshold: 0.78, Seed: 5}.Grid()
	cx, cy := 13.0, 9.0
	ring := func(front bool) {
		for y := 0; y < H; y++ {
			for x := 0; x < W; x++ {
				dx, dy := (float64(x)+0.5-cx)/12.5, (float64(y)+0.5-cy)/3.6
				// Tilt the ring a little.
				dy += dx * 0.35
				d := dx*dx + dy*dy
				if d < 0.62 || d > 1 {
					continue
				}
				if (dy > 0) != front {
					continue
				}
				c := cyan
				if d > 0.84 {
					c = "#3E8FD6"
				}
				g.Set(x, y, c)
			}
		}
	}
	ring(false)
	for y := 0; y < planet.H; y++ {
		for x := 0; x < planet.W; x++ {
			if c := planet.At(x, y); c != "" {
				g.Set(x+7, y+3, c)
			}
		}
	}
	ring(true)
	return SVG(W, H, "", Layer{Grid: g})
}

// Sparkle is the four-point star used in section titles.
func Sparkle() string {
	g := Sprite(map[rune]string{'#': starGold, '+': white, 'o': flameO},
		"...#...",
		"...#...",
		"..o#o..",
		"###+###",
		"..o#o..",
		"...#...",
		"...#...",
	)
	return SVG(g.W, g.H, "", Layer{Grid: g})
}

// Moon is a lone moon for empty states.
func Moon() string {
	g := Planet{R: 10, Outline: "#3A3E7A", Light: "#B4B8EA", Base: "#8C8FD0", Shade: "#5E62A8",
		Feature: "#6E72B8", FeatureDk: "#4E5296", Scale: 2.4, Threshold: 0.66, Seed: 21}.Grid()
	return SVG(g.W, g.H, "", Layer{Grid: g})
}

// Info is a chunky pixel "i" badge.
func Info() string {
	g := Sprite(map[rune]string{'D': "#3A2E8F", 'P': violet, 'L': violetLt, 'W': white},
		".DDDDDD.",
		"DLLPPPPD",
		"DLPWWPPD",
		"DPPPPPPD",
		"DPPWWPPD",
		"DPPWWPPD",
		"DPPWWPPD",
		".DDDDDD.",
	)
	return SVG(g.W, g.H, "", Layer{Grid: g})
}

// Landscape is a small night scene for card media.
func Landscape() string {
	const W, H = 64, 36
	var layers []Layer
	for i, s := range [][2]int{{6, 4}, {14, 12}, {22, 3}, {30, 9}, {40, 5}, {50, 14}, {58, 6}, {4, 18}, {36, 16}, {60, 20}} {
		c := white
		if i%3 == 1 {
			c = starDim
		}
		layers = append(layers, Layer{Grid: dot(c), X: s[0], Y: s[1], Class: []string{"twinkle", "twinkle b", "twinkle c"}[i%3]})
	}
	layers = append(layers,
		Layer{Grid: sparkle(starGold), X: 44, Y: 20, Class: "twinkle b"},
		Layer{Grid: Planet{R: 6, Outline: "#3A2E8F", Light: violetLt, Base: violet, Shade: "#5A41C8",
			Feature: "#7B5CF0", Scale: 1.6, Threshold: 0.72, Seed: 9}.Grid(), X: 24, Y: 6},
	)
	hills := NewGrid(W, 12)
	for x := 0; x < W; x++ {
		h1 := 5 + int(3*math.Sin(float64(x)/7)+2*math.Sin(float64(x)/3.1))
		h2 := 3 + int(2*math.Sin(float64(x+9)/5.3))
		for y := 0; y < 12; y++ {
			switch {
			case y >= 12-h2:
				hills.Set(x, y, surfShade)
			case y >= 12-h1:
				hills.Set(x, y, surfBase)
			}
			if y == 12-h1 {
				hills.Set(x, y, surfLight)
			}
		}
	}
	layers = append(layers, Layer{Grid: hills, X: 0, Y: H - 12})
	return SVG(W, H, sceneCSS, layers...)
}

// All returns every illustration by name.
func All() map[string]string {
	return map[string]string{
		"info":      Info(),
		"landscape": Landscape(),
		"hero":      Hero(),
		"logo":      Logo(),
		"saturn":    Saturn(),
		"sparkle":   Sparkle(),
		"moon":      Moon(),
	}
}
