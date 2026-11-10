package entity

import (
	"image/color"
	"math"

	"pelotongrandtour/internal/camera"
	"pelotongrandtour/internal/physics"
	"pelotongrandtour/internal/render"
	"pelotongrandtour/internal/ui"
)

// PixelsPerMeter converts road distance in meters to virtual screen pixels.
const PixelsPerMeter = 42.0

// RiderRole distinguishes AI peloton archetypes.
type RiderRole int

const (
	RolePlayer RiderRole = iota
	RoleDomestique
	RoleRouleur
	RoleGrimpeur
	RoleSprinter
)

// ItemKind identifies roadside collectibles and hazards.
type ItemKind int

const (
	ItemEnergyGel ItemKind = iota
	ItemBidon
	ItemCobbleHazard
)

// CourseItem represents a pre-allocated pickup or road hazard on the stage.
type CourseItem struct {
	Active bool
	Kind   ItemKind
	RoadX  float64 // Distance in meters from stage start
	Lane   int
	Y      float64
	Anim   *render.AnimationController
}

// WindParticle represents a pre-allocated slipstream ribbon or sprint spark particle.
type WindParticle struct {
	Active        bool
	WorldX        float64
	ScreenY       float64
	VelX, VelY    float64
	Length        float32
	Life, MaxLife float64
	Color         color.RGBA
}

// StageSegment defines a gradient section of a Grand Tour stage.
type StageSegment struct {
	StartMeters float64
	EndMeters   float64
	GradientPct float64 // e.g., 0.0 for flat, 8.5 for +8.5% climb, -4.0 for descent
	Label       string
}

// StageSpec defines one of the 3 Grand Tour stages.
type StageSpec struct {
	Number       int
	Title        string
	Subtitle     string
	LengthMeters float64
	SkyTint      color.RGBA
	RoadColor    color.RGBA
	Segments     []StageSegment
}

// GradientAt returns the road gradient percentage and segment name at distance distMeters.
func (s StageSpec) GradientAt(distMeters float64) (float64, string) {
	for _, seg := range s.Segments {
		if distMeters >= seg.StartMeters && distMeters <= seg.EndMeters {
			return seg.GradientPct, seg.Label
		}
	}
	return 0.0, "Flat Sector"
}

// ElevationAt computes normalized elevation [0.0..1.0] for the HUD course profile ribbon.
func (s StageSpec) ElevationAt(distMeters float64) float64 {
	elev := 0.15
	cursor := 0.0
	for _, seg := range s.Segments {
		if distMeters <= seg.StartMeters {
			break
		}
		end := math.Min(distMeters, seg.EndMeters)
		span := end - cursor
		if span > 0 {
			elev += (span / s.LengthMeters) * (seg.GradientPct * 0.085)
		}
		cursor = end
	}
	return math.Max(0.08, math.Min(0.92, elev))
}

// AllStages returns the 3 Grand Tour stages.
func AllStages() []StageSpec {
	return []StageSpec{
		{
			Number:       1,
			Title:        "Stage 1: Cote d'Azur Coastal Sprint",
			Subtitle:     "Nice Promenade -> Cannes Boulevard (1,600m)",
			LengthMeters: 1600,
			SkyTint:      ui.PremultipliedRGBA(255, 248, 232, 255),
			RoadColor:    ui.PremultipliedRGBA(54, 58, 70, 255),
			Segments: []StageSegment{
				{0, 450, 0.5, "Promenade des Anglais (Flat)"},
				{450, 900, 3.8, "Cap d'Antibes Rollers (+3.8%)"},
				{900, 1200, -2.5, "Coastal Descent (-2.5%)"},
				{1200, 1600, 0.0, "Flamme Rouge Boulevard Sprint"},
			},
		},
		{
			Number:       2,
			Title:        "Stage 2: Col du Galibier Alpine Climb",
			Subtitle:     "Valloire -> Galibier Summit (1,900m)",
			LengthMeters: 1900,
			SkyTint:      ui.PremultipliedRGBA(225, 240, 255, 255),
			RoadColor:    ui.PremultipliedRGBA(48, 52, 64, 255),
			Segments: []StageSegment{
				{0, 400, 2.5, "Alpine Valley Approach (+2.5%)"},
				{400, 1050, 7.8, "Col du Telegraphe (+7.8% Climb)"},
				{1050, 1550, 10.2, "Galibier HC Hairpins (+10.2% HC)"},
				{1550, 1900, 5.5, "Summit Finish Ramp (+5.5%)"},
			},
		},
		{
			Number:       3,
			Title:        "Stage 3: Champs-Elysees Cobble Finale",
			Subtitle:     "Paris Pave Sectors -> Arc de Triomphe (2,200m)",
			LengthMeters: 2200,
			SkyTint:      ui.PremultipliedRGBA(255, 232, 210, 255),
			RoadColor:    ui.PremultipliedRGBA(60, 58, 66, 255),
			Segments: []StageSegment{
				{0, 550, 0.5, "Rue de Rivoli Peloton Launch"},
				{550, 1150, 3.2, "Montmartre Pave Climb (+3.2%)"},
				{1150, 1650, -1.5, "Place de la Concorde Tunnel"},
				{1650, 2200, 1.0, "Champs-Elysees Royal Sprint"},
			},
		},
	}
}

