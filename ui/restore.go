package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"vc/git"
)

// RestoreState represents the state of the restore flow
type RestoreState int

const (
	RestoreStateList RestoreState = iota
	RestoreStateConfirm
	RestoreStateRestoring
	RestoreStateSuccess
	RestoreStateError
	RestoreStateEmpty
)

// RestoreModel is the model for the restore flow
type RestoreModel struct {
	commits  []git.CommitInfo
	cursor   int
	state    RestoreState
	err      error
	selected git.CommitInfo
}

// NewRestoreModel creates a new restore model
func NewRestoreModel() RestoreModel {
	commits, err := git.Log(20)
	
	state := RestoreStateList
	if err != nil || len(commits) == 0 {
		state = RestoreStateEmpty
	}

	return RestoreModel{
		commits: commits,
		cursor:  0,
		state:   state,
	}
}

// Init initializes the restore model
func (m RestoreModel) Init() tea.Cmd {
	return nil
}

// RestoreMsg is sent when a restore operation completes
type RestoreMsg struct {
	Err error
}

// doRestore performs the actual git reset
func doRestore(commitHash string) tea.Cmd {
	return func() tea.Msg {
		err := git.ResetHard(commitHash)
		return RestoreMsg{Err: err}
	}
}

// Update handles messages for the restore model
func (m RestoreModel) Update(msg tea.Msg) (RestoreModel, tea.Cmd) {
	switch msg := msg.(type) {
	case RestoreMsg:
		if msg.Err != nil {
			m.state = RestoreStateError
			m.err = msg.Err
		} else {
			m.state = RestoreStateSuccess
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case RestoreStateList:
			switch {
			case key.Matches(msg, keys.Up):
				if m.cursor > 0 {
					m.cursor--
				}
			case key.Matches(msg, keys.Down):
				if m.cursor < len(m.commits)-1 {
					m.cursor++
				}
			case key.Matches(msg, keys.Enter):
				m.selected = m.commits[m.cursor]
				m.state = RestoreStateConfirm
			}

		case RestoreStateConfirm:
			switch msg.String() {
			case "y", "Y":
				m.state = RestoreStateRestoring
				return m, doRestore(m.selected.FullHash)
			case "n", "N", "esc":
				m.state = RestoreStateList
			}
		}
	}

	return m, nil
}

// View renders the restore flow
func (m RestoreModel) View() string {
	var s string

	s += RenderTitle("Restore Previous State") + "\n\n"

	switch m.state {
	case RestoreStateEmpty:
		s += RenderMuted("No save points found!") + "\n\n"
		s += RenderMuted("Save your progress first before you can restore.") + "\n\n"
		s += HelpText("Press any key to go back")

	case RestoreStateList:
		s += RenderSubtitle("Select a save point to restore:") + "\n\n"

		for i, commit := range m.commits {
			cursor := "  "
			style := ListItemStyle

			if m.cursor == i {
				cursor = MenuCursorStyle.Render("> ")
				style = ListItemSelectedStyle
			}

			// Format: hash - message (time ago)
			line := fmt.Sprintf("%s %s", commit.Hash, commit.Message)
			if len(line) > 60 {
				line = line[:57] + "..."
			}
			
			s += cursor + style.Render(line) + "\n"
			s += "    " + MutedStyle.Render(commit.Timestamp) + "\n\n"
		}

		s += HelpText("↑/↓: navigate • enter: select • esc: cancel")

	case RestoreStateConfirm:
		s += RenderError("⚠ Warning: This will discard current changes!") + "\n\n"
		s += "Restore to: " + HighlightStyle.Render(m.selected.Hash) + "\n"
		s += RenderMuted(m.selected.Message) + "\n\n"
		s += RenderSubtitle("Are you sure? (y/n)") + "\n"

	case RestoreStateRestoring:
		s += RenderHighlight("Restoring...") + "\n"

	case RestoreStateSuccess:
		s += RenderSuccess("✓ Restored!") + "\n\n"
		s += RenderMuted("Your project has been restored to the selected state.") + "\n\n"
		s += HelpText("Press any key to continue")

	case RestoreStateError:
		s += RenderError("✗ Restore failed") + "\n\n"
		if m.err != nil {
			s += RenderMuted(m.err.Error()) + "\n\n"
		}
		s += HelpText("Press any key to go back")
	}

	return BoxStyle.Render(s)
}

// IsDone returns true if the restore flow is complete
func (m RestoreModel) IsDone() bool {
	return m.state == RestoreStateSuccess || m.state == RestoreStateError || m.state == RestoreStateEmpty
}

