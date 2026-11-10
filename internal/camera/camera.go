package camera

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Camera manages the 2D viewport, exponential follow damping, and trauma shake offsets.
type Camera struct {
	X, Y             float64
	TargetX, TargetY float64
	ViewportW        float64
	ViewportH        float64
	Zoom             float64
	Rotation         float64
	FollowSharpness  float64
	ShakeOffsetX     float64
	ShakeOffsetY     float64
	ShakeAngle       float64
}

// NewCamera constructs a validated 2D Camera for the given virtual screen dimensions.
func NewCamera(viewportW, viewportH float64) (*Camera, error) {
	if viewportW <= 0 || viewportH <= 0 {
		return nil, fmt.Errorf("camera: invalid viewport dimensions (%.1f x %.1f): must be > 0", viewportW, viewportH)
	}
	return &Camera{
		ViewportW:       viewportW,
		ViewportH:       viewportH,
		Zoom:            1.0,
		FollowSharpness: 10.0,
	}, nil
}

// ExpDecay performs frame-rate independent exponential interpolation.
func ExpDecay(current, target, sharpness, dt float64) float64 {
	return target + (current-target)*math.Exp(-sharpness*dt)
}

// UpdateFollow advances camera position toward TargetX/TargetY using exponential decay.
func (c *Camera) UpdateFollow(dt float64) {
	c.X = ExpDecay(c.X, c.TargetX, c.FollowSharpness, dt)
	c.Y = ExpDecay(c.Y, c.TargetY, c.FollowSharpness, dt)
}

// Reset clears camera shake and snaps to TargetX, TargetY.
func (c *Camera) Reset() {
	c.X = c.TargetX
	c.Y = c.TargetY
	c.ShakeOffsetX = 0
	c.ShakeOffsetY = 0
	c.ShakeAngle = 0
}

// ViewMatrix returns the World -> Screen transformation matrix.
func (c *Camera) ViewMatrix() ebiten.GeoM {
	var m ebiten.GeoM
	m.Translate(-c.X, -c.Y)
	m.Scale(c.Zoom, c.Zoom)
	m.Rotate(c.Rotation + c.ShakeAngle)
	m.Translate(c.ViewportW*0.5+c.ShakeOffsetX, c.ViewportH*0.5+c.ShakeOffsetY)
	return m
}

// TraumaShaker implements Squirrel Eiserloh's quadratic trauma screen shake.
type TraumaShaker struct {
	Trauma      float64
	DecayRate   float64
	MaxOffsetPx float64
	MaxAngleRad float64
	Time        float64
}

// NewTraumaShaker constructs a TraumaShaker with validated decayRate.
func NewTraumaShaker(decayRate, maxOffsetPx, maxAngleRad float64) (*TraumaShaker, error) {
	if decayRate <= 0 {
		return nil, fmt.Errorf("camera: trauma decayRate must be > 0, got %f", decayRate)
	}
	return &TraumaShaker{
		DecayRate:   decayRate,
		MaxOffsetPx: maxOffsetPx,
		MaxAngleRad: maxAngleRad,
	}, nil
}

// AddTrauma injects impact energy clamped to [0.0, 1.0].
func (ts *TraumaShaker) AddTrauma(amount float64) {
	if amount <= 0 {
		return
	}
	ts.Trauma = math.Min(1.0, ts.Trauma+amount)
}

// Reset flushes all active trauma and camera shake offsets.
func (ts *TraumaShaker) Reset(cam *Camera) {
	ts.Trauma = 0
	if cam != nil {
		cam.ShakeOffsetX = 0
		cam.ShakeOffsetY = 0
		cam.ShakeAngle = 0
	}
}

// Update decays trauma and applies deterministic multi-frequency shake to cam.
func (ts *TraumaShaker) Update(dt float64, cam *Camera) {
	if ts.Trauma <= 0 {
		cam.ShakeOffsetX = 0
		cam.ShakeOffsetY = 0
		cam.ShakeAngle = 0
		return
	}
	ts.Time += dt
	shake := ts.Trauma * ts.Trauma

	nx := math.Sin(ts.Time*37.0)*0.6 + math.Cos(ts.Time*73.0)*0.4
	ny := math.Cos(ts.Time*41.0)*0.6 + math.Sin(ts.Time*89.0)*0.4
	nr := math.Sin(ts.Time*53.0)*0.7 + math.Cos(ts.Time*97.0)*0.3

	cam.ShakeOffsetX = ts.MaxOffsetPx * shake * nx
	cam.ShakeOffsetY = ts.MaxOffsetPx * shake * ny
	cam.ShakeAngle = ts.MaxAngleRad * shake * nr

	ts.Trauma = math.Max(0.0, ts.Trauma-ts.DecayRate*dt)
}

// TimeController manages hitstop freeze frames and time dilation.
type TimeController struct {
	HitstopRemaining float64
	TimeScale        float64
}

// NewTimeController constructs a TimeController at 1.0x normal speed.
func NewTimeController() *TimeController {
	return &TimeController{TimeScale: 1.0}
}

// TriggerHitstop pauses world simulation for durationSeconds.
func (tc *TimeController) TriggerHitstop(durationSeconds float64) {
	if durationSeconds > tc.HitstopRemaining {
		tc.HitstopRemaining = durationSeconds
	}
}

// Step computes the effective world delta time.
func (tc *TimeController) Step(rawDt float64) (worldDt float64, frozen bool) {
	if tc.HitstopRemaining > 0 {
		tc.HitstopRemaining -= rawDt
		if tc.HitstopRemaining < 0 {
			tc.HitstopRemaining = 0
		}
		return 0, true
	}
	return rawDt * tc.TimeScale, false
}

// ScaleSpring implements volume-preserving harmonic squash-and-stretch deformation.
type ScaleSpring struct {
	ScaleX, ScaleY float64
	VelX, VelY     float64
	Stiffness      float64
	Damping        float64
}

// NewScaleSpring constructs a ScaleSpring with stiffness and damping > 0.
func NewScaleSpring(stiffness, damping float64) (*ScaleSpring, error) {
	if stiffness <= 0 || damping <= 0 {
		return nil, fmt.Errorf("camera: spring stiffness (%f) and damping (%f) must be > 0", stiffness, damping)
	}
	return &ScaleSpring{
		ScaleX:    1.0,
		ScaleY:    1.0,
		Stiffness: stiffness,
		Damping:   damping,
	}, nil
}

// Impulse deforms the sprite while conserving 2D area (ScaleX * ScaleY == 1).
func (s *ScaleSpring) Impulse(scaleX float64) {
	if scaleX <= 0.2 {
		scaleX = 0.2
	}
	s.ScaleX = scaleX
	s.ScaleY = 1.0 / scaleX
}

// Update steps Hooke's law toward rest scale (1.0, 1.0).
func (s *ScaleSpring) Update(dt float64) {
	forceX := (1.0 - s.ScaleX) * s.Stiffness
	forceY := (1.0 - s.ScaleY) * s.Stiffness

	s.VelX = (s.VelX + forceX*dt) * math.Exp(-s.Damping*dt)
	s.VelY = (s.VelY + forceY*dt) * math.Exp(-s.Damping*dt)

	s.ScaleX += s.VelX * dt
	s.ScaleY += s.VelY * dt
}
