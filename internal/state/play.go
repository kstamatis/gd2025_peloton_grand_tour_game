package state

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"pelotongrandtour/internal/camera"
	"pelotongrandtour/internal/entity"
	"pelotongrandtour/internal/input"
	"pelotongrandtour/internal/physics"
	"pelotongrandtour/internal/ui"
)

const (
	maxCourseItems   = 48
	maxWindParticles = 96
)

// PlayState implements the core multi-lane peloton race and AI Attract Mode.
type PlayState struct {
	ctx          *Context
	demoMode     bool
	stages       []entity.StageSpec
	stage        entity.StageSpec
	riders       []*entity.Rider
	sortedRiders []*entity.Rider
	player       *entity.Rider
	items        []entity.CourseItem
	particles    []entity.WindParticle

	cam        *camera.Camera
	shaker     *camera.TraumaShaker
	timeCtrl   *camera.TimeController
	bgDrawOpts ebiten.DrawImageOptions

	stageTimeSec      float64
	draftWhooshTimer  float64
	freewheelTimer    float64
	paused            bool
	pauseMenuIdx      int
	stageFinished     bool
	finishBannerTimer float64
	playerRank        int
	stageWon          bool

	// Pre-computed HUD strings updated in Update(dt) for zero-allocation Draw()
	hudSpeedStr    string
	hudCadenceStr  string
	hudStaminaStr  string
	hudGelsStr     string
	hudStageTitle  string
	hudSegmentStr  string
	hudDistRemStr  string
	hudRankStr     string
	hudTimeStr     string
	hudBannerText  string
	hudBannerColor color.RGBA
	hudFinishTitle string
}

// NewPlayState constructs a PlayState (either interactive or autonomous Demo Mode).
func NewPlayState(ctx *Context, demoMode bool) *PlayState {
	cam, _ := camera.NewCamera(VirtualWidth, VirtualHeight)
	shaker, _ := camera.NewTraumaShaker(1.8, 14.0, 0.018)
	return &PlayState{
		ctx:          ctx,
		demoMode:     demoMode,
		stages:       entity.AllStages(),
		items:        make([]entity.CourseItem, maxCourseItems),
		particles:    make([]entity.WindParticle, maxWindParticles),
		cam:          cam,
		shaker:       shaker,
		timeCtrl:     camera.NewTimeController(),
		sortedRiders: make([]*entity.Rider, 0, 16),
	}
}

func (s *PlayState) Enter() error {
	if s.ctx.CurrentStageIndex < 0 || s.ctx.CurrentStageIndex >= len(s.stages) {
		s.ctx.CurrentStageIndex = 0
	}
	s.stage = s.stages[s.ctx.CurrentStageIndex]

	riders, player, err := entity.BuildPeloton(s.ctx.CyclistProto, s.stage.Number)
	if err != nil {
		return err
	}
	s.riders = riders
	s.player = player
	s.sortedRiders = s.sortedRiders[:0]
	s.sortedRiders = append(s.sortedRiders, s.riders...)

	s.stageTimeSec = 0
	s.paused = false
	s.pauseMenuIdx = 0
	s.stageFinished = false
	s.finishBannerTimer = 0
	s.playerRank = 1
	s.stageWon = false

	s.populateCourseItems()
	for i := range s.particles {
		s.particles[i].Active = false
	}

	s.shaker.Reset(s.cam)
	s.cam.TargetX = s.player.RoadX * entity.PixelsPerMeter
	s.cam.TargetY = VirtualHeight * 0.5
	s.cam.Reset()

	if s.ctx.BGM != nil {
		s.ctx.BGM.SetVolume(0.65)
		s.ctx.BGM.Start()
	}

	s.updateHUDStrings(0.0, s.stage.Segments[0].Label)
	return nil
}

func (s *PlayState) populateCourseItems() {
	for i := range s.items {
		s.items[i].Active = false
	}
	idx := 0
	for dist := 140.0; dist < s.stage.LengthMeters-140.0 && idx < len(s.items); dist += 68.0 {
		lane := (idx * 3) % physics.NumLanes
		kind := entity.ItemBidon
		anim := s.ctx.BidonProto.Clone()
		if idx%4 == 1 {
			kind = entity.ItemEnergyGel
			anim = s.ctx.GelProto.Clone()
		} else if idx%3 == 2 && (s.stage.Number == 3 || dist > 450.0) {
			kind = entity.ItemCobbleHazard
			anim = s.ctx.CobbleProto.Clone()
		}
		s.items[idx] = entity.CourseItem{
			Active: true,
			Kind:   kind,
			RoadX:  dist,
			Lane:   lane,
			Y:      physics.LaneCenterY(lane),
			Anim:   anim,
		}
		idx++
	}
}

