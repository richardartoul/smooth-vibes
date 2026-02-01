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
				if m.cursor < 1 { // 2 settings
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

		case SettingsStateSaved, SettingsStateError:
			// Any key goes back to menu
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

		if m.dirty {
			s += HighlightStyle.Render("• Unsaved changes") + "\n\n"
			s += HelpText("↑/↓: navigate • enter/space: toggle • s: save • esc: back")
		} else {
			s += HelpText("↑/↓: navigate • enter/space: toggle • esc: back")
		}

	case SettingsStateEditMaxBackups:
		s += RenderSubtitle("Maximum backups to keep:") + "\n\n"
		s += m.textInput.View() + "\n\n"
		s += RenderMuted("Enter a number between 1 and 1000") + "\n\n"
		s += HelpText("enter: confirm • esc: cancel")

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
		s += HelpText("s: save and exit • y: exit without saving • n: cancel")
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
		s += fmt.Sprintf("%s%s: %s\n", cursor, nameStr, valueStr)

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

// IsDone returns true if the settings screen should close
func (m SettingsModel) IsDone() bool {
	return false // Settings screen doesn't auto-close
}

// WantsBack returns true if user wants to go back (only when not dirty or confirmed)
func (m SettingsModel) WantsBack() bool {
	return m.wantsExit || (m.state == SettingsStateMenu && !m.dirty)
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

