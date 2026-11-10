package ui

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/ericonyourphone/sidea/app"
)

type pixel struct{ r, g, b uint8 }
type point struct{ x, y float64 }

func rgb(hex string) pixel {
	var p pixel
	_, _ = fmt.Sscanf(hex, "#%02x%02x%02x", &p.r, &p.g, &p.b)
	return p
}
func (p pixel) color() lipgloss.Color {
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", p.r, p.g, p.b))
}
func mix(a, b pixel, t float64) pixel {
	return pixel{uint8(float64(a.r)*(1-t) + float64(b.r)*t), uint8(float64(a.g)*(1-t) + float64(b.g)*t), uint8(float64(a.b)*(1-t) + float64(b.b)*t)}
}

type cassetteKey struct {
	artist, song           string
	w, h, phase, container int
	palette                app.Palette
}

var cassetteMemo struct {
	sync.Mutex
	key  cassetteKey
	rows []string
}

// The cassette is drawn as square pixels using two vertical pixels per
// terminal cell. No image protocol, external artwork, or fixed palette is used.
func cassette(artist, song string, w, h, frame int, playing bool, containerW int) []string {
	phase := 0
	if playing {
		phase = (frame / 4) % 12
	}
	key := cassetteKey{artist, song, w, h, phase, containerW, app.CurrentPalette()}
	cassetteMemo.Lock()
	defer cassetteMemo.Unlock()
	if cassetteMemo.rows != nil && cassetteMemo.key == key {
		return cassetteMemo.rows
	}
	rows := drawCassette(artist, song, w, h, phase, containerW)
	cassetteMemo.key, cassetteMemo.rows = key, rows
	return rows
}
func drawCassette(artist, song string, w, h, phase, containerW int) []string {
	ph := h * 2
	p := app.CurrentPalette()
	bg, shell, outline, ink := rgb(p.Bg), rgb(p.Surface), mix(rgb(p.Border), rgb(p.TextSec), .18), rgb(p.Bg)
	paper := mix(rgb(p.Text), bg, .08)
	stripe := rgb(p.Accent)
	pixels := make([]pixel, w*ph)
	for i := range pixels {
		pixels[i] = bg
	}
	put := func(x, y int, c pixel) {
		if x >= 0 && x < w && y >= 0 && y < ph {
			pixels[y*w+x] = c
		}
	}
	rect := func(x0, y0, x1, y1 int, c pixel) {
		for y := max(0, y0); y < min(ph, y1); y++ {
			for x := max(0, x0); x < min(w, x1); x++ {
				put(x, y, c)
			}
		}
	}
	polygon := func(points []point, c pixel) {
		for y := 0; y < ph; y++ {
			for x := 0; x < w; x++ {
				inside := false
				px, py := float64(x)+.5, float64(y)+.5
				for i, j := 0, len(points)-1; i < len(points); j, i = i, i+1 {
					a, b := points[i], points[j]
					if (a.y > py) != (b.y > py) && px < (b.x-a.x)*(py-a.y)/(b.y-a.y)+a.x {
						inside = !inside
					}
				}
				if inside {
					put(x, y, c)
				}
			}
		}
	}
	shape := func(x0, y0, x1, y1, cut float64) []point {
		return []point{{x0 + cut, y0}, {x1 - cut, y0}, {x1, y0 + cut}, {x1, y1 - cut}, {x1 - cut, y1}, {x0 + cut, y1}, {x0, y1 - cut}, {x0, y0 + cut}}
	}
	polygon(shape(1, 1, float64(w), float64(ph), 2), rgb(p.Border))
	polygon(shape(0, 0, float64(w-1), float64(ph-1), 2), outline)
	polygon(shape(1, 1, float64(w-2), float64(ph-2), 1), shell)
	labelBottom := int(float64(ph) * .76)
	polygon(shape(2, 2, float64(w-3), float64(labelBottom), 2), paper)
	bandY := int(float64(ph) * .34)
	rect(2, bandY, w-3, bandY+max(1, ph/16), stripe)
	windowTop := int(float64(ph) * .39)
	windowBottom := int(float64(ph) * .64)
	rect(int(float64(w)*.16), windowTop, int(float64(w)*.83), windowBottom, shell)
	cy := float64(windowTop+windowBottom-1) / 2
	radius := max(2, float64(windowBottom-windowTop)*.46)
	angle := float64(phase) * 2 * math.Pi / 12
	for _, cx := range []float64{float64(w) * .28, float64(w) * .70} {
		for y := windowTop; y < windowBottom; y++ {
			for x := int(cx - radius - 1); x <= int(cx+radius+1); x++ {
				dx, dy := float64(x)+.5-cx, float64(y)+.5-cy
				r := math.Hypot(dx, dy) / radius
				if r > 1 {
					continue
				}
				c := bg
				if r > .72 || r > .25 && math.Cos(math.Atan2(dy, dx)*5+angle) > .05 {
					c = paper
				}
				put(x, y, c)
			}
		}
	}
	bridgeLeft, bridgeRight := int(float64(w)*.44), int(float64(w)*.56)
	rect(bridgeLeft, windowTop+1, bridgeRight, windowBottom-1, bg)
	rect((bridgeLeft+bridgeRight)/2, windowTop+1, (bridgeLeft+bridgeRight)/2+max(1, w/28), windowBottom-1, paper)
	polygon([]point{{float64(w) * .28, float64(ph) * .80}, {float64(w) * .70, float64(ph) * .80}, {float64(w) * .78, float64(ph - 2)}, {float64(w) * .20, float64(ph - 2)}}, mix(shell, bg, .6))
	for _, xf := range []float64{.23, .75} {
		rect(int(float64(w)*xf), ph-5, int(float64(w)*xf)+2, ph-3, bg)
	}
	for _, xf := range []float64{.37, .61} {
		put(int(float64(w)*xf), ph-5, outline)
	}
	rows := make([]string, h)
	for row := range h {
		var line strings.Builder
		// Coalesce adjacent cells with the same two colors to keep terminal
		// output and redraw cost small while the reels turn.
		for x := 0; x < w; {
			top, bottom := pixels[row*2*w+x], pixels[(row*2+1)*w+x]
			end := x + 1
			for end < w && pixels[row*2*w+end] == top && pixels[(row*2+1)*w+end] == bottom {
				end++
			}
			line.WriteString(lipgloss.NewStyle().Foreground(top.color()).Background(bottom.color()).Render(strings.Repeat("▀", end-x)))
			x = end
		}
		rows[row] = line.String()
	}
	stamp := func(row, x int, text string, fg, bg pixel) {
		if row < 0 || row >= h {
			return
		}
		text = label(text, max(0, w-x-3))
		n := lipgloss.Width(text)
		mark := lipgloss.NewStyle().Foreground(fg.color()).Background(bg.color()).Render(text)
		rows[row] = ansi.Cut(rows[row], 0, x) + mark + ansi.Cut(rows[row], x+n, w)
	}
	labelW := max(1, w-14)
	stamp(1, 7, fit(center(label(artist, labelW), labelW), labelW), ink, paper)
	stamp(2, 7, fit(center(label(song, labelW), labelW), labelW), ink, paper)
	stamp(1, 3, " A ", ink, stripe)
	stamp(2, 3, "   ", ink, stripe)
	stamp(min(h-3, windowBottom/2), w-7, "90", ink, paper)
	for i := range rows {
		rows[i] = center(rows[i], containerW)
	}
	return rows
}
