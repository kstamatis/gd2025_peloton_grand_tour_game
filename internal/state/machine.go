package state

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// Machine manages scene transitions and enforces Exit() -> Enter() cleanliness.
type Machine struct {
	states  map[StateID]State
	current State
	active  StateID
}

// NewMachine constructs an empty state Machine.
func NewMachine() *Machine {
	return &Machine{
		states: make(map[StateID]State, 8),
	}
}

// Register binds a StateID to its State implementation.
func (m *Machine) Register(id StateID, s State) {
	m.states[id] = s
}

// Transition exits the current state and enters nextID.
func (m *Machine) Transition(nextID StateID) error {
	next, ok := m.states[nextID]
	if !ok || next == nil {
		return fmt.Errorf("state: unregistered state %q", nextID)
	}
	if m.current != nil {
		if err := m.current.Exit(); err != nil {
			return fmt.Errorf("state: exit %q: %w", m.active, err)
		}
	}
	m.current = next
	m.active = nextID
	if err := m.current.Enter(); err != nil {
		return fmt.Errorf("state: enter %q: %w", nextID, err)
	}
	return nil
}

// ActiveID returns the currently active StateID.
func (m *Machine) ActiveID() StateID {
	return m.active
}

// Update delegates fixed-timestep logic to the active state.
func (m *Machine) Update(dt float64) error {
	if m.current == nil {
		return fmt.Errorf("state: machine has no active state")
	}
	return m.current.Update(dt)
}

// Draw delegates rendering to the active state.
func (m *Machine) Draw(screen *ebiten.Image) {
	if m.current != nil {
		m.current.Draw(screen)
	}
}
