package ui

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// PremultipliedRGBA constructs a valid premultiplied color.RGBA where R, G, B <= A.
func PremultipliedRGBA(r, g, b, a uint8) color.RGBA {
	af := float64(a) / 255.0
	return color.RGBA{
		R: uint8(math.Round(float64(r) * af)),
		G: uint8(math.Round(float64(g) * af)),
		B: uint8(math.Round(float64(b) * af)),
		A: a,
	}
}

var (
	ColPanelBg     = PremultipliedRGBA(14, 18, 32, 215)
	ColPanelBorder = PremultipliedRGBA(250, 208, 44, 235)
	ColTextWhite   = PremultipliedRGBA(248, 250, 255, 255)
	ColTextGold    = PremultipliedRGBA(255, 216, 48, 255)
	ColTextCyan    = PremultipliedRGBA(72, 232, 255, 255)
	ColTextGreen   = PremultipliedRGBA(92, 242, 132, 255)
	ColTextRed     = PremultipliedRGBA(255, 88, 88, 255)
	ColTextMuted   = PremultipliedRGBA(175, 188, 212, 255)
	ColShadow      = PremultipliedRGBA(6, 8, 16, 235)
)

// DrawPanel renders a sleek broadcast telemetry panel with a 2px border.
func DrawPanel(screen *ebiten.Image, x, y, w, h float32, bg, border color.RGBA) {
	vector.FillRect(screen, x, y, w, h, bg, false)
	vector.StrokeRect(screen, x, y, w, h, 2.0, border, false)
}

// DrawProgressBar renders a horizontal gauge bar clamped to [0, 1].
func DrawProgressBar(screen *ebiten.Image, x, y, w, h float32, ratio float64, bgCol, fillCol, borderCol color.RGBA) {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	vector.FillRect(screen, x, y, w, h, bgCol, false)
	fillW := float32(float64(w-4) * ratio)
	if fillW > 0 {
		vector.FillRect(screen, x+2, y+2, fillW, h-4, fillCol, false)
	}
	vector.StrokeRect(screen, x, y, w, h, 1.5, borderCol, false)
}
