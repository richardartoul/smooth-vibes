package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Theme represents a color theme
type Theme struct {
	Name       string
	Primary    string // Main accent color
	Secondary  string // Secondary accent
	Accent     string // Highlight/selection color
	Success    string // Success messages
	Danger     string // Error/danger messages
	Muted      string // Muted/subtle text
	Background string // Background elements
	Text       string // Main text color
	Highlight  string // Highlighted text
}

// Available themes
var Themes = map[string]Theme{
	"coral": {
		Name:       "Coral Sunset",
		Primary:    "#FF6B6B",
		Secondary:  "#4ECDC4",
		Accent:     "#FFE66D",
		Success:    "#95E77E",
		Danger:     "#FF6B6B",
		Muted:      "#6C757D",
		Background: "#2D3436",
		Text:       "#DFE6E9",
		Highlight:  "#A29BFE",
	},
	"ocean": {
		Name:       "Ocean Breeze",
		Primary:    "#0077B6",
		Secondary:  "#00B4D8",
		Accent:     "#90E0EF",
		Success:    "#2EC4B6",
		Danger:     "#E63946",
		Muted:      "#6B7280",
		Background: "#1B263B",
		Text:       "#CAF0F8",
		Highlight:  "#48CAE4",
	},
	"forest": {
		Name:       "Forest",
		Primary:    "#2D6A4F",
		Secondary:  "#40916C",
		Accent:     "#95D5B2",
		Success:    "#74C69D",
		Danger:     "#BC4749",
		Muted:      "#6B705C",
		Background: "#1B4332",
		Text:       "#D8F3DC",
		Highlight:  "#B7E4C7",
	},
	"dracula": {
		Name:       "Dracula",
		Primary:    "#FF79C6",
		Secondary:  "#BD93F9",
		Accent:     "#F1FA8C",
		Success:    "#50FA7B",
		Danger:     "#FF5555",
		Muted:      "#6272A4",
		Background: "#282A36",
		Text:       "#F8F8F2",
		Highlight:  "#8BE9FD",
	},
	"nord": {
		Name:       "Nord",
		Primary:    "#88C0D0",
		Secondary:  "#81A1C1",
		Accent:     "#EBCB8B",
		Success:    "#A3BE8C",
		Danger:     "#BF616A",
		Muted:      "#4C566A",
		Background: "#2E3440",
		Text:       "#ECEFF4",
		Highlight:  "#B48EAD",
	},
	"solarized": {
		Name:       "Solarized Dark",
		Primary:    "#268BD2",
		Secondary:  "#2AA198",
		Accent:     "#B58900",
		Success:    "#859900",
		Danger:     "#DC322F",
		Muted:      "#586E75",
		Background: "#002B36",
		Text:       "#93A1A1",
		Highlight:  "#6C71C4",
	},
	"monokai": {
		Name:       "Monokai",
		Primary:    "#F92672",
		Secondary:  "#66D9EF",
		Accent:     "#E6DB74",
		Success:    "#A6E22E",
		Danger:     "#F92672",
		Muted:      "#75715E",
		Background: "#272822",
		Text:       "#F8F8F2",
		Highlight:  "#AE81FF",
	},
	"cyberpunk": {
		Name:       "Cyberpunk",
		Primary:    "#FF00FF",
		Secondary:  "#00FFFF",
		Accent:     "#FFFF00",
		Success:    "#00FF00",
		Danger:     "#FF0055",
		Muted:      "#666699",
		Background: "#0D0221",
		Text:       "#EEEEFF",
		Highlight:  "#FF6EC7",
	},
	"gruvbox": {
		Name:       "Gruvbox",
		Primary:    "#FB4934",
		Secondary:  "#83A598",
		Accent:     "#FABD2F",
		Success:    "#B8BB26",
		Danger:     "#FB4934",
		Muted:      "#928374",
		Background: "#282828",
		Text:       "#EBDBB2",
		Highlight:  "#D3869B",
	},
	"rosepine": {
		Name:       "Rosé Pine",
		Primary:    "#EBBCBA",
		Secondary:  "#31748F",
		Accent:     "#F6C177",
		Success:    "#9CCFD8",
		Danger:     "#EB6F92",
		Muted:      "#6E6A86",
		Background: "#191724",
		Text:       "#E0DEF4",
		Highlight:  "#C4A7E7",
	},
}

// ThemeNames returns the list of available theme IDs in display order
var ThemeNames = []string{
	"coral", "ocean", "forest", "dracula", "nord",
	"solarized", "monokai", "cyberpunk", "gruvbox", "rosepine",
}

// Config holds application configuration
type Config struct {
	AutoSyncEnabled    bool   `json:"autoSyncEnabled"`
	MaxBackups         int    `json:"maxBackups"`
	ExperimentsEnabled bool   `json:"experimentsEnabled"`
	Theme              string `json:"theme"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() Config {
	return Config{
		AutoSyncEnabled:    false,
		MaxBackups:         10,
		ExperimentsEnabled: false,
		Theme:              "coral",
	}
}

// GetTheme returns the theme for the given name, or default if not found
func GetTheme(name string) Theme {
	if theme, ok := Themes[name]; ok {
		return theme
	}
	return Themes["coral"]
}

// CurrentTheme returns the theme from the current config
func CurrentTheme() Theme {
	cfg, _ := Load()
	return GetTheme(cfg.Theme)
}

// configPath returns the path to the config file
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".smooth", "config.json"), nil
}

// Load reads the config from disk, returning defaults if not found
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if config doesn't exist
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}

	// Ensure MaxBackups has a reasonable minimum
	if cfg.MaxBackups < 1 {
		cfg.MaxBackups = 1
	}

	// Ensure Theme has a valid value
	if cfg.Theme == "" {
		cfg.Theme = "coral"
	} else if _, ok := Themes[cfg.Theme]; !ok {
		cfg.Theme = "coral"
	}

	return cfg, nil
}

// Save writes the config to disk
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Create .smooth directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
