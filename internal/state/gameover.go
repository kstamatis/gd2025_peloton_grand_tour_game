package state

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"pelotongrandtour/internal/render"
	"pelotongrandtour/internal/ui"
)

// GameOverState renders the Voiture Balai (Broom Wagon) elimination screen.
type GameOverState struct {
	ctx        *Context
	riderAnim  *render.AnimationController
	bgDrawOpts ebiten.DrawImageOptions
	elapsed    float64
	menuIdx    int
	scoreLine  string
	reasonLine string
}

// NewGameOverState constructs the GameOverState.
func NewGameOverState(ctx *Context) *GameOverState {
	return &GameOverState{ctx: ctx}
}

func (s *GameOverState) Enter() error {
	s.elapsed = 0
	s.menuIdx = 0
	if s.ctx.CyclistProto != nil {
		s.riderAnim = s.ctx.CyclistProto.Clone()
		_ = s.riderAnim.Play("idle")
	}
	if s.ctx.TourScore > s.ctx.SaveData.HighScore {
		s.ctx.SaveData.HighScore = s.ctx.TourScore
		if s.ctx.Storage != nil {
			_ = s.ctx.Storage.Save("career", s.ctx.SaveData)
		}
	}
	s.scoreLine = fmt.Sprintf("TOUR SCORE AT ELIMINATION: %d PTS   (HIGH SCORE: %d PTS)", s.ctx.TourScore, s.ctx.SaveData.HighScore)
	if s.ctx.FailureReason != "" {
		s.reasonLine = s.ctx.FailureReason
	} else {
		s.reasonLine = "Eliminated outside the General Classification time cut."
	}
	return nil
}

func (s *GameOverState) Update(dt float64) error {
	durDt := time.Duration(dt * float64(time.Second))
	if s.ctx.BGM != nil {
		_ = s.ctx.BGM.Update(durDt)
	}
	s.elapsed += dt
	if s.riderAnim != nil {
		s.riderAnim.Update(durDt)
	}

	in := s.ctx.Input.Poll()
	if in.LaneUpJustPressed || in.LaneDownJustPressed {
		s.menuIdx = 1 - s.menuIdx
		s.ctx.SFX.PlaySFX("ui_select", 80*time.Millisecond)
	}
	if s.elapsed > 0.35 && in.ConfirmJustPressed {
		s.ctx.SFX.PlaySFX("ui_confirm", 80*time.Millisecond)
		if s.menuIdx == 0 {
			return s.ctx.TransitionTo(StatePlay)
		}
		return s.ctx.TransitionTo(StateTitle)
	}
	if s.elapsed >= 10.0 {
		return s.ctx.TransitionTo(StateTitle)
	}
	return nil
}

func (s *GameOverState) Draw(screen *ebiten.Image) {
	screen.Fill(ui.PremultipliedRGBA(24, 14, 22, 255))

	if s.ctx.BackdropImg != nil {
		bw := float64(s.ctx.BackdropImg.Bounds().Dx())
		bh := float64(s.ctx.BackdropImg.Bounds().Dy())
		s.bgDrawOpts.GeoM.Reset()
		s.bgDrawOpts.ColorScale.Reset()
		s.bgDrawOpts.GeoM.Scale(float64(VirtualWidth)/bw, float64(VirtualHeight)/bh)
		s.bgDrawOpts.ColorScale.Scale(0.32, 0.25, 0.28, 1.0)
		screen.DrawImage(s.ctx.BackdropImg, &s.bgDrawOpts)
	}

	ui.DrawPanel(screen, 170, 72, 940, 576, ui.ColPanelBg, ui.ColTextRed)

	s.ctx.Fonts.DrawAligned(screen, "HORS DELAI — RACE ELIMINATED", s.ctx.Fonts.TitleFace, 640, 128, text.AlignCenter, text.AlignCenter, ui.ColTextRed, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, s.reasonLine, s.ctx.Fonts.BodyFace, 640, 188, text.AlignCenter, text.AlignCenter, ui.ColTextWhite, ui.ColShadow)

	if s.riderAnim != nil {
		s.riderAnim.DrawAnchored(screen, 640, 396, 0.68, 0.68, 0, ui.PremultipliedRGBA(210, 210, 225, 255), ebiten.FilterLinear)
	}

	s.ctx.Fonts.DrawAligned(screen, s.scoreLine, s.ctx.Fonts.BodyFace, 640, 452, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)

	retryCol := ui.ColTextWhite
	titleCol := ui.ColTextWhite
	if s.menuIdx == 0 {
		retryCol = ui.ColTextGold
	} else {
		titleCol = ui.ColTextGold
	}
	s.ctx.Fonts.DrawAligned(screen, "▶ RETRY CURRENT STAGE", s.ctx.Fonts.LargeFace, 640, 520, text.AlignCenter, text.AlignCenter, retryCol, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "▶ RETURN TO TITLE SCREEN", s.ctx.Fonts.LargeFace, 640, 572, text.AlignCenter, text.AlignCenter, titleCol, ui.ColShadow)
}

func (s *GameOverState) Exit() error {
	return nil
}
