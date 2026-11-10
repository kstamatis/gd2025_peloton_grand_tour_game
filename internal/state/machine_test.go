package state

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type stubState struct {
	entered int
	updated int
	exited  int
}

func (s *stubState) Enter() error              { s.entered++; return nil }
func (s *stubState) Update(dt float64) error   { s.updated++; return nil }
func (s *stubState) Draw(screen *ebiten.Image) {}
func (s *stubState) Exit() error               { s.exited++; return nil }

func TestMachineLifecycleTransitions(t *testing.T) {
	m := NewMachine()
	sTitle := &stubState{}
	sPlay := &stubState{}
	m.Register(StateTitle, sTitle)
	m.Register(StatePlay, sPlay)

	if err := m.Transition(StateTitle); err != nil {
		t.Fatalf("Transition(StateTitle) failed: %v", err)
	}
	if m.ActiveID() != StateTitle || sTitle.entered != 1 {
		t.Fatalf("expected StateTitle entered once")
	}
	if err := m.Update(1.0 / 60.0); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if err := m.Transition(StatePlay); err != nil {
		t.Fatalf("Transition(StatePlay) failed: %v", err)
	}
	if sTitle.exited != 1 || sPlay.entered != 1 || m.ActiveID() != StatePlay {
		t.Fatalf("expected clean Exit->Enter transition to StatePlay")
	}
}
