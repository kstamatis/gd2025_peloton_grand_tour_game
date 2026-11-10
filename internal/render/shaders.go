package render

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// ShaderPass wraps a compiled Kage shader and pre-allocated DrawRectShaderOptions.
type ShaderPass struct {
	shader     *ebiten.Shader
	op         ebiten.DrawRectShaderOptions
	screenSize []float32
	tintColor  []float32
}

// NewShaderPass compiles a Kage shader source at Boot time and pre-allocates uniform slices.
func NewShaderPass(kageSource []byte) (*ShaderPass, error) {
	if len(kageSource) == 0 {
		return nil, fmt.Errorf("render: empty Kage shader source")
	}
	sh, err := ebiten.NewShader(kageSource)
	if err != nil {
		return nil, fmt.Errorf("render: compile Kage shader: %w", err)
	}
	sp := &ShaderPass{
		shader:     sh,
		screenSize: []float32{1280, 720},
		tintColor:  []float32{1.0, 0.85, 0.15},
	}
	sp.op.Uniforms = make(map[string]any, 4)
	return sp, nil
}

// DrawSpeedVignette renders the full-screen speed/effort Kage shader overlay with zero heap allocations.
func (sp *ShaderPass) DrawSpeedVignette(dst *ebiten.Image, width, height int, intensity, timeSec float32, r, g, b float32) {
	if sp == nil || sp.shader == nil || intensity <= 0.01 {
		return
	}
	sp.screenSize[0] = float32(width)
	sp.screenSize[1] = float32(height)
	sp.tintColor[0] = r
	sp.tintColor[1] = g
	sp.tintColor[2] = b

	sp.op.GeoM.Reset()
	sp.op.Uniforms["ScreenSize"] = sp.screenSize
	sp.op.Uniforms["Intensity"] = intensity
	sp.op.Uniforms["Time"] = timeSec
	sp.op.Uniforms["TintColor"] = sp.tintColor
	dst.DrawRectShader(width, height, sp.shader, &sp.op)
}
