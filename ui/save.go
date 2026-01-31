package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vc/git"
)

// SaveState represents the state of the save flow
type SaveState int

const (
	SaveStateInput SaveState = iota
	SaveStateSaving
	SaveStateSuccess
	SaveStateError
	SaveStateNoChanges
)

// SaveModel is the model for the save flow
type SaveModel struct {
	textInput textinput.Model
	state     SaveState
	err       error
}

// NewSaveModel creates a new save model
func NewSaveModel() SaveModel {
	ti := textinput.New()
	ti.Placeholder = "What did you work on?"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorAccent)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorText)

	state := SaveStateInput
	if !git.HasChanges() {
		state = SaveStateNoChanges
	}

	return SaveModel{
		textInput: ti,
		state:     state,
	}
}

// Init initializes the save model
func (m SaveModel) Init() tea.Cmd {
	return textinput.Blink
}

// SaveMsg is sent when a save operation completes
type SaveMsg struct {
	Err error
}

// doSave performs the actual git operations
func doSave(message string) tea.Cmd {
	return func() tea.Msg {
		// Stage all changes
		if err := git.AddAll(); err != nil {
			return SaveMsg{Err: err}
		}

		// Commit with message
		if err := git.Commit(message); err != nil {
			return SaveMsg{Err: err}
		}

		return SaveMsg{Err: nil}
	}
}

// Update handles messages for the save model
func (m SaveModel) Update(msg tea.Msg) (SaveModel, tea.Cmd) {
	switch msg := msg.(type) {
	case SaveMsg:
		if msg.Err != nil {
			m.state = SaveStateError
			m.err = msg.Err
		} else {
			m.state = SaveStateSuccess
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.state == SaveStateInput && m.textInput.Value() != "" {
				m.state = SaveStateSaving
				return m, doSave(m.textInput.Value())
			}
		}
	}

	if m.state == SaveStateInput {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the save flow
func (m SaveModel) View() string {
	var s string

	s += RenderTitle("Save Progress") + "\n\n"

	switch m.state {
	case SaveStateNoChanges:
		s += RenderMuted("No changes to save!") + "\n\n"
		s += RenderMuted("Your work is already saved.") + "\n\n"
		s += HelpText("Press any key to go back")

	case SaveStateInput:
		s += RenderSubtitle("Describe what you worked on:") + "\n\n"
		s += m.textInput.View() + "\n\n"
		s += HelpText("enter: save • esc: cancel")

	case SaveStateSaving:
		s += RenderHighlight("Saving your progress...") + "\n"

	case SaveStateSuccess:
		s += RenderSuccess("✓ Progress saved!") + "\n\n"
		s += RenderMuted("Your work has been safely stored.") + "\n\n"
		s += HelpText("Press any key to continue")

	case SaveStateError:
		s += RenderError("✗ Failed to save") + "\n\n"
		if m.err != nil {
			s += RenderMuted(m.err.Error()) + "\n\n"
		}
		s += HelpText("Press any key to go back")
	}

	return BoxStyle.Render(s)
}

// IsDone returns true if the save flow is complete
func (m SaveModel) IsDone() bool {
	return m.state == SaveStateSuccess || m.state == SaveStateError || m.state == SaveStateNoChanges
}

