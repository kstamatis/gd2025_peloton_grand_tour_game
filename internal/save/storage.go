package save

import (
	"encoding/json"
	"fmt"
)

// GameSaveData stores persistent Grand Tour records across sessions.
type GameSaveData struct {
	Version        int     `json:"version"`
	HighScore      int     `json:"high_score"`
	StagesCleared  int     `json:"stages_cleared"`
	BestTourTimeMs int64   `json:"best_tour_time_ms"`
	TopSpeedKmh    float64 `json:"top_speed_kmh"`
}

// Storage defines the platform-agnostic save contract.
type Storage interface {
	Save(slot string, data GameSaveData) error
	Load(slot string) (GameSaveData, error)
}

// DefaultSaveData returns a fresh valid schema v1 GameSaveData.
func DefaultSaveData() GameSaveData {
	return GameSaveData{
		Version:        1,
		HighScore:      0,
		StagesCleared:  0,
		BestTourTimeMs: 0,
		TopSpeedKmh:    0,
	}
}

// MarshalSave validates and serializes GameSaveData to indented JSON.
func MarshalSave(data GameSaveData) ([]byte, error) {
	if data.Version <= 0 {
		return nil, fmt.Errorf("save: invalid schema version %d (must be >= 1)", data.Version)
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("save: marshal GameSaveData: %w", err)
	}
	return raw, nil
}

// UnmarshalSave deserializes and validates GameSaveData from JSON bytes.
func UnmarshalSave(raw []byte) (GameSaveData, error) {
	var data GameSaveData
	if err := json.Unmarshal(raw, &data); err != nil {
		return GameSaveData{}, fmt.Errorf("save: unmarshal GameSaveData: %w", err)
	}
	if data.Version <= 0 {
		return GameSaveData{}, fmt.Errorf("save: corrupt or missing schema version %d", data.Version)
	}
	return data, nil
}
