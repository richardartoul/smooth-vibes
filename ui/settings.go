package ui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vc/config"
)

// SettingsState represents the state of the settings screen
type SettingsState int

const (
	SettingsStateMenu SettingsState = iota
	SettingsStateEditMaxBackups
	SettingsStateSaving
	SettingsStateSaved
	SettingsStateError
	SettingsStateConfirmExit
)

// SettingsModel is the model for the settings screen
type SettingsModel struct {
	cfg       config.Config
	cursor    int
	state     SettingsState
	textInput textinput.Model
	err       error
	dirty     bool // whether config has been modified
	wantsExit bool // whether user confirmed exit
}

// NewSettingsModel creates a new settings model
func NewSettingsModel() SettingsModel {
	cfg, _ := config.Load()

	ti := textinput.New()
	ti.Placeholder = "10"
	ti.CharLimit = 4
	ti.Width = 10
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorAccent)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorText)

	return SettingsModel{
		cfg:       cfg,
		cursor:    0,
		state:     SettingsStateMenu,
		textInput: ti,
	}
}

// Init initializes the settings model
func (m SettingsModel) Init() tea.Cmd {
	return nil
}

// SettingsSaveMsg is sent when settings are saved
type SettingsSaveMsg struct {
	Err error
}

// doSaveSettings saves the config
func doSaveSettings(cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		err := config.Save(cfg)
		return SettingsSaveMsg{Err: err}
	}
}

// Update handles messages for the settings model
func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case SettingsSaveMsg:
		if msg.Err != nil {
			m.state = SettingsStateError
			m.err = msg.Err
		} else {
			m.state = SettingsStateSaved
			m.dirty = false
			// Apply theme now that it's saved
			ApplyTheme(config.GetTheme(m.cfg.Theme))
			// If we were saving before exit, mark exit now
			if m.wantsExit {
				return m, nil
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case SettingsStateMenu:
			switch {
			case key.Matches(msg, keys.Up):
				if m.cursor > 0 {
					m.cursor--
				}
			case key.Matches(msg, keys.Down):
				if m.cursor < 3 { // 4 settings
					m.cursor++
				}
			case key.Matches(msg, keys.Enter), msg.String() == " ":
				switch m.cursor {
				case 0: // Auto-sync toggle
					m.cfg.AutoSyncEnabled = !m.cfg.AutoSyncEnabled
					m.dirty = true
				case 1: // Max backups - switch to edit mode
					m.state = SettingsStateEditMaxBackups
					m.textInput.SetValue(fmt.Sprintf("%d", m.cfg.MaxBackups))
					m.textInput.Focus()
					return m, textinput.Blink
				case 2: // Experiments toggle
					m.cfg.ExperimentsEnabled = !m.cfg.ExperimentsEnabled
					m.dirty = true
					// case 3 (Theme) - do nothing on enter/space, use arrows only
				}
			case msg.String() == "right":
				// Right arrow cycles theme forward
				if m.cursor == 3 {
					m.cfg.Theme = nextTheme(m.cfg.Theme)
					m.dirty = true
				}
			case msg.String() == "left":
				// Left arrow cycles theme backward
				if m.cursor == 3 {
					m.cfg.Theme = prevTheme(m.cfg.Theme)
					m.dirty = true
				}
			case msg.String() == "s":
				// Save settings
				if m.dirty {
					m.state = SettingsStateSaving
					return m, doSaveSettings(m.cfg)
				}
			}

		case SettingsStateEditMaxBackups:
			switch msg.String() {
			case "enter":
				val, err := strconv.Atoi(m.textInput.Value())
				if err != nil || val < 1 {
					val = 1
				}
				if val > 1000 {
					val = 1000
				}
				m.cfg.MaxBackups = val
				m.dirty = true
				m.state = SettingsStateMenu
				return m, nil
			case "esc":
				m.state = SettingsStateMenu
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}

		case SettingsStateSaved:
			// Any key goes back to main menu
			m.wantsExit = true
			return m, nil

		case SettingsStateError:
			// Any key goes back to settings menu
			m.state = SettingsStateMenu
			return m, nil

		case SettingsStateConfirmExit:
			switch msg.String() {
			case "y", "Y":
				// Exit without saving
				m.wantsExit = true
				m.dirty = false
			case "n", "N", "esc":
				m.state = SettingsStateMenu
			case "s", "S":
				// Save and exit
				m.state = SettingsStateSaving
				m.wantsExit = true
				return m, doSaveSettings(m.cfg)
			}
		}
	}

	return m, nil
}