func (s *PlayState) Update(dt float64) error {
	durDt := time.Duration(dt * float64(time.Second))
	if s.ctx.BGM != nil {
		_ = s.ctx.BGM.Update(durDt)
	}

	in := s.ctx.Input.Poll()

	// Instant interrupt in Demo Attract Mode
	if s.demoMode && in.AnyActivity {
		return s.ctx.TransitionTo(StateTitle)
	}

	// Pause menu handling in interactive PlayState
	if !s.demoMode && !s.stageFinished && in.PauseJustPressed {
		s.paused = !s.paused
		s.ctx.SFX.PlaySFX("ui_select", 80*time.Millisecond)
		return nil
	}
	if s.paused {
		if in.LaneUpJustPressed || in.LaneDownJustPressed {
			s.pauseMenuIdx = 1 - s.pauseMenuIdx
			s.ctx.SFX.PlaySFX("ui_select", 80*time.Millisecond)
		}
		if in.ConfirmJustPressed {
			s.ctx.SFX.PlaySFX("ui_confirm", 80*time.Millisecond)
			if s.pauseMenuIdx == 0 {
				s.paused = false
			} else {
				return s.ctx.TransitionTo(StateTitle)
			}
		}
		return nil
	}

	worldDt, frozen := s.timeCtrl.Step(dt)
	if frozen {
		s.shaker.Update(dt, s.cam)
		return nil
	}
	worldDur := time.Duration(worldDt * float64(time.Second))

	if s.stageFinished {
		s.finishBannerTimer += worldDt
		for _, r := range s.riders {
			r.UpdateDesiredAnimation(true, r == s.player && s.stageWon)
			r.Anim.Update(worldDur)
			r.Spring.Update(worldDt)
		}
		if s.finishBannerTimer >= 3.2 || in.ConfirmJustPressed {
			return s.advanceAfterStage()
		}
		return nil
	}

	s.stageTimeSec += worldDt
	s.ctx.TourTimeSec += worldDt

	// Update course items
	for i := range s.items {
		if s.items[i].Active && s.items[i].Anim != nil {
			s.items[i].Anim.Update(worldDur)
		}
	}

	// Step player or autonomous demo AI controls
	if s.demoMode {
		s.updateDemoAI(worldDt)
	} else {
		s.handlePlayerControls(in)
	}

	// Compute slipstream drafting factors & step rider physics
	for _, r := range s.riders {
		s.updateRiderDraft(r)
	}
	for _, r := range s.riders {
		if !r.IsPlayer {
			s.updateRivalAI(r, worldDt)
		}
		s.stepRiderPhysics(r, worldDt, in)
	}

	// Check roadside pickups and cobble hazards for player
	s.checkPlayerItems()

	// Update slipstream & sprint wind particles
	s.updateParticles(worldDt)

	// Compute player live rank (1 = furthest ahead)
	rank := 1
	for _, r := range s.riders {
		if r != s.player && r.RoadX > s.player.RoadX {
			rank++
		}
	}
	s.playerRank = rank

	// Sort riders back-to-front by Y for painter's order in Draw()
	sort.Slice(s.sortedRiders, func(i, j int) bool {
		if math.Abs(s.sortedRiders[i].Y-s.sortedRiders[j].Y) > 1.0 {
			return s.sortedRiders[i].Y < s.sortedRiders[j].Y
		}
		return s.sortedRiders[i].RoadX < s.sortedRiders[j].RoadX
	})

	// Check stage completion or catastrophic elimination
	if s.player.RoadX >= s.stage.LengthMeters {
		s.stageFinished = true
		s.finishBannerTimer = 0
		s.ctx.LastStageRank = s.playerRank
		s.stageWon = s.playerRank <= 3
		s.hudFinishTitle = fmt.Sprintf("STAGE %d COMPLETE — RANK %d / %d", s.stage.Number, s.playerRank, len(s.riders))
		stagePts := math.Max(100, float64(1800-(s.playerRank-1)*250)+s.player.Stamina*5.0)
		s.ctx.TourScore += int(stagePts)
		s.ctx.SFX.PlaySFX("fanfare_finish", 150*time.Millisecond)
		s.timeCtrl.TriggerHitstop(0.08)
	} else {
		// Check if leader finished far ahead while player bonked
		leaderX := s.player.RoadX
		for _, r := range s.riders {
			if r.RoadX > leaderX {
				leaderX = r.RoadX
			}
		}
		if leaderX-s.player.RoadX > 95.0 && s.player.BonkedTimer > 0 && !s.demoMode {
			s.ctx.FailureReason = "Dropped by the Peloton after bonking! The Voiture Balai broom wagon swept you up."
			return s.ctx.TransitionTo(StateGameOver)
		}
	}

	// Update camera follow & trauma shake
	leadOffsetPx := 260.0 + (s.player.SpeedMps-14.0)*14.0
	s.cam.TargetX = s.player.RoadX*entity.PixelsPerMeter + leadOffsetPx
	s.cam.TargetY = VirtualHeight * 0.5
	s.cam.UpdateFollow(worldDt)
	s.shaker.Update(worldDt, s.cam)

	gradPct, segLabel := s.stage.GradientAt(s.player.RoadX)
	s.updateHUDStrings(gradPct, segLabel)

	return nil
}

