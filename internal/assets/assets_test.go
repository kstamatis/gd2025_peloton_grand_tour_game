package assets

import "testing"

func TestLoadAllEmbeddedAssets(t *testing.T) {
	b, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() failed: %v", err)
	}
	if b.CyclistSheet == nil || b.EnergyGelSheet == nil || b.BidonSheet == nil || b.CobbleSheet == nil || b.StageBackdrop == nil {
		t.Fatalf("expected all 5 embedded sprite/backdrop images to be non-nil")
	}
	if len(b.MusicPCM) == 0 || len(b.MusicPCM)%4 != 0 {
		t.Fatalf("expected non-empty 4-byte aligned MusicPCM, got %d bytes", len(b.MusicPCM))
	}
	if len(b.SFXCues) != 9 {
		t.Fatalf("expected 9 embedded SFX cues, got %d", len(b.SFXCues))
	}
	for name, pcm := range b.SFXCues {
		if len(pcm) == 0 || len(pcm)%4 != 0 {
			t.Fatalf("SFX cue %q is not 4-byte aligned (len=%d)", name, len(pcm))
		}
	}
}
