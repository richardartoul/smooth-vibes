package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"vc/git"
)

// BackupsState represents the state of the backups flow
type BackupsState int

const (
	BackupsStateList BackupsState = iota
	BackupsStateConfirm
	BackupsStateRestoring
	BackupsStateSuccess
	BackupsStateError
	BackupsStateEmpty
)

// BackupsModel is the model for the backups flow
type BackupsModel struct {
	backups  []git.BackupInfo
	cursor   int
	state    BackupsState
	err      error
	selected git.BackupInfo
	branch   string
	width    int
	height   int
}

// NewBackupsModel creates a new backups model
func NewBackupsModel() BackupsModel {
	branch, _ := git.CurrentBranch()
	backups, _ := git.ListBackups(branch)

	state := BackupsStateList
	if len(backups) == 0 {
		state = BackupsStateEmpty
	}

	return BackupsModel{
		backups: backups,
		cursor:  0,
		state:   state,
		branch:  branch,
	}
}

// Init initializes the backups model
func (m BackupsModel) Init() tea.Cmd {
	return nil
}

// BackupsMsg is sent when a backup operation completes
type BackupsMsg struct {
	Err error
}

// doRestoreBackup performs the backup restoration
func doRestoreBackup(backupBranch string) tea.Cmd {
	return func() tea.Msg {
		err := git.RestoreBackup(backupBranch)
		return BackupsMsg{Err: err}
	}
}

// Update handles messages for the backups model
func (m BackupsModel) Update(msg tea.Msg) (BackupsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case BackupsMsg:
		if msg.Err != nil {
			m.state = BackupsStateError
			m.err = msg.Err
		} else {
			m.state = BackupsStateSuccess
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case BackupsStateList:
			switch {
			case key.Matches(msg, keys.Up):
				if m.cursor > 0 {
					m.cursor--
				}
			case key.Matches(msg, keys.Down):
				if m.cursor < len(m.backups)-1 {
					m.cursor++
				}
			case key.Matches(msg, keys.Enter):
				m.selected = m.backups[m.cursor]
				m.state = BackupsStateConfirm
			}

		case BackupsStateConfirm:
			switch msg.String() {
			case "y", "Y":
				m.state = BackupsStateRestoring
				return m, doRestoreBackup(m.selected.Name)
			case "n", "N", "esc":
				m.state = BackupsStateList
			}
		}
	}

	return m, nil
}

// View renders the backups flow
func (m BackupsModel) View() string {
	var s string

	s += RenderTitle("Backups") + "\n\n"
	s += RenderMuted(fmt.Sprintf("Showing backups for: %s", m.branch)) + "\n\n"

	switch m.state {
	case BackupsStateEmpty:
		s += RenderMuted("No backups found for this branch!") + "\n\n"
		s += RenderMuted("Backups are created automatically when you restore") + "\n"
		s += RenderMuted("to a previous state.") + "\n\n"
		s += HelpText("Press any key to go back")

	case BackupsStateList:
		s += RenderSubtitle("Select a backup to restore:") + "\n\n"

		// Calculate maxVisible based on terminal height
		maxVisible := 8
		if m.height > 0 {
			available := m.height - 12 // Reserve space for chrome
			maxVisible = available / 3 // Each item is ~3 lines
			if maxVisible < 2 {
				maxVisible = 2
			}
			if maxVisible > 12 {
				maxVisible = 12
			}
		}

		start := 0
		if m.cursor >= maxVisible {
			start = m.cursor - maxVisible + 1
		}

		for i := start; i < len(m.backups) && i < start+maxVisible; i++ {
			backup := m.backups[i]
			cursor := "  "
			style := ListItemStyle

			if m.cursor == i {
				cursor = MenuCursorStyle.Render("> ")
				style = ListItemSelectedStyle
			}

			// Format timestamp nicely
			timestamp := formatBackupTimestamp(backup.Timestamp)

			// Format: timestamp - message
			line := fmt.Sprintf("%s  %s", timestamp, backup.Message)
			if len(line) > 55 {
				line = line[:52] + "..."
			}

			s += cursor + style.Render(line) + "\n"
			s += "    " + MutedStyle.Render(backup.CommitHash) + "\n\n"
		}

		if len(m.backups) > maxVisible {
			s += MutedStyle.Render(fmt.Sprintf("  ... %d total backups\n", len(m.backups)))
		}

		s += HelpText("↑/↓: navigate • enter: restore • esc: cancel")

	case BackupsStateConfirm:
		s += RenderError("⚠ Warning: This will discard current changes!") + "\n\n"
		s += "Restore backup from: " + HighlightStyle.Render(formatBackupTimestamp(m.selected.Timestamp)) + "\n"
		s += RenderMuted(m.selected.Message) + "\n\n"
		s += RenderSubtitle("Are you sure? (y/n)") + "\n"

	case BackupsStateRestoring:
		s += RenderHighlight("Restoring from backup...") + "\n"

	case BackupsStateSuccess:
		s += RenderSuccess("✓ Restored from backup!") + "\n\n"
		s += RenderMuted("Your project has been restored to the backup state.") + "\n\n"
		s += HelpText("Press any key to continue")

	case BackupsStateError:
		s += RenderError("✗ Restore failed") + "\n\n"
		if m.err != nil {
			s += RenderMuted(m.err.Error()) + "\n\n"
		}
		s += HelpText("Press any key to go back")
	}

	return BoxStyle.Render(s)
}

// IsDone returns true if the backups flow is complete
func (m BackupsModel) IsDone() bool {
	return m.state == BackupsStateSuccess || m.state == BackupsStateError || m.state == BackupsStateEmpty
}

// formatBackupTimestamp formats the timestamp for display
func formatBackupTimestamp(timestamp string) string {
	// Input format: 20060102-150405
	if len(timestamp) >= 15 {
		date := timestamp[:8]
		time := timestamp[9:15]
		// Format as: 2006-01-02 15:04:05
		return fmt.Sprintf("%s-%s-%s %s:%s:%s",
			date[:4], date[4:6], date[6:8],
			time[:2], time[2:4], time[4:6])
	}
	return timestamp
}