// Rider represents the player or an AI rival cyclist in the peloton.
type Rider struct {
	Name          string
	Team          string
	Role          RiderRole
	IsPlayer      bool
	RoadX         float64 // Distance in meters from stage start
	Lane          int     // Target discrete road lane [0..3]
	Y             float64 // Smoothly interpolated virtual screen Y
	SpeedMps      float64 // Current speed in meters/second (15 m/s = 54 km/h)
	BasePower     float64 // Rider baseline cruising speed in m/s
	Stamina       float64 // [0.0 .. 100.0]
	MaxStamina    float64
	DraftFactor   float64 // [0.0 .. 1.0] slipstream reduction from rider ahead
	Sprinting     bool
	Tucking       bool
	BonkedTimer   float64
	CobbleTimer   float64
	GelBoostTimer float64
	Gels          int
	CadenceRPM    int
	GearLabel     string
	AITimer       float64
	Finished      bool
	FinishTimeSec float64
	Tint          color.RGBA
	Anim          *render.AnimationController
	Spring        *camera.ScaleSpring
}

// SpeedKmh returns the rider's speed in kilometers per hour.
func (r *Rider) SpeedKmh() float64 {
	return r.SpeedMps * 3.6
}

// UpdateDesiredAnimation selects the matching sprite animation tag for the rider's current state.
func (r *Rider) UpdateDesiredAnimation(stageFinished bool, wonStage bool) {
	if r.Anim == nil {
		return
	}
	if stageFinished && wonStage {
		_ = r.Anim.Play("victory")
		return
	}
	if r.Sprinting || r.GelBoostTimer > 0 {
		_ = r.Anim.Play("sprint")
		return
	}
	if r.Tucking || r.DraftFactor >= 0.48 {
		_ = r.Anim.Play("tuck")
		return
	}
	if r.SpeedMps < 5.0 {
		_ = r.Anim.Play("idle")
		return
	}
	_ = r.Anim.Play("cruise")
}

// BuildPeloton initializes the player and 11 rival pro riders for a stage.
func BuildPeloton(baseAnim *render.AnimationController, stageNum int) ([]*Rider, *Rider, error) {
	rivalsSpec := []struct {
		name  string
		team  string
		role  RiderRole
		power float64
		tint  color.RGBA
	}{
		{"T. Pogacar", "UAE Emirates", RoleGrimpeur, 15.85, ui.PremultipliedRGBA(255, 225, 235, 255)},
		{"J. Vingegaard", "Visma Lease", RoleGrimpeur, 15.75, ui.PremultipliedRGBA(210, 245, 255, 255)},
		{"M. van der Poel", "Alpecin Deceuninck", RoleSprinter, 15.80, ui.PremultipliedRGBA(185, 220, 255, 255)},
		{"W. van Aert", "Visma Lease", RoleRouleur, 15.65, ui.PremultipliedRGBA(190, 255, 210, 255)},
		{"R. Evenepoel", "Soudal Quick-Step", RoleRouleur, 15.70, ui.PremultipliedRGBA(200, 230, 255, 255)},
		{"J. Philipsen", "Alpecin Sprint", RoleSprinter, 15.60, ui.PremultipliedRGBA(255, 195, 195, 255)},
		{"P. Roglic", "RedBull Bora", RoleGrimpeur, 15.55, ui.PremultipliedRGBA(195, 255, 235, 255)},
		{"B. Girmay", "Intermarche", RoleSprinter, 15.50, ui.PremultipliedRGBA(215, 255, 190, 255)},
		{"M. Pedersen", "Lidl Trek", RoleRouleur, 15.40, ui.PremultipliedRGBA(255, 210, 245, 255)},
		{"C. Laporte", "Team Domestique", RoleDomestique, 15.25, ui.PremultipliedRGBA(225, 225, 255, 255)},
		{"K. Asgreen", "Leadout Train", RoleDomestique, 15.15, ui.PremultipliedRGBA(240, 220, 200, 255)},
	}

	stageBonus := float64(stageNum-1) * 0.28
	riders := make([]*Rider, 0, len(rivalsSpec)+1)

	playerSpring, err := camera.NewScaleSpring(280.0, 16.0)
	if err != nil {
		return nil, nil, err
	}
	player := &Rider{
		Name:       "YOU (Maillot Jaune)",
		Team:       "Antigravity Pro Cycling",
		Role:       RolePlayer,
		IsPlayer:   true,
		RoadX:      12.0,
		Lane:       1,
		Y:          physics.LaneCenterY(1),
		SpeedMps:   14.2,
		BasePower:  16.15 + stageBonus,
		Stamina:    100.0,
		MaxStamina: 100.0,
		Gels:       2,
		CadenceRPM: 92,
		GearLabel:  "53x14",
		Tint:       ui.PremultipliedRGBA(255, 255, 255, 255),
		Anim:       baseAnim.Clone(),
		Spring:     playerSpring,
	}
	riders = append(riders, player)

	for i, spec := range rivalsSpec {
		sp, err := camera.NewScaleSpring(260.0, 16.0)
		if err != nil {
			return nil, nil, err
		}
		lane := i % physics.NumLanes
		startX := 16.0 + float64(i)*4.2
		r := &Rider{
			Name:       spec.name,
			Team:       spec.team,
			Role:       spec.role,
			IsPlayer:   false,
			RoadX:      startX,
			Lane:       lane,
			Y:          physics.LaneCenterY(lane),
			SpeedMps:   14.4,
			BasePower:  spec.power + stageBonus,
			Stamina:    100.0,
			MaxStamina: 100.0,
			Gels:       1,
			CadenceRPM: 90,
			GearLabel:  "53x15",
			AITimer:    float64(i) * 0.37,
			Tint:       spec.tint,
			Anim:       baseAnim.Clone(),
			Spring:     sp,
		}
		riders = append(riders, r)
	}

	return riders, player, nil
}
