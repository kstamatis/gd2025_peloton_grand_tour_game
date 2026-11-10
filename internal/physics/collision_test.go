package physics

import "testing"

func TestLaneCenterYAndNearestLane(t *testing.T) {
	for lane := 0; lane < NumLanes; lane++ {
		y := LaneCenterY(lane)
		got := NearestLane(y)
		if got != lane {
			t.Fatalf("NearestLane(LaneCenterY(%d)=%.1f) = %d, want %d", lane, y, got, lane)
		}
	}
	if NearestLane(-100) != 0 {
		t.Fatalf("expected top clamp to lane 0")
	}
	if NearestLane(9999) != NumLanes-1 {
		t.Fatalf("expected bottom clamp to lane %d", NumLanes-1)
	}
}

func TestComputeSlipstreamFactor(t *testing.T) {
	laneY := LaneCenterY(1)

	// Optimal slipstream sweet spot (3.6m behind, identical lane Y)
	opt := ComputeSlipstreamFactor(10.0, laneY, 13.6, laneY)
	if opt < 0.99 {
		t.Fatalf("expected optimal slipstream ~1.0 at 3.6m behind in same lane, got %f", opt)
	}

	// Ahead of leader -> zero draft
	ahead := ComputeSlipstreamFactor(15.0, laneY, 13.6, laneY)
	if ahead != 0 {
		t.Fatalf("expected 0 slipstream when ahead of leader, got %f", ahead)
	}

	// Too far behind (>7.8m) -> zero draft
	far := ComputeSlipstreamFactor(2.0, laneY, 13.6, laneY)
	if far != 0 {
		t.Fatalf("expected 0 slipstream when >7.8m behind leader, got %f", far)
	}

	// Adjacent lane (>34px lateral spread) -> zero draft
	adjLane := ComputeSlipstreamFactor(10.0, LaneCenterY(2), 13.6, laneY)
	if adjLane != 0 {
		t.Fatalf("expected 0 slipstream across full lane separation, got %f", adjLane)
	}
}

func TestCheckPickupOverlap(t *testing.T) {
	if !CheckPickupOverlap(100.0, 500.0, 101.2, 515.0, 1.85, 28.0) {
		t.Fatalf("expected overlap within radii")
	}
	if CheckPickupOverlap(100.0, 500.0, 105.0, 515.0, 1.85, 28.0) {
		t.Fatalf("expected no overlap when X delta exceeds radiusX")
	}
}
