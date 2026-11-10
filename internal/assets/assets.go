package assets

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	_ "image/png"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"pelotongrandtour/internal/audio/procmusic"
	"pelotongrandtour/internal/audio/procsfx"
)

//go:embed sprites/*.png sfx/*.sfx music/*.music fonts/*.ttf shaders/*.kage
var EmbeddedFS embed.FS

// Bundle holds all validated, decoded game assets loaded during the Boot state.
type Bundle struct {
	CyclistSheet   image.Image
	EnergyGelSheet image.Image
	BidonSheet     image.Image
	CobbleSheet    image.Image
	StageBackdrop  image.Image
	FontTTF        []byte
	SpeedShaderSrc []byte
	MusicPCM       []byte
	SFXCues        map[string][]byte
}

// LoadAll validates and decodes all embedded game assets with fail-fast error reporting.
func LoadAll() (*Bundle, error) {
	cyclistImg, err := loadAndValidatePNG("sprites/cyclist_pro.png")
	if err != nil {
		return nil, err
	}
	gelImg, err := loadAndValidatePNG("sprites/energy_gel.png")
	if err != nil {
		return nil, err
	}
	bidonImg, err := loadAndValidatePNG("sprites/bidon_bottle.png")
	if err != nil {
		return nil, err
	}
	cobbleImg, err := loadAndValidatePNG("sprites/cobble_hazard.png")
	if err != nil {
		return nil, err
	}
	backdropImg, err := loadAndValidatePNG("sprites/stage_backdrop.png")
	if err != nil {
		return nil, err
	}

	fontBytes, err := fs.ReadFile(EmbeddedFS, "fonts/font.ttf")
	if err != nil {
		return nil, fmt.Errorf("assets: read fonts/font.ttf: %w", err)
	}
	if len(fontBytes) < 64 {
		return nil, fmt.Errorf("assets: fonts/font.ttf is truncated (%d bytes)", len(fontBytes))
	}

	shaderBytes, err := fs.ReadFile(EmbeddedFS, "shaders/speed_vignette.kage")
	if err != nil {
		return nil, fmt.Errorf("assets: read shaders/speed_vignette.kage: %w", err)
	}
	if !bytes.Contains(shaderBytes, []byte("//kage:unit pixels")) {
		return nil, fmt.Errorf("assets: shader missing mandatory //kage:unit pixels directive")
	}

	musicSrc, err := fs.ReadFile(EmbeddedFS, "music/grand_tour_theme.music")
	if err != nil {
		return nil, fmt.Errorf("assets: read music/grand_tour_theme.music: %w", err)
	}
	musicPCM, err := procmusic.CompileDSL(string(musicSrc))
	if err != nil {
		return nil, fmt.Errorf("assets: compile music/grand_tour_theme.music: %w", err)
	}
	alignedMusicLen := len(musicPCM) &^ 3
	if alignedMusicLen < 4 {
		return nil, fmt.Errorf("assets: compiled music PCM is empty")
	}
	musicPCM = musicPCM[:alignedMusicLen]

	sfxEntries, err := fs.ReadDir(EmbeddedFS, "sfx")
	if err != nil {
		return nil, fmt.Errorf("assets: read sfx dir: %w", err)
	}
	sfxCues := make(map[string][]byte, len(sfxEntries))
	for _, entry := range sfxEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sfx") {
			continue
		}
		path := "sfx/" + entry.Name()
		raw, err := fs.ReadFile(EmbeddedFS, path)
		if err != nil {
			return nil, fmt.Errorf("assets: read %q: %w", path, err)
		}
		pcm, err := procsfx.CompileDSL(string(raw))
		if err != nil {
			return nil, fmt.Errorf("assets: compile SFX %q: %w", path, err)
		}
		rep, err := procsfx.ValidateProductionSFX(pcm)
		if err != nil || !rep.Passed {
			return nil, fmt.Errorf("assets: SFX %q failed production validation: %v", path, err)
		}
		cueName := strings.TrimSuffix(filepath.Base(entry.Name()), ".sfx")
		sfxCues[cueName] = pcm[:len(pcm)&^3]
	}

	if len(sfxCues) == 0 {
		return nil, fmt.Errorf("assets: zero SFX cues loaded")
	}

	return &Bundle{
		CyclistSheet:   cyclistImg,
		EnergyGelSheet: gelImg,
		BidonSheet:     bidonImg,
		CobbleSheet:    cobbleImg,
		StageBackdrop:  backdropImg,
		FontTTF:        fontBytes,
		SpeedShaderSrc: shaderBytes,
		MusicPCM:       musicPCM,
		SFXCues:        sfxCues,
	}, nil
}

func loadAndValidatePNG(path string) (image.Image, error) {
	raw, err := fs.ReadFile(EmbeddedFS, path)
	if err != nil {
		return nil, fmt.Errorf("assets: read %q: %w", path, err)
	}
	mimeType := http.DetectContentType(raw)
	if mimeType != "image/png" {
		return nil, fmt.Errorf("assets: expected image/png for %q, detected %q", path, mimeType)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("assets: decode PNG %q: %w", path, err)
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil, fmt.Errorf("assets: invalid dimensions %dx%d for %q", b.Dx(), b.Dy(), path)
	}
	return img, nil
}
