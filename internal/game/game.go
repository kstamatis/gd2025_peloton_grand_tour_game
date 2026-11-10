package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"pelotongrandtour/internal/input"
	"pelotongrandtour/internal/state"
)

const (
	VirtualWidth  = state.VirtualWidth
	VirtualHeight = state.VirtualHeight
	FixedDt       = 1.0 / 60.0
)

// Game implements ebiten.Game and delegates scene lifecycle to state.Machine.
type Game struct {
	ctx     *state.Context
	machine *state.Machine
}

// New constructs the top-level Game instance and boots into StateBoot.
func New() (*Game, error) {
	ctx := &state.Context{
		Input: input.NewManager(),
	}
	machine := state.NewMachine()
	ctx.TransitionTo = machine.Transition

	machine.Register(state.StateBoot, state.NewBootState(ctx))
	machine.Register(state.StateIntro, state.NewIntroState(ctx))
	machine.Register(state.StateTitle, state.NewTitleState(ctx))
	machine.Register(state.StatePlay, state.NewPlayState(ctx, false))
	machine.Register(state.StateDemo, state.NewPlayState(ctx, true))
	machine.Register(state.StateWin, state.NewWinState(ctx))
	machine.Register(state.StateGameOver, state.NewGameOverState(ctx))

	if err := machine.Transition(state.StateBoot); err != nil {
		return nil, err
	}

	return &Game{
		ctx:     ctx,
		machine: machine,
	}, nil
}

// Update advances game logic at a fixed 60 TPS cadence.
func (g *Game) Update() error {
	// Toggle fullscreen with F11 or Alt+Enter
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) ||
		(ebiten.IsKeyPressed(ebiten.KeyAlt) && inpututil.IsKeyJustPressed(ebiten.KeyEnter)) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	return g.machine.Update(FixedDt)
}

// Draw renders the active state onto the 1280x720 virtual screen.
func (g *Game) Draw(screen *ebiten.Image) {
	g.machine.Draw(screen)
}

// Layout locks the logical resolution to 1280x720 regardless of physical window size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return VirtualWidth, VirtualHeight
}
