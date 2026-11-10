package render

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// LoopMode controls how an AnimationTag advances through its frames.
type LoopMode int

const (
	LoopForward LoopMode = iota
	LoopReverse
	LoopPingPong
	LoopOnce
)

// GridSpriteSheet slices an *ebiten.Image atlas into uniform sub-image frames once at boot.
type GridSpriteSheet struct {
	Sheet       *ebiten.Image
	FrameWidth  int
	FrameHeight int
	Cols        int
	Rows        int
	Frames      []*ebiten.Image
}

// NewGridSpriteSheet validates and slices an *ebiten.Image into uniform grid cells.
func NewGridSpriteSheet(sheet *ebiten.Image, frameW, frameH int) (*GridSpriteSheet, error) {
	if sheet == nil {
		return nil, fmt.Errorf("render: sprite sheet image is nil")
	}
	if frameW <= 0 || frameH <= 0 {
		return nil, fmt.Errorf("render: invalid frame dimensions %dx%d", frameW, frameH)
	}
	b := sheet.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < frameW || h < frameH || w%frameW != 0 || h%frameH != 0 {
		return nil, fmt.Errorf("render: sheet size %dx%d is not an exact multiple of frame size %dx%d", w, h, frameW, frameH)
	}
	cols := w / frameW
	rows := h / frameH
	frames := make([]*ebiten.Image, 0, cols*rows)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			x0 := b.Min.X + c*frameW
			y0 := b.Min.Y + r*frameH
			rect := image.Rect(x0, y0, x0+frameW, y0+frameH)
			sub, ok := sheet.SubImage(rect).(*ebiten.Image)
			if !ok || sub == nil {
				return nil, fmt.Errorf("render: failed to slice sub-image at col=%d row=%d", c, r)
			}
			frames = append(frames, sub)
		}
	}
	return &GridSpriteSheet{
		Sheet:       sheet,
		FrameWidth:  frameW,
		FrameHeight: frameH,
		Cols:        cols,
		Rows:        rows,
		Frames:      frames,
	}, nil
}

// FrameCount returns the total number of sliced frames in the sheet.
func (s *GridSpriteSheet) FrameCount() int {
	if s == nil {
		return 0
	}
	return len(s.Frames)
}

// Frame returns the pre-sliced sub-image for index.
func (s *GridSpriteSheet) Frame(index int) *ebiten.Image {
	if s == nil || index < 0 || index >= len(s.Frames) {
		return nil
	}
	return s.Frames[index]
}

// AnimationTag defines a named animation sequence across a range of frames.
type AnimationTag struct {
	Name          string
	StartFrame    int
	EndFrame      int
	FrameDuration time.Duration
	LoopMode      LoopMode
}

// AnimationController drives zero-allocation sprite animation playback.
type AnimationController struct {
	Sheet        *GridSpriteSheet
	Tags         map[string]*AnimationTag
	ActiveTag    *AnimationTag
	CurrentFrame int
	Elapsed      time.Duration
	PingPongDir  int
	Finished     bool
	drawOp       ebiten.DrawImageOptions
}

// NewAnimationController constructs an AnimationController for a pre-sliced GridSpriteSheet.
func NewAnimationController(sheet *GridSpriteSheet) *AnimationController {
	return &AnimationController{
		Sheet:       sheet,
		Tags:        make(map[string]*AnimationTag, 8),
		PingPongDir: 1,
	}
}

// Clone creates an independent AnimationController sharing the same pre-sliced GridSpriteSheet and tags.
func (ac *AnimationController) Clone() *AnimationController {
	cp := &AnimationController{
		Sheet:       ac.Sheet,
		Tags:        ac.Tags,
		PingPongDir: 1,
	}
	if ac.ActiveTag != nil {
		_ = cp.Play(ac.ActiveTag.Name)
	}
	return cp
}

