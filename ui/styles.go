package ui

import "github.com/charmbracelet/lipgloss"

// Color palette - warm, friendly vibes
var (
	ColorPrimary    = lipgloss.Color("#FF6B6B") // Coral red
	ColorSecondary  = lipgloss.Color("#4ECDC4") // Teal
	ColorAccent     = lipgloss.Color("#FFE66D") // Sunny yellow
	ColorSuccess    = lipgloss.Color("#95E77E") // Fresh green
	ColorDanger     = lipgloss.Color("#FF6B6B") // Coral red
	ColorMuted      = lipgloss.Color("#6C757D") // Gray
	ColorBackground = lipgloss.Color("#2D3436") // Dark charcoal
	ColorText       = lipgloss.Color("#DFE6E9") // Light gray
	ColorHighlight  = lipgloss.Color("#A29BFE") // Lavender
)

// Text styles
var (
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
)

// Menu styles
var (
	MenuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	MenuItemSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(ColorAccent).
				Bold(true)

	MenuCursorStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)
)

// Box styles
var (
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
			Padding(1, 2)

	HeaderBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 2).
			MarginBottom(1)
)

// Input styles
var (
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorSecondary).
			Padding(0, 1)

	InputFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorAccent).
				Padding(0, 1)
)

// List item styles
var (
	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	ListItemSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(ColorAccent).
				Background(lipgloss.Color("#3D4446"))

	ListItemDescStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				PaddingLeft(4)
)

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
 ███████╗███╗   ███╗ ██████╗  ██████╗ ████████╗██╗  ██╗
 ██╔════╝████╗ ████║██╔═══██╗██╔═══██╗╚══██╔══╝██║  ██║
 ███████╗██╔████╔██║██║   ██║██║   ██║   ██║   ███████║
 ╚════██║██║╚██╔╝██║██║   ██║██║   ██║   ██║   ██╔══██║
 ███████║██║ ╚═╝ ██║╚██████╔╝╚██████╔╝   ██║   ██║  ██║
 ╚══════╝╚═╝     ╚═╝ ╚═════╝  ╚═════╝    ╚═╝   ╚═╝  ╚═╝`

	return lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(banner)
}

// HelpText renders the bottom help text
func HelpText(text string) string {
	return lipgloss.NewStyle().
		Foreground(ColorMuted).
		MarginTop(1).
		Render(text)
}
