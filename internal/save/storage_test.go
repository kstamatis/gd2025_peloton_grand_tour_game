package save

import "testing"

func TestMarshalUnmarshalSave(t *testing.T) {
	orig := GameSaveData{
		Version:        1,
		HighScore:      4850,
		StagesCleared:  3,
		BestTourTimeMs: 214500,
		TopSpeedKmh:    68.4,
	}
	raw, err := MarshalSave(orig)
	if err != nil {
		t.Fatalf("MarshalSave failed: %v", err)
	}
	got, err := UnmarshalSave(raw)
	if err != nil {
		t.Fatalf("UnmarshalSave failed: %v", err)
	}
	if got.HighScore != orig.HighScore || got.StagesCleared != orig.StagesCleared || got.Version != orig.Version {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, orig)
	}
}
