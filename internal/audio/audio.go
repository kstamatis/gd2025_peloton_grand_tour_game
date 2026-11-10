package audio

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	ebitaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	SampleRate     = 44100
	BytesPerSample = 4 // 16-bit signed Little-Endian stereo PCM (2 bytes L + 2 bytes R)
)

// AlignPCM masks a stereo PCM slice to a strict 4-byte sample frame boundary (&^ 3).
func AlignPCM(pcm []byte) []byte {
	return pcm[:len(pcm)&^3]
}

// AlignPCMByteOffset converts an elapsed duration into a 4-byte aligned stereo PCM byte offset
// wrapped cleanly within loopLengthBytes.
func AlignPCMByteOffset(elapsed time.Duration, sampleRate int, loopLengthBytes int64) (int64, error) {
	if sampleRate <= 0 {
		return 0, fmt.Errorf("audio: invalid sampleRate %d", sampleRate)
	}
	if loopLengthBytes <= 0 || loopLengthBytes%BytesPerSample != 0 {
		return 0, fmt.Errorf("audio: invalid loopLengthBytes %d (must be multiple of 4)", loopLengthBytes)
	}

	totalSamples := int64(elapsed.Seconds() * float64(sampleRate))
	rawByteOffset := totalSamples * BytesPerSample
	wrappedOffset := rawByteOffset % loopLengthBytes
	if wrappedOffset < 0 {
		wrappedOffset = 0
	}
	alignedOffset := wrappedOffset &^ 3
	return alignedOffset, nil
}

// SyncedBGMPlayer manages 4-byte aligned infinite-loop BGM playback and WebAssembly autoplay unlock seeking.
type SyncedBGMPlayer struct {
	ctx             *ebitaudio.Context
	loopStream      *ebitaudio.InfiniteLoop
	player          *ebitaudio.Player
	loopLengthBytes int64
	elapsed         time.Duration
	unlocked        bool
	playing         bool
	pressedKeysBuf  []ebiten.Key
	touchIDsBuf     []ebiten.TouchID
}

// NewSyncedBGMPlayer creates a looped BGM player from 4-byte aligned stereo PCM bytes.
func NewSyncedBGMPlayer(ctx *ebitaudio.Context, pcmBytes []byte) (*SyncedBGMPlayer, error) {
	if ctx == nil {
		return nil, fmt.Errorf("audio: nil audio.Context")
	}
	alignedLen := int64(len(pcmBytes) &^ 3)
	if alignedLen <= 0 {
		return nil, fmt.Errorf("audio: pcmBytes length (%d) must be >= 4 bytes", len(pcmBytes))
	}

	stream := bytes.NewReader(pcmBytes[:alignedLen])
	loop := ebitaudio.NewInfiniteLoop(stream, alignedLen)
	p, err := ctx.NewPlayer(loop)
	if err != nil {
		return nil, fmt.Errorf("audio: create BGM player: %w", err)
	}
	p.SetVolume(0.65)

	return &SyncedBGMPlayer{
		ctx:             ctx,
		loopStream:      loop,
		player:          p,
		loopLengthBytes: alignedLen,
		pressedKeysBuf:  make([]ebiten.Key, 0, 16),
		touchIDsBuf:     make([]ebiten.TouchID, 0, 8),
	}, nil
}

// Start begins or resumes BGM playback (seeking on first user gesture in WebAssembly).
func (bgm *SyncedBGMPlayer) Start() {
	if bgm == nil {
		return
	}
	bgm.playing = true
	if bgm.unlocked && !bgm.player.IsPlaying() {
		bgm.player.Play()
	}
}

// Pause pauses BGM playback cleanly on scene transitions or pause overlays.
func (bgm *SyncedBGMPlayer) Pause() {
	if bgm == nil {
		return
	}
	bgm.playing = false
	if bgm.player != nil && bgm.player.IsPlaying() {
		bgm.player.Pause()
	}
}

// SetVolume adjusts the BGM master volume in [0.0, 1.0].
func (bgm *SyncedBGMPlayer) SetVolume(vol float64) {
	if bgm != nil && bgm.player != nil {
		bgm.player.SetVolume(vol)
	}
}

// Update advances the internal clock and unlocks/seeks audio on first user gesture.
func (bgm *SyncedBGMPlayer) Update(dt time.Duration) error {
	if bgm == nil || !bgm.playing {
		return nil
	}
	bgm.elapsed += dt

	if bgm.unlocked {
		if !bgm.player.IsPlaying() {
			bgm.player.Play()
		}
		return nil
	}

	if bgm.hasUserGesture() {
		offsetBytes, err := AlignPCMByteOffset(bgm.elapsed, bgm.ctx.SampleRate(), bgm.loopLengthBytes)
		if err != nil {
			return fmt.Errorf("audio: compute late-sync offset: %w", err)
		}
		if _, err := bgm.loopStream.Seek(offsetBytes, io.SeekStart); err != nil {
			return fmt.Errorf("audio: seek BGM stream to %d bytes: %w", offsetBytes, err)
		}
		bgm.player.Play()
		bgm.unlocked = true
	}
	return nil
}

func (bgm *SyncedBGMPlayer) hasUserGesture() bool {
	bgm.pressedKeysBuf = inpututil.AppendJustPressedKeys(bgm.pressedKeysBuf[:0])
	if len(bgm.pressedKeysBuf) > 0 {
		return true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		return true
	}
	bgm.touchIDsBuf = inpututil.AppendJustPressedTouchIDs(bgm.touchIDsBuf[:0])
	return len(bgm.touchIDsBuf) > 0
}

// SFXBank manages pre-compiled 4-byte aligned procedural SFX cues with per-cue cooldowns.
type SFXBank struct {
	ctx        *ebitaudio.Context
	cues       map[string][]byte
	lastPlayed map[string]time.Time
}

// NewSFXBank constructs a new SFXBank sharing the unified 44.1 kHz audio.Context.
func NewSFXBank(ctx *ebitaudio.Context, cues map[string][]byte) (*SFXBank, error) {
	if ctx == nil {
		return nil, fmt.Errorf("audio: nil audio.Context for SFXBank")
	}
	bank := &SFXBank{
		ctx:        ctx,
		cues:       make(map[string][]byte, len(cues)),
		lastPlayed: make(map[string]time.Time, len(cues)),
	}
	for name, pcm := range cues {
		if err := bank.RegisterCue(name, pcm); err != nil {
			return nil, err
		}
	}
	return bank, nil
}

// RegisterCue registers a 4-byte aligned PCM cue in the bank.
func (b *SFXBank) RegisterCue(name string, pcm []byte) error {
	aligned := AlignPCM(pcm)
	if name == "" || len(aligned) == 0 {
		return fmt.Errorf("audio: invalid SFX cue %q (aligned length=%d)", name, len(aligned))
	}
	b.cues[name] = aligned
	return nil
}

// PlaySFX plays a named SFX cue if its cooldown window has elapsed.
func (b *SFXBank) PlaySFX(name string, cooldown time.Duration) {
	if b == nil || b.ctx == nil {
		return
	}
	pcm, ok := b.cues[name]
	if !ok || len(pcm) < 4 {
		return
	}
	now := time.Now()
	if last, exists := b.lastPlayed[name]; exists && now.Sub(last) < cooldown {
		return
	}
	b.lastPlayed[name] = now
	p := b.ctx.NewPlayerFromBytes(pcm)
	p.SetVolume(0.80)
	p.Play()
}