// View renders the settings screen
func (m SettingsModel) View() string {
	var s string

	s += RenderTitle("Settings") + "\n\n"

	switch m.state {
	case SettingsStateMenu:
		s += m.renderSettingsList() + "\n"

		// Show theme preview when hovering over theme option
		if m.cursor == 3 {
			s += m.renderThemePreview() + "\n"
		}

		if m.dirty {
			s += HighlightStyle.Render("• Unsaved changes") + "\n\n"
			if m.cursor == 3 {
				s += HelpBar([][]string{{"↑↓", "navigate"}, {"←→", "cycle theme"}, {"s", "save"}, {"esc", "back"}})
			} else {
				s += HelpBar([][]string{{"↑↓", "navigate"}, {"enter", "toggle"}, {"s", "save"}, {"esc", "back"}})
			}
		} else {
			if m.cursor == 3 {
				s += HelpBar([][]string{{"↑↓", "navigate"}, {"←→", "cycle theme"}, {"esc", "back"}})
			} else {
				s += HelpBar([][]string{{"↑↓", "navigate"}, {"enter", "toggle"}, {"esc", "back"}})
			}
		}

	case SettingsStateEditMaxBackups:
		s += RenderSubtitle("Maximum backups to keep:") + "\n\n"
		s += m.textInput.View() + "\n\n"
		s += RenderMuted("Enter a number between 1 and 1000") + "\n\n"
		s += HelpBar([][]string{{"enter", "confirm"}, {"esc", "cancel"}})

	case SettingsStateSaving:
		s += RenderHighlight("Saving settings...") + "\n"

	case SettingsStateSaved:
		s += RenderSuccess("✓ Settings saved!") + "\n\n"
		s += HelpText("Press any key to continue")

	case SettingsStateError:
		s += RenderError("✗ Failed to save settings") + "\n\n"
		if m.err != nil {
			s += RenderMuted(m.err.Error()) + "\n\n"
		}
		s += HelpText("Press any key to go back")

	case SettingsStateConfirmExit:
		s += RenderError("⚠ You have unsaved changes!") + "\n\n"
		s += RenderMuted("Do you want to save before leaving?") + "\n\n"
		s += HelpBar([][]string{{"s", "save and exit"}, {"y", "exit without saving"}, {"n", "cancel"}})
	}

	return BoxStyle.Render(s)
}

// renderSettingsList renders the list of settings
func (m SettingsModel) renderSettingsList() string {
	var s string

	settings := []struct {
		name        string
		description string
		value       string
	}{
		{
			name:        "Auto-sync to GitHub",
			description: "Automatically push to GitHub after each save",
			value:       formatBool(m.cfg.AutoSyncEnabled),
		},
		{
			name:        "Maximum backups",
			description: "Number of backups to keep per branch",
			value:       fmt.Sprintf("%d", m.cfg.MaxBackups),
		},
		{
			name:        "Experiments feature",
			description: "Enable experimental branches for trying new ideas",
			value:       formatBool(m.cfg.ExperimentsEnabled),
		},
		{
			name:        "Theme",
			description: "Color scheme for the interface",
			value:       config.GetTheme(m.cfg.Theme).Name,
		},
	}

	for i, setting := range settings {
		cursor := "  "
		style := MenuItemStyle

		if m.cursor == i {
			cursor = MenuCursorStyle.Render("> ")
			style = MenuItemSelectedStyle
		}

		// Setting name and value
		nameStr := style.Render(setting.name)
		valueStr := HighlightStyle.Render(setting.value)

		// Theme setting gets arrow indicators
		if i == 3 { // Theme
			if m.cursor == i {
				// Show arrows when selected
				s += fmt.Sprintf("%s%s: ← %s →\n", cursor, nameStr, valueStr)
			} else {
				s += fmt.Sprintf("%s%s: %s\n", cursor, nameStr, valueStr)
			}
		} else {
			s += fmt.Sprintf("%s%s: %s\n", cursor, nameStr, valueStr)
		}

		// Description
		s += "    " + MutedStyle.Render(setting.description) + "\n\n"
	}

	return s
}

// formatBool formats a boolean for display
func formatBool(b bool) string {
	if b {
		return "On"
	}
	return "Off"
}

// nextTheme returns the next theme in the cycle
func nextTheme(current string) string {
	for i, name := range config.ThemeNames {
		if name == current {
			nextIdx := (i + 1) % len(config.ThemeNames)
			return config.ThemeNames[nextIdx]
		}
	}
	return config.ThemeNames[0]
}

// prevTheme returns the previous theme in the cycle
func prevTheme(current string) string {
	for i, name := range config.ThemeNames {
		if name == current {
			prevIdx := i - 1
			if prevIdx < 0 {
				prevIdx = len(config.ThemeNames) - 1
			}
			return config.ThemeNames[prevIdx]
		}
	}
	return config.ThemeNames[0]
}

// renderThemePreview renders a preview of the selected theme's colors
func (m SettingsModel) renderThemePreview() string {
	theme := config.GetTheme(m.cfg.Theme)

	// Create styles using the theme colors directly
	primaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Primary)).Bold(true)
	secondaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Secondary))
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent)).Bold(true)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Success)).Bold(true)
	dangerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Danger)).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Text))
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Highlight)).Bold(true)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Secondary)).
		Padding(0, 1)

	var preview string
	preview += primaryStyle.Render("■") + " Primary   "
	preview += secondaryStyle.Render("■") + " Secondary   "
	preview += accentStyle.Render("■") + " Accent\n"
	preview += successStyle.Render("■") + " Success   "
	preview += dangerStyle.Render("■") + " Danger     "
	preview += highlightStyle.Render("■") + " Highlight\n"
	preview += mutedStyle.Render("■") + " Muted     "
	preview += textStyle.Render("■") + " Text\n\n"
	preview += primaryStyle.Render("Title Text") + "  "
	preview += secondaryStyle.Render("Subtitle") + "\n"
	preview += accentStyle.Render("> Selected item") + "\n"
	preview += successStyle.Render("✓ Success!") + "  "
	preview += dangerStyle.Render("✗ Error") + "\n"
	preview += mutedStyle.Render("Muted helper text")

	return boxStyle.Render(preview) + "\n"
}

// IsDone returns true if the settings screen should close
func (m SettingsModel) IsDone() bool {
	return false // Settings screen doesn't auto-close
}

// WantsBack returns true if user confirmed they want to go back
func (m SettingsModel) WantsBack() bool {
	return m.wantsExit
}

// HasUnsavedChanges returns true if there are unsaved changes
func (m SettingsModel) HasUnsavedChanges() bool {
	return m.dirty
}

// PromptExit triggers the exit confirmation prompt
func (m *SettingsModel) PromptExit() {
	if m.dirty {
		m.state = SettingsStateConfirmExit
	}
}
