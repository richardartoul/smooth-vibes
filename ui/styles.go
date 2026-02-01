package ui

import (
	"github.com/charmbracelet/lipgloss"

	"vc/config"
)

// Color variables - updated by ApplyTheme
var (
	ColorPrimary    lipgloss.Color
	ColorSecondary  lipgloss.Color
	ColorAccent     lipgloss.Color
	ColorSuccess    lipgloss.Color
	ColorDanger     lipgloss.Color
	ColorMuted      lipgloss.Color
	ColorBackground lipgloss.Color
	ColorText       lipgloss.Color
	ColorHighlight  lipgloss.Color
)

// Text styles - updated by ApplyTheme
var (
	TitleStyle     lipgloss.Style
	SubtitleStyle  lipgloss.Style
	NormalStyle    lipgloss.Style
	MutedStyle     lipgloss.Style
	SuccessStyle   lipgloss.Style
	ErrorStyle     lipgloss.Style
	HighlightStyle lipgloss.Style
)

// Menu styles - updated by ApplyTheme
var (
	MenuItemStyle         lipgloss.Style
	MenuItemSelectedStyle lipgloss.Style
	MenuCursorStyle       lipgloss.Style
)

// Box styles - updated by ApplyTheme
var (
	BoxStyle       lipgloss.Style
	HeaderBoxStyle lipgloss.Style
)

// Input styles - updated by ApplyTheme
var (
	InputStyle        lipgloss.Style
	InputFocusedStyle lipgloss.Style
)

// List item styles - updated by ApplyTheme
var (
	ListItemStyle         lipgloss.Style
	ListItemSelectedStyle lipgloss.Style
	ListItemDescStyle     lipgloss.Style
)

func init() {
	// Apply default theme on startup
	ApplyTheme(config.CurrentTheme())
}

// ApplyTheme updates all styles based on the given theme
func ApplyTheme(theme config.Theme) {
	// Update colors
	ColorPrimary = lipgloss.Color(theme.Primary)
	ColorSecondary = lipgloss.Color(theme.Secondary)
	ColorAccent = lipgloss.Color(theme.Accent)
	ColorSuccess = lipgloss.Color(theme.Success)
	ColorDanger = lipgloss.Color(theme.Danger)
	ColorMuted = lipgloss.Color(theme.Muted)
	ColorBackground = lipgloss.Color(theme.Background)
	ColorText = lipgloss.Color(theme.Text)
	ColorHighlight = lipgloss.Color(theme.Highlight)

	// Update text styles
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Italic(true)

	NormalStyle = lipgloss.NewStyle().
		Foreground(ColorText)

	MutedStyle = lipgloss.NewStyle().
		Foreground(ColorMuted)

	SuccessStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	HighlightStyle = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Bold(true)

	// Update menu styles
	MenuItemStyle = lipgloss.NewStyle().
		PaddingLeft(2)

	MenuItemSelectedStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(ColorAccent).
		Bold(true)

	MenuCursorStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	// Update box styles
	BoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(1, 2)

	HeaderBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		MarginBottom(1)

	// Update input styles
	InputStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorSecondary).
		Padding(0, 1)

	InputFocusedStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorAccent).
		Padding(0, 1)

	// Update list item styles
	ListItemStyle = lipgloss.NewStyle().
		PaddingLeft(2)

	// Derive a subtle background from the theme
	ListItemSelectedStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(ColorAccent).
		Background(ColorBackground)

	ListItemDescStyle = lipgloss.NewStyle().
		Foreground(ColorMuted).
		PaddingLeft(4)
}

// ReloadTheme reloads the theme from config
func ReloadTheme() {
	ApplyTheme(config.CurrentTheme())
}

// Helper functions
func RenderTitle(text string) string {
	return TitleStyle.Render(text)
}

func RenderSubtitle(text string) string {
	return SubtitleStyle.Render(text)
}

func RenderSuccess(text string) string {
	return SuccessStyle.Render(text)
}

func RenderError(text string) string {
	return ErrorStyle.Render(text)
}

func RenderMuted(text string) string {
	return MutedStyle.Render(text)
}

func RenderHighlight(text string) string {
	return HighlightStyle.Render(text)
}

// Banner renders the app banner
func Banner() string {
	banner := `
 ███████ ███   ███  ██████   ██████  ████████ ██  ██
 ██      ████ ████ ██    ██ ██    ██    ██    ██  ██
 ███████ ██ ███ ██ ██    ██ ██    ██    ██    ██████
      ██ ██  █  ██ ██    ██ ██    ██    ██    ██  ██
 ███████ ██     ██  ██████   ██████     ██    ██  ██`

	return lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(banner)
}

// HelpKey renders a keyboard key with subtle styling
func HelpKey(key string) string {
	return lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Render(key)
}

// HelpText renders the bottom help text with styled keys
// Format: "key1: action1 • key2: action2" or use HelpBar for more control
func HelpText(text string) string {
	return lipgloss.NewStyle().
		Foreground(ColorMuted).
		MarginTop(1).
		Render(text)
}

// HelpBar renders a cleaner help bar with styled key hints
func HelpBar(hints [][]string) string {
	var parts []string
	for _, hint := range hints {
		if len(hint) >= 2 {
			key := HelpKey(hint[0])
			action := MutedStyle.Render(hint[1])
			parts = append(parts, key+" "+action)
		}
	}

	separator := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render("  ·  ")

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += separator
		}
		result += part
	}

	return lipgloss.NewStyle().
		MarginTop(1).
		Render(result)
}
