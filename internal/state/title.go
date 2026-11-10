package state

import (
	"fmt"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"pelotongrandtour/internal/ui"
)

const titleIdleTimeoutSec = 10.0

// TitleState renders the main menu, control guide, career records, and 10s attract-mode timer.
type TitleState struct {
	ctx           *Context
	idleTimer     float64
	scrollX       float64
	bgDrawOp      ebiten.DrawImageOptions
	bestRecordStr string
}

// NewTitleState constructs the TitleState.
func NewTitleState(ctx *Context) *TitleState {
	return &TitleState{ctx: ctx}
}

// Enter starts BGM and resets the 10-second attract-mode idle timer.
func (s *TitleState) Enter() error {
	s.idleTimer = 0
	if s.ctx.BGM != nil {
		s.ctx.BGM.SetVolume(0.65)
		s.ctx.BGM.Start()
	}
	if s.ctx.CyclistProto != nil {
		_ = s.ctx.CyclistProto.Play("cruise")
	}
	if s.ctx.GelProto != nil {
		_ = s.ctx.GelProto.Play("spin")
	}
	if s.ctx.BidonProto != nil {
		_ = s.ctx.BidonProto.Play("spin")
	}
	s.bestRecordStr = fmt.Sprintf("CAREER HIGH SCORE: %d PTS   |   STAGES CLEARED: %d / 3",
		s.ctx.SaveData.HighScore, s.ctx.SaveData.StagesCleared)
	return nil
}

// Update handles menu confirmation and the 10s idle attract-mode transition.
func (s *TitleState) Update(dt float64) error {
	s.idleTimer += dt
	s.scrollX += dt * 90.0

	if s.ctx.BGM != nil {
		_ = s.ctx.BGM.Update(time.Duration(dt * float64(time.Second)))
	}
	if s.ctx.CyclistProto != nil {
		s.ctx.CyclistProto.Update(time.Duration(dt * float64(time.Second)))
	}
	if s.ctx.GelProto != nil {
		s.ctx.GelProto.Update(time.Duration(dt * float64(time.Second)))
	}
	if s.ctx.BidonProto != nil {
		s.ctx.BidonProto.Update(time.Duration(dt * float64(time.Second)))
	}

	in := s.ctx.Input.Poll()
	if in.ConfirmJustPressed {
		s.ctx.SFX.PlaySFX("ui_confirm", 80*time.Millisecond)
		s.ctx.CurrentStageIndex = 0
		s.ctx.TourTimeSec = 0
		s.ctx.TourScore = 0
		return s.ctx.TransitionTo(StatePlay)
	}
	if in.AnyActivity {
		s.idleTimer = 0
	}

	// Arcade Attract Mode: after 10s of idle on Title Screen, alternate between Demo Mode and Intro
	if s.idleTimer >= titleIdleTimeoutSec {
		s.idleTimer = 0
		s.ctx.AttractToggle = !s.ctx.AttractToggle
		if s.ctx.AttractToggle {
			return s.ctx.TransitionTo(StateDemo)
		}
		return s.ctx.TransitionTo(StateIntro)
	}
	return nil
}

