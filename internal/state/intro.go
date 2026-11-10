package state

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"pelotongrandtour/internal/ui"
)

// IntroState presents the cinematic Grand Tour broadcast opening sequence.
type IntroState struct {
	ctx            *Context
	timer          float64
	heroAnim       *ebiten.DrawImageOptions
	cyclistPreview *ebiten.Image
}

// NewIntroState constructs the IntroState.
func NewIntroState(ctx *Context) *IntroState {
	return &IntroState{
		ctx:      ctx,
		heroAnim: &ebiten.DrawImageOptions{},
	}
}

// Enter starts BGM and resets intro timers.
func (s *IntroState) Enter() error {
	s.timer = 0
	if s.ctx.BGM != nil {
		s.ctx.BGM.Start()
	}
	if s.ctx.CyclistProto != nil {
		_ = s.ctx.CyclistProto.Play("sprint")
	}
	return nil
}

// Update advances the intro choreography and skips to TitleState on user input or completion.
func (s *IntroState) Update(dt float64) error {
	s.timer += dt
	if s.ctx.BGM != nil {
		_ = s.ctx.BGM.Update(time.Duration(dt * float64(time.Second)))
	}
	if s.ctx.CyclistProto != nil {
		s.ctx.CyclistProto.Update(time.Duration(dt * float64(time.Second)))
	}

	in := s.ctx.Input.Poll()
	if s.timer > 0.25 && in.AnyActivity {
		s.ctx.SFX.PlaySFX("ui_confirm", 80*time.Millisecond)
		return s.ctx.TransitionTo(StateTitle)
	}
	if s.timer >= 5.5 {
		return s.ctx.TransitionTo(StateTitle)
	}
	return nil
}

// Draw renders the broadcast intro sequence.
func (s *IntroState) Draw(screen *ebiten.Image) {
	screen.Fill(ui.PremultipliedRGBA(12, 18, 34, 255))

	if s.ctx.BackdropImg != nil {
		b := s.ctx.BackdropImg.Bounds()
		scaleX := float64(VirtualWidth) / float64(b.Dx())
		scaleY := float64(VirtualHeight) / float64(b.Dy())
		s.heroAnim.GeoM.Reset()
		s.heroAnim.GeoM.Scale(scaleX, scaleY)
		s.heroAnim.ColorScale.Reset()
		s.heroAnim.ColorScale.Scale(0.55, 0.55, 0.65, 1.0)
		screen.DrawImage(s.ctx.BackdropImg, s.heroAnim)
	}

	// Dark broadcast letterbox bars
	vector.FillRect(screen, 0, 0, VirtualWidth, 95, ui.PremultipliedRGBA(8, 10, 20, 235), false)
	vector.FillRect(screen, 0, VirtualHeight-95, VirtualWidth, 95, ui.PremultipliedRGBA(8, 10, 20, 235), false)
	vector.StrokeLine(screen, 0, 95, VirtualWidth, 95, 2, ui.ColPanelBorder, false)
	vector.StrokeLine(screen, 0, VirtualHeight-95, VirtualWidth, VirtualHeight-95, 2, ui.ColPanelBorder, false)

	// Animated cyclist sweeping across screen
	if s.ctx.CyclistProto != nil {
		riderX := 240.0 + s.timer*155.0
		s.ctx.CyclistProto.DrawAnchored(screen, riderX, 540, 0.95, 0.95, 0, nil, ebiten.FilterLinear)
	}

	ui.DrawPanel(screen, 220, 150, 840, 220, ui.ColPanelBg, ui.ColPanelBorder)
	s.ctx.Fonts.DrawAligned(screen, "LE GRAND TOUR EDITION SPECIALE", s.ctx.Fonts.BodyFace, VirtualWidth*0.5, 185, text.AlignCenter, text.AlignCenter, ui.ColTextCyan, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "PELOTON GRAND TOUR: BREAKAWAY", s.ctx.Fonts.TitleFace, VirtualWidth*0.5, 240, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "3 STAGES: COTE D'AZUR  |  COL DU GALIBIER  |  CHAMPS-ELYSEES", s.ctx.Fonts.BodyFace, VirtualWidth*0.5, 295, text.AlignCenter, text.AlignCenter, ui.ColTextWhite, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "Tuck into rival slipstreams (-65% drag), save Stamina, and launch the winning Sprint!", s.ctx.Fonts.SmallFace, VirtualWidth*0.5, 338, text.AlignCenter, text.AlignCenter, ui.ColTextMuted, ui.ColShadow)

	s.ctx.Fonts.DrawAligned(screen, "PRESS SPACE / ENTER / CLICK TO SKIP INTRO", s.ctx.Fonts.BodyFace, VirtualWidth*0.5, VirtualHeight-48, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
}

// Exit cleans up intro timers.
func (s *IntroState) Exit() error {
	s.timer = 0
	return nil
}
