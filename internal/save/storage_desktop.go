//go:build !js

package save

import (
	"fmt"
	"os"
	"path/filepath"
)

// DesktopStorage persists save slots atomically inside os.UserConfigDir().
type DesktopStorage struct {
	baseDir string
}

// NewStorage creates a DesktopStorage rooted at os.UserConfigDir()/appID.
func NewStorage(appID string) (Storage, error) {
	if appID == "" {
		return nil, fmt.Errorf("save: appID must not be empty")
	}
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir = os.TempDir()
	}
	dir := filepath.Join(cfgDir, appID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("save: create save directory %q: %w", dir, err)
	}
	return &DesktopStorage{baseDir: dir}, nil
}

// Save writes data to a temporary file and renames it atomically.
func (ds *DesktopStorage) Save(slot string, data GameSaveData) error {
	if slot == "" {
		return fmt.Errorf("save: slot name must not be empty")
	}
	raw, err := MarshalSave(data)
	if err != nil {
		return err
	}
	targetPath := filepath.Join(ds.baseDir, slot+".json")
	tmpPath := targetPath + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o600); err != nil {
		return fmt.Errorf("save: write temp save file %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("save: atomic rename %q -> %q: %w", tmpPath, targetPath, err)
	}
	return nil
}

// Load reads and validates a save slot from disk.
func (ds *DesktopStorage) Load(slot string) (GameSaveData, error) {
	if slot == "" {
		return GameSaveData{}, fmt.Errorf("save: slot name must not be empty")
	}
	targetPath := filepath.Join(ds.baseDir, slot+".json")
	raw, err := os.ReadFile(targetPath)
	if err != nil {
		return GameSaveData{}, fmt.Errorf("save: read save file %q: %w", targetPath, err)
	}
	return UnmarshalSave(raw)
}
