package state

import (
	"github.com/hajimehoshi/ebiten/v2"
	ebitaudio "github.com/hajimehoshi/ebiten/v2/audio"

	"pelotongrandtour/internal/audio"
	"pelotongrandtour/internal/input"
	"pelotongrandtour/internal/render"
	"pelotongrandtour/internal/save"
	"pelotongrandtour/internal/ui"
)

const (
	VirtualWidth  = 1280
	VirtualHeight = 720
)

// StateID identifies a distinct scene in the game's finite state machine.
type StateID string

const (
	StateBoot     StateID = "boot"
	StateIntro    StateID = "intro"
	StateTitle    StateID = "title"
	StatePlay     StateID = "play"
	StateDemo     StateID = "demo"
	StateWin      StateID = "win"
	StateGameOver StateID = "gameover"
)

// State defines the strict lifecycle hooks required for every scene.
type State interface {
	Enter() error
	Update(dt float64) error
	Draw(screen *ebiten.Image)
	Exit() error
}

// Context holds shared runtime subsystems initialized during BootState.
type Context struct {
	AudioCtx          *ebitaudio.Context
	BGM               *audio.SyncedBGMPlayer
	SFX               *audio.SFXBank
	Fonts             *ui.FontBank
	SpeedShader       *render.ShaderPass
	CyclistProto      *render.AnimationController
	GelProto          *render.AnimationController
	BidonProto        *render.AnimationController
	CobbleProto       *render.AnimationController
	BackdropImg       *ebiten.Image
	Input             *input.Manager
	Storage           save.Storage
	SaveData          save.GameSaveData
	CurrentStageIndex int
	TourTimeSec       float64
	TourScore         int
	LastStageRank     int
	FailureReason     string
	AttractToggle     bool
	TransitionTo      func(id StateID) error
}
