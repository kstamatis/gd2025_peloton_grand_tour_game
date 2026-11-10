package ui

import (
	"bytes"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// FontBank manages pre-allocated text/v2 font faces and DrawOptions for zero-allocation HUD rendering.
type FontBank struct {
	source    *text.GoTextFaceSource
	SmallFace *text.GoTextFace
	BodyFace  *text.GoTextFace
	LargeFace *text.GoTextFace
	TitleFace *text.GoTextFace
	drawOp    text.DrawOptions
}

// NewFontBank parses embedded TTF bytes and constructs small, body, large, and title font faces.
func NewFontBank(ttfBytes []byte) (*FontBank, error) {
	if len(ttfBytes) == 0 {
		return nil, fmt.Errorf("ui: empty TTF font bytes")
	}
	src, err := text.NewGoTextFaceSource(bytes.NewReader(ttfBytes))
	if err != nil {
		return nil, fmt.Errorf("ui: parse TTF font source: %w", err)
	}
	return &FontBank{
		source: src,
		SmallFace: &text.GoTextFace{
			Source: src,
			Size:   14,
		},
		BodyFace: &text.GoTextFace{
			Source: src,
			Size:   18,
		},
		LargeFace: &text.GoTextFace{
			Source: src,
			Size:   26,
		},
		TitleFace: &text.GoTextFace{
			Source: src,
			Size:   42,
		},
	}, nil
}

// DrawAligned renders shadowed text using ebiten/v2/text/v2 with zero heap allocations.
func (fb *FontBank) DrawAligned(
	screen *ebiten.Image,
	str string,
	face *text.GoTextFace,
	x, y float64,
	hAlign, vAlign text.Align,
	fgColor, shadowColor color.Color,
) {
	if fb == nil || face == nil || str == "" {
		return
	}
	fb.drawOp.GeoM.Reset()
	fb.drawOp.GeoM.Translate(x+2, y+2)
	fb.drawOp.ColorScale.Reset()
	fb.drawOp.ColorScale.ScaleWithColor(shadowColor)
	fb.drawOp.LayoutOptions.PrimaryAlign = hAlign
	fb.drawOp.LayoutOptions.SecondaryAlign = vAlign
	fb.drawOp.LayoutOptions.LineSpacing = face.Size * 1.25
	text.Draw(screen, str, face, &fb.drawOp)

	fb.drawOp.GeoM.Reset()
	fb.drawOp.GeoM.Translate(x, y)
	fb.drawOp.ColorScale.Reset()
	fb.drawOp.ColorScale.ScaleWithColor(fgColor)
	text.Draw(screen, str, face, &fb.drawOp)
}
