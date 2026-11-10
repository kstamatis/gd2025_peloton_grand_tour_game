package state

import (
	"fmt"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	ebitaudio "github.com/hajimehoshi/ebiten/v2/audio"

	"pelotongrandtour/internal/assets"
	"pelotongrandtour/internal/audio"
	"pelotongrandtour/internal/render"
	"pelotongrandtour/internal/save"
	"pelotongrandtour/internal/ui"
)

// BootState validates and preloads all embedded sprites, shaders, fonts, and PCM audio.
type BootState struct {
	ctx    *Context
	loaded bool
	timer  float64
}

// NewBootState constructs the fail-fast BootState.
func NewBootState(ctx *Context) *BootState {
	return &BootState{ctx: ctx}
}

// Enter loads and validates all embedded assets at startup.
func (s *BootState) Enter() error {
	if s.loaded {
		s.timer = 0
		return nil
	}

	bundle, err := assets.LoadAll()
	if err != nil {
		return fmt.Errorf("boot: load embedded assets: %w", err)
	}

	fonts, err := ui.NewFontBank(bundle.FontTTF)
	if err != nil {
		return fmt.Errorf("boot: init font bank: %w", err)
	}
	s.ctx.Fonts = fonts

	shaderPass, err := render.NewShaderPass(bundle.SpeedShaderSrc)
	if err != nil {
		return fmt.Errorf("boot: init speed shader: %w", err)
	}
	s.ctx.SpeedShader = shaderPass

	// Pre-slice 256x256 2D HD Cyclist Sprite Sheet (4x4 = 16 frames)
	cyclistAtlas := ebiten.NewImageFromImage(bundle.CyclistSheet)
	cyclistGrid, err := render.NewGridSpriteSheet(cyclistAtlas, 256, 256)
	if err != nil {
		return fmt.Errorf("boot: slice cyclist sheet: %w", err)
	}
	cyclistAnim := render.NewAnimationController(cyclistGrid)
	if err := cyclistAnim.AddTag("idle", 0, 3, 100*time.Millisecond, render.LoopPingPong); err != nil {
		return err
	}
	if err := cyclistAnim.AddTag("cruise", 0, 3, 85*time.Millisecond, render.LoopForward); err != nil {
		return err
	}
	if err := cyclistAnim.AddTag("sprint", 4, 7, 70*time.Millisecond, render.LoopForward); err != nil {
		return err
	}
	if err := cyclistAnim.AddTag("tuck", 8, 11, 85*time.Millisecond, render.LoopForward); err != nil {
		return err
	}
	if err := cyclistAnim.AddTag("victory", 12, 15, 110*time.Millisecond, render.LoopPingPong); err != nil {
		return err
	}
	_ = cyclistAnim.Play("cruise")
	s.ctx.CyclistProto = cyclistAnim

	// Pre-slice 64x64 Energy Gel procedural sprite sheet
	gelAtlas := ebiten.NewImageFromImage(bundle.EnergyGelSheet)
	gelGrid, err := render.NewGridSpriteSheet(gelAtlas, 64, 64)
	if err != nil {
		return fmt.Errorf("boot: slice energy_gel sheet: %w", err)
	}
	gelAnim := render.NewAnimationController(gelGrid)
	if err := gelAnim.AddTag("spin", 0, gelGrid.FrameCount()-1, 100*time.Millisecond, render.LoopForward); err != nil {
		return err
	}
	_ = gelAnim.Play("spin")
	s.ctx.GelProto = gelAnim

	// Pre-slice 64x64 Bidon bottle procedural sprite sheet
	bidonAtlas := ebiten.NewImageFromImage(bundle.BidonSheet)
	bidonGrid, err := render.NewGridSpriteSheet(bidonAtlas, 64, 64)
	if err != nil {
		return fmt.Errorf("boot: slice bidon_bottle sheet: %w", err)
	}
	bidonAnim := render.NewAnimationController(bidonGrid)
	if err := bidonAnim.AddTag("spin", 0, bidonGrid.FrameCount()-1, 100*time.Millisecond, render.LoopForward); err != nil {
		return err
	}
	_ = bidonAnim.Play("spin")
	s.ctx.BidonProto = bidonAnim

	// Pre-slice 64x64 Cobble hazard procedural sprite sheet
	cobbleAtlas := ebiten.NewImageFromImage(bundle.CobbleSheet)
	cobbleGrid, err := render.NewGridSpriteSheet(cobbleAtlas, 64, 64)
	if err != nil {
		return fmt.Errorf("boot: slice cobble_hazard sheet: %w", err)
	}
	cobbleAnim := render.NewAnimationController(cobbleGrid)
	if err := cobbleAnim.AddTag("idle", 0, cobbleGrid.FrameCount()-1, 125*time.Millisecond, render.LoopForward); err != nil {
		return err
	}
	_ = cobbleAnim.Play("idle")
	s.ctx.CobbleProto = cobbleAnim

	s.ctx.BackdropImg = ebiten.NewImageFromImage(bundle.StageBackdrop)

	if s.ctx.AudioCtx == nil {
		s.ctx.AudioCtx = ebitaudio.NewContext(audio.SampleRate)
	}
	bgmPlayer, err := audio.NewSyncedBGMPlayer(s.ctx.AudioCtx, bundle.MusicPCM)
	if err != nil {
		return fmt.Errorf("boot: init BGM player: %w", err)
	}
	s.ctx.BGM = bgmPlayer

	sfxBank, err := audio.NewSFXBank(s.ctx.AudioCtx, bundle.SFXCues)
	if err != nil {
		return fmt.Errorf("boot: init SFX bank: %w", err)
	}
	s.ctx.SFX = sfxBank

	storage, err := save.NewStorage("peloton-grand-tour")
	if err == nil {
		s.ctx.Storage = storage
		if loadedSave, loadErr := storage.Load("career"); loadErr == nil {
			s.ctx.SaveData = loadedSave
		} else {
			s.ctx.SaveData = save.DefaultSaveData()
		}
	} else {
		s.ctx.SaveData = save.DefaultSaveData()
	}

	s.loaded = true
	s.timer = 0
	log.Printf("boot: loaded 5 sprite sheets, 1 Kage shader, and %d procedural audio cues (BGM %d bytes 4-byte aligned)",
		len(bundle.SFXCues), len(bundle.MusicPCM))
	return nil
}

// Update transitions to IntroState after the studio splash display.
func (s *BootState) Update(dt float64) error {
	s.timer += dt
	if s.timer >= 0.45 {
		return s.ctx.TransitionTo(StateIntro)
	}
	return nil
}

// Draw renders the Antigravity 2D Game Studio boot splash.
func (s *BootState) Draw(screen *ebiten.Image) {
	screen.Fill(ui.PremultipliedRGBA(10, 14, 26, 255))
	if s.ctx.Fonts != nil {
		s.ctx.Fonts.DrawAligned(
			screen,
			"ANTIGRAVITY 2D GAME STUDIO",
			s.ctx.Fonts.LargeFace,
			VirtualWidth*0.5,
			VirtualHeight*0.46,
			1, 1,
			ui.ColTextGold,
			ui.ColShadow,
		)
		s.ctx.Fonts.DrawAligned(
			screen,
			"INITIALIZING PELOTON TELEMETRY & YM2612 SYNTH...",
			s.ctx.Fonts.BodyFace,
			VirtualWidth*0.5,
			VirtualHeight*0.55,
			1, 1,
			ui.ColTextCyan,
			ui.ColShadow,
		)
	}
}

// Exit cleans up transient boot state.
func (s *BootState) Exit() error {
	s.timer = 0
	return nil
}