// Draw renders the Title Screen with scrolling Alpine backdrop, 2D HD cyclist showcase, and controls.
func (s *TitleState) Draw(screen *ebiten.Image) {
	screen.Fill(ui.PremultipliedRGBA(18, 26, 48, 255))

	if s.ctx.BackdropImg != nil {
		b := s.ctx.BackdropImg.Bounds()
		scaleX := float64(VirtualWidth) / float64(b.Dx())
		scaleY := float64(480) / float64(b.Dy())
		offset := math.Mod(s.scrollX, float64(VirtualWidth))
		for i := 0; i < 2; i++ {
			s.bgDrawOp.GeoM.Reset()
			s.bgDrawOp.GeoM.Scale(scaleX, scaleY)
			s.bgDrawOp.GeoM.Translate(-offset+float64(i*VirtualWidth), 0)
			screen.DrawImage(s.ctx.BackdropImg, &s.bgDrawOp)
		}
	}

	// Road surface preview at bottom
	vector.FillRect(screen, 0, 460, VirtualWidth, 260, ui.PremultipliedRGBA(46, 50, 62, 255), false)
	vector.FillRect(screen, 0, 452, VirtualWidth, 8, ui.ColTextGold, false)
	for x := -int(s.scrollX*3) % 80; x < VirtualWidth; x += 80 {
		vector.FillRect(screen, float32(x), 545, 42, 4, ui.PremultipliedRGBA(220, 225, 240, 180), false)
		vector.FillRect(screen, float32(x), 625, 42, 4, ui.PremultipliedRGBA(220, 225, 240, 180), false)
	}

	// Showcase 2D HD Cyclist + Procedural Pickups
	if s.ctx.CyclistProto != nil {
		s.ctx.CyclistProto.DrawAnchored(screen, 310, 645, 0.92, 0.92, 0, nil, ebiten.FilterLinear)
	}
	if s.ctx.GelProto != nil {
		s.ctx.GelProto.DrawAnchored(screen, 535, 595, 1.15, 1.15, 0, nil, ebiten.FilterNearest)
	}
	if s.ctx.BidonProto != nil {
		s.ctx.BidonProto.DrawAnchored(screen, 620, 595, 1.15, 1.15, 0, nil, ebiten.FilterNearest)
	}

	// Title Banner
	ui.DrawPanel(screen, 80, 36, 1120, 136, ui.ColPanelBg, ui.ColPanelBorder)
	s.ctx.Fonts.DrawAligned(screen, "PELOTON GRAND TOUR: BREAKAWAY", s.ctx.Fonts.TitleFace, VirtualWidth*0.5, 82, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "2D HD ARCADE CYCLING  |  DRAFT IN THE SLIPSTREAM  |  CONQUER 3 GRAND TOUR STAGES", s.ctx.Fonts.BodyFace, VirtualWidth*0.5, 136, text.AlignCenter, text.AlignCenter, ui.ColTextCyan, ui.ColShadow)

	// Right Panel: Controls & Tactics Guide
	ui.DrawPanel(screen, 690, 200, 510, 360, ui.ColPanelBg, ui.ColPanelBorder)
	s.ctx.Fonts.DrawAligned(screen, "RACE CONTROLS & PELOTON TACTICS", s.ctx.Fonts.LargeFace, 945, 234, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "W / S or UP / DOWN : Switch Road Lanes (4 Lanes)", s.ctx.Fonts.BodyFace, 718, 282, text.AlignStart, text.AlignCenter, ui.ColTextWhite, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "D / A or RIGHT / LEFT : Tempo Effort / Coast & Recover", s.ctx.Fonts.BodyFace, 718, 316, text.AlignStart, text.AlignCenter, ui.ColTextWhite, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "SPACE / Z / L-CLICK : Aero Sprint Surge (+38% Speed)", s.ctx.Fonts.BodyFace, 718, 350, text.AlignStart, text.AlignCenter, ui.ColTextCyan, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "X / SHIFT / E / R-CLICK : Consume Energy Gel (+35% Stamina)", s.ctx.Fonts.BodyFace, 718, 384, text.AlignStart, text.AlignCenter, ui.ColTextGreen, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "SLIPSTREAM DRAFTING : Ride 1.5m-6m behind a rival rider", s.ctx.Fonts.SmallFace, 718, 426, text.AlignStart, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, "to cut wind drag by 65% and regenerate Stamina!", s.ctx.Fonts.SmallFace, 718, 448, text.AlignStart, text.AlignCenter, ui.ColTextGold, ui.ColShadow)

	s.ctx.Fonts.DrawAligned(screen, s.bestRecordStr, s.ctx.Fonts.SmallFace, 945, 516, text.AlignCenter, text.AlignCenter, ui.ColTextMuted, ui.ColShadow)

	// Pulsing Start Prompt
	pulse := 0.65 + 0.35*math.Sin(s.scrollX*0.08)
	promptCol := ui.PremultipliedRGBA(255, 220, 64, uint8(255*pulse))
	ui.DrawPanel(screen, 310, 608, 660, 68, ui.ColPanelBg, ui.ColPanelBorder)
	s.ctx.Fonts.DrawAligned(screen, "PRESS SPACE / ENTER / CLICK TO START STAGE 1", s.ctx.Fonts.LargeFace, VirtualWidth*0.5, 642, text.AlignCenter, text.AlignCenter, promptCol, ui.ColShadow)
}

// Exit resets title timers.
func (s *TitleState) Exit() error {
	s.idleTimer = 0
	return nil
}