func (s *PlayState) advanceAfterStage() error {
	if s.demoMode {
		return s.ctx.TransitionTo(StateTitle)
	}
	// Must finish in top 5 to qualify for next Grand Tour stage
	if s.playerRank > 5 {
		s.ctx.FailureReason = fmt.Sprintf("Finished %dth on Stage %d (Top 5 Podium Cutoff required for GC classification).", s.playerRank, s.stage.Number)
		return s.ctx.TransitionTo(StateGameOver)
	}
	if s.ctx.CurrentStageIndex+1 < len(s.stages) {
		s.ctx.CurrentStageIndex++
		if s.ctx.CurrentStageIndex > s.ctx.SaveData.StagesCleared {
			s.ctx.SaveData.StagesCleared = s.ctx.CurrentStageIndex
		}
		if s.ctx.Storage != nil {
			_ = s.ctx.Storage.Save("career", s.ctx.SaveData)
		}
		return s.Enter()
	}
	return s.ctx.TransitionTo(StateWin)
}

func (s *PlayState) handlePlayerControls(in input.Intent) {
	p := s.player
	if in.LaneUpJustPressed && p.Lane > 0 {
		p.Lane--
		p.Spring.Impulse(0.90)
		s.ctx.SFX.PlaySFX("gear_shift", 70*time.Millisecond)
	} else if in.LaneDownJustPressed && p.Lane < physics.NumLanes-1 {
		p.Lane++
		p.Spring.Impulse(1.10)
		s.ctx.SFX.PlaySFX("gear_shift", 70*time.Millisecond)
	}

	if in.UseGelJustPressed && p.Gels > 0 && p.GelBoostTimer <= 0 {
		p.Gels--
		p.Stamina = math.Min(p.MaxStamina, p.Stamina+42.0)
		p.GelBoostTimer = 3.5
		p.BonkedTimer = 0
		p.Spring.Impulse(1.24)
		s.shaker.AddTrauma(0.22)
		s.ctx.SFX.PlaySFX("gel_pickup", 90*time.Millisecond)
	}

	p.Tucking = in.PaceDownHeld && !in.SprintHeld
	wantsSprint := in.SprintHeld && p.Stamina > 4.0 && p.BonkedTimer <= 0
	if wantsSprint && !p.Sprinting && in.SprintJustPressed {
		p.Spring.Impulse(1.20)
		s.shaker.AddTrauma(0.18)
		s.ctx.SFX.PlaySFX("sprint_surge", 120*time.Millisecond)
	}
	p.Sprinting = wantsSprint

	if in.PaceUpHeld {
		p.GearLabel = "53x11"
	} else if p.Tucking {
		p.GearLabel = "39x25"
	} else {
		p.GearLabel = "53x14"
	}
}

func (s *PlayState) updateDemoAI(dt float64) {
	p := s.player
	p.AITimer -= dt
	if p.AITimer <= 0 {
		p.AITimer = 0.65
		bestLane := p.Lane
		bestDist := 999.0
		for _, r := range s.riders {
			if r == p {
				continue
			}
			gap := r.RoadX - p.RoadX
			if gap > 1.8 && gap < 14.0 && gap < bestDist {
				bestDist = gap
				bestLane = r.Lane
			}
		}
		if bestLane < p.Lane {
			p.Lane--
		} else if bestLane > p.Lane {
			p.Lane++
		}
	}
	if p.Stamina < 40.0 && p.Gels > 0 && p.GelBoostTimer <= 0 {
		p.Gels--
		p.Stamina = math.Min(p.MaxStamina, p.Stamina+42.0)
		p.GelBoostTimer = 3.2
	}
	rem := s.stage.LengthMeters - p.RoadX
	p.Sprinting = (rem < 350.0 && p.Stamina > 20.0) || (p.Stamina > 78.0 && math.Sin(s.stageTimeSec*1.7) > 0.55)
	p.Tucking = !p.Sprinting && p.DraftFactor > 0.45
}

func (s *PlayState) updateRivalAI(r *entity.Rider, dt float64) {
	r.AITimer -= dt
	if r.AITimer <= 0 {
		r.AITimer = 0.85 + float64(r.Lane)*0.18
		for i := range s.items {
			it := &s.items[i]
			if it.Active && it.Kind == entity.ItemCobbleHazard && it.Lane == r.Lane {
				gap := it.RoadX - r.RoadX
				if gap > 2.0 && gap < 22.0 {
					if r.Lane > 0 {
						r.Lane--
					} else {
						r.Lane++
					}
					break
				}
			}
		}
	}

	gradPct, _ := s.stage.GradientAt(r.RoadX)
	rem := s.stage.LengthMeters - r.RoadX
	wantsSprint := false
	switch r.Role {
	case entity.RoleGrimpeur:
		wantsSprint = gradPct >= 4.5 && r.Stamina > 28.0
	case entity.RoleSprinter:
		wantsSprint = rem < 420.0 && r.Stamina > 18.0
	case entity.RoleRouleur:
		wantsSprint = r.DraftFactor < 0.25 && r.Stamina > 62.0
	}
	r.Sprinting = wantsSprint && r.BonkedTimer <= 0
	r.Tucking = r.DraftFactor > 0.55 && !r.Sprinting
}

