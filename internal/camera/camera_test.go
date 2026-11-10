package camera

import (
	"math"
	"testing"
)

func TestCameraValidationAndFollow(t *testing.T) {
	if _, err := NewCamera(0, 720); err == nil {
		t.Fatalf("expected error for invalid viewport width")
	}
	cam, err := NewCamera(1280, 720)
	if err != nil {
		t.Fatalf("NewCamera failed: %v", err)
	}
	cam.TargetX = 500
	cam.TargetY = 360
	cam.UpdateFollow(0.1)
	if cam.X <= 0 || cam.X >= 500 {
		t.Fatalf("expected exponential follow toward 500, got %f", cam.X)
	}
}

func TestTraumaShakerAndReset(t *testing.T) {
	if _, err := NewTraumaShaker(0, 10, 0.1); err == nil {
		t.Fatalf("expected error for zero decayRate")
	}
	cam, _ := NewCamera(1280, 720)
	ts, err := NewTraumaShaker(2.0, 12.0, 0.02)
	if err != nil {
		t.Fatalf("NewTraumaShaker failed: %v", err)
	}
	ts.AddTrauma(0.6)
	ts.Update(0.016, cam)
	if ts.Trauma >= 0.6 {
		t.Fatalf("expected trauma to decay, got %f", ts.Trauma)
	}
	ts.Reset(cam)
	if ts.Trauma != 0 || cam.ShakeOffsetX != 0 || cam.ShakeOffsetY != 0 {
		t.Fatalf("expected Reset to zero trauma and camera shake")
	}
}

func TestScaleSpringVolumeConservation(t *testing.T) {
	sp, err := NewScaleSpring(250, 15)
	if err != nil {
		t.Fatalf("NewScaleSpring failed: %v", err)
	}
	sp.Impulse(1.25)
	area := sp.ScaleX * sp.ScaleY
	if math.Abs(area-1.0) > 1e-6 {
		t.Fatalf("expected area conservation ScaleX*ScaleY == 1.0, got %f", area)
	}
	for i := 0; i < 60; i++ {
		sp.Update(1.0 / 60.0)
	}
	if math.Abs(sp.ScaleX-1.0) > 0.05 || math.Abs(sp.ScaleY-1.0) > 0.05 {
		t.Fatalf("expected spring to settle near (1,1), got (%f,%f)", sp.ScaleX, sp.ScaleY)
	}
}
