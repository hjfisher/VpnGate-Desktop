package data

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppSettings are persisted across restarts.
type AppSettings struct {
	Language           string   `json:"language"`
	Theme              string   `json:"theme"` // "system" | "light" | "dark"
	SortBy             string   `json:"sort_by"` // score | ping | speed | sessions
	SortAscending      bool     `json:"sort_ascending"`
	AutoRefreshMinutes int      `json:"auto_refresh_minutes"` // 0 = off
	UseMirror          bool     `json:"use_mirror"`
	ExportFolder       string   `json:"export_folder"` // empty => default Downloads/VPNGate
	Favorites          []string `json:"favorites"`
}

func DefaultSettings() AppSettings {
	return AppSettings{
		Language:  "en",
		Theme:     "system",
		SortBy:    "score",
		SortAscending: false,
		AutoRefreshMinutes: 0,
		UseMirror:false,
	}
}

// SettingsStore reads and writes AppSettings JSON in the app config dir.
type SettingsStore struct {
	path string
}

func NewSettingsStore(configDir string) *SettingsStore {
	return &SettingsStore{path: filepath.Join(configDir, "settings.json")}
}

func (s *SettingsStore) Load() AppSettings {
	settings := DefaultSettings()
	data, err := os.ReadFile(s.path)
	if err != nil {
		return settings
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return DefaultSettings()
	}
	// Merge defaults for any field left zero-valued in an old file.
	def := DefaultSettings()
	if settings.SortBy == "" {
		settings.SortBy = def.SortBy
	}
	if settings.Theme == "" {
		settings.Theme = def.Theme
	}
	if settings.ExportFolder == "" {
		settings.ExportFolder = def.ExportFolder
	}
	return settings
}

func (s *SettingsStore) Save(settings AppSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(s.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(s.path, data, 0o644)
}