func (s *PlayState) updateRiderDraft(r *entity.Rider) {
	best := 0.0
	for _, other := range s.riders {
		if other == r {
			continue
		}
		f := physics.ComputeSlipstreamFactor(r.RoadX, r.Y, other.RoadX, other.Y)
		if f > best {
			best = f
		}
	}
	r.DraftFactor = best
}

func (s *PlayState) stepRiderPhysics(r *entity.Rider, dt float64, in input.Intent) {
	targetY := physics.LaneCenterY(r.Lane)
	r.Y = camera.ExpDecay(r.Y, targetY, 11.5, dt)

	if r.BonkedTimer > 0 {
		r.BonkedTimer = math.Max(0, r.BonkedTimer-dt)
	}
	if r.CobbleTimer > 0 {
		r.CobbleTimer = math.Max(0, r.CobbleTimer-dt)
	}
	if r.GelBoostTimer > 0 {
		r.GelBoostTimer = math.Max(0, r.GelBoostTimer-dt)
	}

	gradPct, _ := s.stage.GradientAt(r.RoadX)
	slopePenalty := gradPct * 0.34
	if r.Role == entity.RoleGrimpeur && gradPct > 3.0 {
		slopePenalty *= 0.68
	}

	targetSpeed := r.BasePower - slopePenalty

	// Slipstream aerodynamic drag reduction (up to +2.35 m/s boost when tucked in slipstream)
	targetSpeed += r.DraftFactor * 2.35

	// Player pace control
	if r.IsPlayer && !s.demoMode {
		if in.PaceUpHeld {
			targetSpeed += 1.05
		} else if in.PaceDownHeld {
			targetSpeed -= 1.25
		}
	}

	if r.Sprinting {
		sprintBonus := 3.85
		if r.Role == entity.RoleSprinter {
			sprintBonus = 4.25
		}
		targetSpeed += sprintBonus
		drain := (15.5 - r.DraftFactor*6.5) * dt
		r.Stamina = math.Max(0, r.Stamina-drain)
		if r.Stamina <= 0.2 {
			r.Stamina = 0
			r.BonkedTimer = 3.2
			r.Sprinting = false
			if r.IsPlayer {
				s.shaker.AddTrauma(0.45)
			}
		}
	} else {
		var netStaminaRate float64
		if r.DraftFactor >= 0.35 {
			netStaminaRate = 7.5 * r.DraftFactor
			if r.IsPlayer {
				s.draftWhooshTimer -= dt
				if s.draftWhooshTimer <= 0 {
					s.draftWhooshTimer = 1.35
					s.ctx.SFX.PlaySFX("draft_whoosh", 250*time.Millisecond)
				}
			}
		} else if r.Tucking {
			netStaminaRate = 5.2
			if r.IsPlayer {
				s.freewheelTimer -= dt
				if s.freewheelTimer <= 0 {
					s.freewheelTimer = 1.6
					s.ctx.SFX.PlaySFX("freewheel_click", 250*time.Millisecond)
				}
			}
		} else {
			netStaminaRate = -1.6 + r.DraftFactor*4.0
		}
		if gradPct > 5.0 && r.DraftFactor < 0.3 {
			netStaminaRate -= 2.2
		}
		r.Stamina = math.Max(0, math.Min(r.MaxStamina, r.Stamina+netStaminaRate*dt))
	}

	if r.GelBoostTimer > 0 {
		targetSpeed += 2.65
	}
	if r.BonkedTimer > 0 {
		targetSpeed *= 0.62
	}
	if r.CobbleTimer > 0 {
		targetSpeed *= 0.74
	}

	if targetSpeed < 6.5 {
		targetSpeed = 6.5
	}

	r.SpeedMps = camera.ExpDecay(r.SpeedMps, targetSpeed, 5.5, dt)
	r.RoadX += r.SpeedMps * dt
	r.CadenceRPM = int(68 + (r.SpeedMps-10.0)*4.8)
	if r.Sprinting {
		r.CadenceRPM += 18
	}

	if r.IsPlayer && r.SpeedKmh() > s.ctx.SaveData.TopSpeedKmh {
		s.ctx.SaveData.TopSpeedKmh = r.SpeedKmh()
	}

	r.UpdateDesiredAnimation(false, false)
	r.Anim.Update(time.Duration(dt * float64(time.Second)))
	r.Spring.Update(dt)

	// Spawn slipstream or sprint particles
	if (r.DraftFactor > 0.45 || r.Sprinting || r.GelBoostTimer > 0) && math.Mod(r.RoadX*10, 2.0) < 0.6 {
		col := ui.ColTextCyan
		if r.Sprinting || r.GelBoostTimer > 0 {
			col = ui.ColTextGold
		}
		s.spawnWindParticle(r.RoadX*entity.PixelsPerMeter-32, r.Y-28, col)
	}
}

