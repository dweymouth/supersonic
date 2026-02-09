package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const eqPresetsDir = "eq_presets"

// EQPreset represents an equalizer preset that can be saved/loaded
type EQPreset struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`           // "ISO10Band" or "ISO15Band"
	Preamp    float64   `json:"preamp"`
	Bands     []float64 `json:"bands"`
	IsBuiltin bool      `json:"-"` // not saved to file, determined at load time
}

// EQPresetManager handles loading and saving EQ presets
type EQPresetManager struct {
	presetsDir string
	builtinPresets []EQPreset
}

// NewEQPresetManager creates a new preset manager
func NewEQPresetManager(configDir string) *EQPresetManager {
	return &EQPresetManager{
		presetsDir: filepath.Join(configDir, eqPresetsDir),
		builtinPresets: getBuiltinPresets(),
	}
}

// getBuiltinPresets returns the built-in equalizer presets
func getBuiltinPresets() []EQPreset {
	return []EQPreset{
		{Name: "Flat", Type: "ISO15Band", Preamp: 0, Bands: []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, IsBuiltin: true},
		{Name: "Rock", Type: "ISO15Band", Preamp: 0, Bands: []float64{5, 4, 3, 1, -1, -1, 0, 2, 3, 4, 4, 4, 3, 2, 2}, IsBuiltin: true},
		{Name: "Pop", Type: "ISO15Band", Preamp: 0, Bands: []float64{-1, -1, 0, 2, 4, 4, 2, 0, -1, -1, 0, 1, 2, 3, 3}, IsBuiltin: true},
		{Name: "Jazz", Type: "ISO15Band", Preamp: 0, Bands: []float64{4, 3, 1, 2, -2, -2, 0, 2, 3, 3, 3, 4, 4, 4, 4}, IsBuiltin: true},
		{Name: "Classical", Type: "ISO15Band", Preamp: 0, Bands: []float64{5, 4, 3, 2, -1, -1, 0, 2, 3, 3, 3, 2, 2, 2, -1}, IsBuiltin: true},
		{Name: "Bass Boost", Type: "ISO15Band", Preamp: 0, Bands: []float64{6, 5, 4, 3, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, IsBuiltin: true},
		{Name: "Treble Boost", Type: "ISO15Band", Preamp: 0, Bands: []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 3, 4, 5, 6, 6}, IsBuiltin: true},
		{Name: "Vocal", Type: "ISO15Band", Preamp: 0, Bands: []float64{-2, -3, -3, 1, 4, 4, 4, 3, 2, 1, 0, -1, -2, -2, -3}, IsBuiltin: true},
		{Name: "Electronic", Type: "ISO15Band", Preamp: 0, Bands: []float64{5, 4, 2, 0, -2, -2, 0, 2, 3, 4, 4, 3, 4, 4, 3}, IsBuiltin: true},
		{Name: "Acoustic", Type: "ISO15Band", Preamp: 0, Bands: []float64{5, 4, 3, 1, 2, 1, 1, 2, 2, 2, 1, 2, 2, 3, 2}, IsBuiltin: true},
		{Name: "R&B", Type: "ISO15Band", Preamp: 0, Bands: []float64{3, 6, 5, 2, -2, -2, 2, 3, 2, 2, 3, 3, 3, 3, 4}, IsBuiltin: true},
		{Name: "Loudness", Type: "ISO15Band", Preamp: 0, Bands: []float64{6, 5, 3, 0, -1, -1, -1, -1, 0, 1, 2, 4, 5, 5, 3}, IsBuiltin: true},
	}
}

// LoadPresets loads all presets (builtin + user-defined)
func (m *EQPresetManager) LoadPresets() ([]EQPreset, error) {
	// Start with builtin presets
	presets := make([]EQPreset, len(m.builtinPresets))
	copy(presets, m.builtinPresets)

	// Create presets directory if it doesn't exist
	if err := os.MkdirAll(m.presetsDir, 0o755); err != nil {
		return presets, err
	}

	// Load user presets
	entries, err := os.ReadDir(m.presetsDir)
	if err != nil {
		return presets, nil // Return builtins if can't read dir
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		presetPath := filepath.Join(m.presetsDir, entry.Name())
		data, err := os.ReadFile(presetPath)
		if err != nil {
			continue // Skip invalid files
		}

		var preset EQPreset
		if err := json.Unmarshal(data, &preset); err != nil {
			continue // Skip invalid JSON
		}

		preset.IsBuiltin = false
		presets = append(presets, preset)
	}

	// Sort: builtins first, then user presets alphabetically
	sort.Slice(presets, func(i, j int) bool {
		if presets[i].IsBuiltin != presets[j].IsBuiltin {
			return presets[i].IsBuiltin
		}
		return presets[i].Name < presets[j].Name
	})

	return presets, nil
}

// SavePreset saves a preset to disk
func (m *EQPresetManager) SavePreset(preset EQPreset) error {
	if preset.IsBuiltin {
		return fmt.Errorf("cannot overwrite builtin preset")
	}

	// Create presets directory if it doesn't exist
	if err := os.MkdirAll(m.presetsDir, 0o755); err != nil {
		return err
	}

	// Sanitize filename
	filename := sanitizeFilename(preset.Name) + ".json"
	presetPath := filepath.Join(m.presetsDir, filename)

	data, err := json.MarshalIndent(preset, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(presetPath, data, 0o644)
}

// DeletePreset deletes a user preset
func (m *EQPresetManager) DeletePreset(name string) error {
	// Check if it's a builtin preset
	for _, bp := range m.builtinPresets {
		if bp.Name == name {
			return fmt.Errorf("cannot delete builtin preset")
		}
	}

	filename := sanitizeFilename(name) + ".json"
	presetPath := filepath.Join(m.presetsDir, filename)

	return os.Remove(presetPath)
}

// sanitizeFilename removes characters that aren't safe for filenames
func sanitizeFilename(name string) string {
	// Simple sanitization - could be enhanced
	var result []rune
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == ' ' {
			result = append(result, r)
		}
	}
	return string(result)
}
