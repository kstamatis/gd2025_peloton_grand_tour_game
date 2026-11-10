package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Intent captures all logical player actions for a single 60 TPS update tick.
// Note: F8 and backtick (`) are deliberately left unbound for the Game Dev Studio 10s DVR scrubber.
type Intent struct {
	LaneUpJustPressed   bool
	LaneDownJustPressed bool
	LaneUpHeld          bool
	LaneDownHeld        bool
	PaceUpHeld          bool
	PaceDownHeld        bool
	SprintHeld          bool
	SprintJustPressed   bool
	UseGelJustPressed   bool
	ConfirmJustPressed  bool
	PauseJustPressed    bool
	AnyActivity         bool
}

// Manager polls keyboard, gamepad, mouse, and touch inputs with zero per-frame allocations.
type Manager struct {
	gamepadIDs []ebiten.GamepadID
	keysBuf    []ebiten.Key
	touchesBuf []ebiten.TouchID
	prevStickY float64
}

// NewManager constructs an input Manager with pre-allocated polling slices.
func NewManager() *Manager {
	return &Manager{
		gamepadIDs: make([]ebiten.GamepadID, 0, 4),
		keysBuf:    make([]ebiten.Key, 0, 16),
		touchesBuf: make([]ebiten.TouchID, 0, 8),
	}
}

// Poll samples the current frame's input state.
func (m *Manager) Poll() Intent {
	var in Intent

	in.LaneUpJustPressed = inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyW)
	in.LaneDownJustPressed = inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyS)
	in.LaneUpHeld = ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW)
	in.LaneDownHeld = ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS)

	in.PaceUpHeld = ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD)
	in.PaceDownHeld = ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA)

	in.SprintHeld = ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyZ) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	in.SprintJustPressed = inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)

	in.UseGelJustPressed = inpututil.IsKeyJustPressed(ebiten.KeyX) ||
		inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeyShiftRight) ||
		inpututil.IsKeyJustPressed(ebiten.KeyE) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)

	in.ConfirmJustPressed = inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyZ) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)

	in.PauseJustPressed = inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyP)

	// Poll first connected gamepad if present
	m.gamepadIDs = ebiten.AppendGamepadIDs(m.gamepadIDs[:0])
	if len(m.gamepadIDs) > 0 {
		id := m.gamepadIDs[0]
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			in.LaneUpJustPressed = true
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			in.LaneDownJustPressed = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			in.LaneUpHeld = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			in.LaneDownHeld = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
			in.PaceUpHeld = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
			in.PaceDownHeld = true
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightBottom) {
			in.SprintHeld = true
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom) {
			in.SprintJustPressed = true
			in.ConfirmJustPressed = true
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightRight) {
			in.UseGelJustPressed = true
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight) {
			in.PauseJustPressed = true
		}

		axX := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		axY := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
		if axX > 0.35 {
			in.PaceUpHeld = true
		} else if axX < -0.35 {
			in.PaceDownHeld = true
		}
		if axY < -0.50 && m.prevStickY >= -0.50 {
			in.LaneUpJustPressed = true
		} else if axY > 0.50 && m.prevStickY <= 0.50 {
			in.LaneDownJustPressed = true
		}
		m.prevStickY = axY
	}

	m.keysBuf = inpututil.AppendJustPressedKeys(m.keysBuf[:0])
	m.touchesBuf = inpututil.AppendJustPressedTouchIDs(m.touchesBuf[:0])
	in.AnyActivity = len(m.keysBuf) > 0 ||
		len(m.touchesBuf) > 0 ||
		in.LaneUpHeld || in.LaneDownHeld || in.PaceUpHeld || in.PaceDownHeld ||
		in.SprintHeld || in.UseGelJustPressed || in.ConfirmJustPressed || in.PauseJustPressed

	return in
}