func (s *PlayState) checkPlayerItems() {
	p := s.player
	for i := range s.items {
		it := &s.items[i]
		if !it.Active {
			continue
		}
		if physics.CheckPickupOverlap(p.RoadX, p.Y, it.RoadX, it.Y, 1.85, 28.0) {
			switch it.Kind {
			case entity.ItemEnergyGel:
				it.Active = false
				p.Gels = int(math.Min(5, float64(p.Gels+1)))
				p.Stamina = math.Min(p.MaxStamina, p.Stamina+18.0)
				p.Spring.Impulse(1.16)
				s.ctx.TourScore += 150
				s.ctx.SFX.PlaySFX("gel_pickup", 80*time.Millisecond)
			case entity.ItemBidon:
				it.Active = false
				p.Stamina = math.Min(p.MaxStamina, p.Stamina+26.0)
				p.BonkedTimer = 0
				p.Spring.Impulse(1.14)
				s.ctx.TourScore += 100
				s.ctx.SFX.PlaySFX("gel_pickup", 80*time.Millisecond)
			case entity.ItemCobbleHazard:
				if p.CobbleTimer <= 0 {
					p.CobbleTimer = 1.15
					p.Stamina = math.Max(0, p.Stamina-9.0)
					p.Spring.Impulse(0.82)
					s.shaker.AddTrauma(0.38)
					s.timeCtrl.TriggerHitstop(0.035)
					s.ctx.SFX.PlaySFX("cobble_rattle", 120*time.Millisecond)
				}
			}
		}
	}
}

func (s *PlayState) spawnWindParticle(worldX, screenY float64, col color.RGBA) {
	for i := range s.particles {
		pt := &s.particles[i]
		if !pt.Active {
			pt.Active = true
			pt.WorldX = worldX
			pt.ScreenY = screenY + float64((i%5)-2)*6.0
			pt.VelX = -320.0 - float64(i%4)*45.0
			pt.VelY = float64((i%3)-1) * 12.0
			pt.Length = 28.0 + float32(i%3)*12.0
			pt.MaxLife = 0.32
			pt.Life = pt.MaxLife
			pt.Color = col
			return
		}
	}
}

func (s *PlayState) updateParticles(dt float64) {
	for i := range s.particles {
		pt := &s.particles[i]
		if !pt.Active {
			continue
		}
		pt.Life -= dt
		if pt.Life <= 0 {
			pt.Active = false
			continue
		}
		pt.WorldX += pt.VelX * dt
		pt.ScreenY += pt.VelY * dt
	}
}

func (s *PlayState) updateHUDStrings(gradPct float64, segLabel string) {
	p := s.player
	s.hudSpeedStr = fmt.Sprintf("%.1f km/h", p.SpeedKmh())
	s.hudCadenceStr = fmt.Sprintf("%d RPM  |  GEAR %s  |  SLOPE %+.1f%%", p.CadenceRPM, p.GearLabel, gradPct)
	s.hudStaminaStr = fmt.Sprintf("STAMINA %.0f%%  (DRAFT -%.0f%% DRAG)", p.Stamina, p.DraftFactor*65.0)
	s.hudGelsStr = fmt.Sprintf("ENERGY GELS [X]: %d   |   SCORE: %d", p.Gels, s.ctx.TourScore)
	s.hudStageTitle = s.stage.Title
	s.hudSegmentStr = segLabel

	rem := math.Max(0, s.stage.LengthMeters-p.RoadX)
	s.hudDistRemStr = fmt.Sprintf("%.0fm TO FINISH", rem)
	s.hudRankStr = fmt.Sprintf("POS %d / %d", s.playerRank, len(s.riders))
	s.hudTimeStr = fmt.Sprintf("TIME %02d:%04.1f", int(s.stageTimeSec)/60, math.Mod(s.stageTimeSec, 60.0))

	if s.demoMode {
		s.hudBannerText = "ATTRACT DEMO MODE — PRESS ANY KEY OR CLICK TO START YOUR GRAND TOUR"
		s.hudBannerColor = ui.ColTextGold
	} else if p.BonkedTimer > 0 {
		s.hudBannerText = "BONKED! STAMINA DEPLETED — TUCK INTO SLIPSTREAM OR USE GEL [X]!"
		s.hudBannerColor = ui.ColTextRed
	} else if p.CobbleTimer > 0 {
		s.hudBannerText = "PAVE COBBLE SECTOR IMPACT! STEER CLEAR OF ROUGH STONES [W/S]"
		s.hudBannerColor = ui.ColTextRed
	} else if p.Sprinting || p.GelBoostTimer > 0 {
		s.hudBannerText = "AERO SPRINT ATTACK! FULL WATTAGE SURGE!"
		s.hudBannerColor = ui.ColTextGold
	} else if p.DraftFactor >= 0.45 {
		s.hudBannerText = "SLIPSTREAM LOCKED — -65% WIND DRAG & RECOVERING STAMINA"
		s.hudBannerColor = ui.ColTextCyan
	} else if rem <= 400.0 {
		s.hudBannerText = "FLAMME ROUGE! FINAL 400m — LAUNCH YOUR SPRINT [SPACE]!"
		s.hudBannerColor = ui.ColTextGold
	} else {
		s.hudBannerText = "IN THE WIND — TUCK BEHIND A RIVAL WHEEL TO DRAFT & SAVE ENERGY"
		s.hudBannerColor = ui.ColTextMuted
	}
}

