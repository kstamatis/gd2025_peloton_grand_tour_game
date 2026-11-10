package physics

import "math"

const (
	NumLanes     = 4
	LaneTopY     = 478.0
	LaneSpacingY = 52.0
)

// LaneCenterY returns the virtual screen Y coordinate for a given road lane [0..3].
func LaneCenterY(lane int) float64 {
	if lane < 0 {
		lane = 0
	}
	if lane >= NumLanes {
		lane = NumLanes - 1
	}
	return LaneTopY + float64(lane)*LaneSpacingY
}

// NearestLane converts a smooth Y coordinate to the closest discrete road lane [0..3].
func NearestLane(y float64) int {
	idx := int(math.Round((y - LaneTopY) / LaneSpacingY))
	if idx < 0 {
		return 0
	}
	if idx >= NumLanes {
		return NumLanes - 1
	}
	return idx
}

// ComputeSlipstreamFactor calculates how strongly a rider at (riderX, riderY) drafts
// behind a leader at (leaderX, leaderY).
// Returns a value in [0.0, 1.0], where 1.0 is optimal wheel-sucking slipstream (1.4m..5.5m behind, same lane).
func ComputeSlipstreamFactor(riderX, riderY, leaderX, leaderY float64) float64 {
	gapX := leaderX - riderX
	// Must be behind the leader's rear wheel (1.1m to 7.8m gap)
	if gapX < 1.1 || gapX > 7.8 {
		return 0
	}
	dy := math.Abs(leaderY - riderY)
	const maxLatSpread = 34.0
	if dy >= maxLatSpread {
		return 0
	}
	latFactor := 1.0 - (dy / maxLatSpread)

	var distFactor float64
	if gapX <= 3.6 {
		// Sweet spot 1.1m..3.6m behind
		distFactor = 0.75 + 0.25*((gapX-1.1)/2.5)
	} else {
		// Linearly tapers off from 3.6m..7.8m
		distFactor = 1.0 - ((gapX - 3.6) / 4.2)
	}
	if distFactor < 0 {
		distFactor = 0
	}
	if distFactor > 1 {
		distFactor = 1
	}
	return distFactor * latFactor
}

// CheckPickupOverlap checks whether a cyclist at (riderX, riderY) overlaps a roadside item at (itemX, itemY).
func CheckPickupOverlap(riderX, riderY, itemX, itemY, radiusX, radiusY float64) bool {
	return math.Abs(riderX-itemX) <= radiusX && math.Abs(riderY-itemY) <= radiusY
}
