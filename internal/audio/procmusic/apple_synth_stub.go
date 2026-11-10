package procmusic

import (
	"fmt"
	"runtime"
)

// SynthesizeScoreApple fails fast when the pure-Go YM2612 engine is compiled without darwin Cgo frameworks.
func SynthesizeScoreApple(score *Score) ([]byte, error) {
	return nil, fmt.Errorf("apple high-fidelity synthesis is disabled in pure-Go build (%s/%s)", runtime.GOOS, runtime.GOARCH)
}

// SynthesizeScoreAppleWithStems fails fast when the pure-Go YM2612 engine is compiled without darwin Cgo frameworks.
func SynthesizeScoreAppleWithStems(score *Score) ([]byte, [][]byte, error) {
	return nil, nil, fmt.Errorf("apple high-fidelity synthesis is disabled in pure-Go build (%s/%s)", runtime.GOOS, runtime.GOARCH)
}