func (s *PlayState) Draw(screen *ebiten.Image) {
	screen.Fill(s.stage.SkyTint)

	camLeft := s.cam.X - VirtualWidth*0.5 + s.cam.ShakeOffsetX
	camTop := s.cam.Y - VirtualHeight*0.5 + s.cam.ShakeOffsetY

	// 1. Scrolling Parallax Scenic Backdrop
	if s.ctx.BackdropImg != nil {
		bw := float64(s.ctx.BackdropImg.Bounds().Dx())
		bh := float64(s.ctx.BackdropImg.Bounds().Dy())
		scaleX := float64(VirtualWidth) / bw
		scaleY := 430.0 / bh
		scrollPx := math.Mod(camLeft*0.18, float64(VirtualWidth))
		if scrollPx < 0 {
			scrollPx += float64(VirtualWidth)
		}
		for tile := 0; tile < 2; tile++ {
			s.bgDrawOpts.GeoM.Reset()
			s.bgDrawOpts.ColorScale.Reset()
			s.bgDrawOpts.GeoM.Scale(scaleX, scaleY)
			s.bgDrawOpts.GeoM.Translate(-scrollPx+float64(tile*VirtualWidth), -camTop*0.2)
			screen.DrawImage(s.ctx.BackdropImg, &s.bgDrawOpts)
		}
	}

	// 2. Road Verge, Crowd Barrier, and 4-Lane Asphalt Ribbon
	roadTop := float32(436.0 - camTop)
	roadBottom := float32(674.0 - camTop)
	vector.FillRect(screen, 0, roadTop-26, VirtualWidth, 26, ui.PremultipliedRGBA(42, 118, 64, 255), false)
	vector.FillRect(screen, 0, roadTop-8, VirtualWidth, 8, ui.PremultipliedRGBA(225, 52, 52, 255), false)
	vector.FillRect(screen, 0, roadTop, VirtualWidth, roadBottom-roadTop, s.stage.RoadColor, false)
	vector.FillRect(screen, 0, roadBottom, VirtualWidth, VirtualHeight-roadBottom, ui.PremultipliedRGBA(36, 92, 52, 255), false)

	// Dashed lane dividers & distance markers
	stripeCycle := 120.0
	startStripe := math.Floor(camLeft/stripeCycle) * stripeCycle
	for wx := startStripe; wx < camLeft+VirtualWidth+stripeCycle; wx += stripeCycle {
		sx := float32(wx - camLeft)
		vector.FillRect(screen, sx, roadTop-8, 56, 8, ui.ColTextWhite, false)
		for lane := 1; lane < physics.NumLanes; lane++ {
			ly := float32(physics.LaneTopY + float64(lane)*physics.LaneSpacingY - physics.LaneSpacingY*0.5 - camTop)
			vector.FillRect(screen, sx, ly, 48, 3, ui.PremultipliedRGBA(190, 198, 215, 165), false)
		}
	}

	// 3. Flamme Rouge (400m to go) & Finish Line Arches
	flammeX := float32((s.stage.LengthMeters-400.0)*entity.PixelsPerMeter - camLeft)
	if flammeX > -120 && flammeX < VirtualWidth+120 {
		vector.FillRect(screen, flammeX-6, roadTop-120, 12, roadBottom-roadTop+120, ui.ColTextRed, false)
		vector.FillRect(screen, flammeX-75, roadTop-125, 150, 28, ui.ColTextRed, false)
		s.ctx.Fonts.DrawAligned(screen, "FLAMME ROUGE 400m", s.ctx.Fonts.SmallFace, float64(flammeX), float64(roadTop-111), text.AlignCenter, text.AlignCenter, ui.ColTextWhite, ui.ColShadow)
	}

	finishX := float32(s.stage.LengthMeters*entity.PixelsPerMeter - camLeft)
	if finishX > -140 && finishX < VirtualWidth+140 {
		vector.FillRect(screen, finishX-12, roadTop, 24, roadBottom-roadTop, ui.ColTextWhite, false)
		vector.FillRect(screen, finishX-4, roadTop, 8, roadBottom-roadTop, ui.ColShadow, false)
		vector.FillRect(screen, finishX-90, roadTop-140, 180, 34, ui.ColTextGold, false)
		s.ctx.Fonts.DrawAligned(screen, "ARRIVEE / FINISH", s.ctx.Fonts.BodyFace, float64(finishX), float64(roadTop-123), text.AlignCenter, text.AlignCenter, ui.ColShadow, ui.ColTextWhite)
	}

	// 4. Roadside Course Items (Bidons, Energy Gels, Cobble Hazards)
	for i := range s.items {
		it := &s.items[i]
		if !it.Active || it.Anim == nil {
			continue
		}
		sx := it.RoadX*entity.PixelsPerMeter - camLeft
		if sx < -90 || sx > VirtualWidth+90 {
			continue
		}
		sy := it.Y - camTop
		it.Anim.DrawAnchored(screen, sx, sy+16, 0.9, 0.9, 0, nil, ebiten.FilterNearest)
	}

	// 5. Slipstream & Sprint Wind Particles
	for i := range s.particles {
		pt := &s.particles[i]
		if !pt.Active {
			continue
		}
		sx := float32(pt.WorldX - camLeft)
		sy := float32(pt.ScreenY - camTop)
		vector.FillRect(screen, sx, sy, pt.Length, 3.0, pt.Color, false)
	}

	// 6. Depth-Sorted Peloton Cyclists
	for _, r := range s.sortedRiders {
		sx := r.RoadX*entity.PixelsPerMeter - camLeft
		if sx < -180 || sx > VirtualWidth+180 {
			continue
		}
		sy := r.Y - camTop

		// Contact road shadow
		vector.FillRect(screen, float32(sx-42), float32(sy+12), 84, 10, ui.PremultipliedRGBA(10, 12, 20, 130), false)

		// Slipstream cone indicator when player is drafting
		if r.IsPlayer && r.DraftFactor >= 0.35 {
			vector.FillRect(screen, float32(sx+32), float32(sy-24), 48, 4, ui.ColTextCyan, false)
		}

		// Draw 256x256 cyclist sprite scaled to ~118px road height with ScaleSpring deformation
		baseScale := 0.46
		r.Anim.DrawAnchored(
			screen,
			sx,
			sy+24,
			baseScale*r.Spring.ScaleX,
			baseScale*r.Spring.ScaleY,
			0,
			r.Tint,
			ebiten.FilterLinear,
		)

		// Player Maillot Jaune Overhead Chevron Marker
		if r.IsPlayer {
			s.ctx.Fonts.DrawAligned(screen, "▼ YOU", s.ctx.Fonts.SmallFace, sx, sy-96, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
		}
	}

	// 7. Apply Kage Speed Vignette Overlay when Sprinting or Gel Boosting
	sprintIntensity := float32(0.0)
	if s.player.Sprinting || s.player.GelBoostTimer > 0 {
		sprintIntensity = 0.85
	} else if s.player.SpeedKmh() > 58.0 {
		sprintIntensity = float32((s.player.SpeedKmh() - 58.0) / 18.0)
	}
	if sprintIntensity > 0.05 && s.ctx.SpeedShader != nil {
		s.ctx.SpeedShader.DrawSpeedVignette(screen, VirtualWidth, VirtualHeight, sprintIntensity, float32(s.stageTimeSec), 1.0, 0.85, 0.15)
	}

	// 8. Broadcast Telemetry HUD Overlay
	s.drawHUD(screen)
}

func (s *PlayState) drawHUD(screen *ebiten.Image) {
	// Top-Left: Rider Telemetry & Stamina Gauge
	ui.DrawPanel(screen, 20, 16, 390, 122, ui.ColPanelBg, ui.ColPanelBorder)
	s.ctx.Fonts.DrawAligned(screen, s.hudSpeedStr, s.ctx.Fonts.LargeFace, 34, 40, text.AlignStart, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, s.hudCadenceStr, s.ctx.Fonts.SmallFace, 34, 70, text.AlignStart, text.AlignCenter, ui.ColTextWhite, ui.ColShadow)

	stamRatio := s.player.Stamina / s.player.MaxStamina
	stamCol := ui.ColTextGreen
	if stamRatio < 0.28 {
		stamCol = ui.ColTextRed
	} else if stamRatio < 0.55 {
		stamCol = ui.ColTextGold
	}
	ui.DrawProgressBar(screen, 34, 86, 360, 18, stamRatio, ui.ColShadow, stamCol, ui.ColTextWhite)
	s.ctx.Fonts.DrawAligned(screen, s.hudStaminaStr, s.ctx.Fonts.SmallFace, 40, 95, text.AlignStart, text.AlignCenter, ui.ColTextWhite, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, s.hudGelsStr, s.ctx.Fonts.SmallFace, 34, 120, text.AlignStart, text.AlignCenter, ui.ColTextCyan, ui.ColShadow)

	// Top-Center: Stage Elevation Profile & Live Peloton Radar Ribbon
	ui.DrawPanel(screen, 426, 16, 428, 122, ui.ColPanelBg, ui.ColPanelBorder)
	s.ctx.Fonts.DrawAligned(screen, s.hudStageTitle, s.ctx.Fonts.SmallFace, 640, 32, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, s.hudSegmentStr, s.ctx.Fonts.SmallFace, 640, 52, text.AlignCenter, text.AlignCenter, ui.ColTextWhite, ui.ColShadow)

	// Draw course elevation profile bars across 442..838
	const profileBins = 24
	binW := float32(396.0 / profileBins)
	for b := 0; b < profileBins; b++ {
		dist := (float64(b) / float64(profileBins)) * s.stage.LengthMeters
		elev := s.stage.ElevationAt(dist)
		bh := float32(elev * 44.0)
		bx := float32(442.0) + float32(b)*binW
		by := float32(124.0) - bh
		vector.FillRect(screen, bx, by, binW-2, bh, ui.PremultipliedRGBA(90, 130, 185, 180), false)
	}
	// Draw rival & player dots on profile ribbon
	for _, r := range s.riders {
		ratio := math.Max(0, math.Min(1, r.RoadX/s.stage.LengthMeters))
		rx := float32(442.0 + ratio*392.0)
		if r.IsPlayer {
			vector.FillRect(screen, rx-4, 72, 8, 54, ui.ColTextGold, false)
		} else {
			vector.FillRect(screen, rx-2, 92, 4, 34, ui.ColTextCyan, false)
		}
	}

	// Top-Right: Live GC Classification & Stage Distance
	ui.DrawPanel(screen, 870, 16, 390, 122, ui.ColPanelBg, ui.ColPanelBorder)
	s.ctx.Fonts.DrawAligned(screen, s.hudRankStr, s.ctx.Fonts.LargeFace, 1065, 42, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, s.hudDistRemStr, s.ctx.Fonts.BodyFace, 1065, 78, text.AlignCenter, text.AlignCenter, ui.ColTextWhite, ui.ColShadow)
	s.ctx.Fonts.DrawAligned(screen, s.hudTimeStr, s.ctx.Fonts.SmallFace, 1065, 112, text.AlignCenter, text.AlignCenter, ui.ColTextCyan, ui.ColShadow)

	// Bottom Tactical Telemetry Banner
	ui.DrawPanel(screen, 190, 666, 900, 40, ui.ColPanelBg, ui.ColPanelBorder)
	s.ctx.Fonts.DrawAligned(screen, s.hudBannerText, s.ctx.Fonts.BodyFace, 640, 686, text.AlignCenter, text.AlignCenter, s.hudBannerColor, ui.ColShadow)

	// Stage Finish Overlay Banner
	if s.stageFinished {
		ui.DrawPanel(screen, 260, 220, 760, 250, ui.ColPanelBg, ui.ColPanelBorder)
		s.ctx.Fonts.DrawAligned(screen, s.hudFinishTitle, s.ctx.Fonts.LargeFace, 640, 275, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
		sub := "PODIUM QUALIFIED! PREPARING NEXT GRAND TOUR STAGE..."
		if s.playerRank > 5 {
			sub = "OUTSIDE TOP 5 GC CUTOFF — ELIMINATED FROM GRAND TOUR"
		}
		s.ctx.Fonts.DrawAligned(screen, sub, s.ctx.Fonts.BodyFace, 640, 345, text.AlignCenter, text.AlignCenter, ui.ColTextWhite, ui.ColShadow)
		s.ctx.Fonts.DrawAligned(screen, "PRESS [SPACE] OR [ENTER] TO CONTINUE", s.ctx.Fonts.BodyFace, 640, 415, text.AlignCenter, text.AlignCenter, ui.ColTextCyan, ui.ColShadow)
	}

	// Pause Modal Overlay
	if s.paused {
		ui.DrawPanel(screen, 390, 230, 500, 240, ui.ColPanelBg, ui.ColPanelBorder)
		s.ctx.Fonts.DrawAligned(screen, "RACE PAUSED", s.ctx.Fonts.LargeFace, 640, 275, text.AlignCenter, text.AlignCenter, ui.ColTextGold, ui.ColShadow)
		opt0Col := ui.ColTextWhite
		opt1Col := ui.ColTextWhite
		if s.pauseMenuIdx == 0 {
			opt0Col = ui.ColTextGold
		} else {
			opt1Col = ui.ColTextGold
		}
		s.ctx.Fonts.DrawAligned(screen, "▶ RESUME STAGE", s.ctx.Fonts.BodyFace, 640, 345, text.AlignCenter, text.AlignCenter, opt0Col, ui.ColShadow)
		s.ctx.Fonts.DrawAligned(screen, "▶ ABANDON TO TITLE SCREEN", s.ctx.Fonts.BodyFace, 640, 395, text.AlignCenter, text.AlignCenter, opt1Col, ui.ColShadow)
	}
}

func (s *PlayState) Exit() error {
	return nil
}
