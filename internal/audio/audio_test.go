package audio

import (
	"testing"
	"time"
)

func TestAlignPCMByteOffset(t *testing.T) {
	const loopLen = 44100 * 4 * 10 // 10 seconds of stereo 16-bit PCM
	durations := []time.Duration{
		0,
		1 * time.Millisecond,
		17 * time.Millisecond,
		333 * time.Millisecond,
		12500 * time.Millisecond,
	}
	for _, d := range durations {
		got, err := AlignPCMByteOffset(d, SampleRate, loopLen)
		if err != nil {
			t.Fatalf("AlignPCMByteOffset(%v) unexpected error: %v", d, err)
		}
		if got < 0 || got >= loopLen || got%4 != 0 {
			t.Fatalf("AlignPCMByteOffset(%v) = %d, want [0..%d) divisible by 4", d, got, loopLen)
		}
	}
	if _, err := AlignPCMByteOffset(time.Second, 0, loopLen); err == nil {
		t.Fatalf("expected error for invalid sampleRate 0")
	}
	if _, err := AlignPCMByteOffset(time.Second, SampleRate, 1003); err == nil {
		t.Fatalf("expected error for unaligned loopLengthBytes")
	}
}

func TestAlignPCM(t *testing.T) {
	raw := []byte{1, 2, 3, 4, 5, 6, 7}
	aligned := AlignPCM(raw)
	if len(aligned) != 4 {
		t.Fatalf("AlignPCM(7 bytes) length = %d, want 4", len(aligned))
	}
}