// AddTag registers a named animation tag after validating frame bounds and timing.
func (ac *AnimationController) AddTag(name string, startFrame, endFrame int, frameDuration time.Duration, mode LoopMode) error {
	if ac.Sheet == nil {
		return fmt.Errorf("render: cannot add tag %q to nil sheet", name)
	}
	if startFrame < 0 || endFrame < startFrame || endFrame >= ac.Sheet.FrameCount() {
		return fmt.Errorf("render: invalid frame range [%d..%d] for tag %q", startFrame, endFrame, name)
	}
	if frameDuration <= 0 {
		return fmt.Errorf("render: invalid frameDuration %v for tag %q", frameDuration, name)
	}
	ac.Tags[name] = &AnimationTag{
		Name:          name,
		StartFrame:    startFrame,
		EndFrame:      endFrame,
		FrameDuration: frameDuration,
		LoopMode:      mode,
	}
	return nil
}

// Play switches to the named animation tag if it is not already active.
func (ac *AnimationController) Play(name string) error {
	tag, ok := ac.Tags[name]
	if !ok {
		return fmt.Errorf("render: animation tag %q not found", name)
	}
	if ac.ActiveTag == tag && !ac.Finished {
		return nil
	}
	ac.ActiveTag = tag
	ac.Elapsed = 0
	ac.Finished = false
	ac.PingPongDir = 1
	if tag.LoopMode == LoopReverse {
		ac.CurrentFrame = tag.EndFrame
	} else {
		ac.CurrentFrame = tag.StartFrame
	}
	return nil
}

// Update advances the animation state machine by dt.
func (ac *AnimationController) Update(dt time.Duration) {
	if ac.ActiveTag == nil || ac.Finished {
		return
	}
	tag := ac.ActiveTag
	if tag.StartFrame == tag.EndFrame {
		return
	}
	ac.Elapsed += dt
	for ac.Elapsed >= tag.FrameDuration {
		ac.Elapsed -= tag.FrameDuration
		switch tag.LoopMode {
		case LoopForward:
			ac.CurrentFrame++
			if ac.CurrentFrame > tag.EndFrame {
				ac.CurrentFrame = tag.StartFrame
			}
		case LoopReverse:
			ac.CurrentFrame--
			if ac.CurrentFrame < tag.StartFrame {
				ac.CurrentFrame = tag.EndFrame
			}
		case LoopOnce:
			if ac.CurrentFrame < tag.EndFrame {
				ac.CurrentFrame++
			} else {
				ac.Finished = true
			}
		case LoopPingPong:
			ac.CurrentFrame += ac.PingPongDir
			if ac.CurrentFrame >= tag.EndFrame {
				ac.CurrentFrame = tag.EndFrame
				ac.PingPongDir = -1
			} else if ac.CurrentFrame <= tag.StartFrame && ac.PingPongDir < 0 {
				ac.CurrentFrame = tag.StartFrame
				ac.PingPongDir = 1
			}
		}
	}
}

// DrawAnchored draws the active frame anchored at bottom-center (x, y) with custom scale, rotation, tint, and filter.
func (ac *AnimationController) DrawAnchored(
	screen *ebiten.Image,
	x, y float64,
	scaleX, scaleY float64,
	rotRad float64,
	tint color.Color,
	filter ebiten.Filter,
) {
	if ac == nil || ac.Sheet == nil {
		return
	}
	fr := ac.Sheet.Frame(ac.CurrentFrame)
	if fr == nil {
		return
	}
	fw := float64(ac.Sheet.FrameWidth)
	fh := float64(ac.Sheet.FrameHeight)

	ac.drawOp.GeoM.Reset()
	ac.drawOp.GeoM.Translate(-fw*0.5, -fh+8.0)
	ac.drawOp.GeoM.Scale(scaleX, scaleY)
	if rotRad != 0 {
		ac.drawOp.GeoM.Rotate(rotRad)
	}
	ac.drawOp.GeoM.Translate(x, y)

	ac.drawOp.ColorScale.Reset()
	if tint != nil {
		ac.drawOp.ColorScale.ScaleWithColor(tint)
	}
	ac.drawOp.Filter = filter
	screen.DrawImage(fr, &ac.drawOp)
}
