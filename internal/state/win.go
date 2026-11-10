package state

import (
	"fmt"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"pelotongrandtour/internal/render"
	"pelotongrandtour/internal/ui"
)

// WinState renders the General Classification Maillot Jaune Podium Celebration.
type WinState struct {
	ctx        *Context
	podiumAnim *render.AnimationController
	bgDrawOpts ebiten.DrawImageOptions
	elapsed    float64
	scoreLine  string
	timeLine   string
	recordLine string
}

// NewWinState constructs the Grand Tour Victory screen.
func NewWinState(ctx *Context) *WinState {
	return &WinState{ctx: ctx}
}

func (s *WinState) Enter() error {
	s.elapsed = 0
	if s.ctx.CyclistProto != nil {
		s.podiumAnim = s.ctx.CyclistProto.Clone()
		_ = s.podiumAnim.Play("victory")
	}

	isNewRecord := false
	if s.ctx.TourScore > s.ctx.SaveData.HighScore {
		s.ctx.SaveData.HighScore = s.ctx.TourScore
		isNewRecord = true
	}
	tourTimeMs := int64(s.ctx.TourTimeSec * 1000)
	if s.ctx.SaveData.BestTourTimeMs <= 0 || tourTimeMs < s.ctx.SaveData.BestTourTimeMs {
		s.ctx.SaveData.BestTourTimeMs = tourTimeMs
	}
	s.ctx.SaveData.StagesCleared = 3
	if s.ctx.Storage != nil {
		_ = s.ctx.Storage.Save("career", s.ctx.SaveData)
	}

	s.scoreLine = fmt.Sprintf("FINAL GRAND TOUR SCORE: %d PTS   (FINAL STAGE RANK: #%d)", s.ctx.TourScore, s.ctx.LastStageRank)
	s.timeLine = fmt.Sprintf("CUMULATIVE GC TIME: %02d:%04.1f", int(s.ctx.TourTimeSec)/60, math.Mod(s.ctx.TourTimeSec, 60.0))
	if isNewRecord {
		s.recordLine = "★ NEW ALL-TIME GRAND TOUR RECORD! ★"
	} else {
		s.recordLine = fmt.Sprintf("ALL-TIME HIGH SCORE: %d PTS", s.ctx.SaveData.HighScore)
	}

	s.ctx.SFX.PlaySFX("fanfare_finish", 120*time.Millisecond)
	return nil
}

func (s *WinState) Update(dt float64) error {
	durDt := time.Duration(dt * float64(time.Second))
	if s.ctx.BGM != nil {
		_ = s.ctx.BGM.Update(durDt)
	}
	s.elapsed += dt
	if s.podiumAnim != nil {
		s.podiumAnim.Update(durDt)
	}

	in := s.ctx.Input.Poll()
	if s.elapsed > 0.4 && in.ConfirmJustPressed {
		s.ctx.SFX.PlaySFX("ui_confirm", 80*time.Millisecond)
		return s.ctx.TransitionTo(StateTitle)
	}
	if s.elapsed >= 12.0 {
		return s.ctx.TransitionTo(StateTitle)
	}
	return nil
}

func (s *WinState) Draw(screen *ebiten.Image) {
	screen.Fill(ui.PremultipliedRGBA(18, 22, 42, 255))

	if s.ctx.BackdropImg != nil {
		bw := float64(s.ctx.BackdropImg.Bounds().Dx())
		bh := float64(s.ctx.BackdropImg.Bounds().Dy())
		s.bgDrawOpts.GeoM.Reset()
		s.bgDrawOpts.ColorScale.Reset()
		s.bgDrawOpts.GeoM.Scale(float64(VirtualWidth)/bw, float64(VirtualHeight)/bh)
		s.bgDrawOpts.ColorScale.Scale(0.48, 0.48, 0.56, 1.0)
		screen.DrawImage(s.ctx.BackdropImg, &s.bgDrawOpts)
	}

	ui.DrawPanel(screen, 160, 56, 960, 608, ui.ColPanelBg, ui.ColPanelBorder)

	s.ctx.Fonts.DrawAligned(screen, "MAILLOT JAUNE CHAMPION!", s.ctx.Fonts.TitleFace, 640, 110, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "GENERAL CLASSIFICATION WINNER — PELOTON GRAND TOUR", s.ctx.Fonts.BodyFace, 640, 166, text.AlignCenter, text.AlignCenter, ui.ColTextCyan, ui.ColShadow)

	// Golden podium step & celebrating cyclist
	vector.FillRect(screen, 520, 405, 240, 32, ui.ColTextGold, false)
	vector.StrokeRect(screen, 520, 405, 240, 32, 2.0, ui.ColTextWhite, false)
	s.ctx.Fonts.DrawAligned(screen, "1ST PLACE GC PODIUM", s.ctx.Fonts.SmallFace, 640, 421, text.AlignCenter, text.AlignCenter, ui.ColShadow, ui.ColTextWhite)

	if s.podiumAnim != nil {
		s.podiumAnim.DrawAnchored(screen, 640, 404, 0.78, 0.78, 0, ui.ColTextWhite, ebiten.FilterLinear)
	}

	s.ctx.Fonts.DrawAligned(screen, s.scoreLine, s.ctx.Fonts.LargeFace, 640, 480, text.AlignCenter, text.AlignCenter, ui.ColTextWhite, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, s.timeLine, s.ctx.Fonts.BodyFace, 640, 526, text.AlignCenter, text.AlignCenter, ui.ColTextCyan, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, s.recordLine, s.ctx.Fonts.BodyFace, 640, 566, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)

	s.ctx.Fonts.DrawAligned(screen, "PRESS [SPACE] OR [ENTER] TO RETURN TO TITLE", s.ctx.Fonts.BodyFace, 640, 622, text.AlignCenter, text.AlignCenter, ui.ColTextGreen, ui.ColShadow)
}

func (s *WinState) Exit() error {
	return nil
}
