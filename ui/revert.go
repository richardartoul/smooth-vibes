package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"vc/config"
	"vc/git"
)

// RevertState represents the state of the revert flow
type RevertState int

const (
	RevertStateList RevertState = iota
	RevertStateConfirm
	RevertStateReverting
	RevertStateSuccess
	RevertStateError
	RevertStateEmpty
)

// RevertModel is the model for the revert flow
type RevertModel struct {
	commits    []git.CommitInfo
	cursor     int
	state      RevertState
	err        error
	selected   git.CommitInfo
	branch     string
	backupName string
	width      int
	height     int
}

// NewRevertModel creates a new revert model
func NewRevertModel() RevertModel {
	commits, err := git.Log(20)
	branch, _ := git.CurrentBranch()

	state := RevertStateList
	if err != nil || len(commits) == 0 {
		state = RevertStateEmpty
	}

	return RevertModel{
		commits: commits,
		cursor:  0,
		state:   state,
		branch:  branch,
	}
}

// Init initializes the revert model
func (m RevertModel) Init() tea.Cmd {
	return nil
}

// RevertMsg is sent when a revert operation completes
type RevertMsg struct {
	Err        error
	BackupName string
}

// doRevert creates a backup then performs the git reset
func doRevert(commitHash string, branch string) tea.Cmd {
	return func() tea.Msg {
		// Create a backup first
		backupName, err := git.CreateBackup(branch)
		if err != nil {
			return RevertMsg{Err: fmt.Errorf("failed to create backup: %w", err)}
		}

		// Trim old backups based on config
		cfg, _ := config.Load()
		git.TrimBackups(branch, cfg.MaxBackups)

		// Now do the reset
		err = git.ResetHard(commitHash)
		if err != nil {
			return RevertMsg{Err: err, BackupName: backupName}
		}

		return RevertMsg{Err: nil, BackupName: backupName}
	}
}

// Update handles messages for the revert model
func (m RevertModel) Update(msg tea.Msg) (RevertModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case RevertMsg:
		m.backupName = msg.BackupName
		if msg.Err != nil {
			m.state = RevertStateError
			m.err = msg.Err
		} else {
			m.state = RevertStateSuccess
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case RevertStateList:
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
				m.state = RevertStateConfirm
			}

		case RevertStateConfirm:
			switch msg.String() {
			case "y", "Y":
				m.state = RevertStateReverting
				return m, doRevert(m.selected.FullHash, m.branch)
			case "n", "N", "esc":
				m.state = RevertStateList
			}
		}
	}

	return m, nil
}

// View renders the revert flow
func (m RevertModel) View() string {
	var s string

	s += RenderTitle("Revert") + "\n\n"

	switch m.state {
	case RevertStateEmpty:
		s += RenderMuted("No save points found!") + "\n\n"
		s += RenderMuted("Save your progress first before you can revert.") + "\n\n"
		s += HelpText("Press any key to go back")

	case RevertStateList:
		s += RenderSubtitle("Select a save point to revert to:") + "\n\n"

		// Calculate maxVisible based on terminal height
		// Each item takes ~3 lines, reserve ~8 lines for chrome (title, subtitle, help, borders)
		maxVisible := 8
		if m.height > 0 {
			available := m.height - 10 // Reserve space for chrome
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

		for i := start; i < len(m.commits) && i < start+maxVisible; i++ {
			commit := m.commits[i]
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

		if len(m.commits) > maxVisible {
			s += MutedStyle.Render(fmt.Sprintf("  ... %d total saves\n", len(m.commits)))
		}

		s += HelpBar([][]string{{"↑↓", "navigate"}, {"enter", "select"}, {"esc", "cancel"}})

	case RevertStateConfirm:
		s += RenderError("⚠ Warning: This will discard current changes!") + "\n\n"
		s += "Revert to: " + HighlightStyle.Render(m.selected.Hash) + "\n"
		s += RenderMuted(m.selected.Message) + "\n\n"
		s += RenderMuted("A backup will be created before reverting.") + "\n\n"
		s += RenderSubtitle("Are you sure? (y/n)") + "\n"

	case RevertStateReverting:
		s += RenderHighlight("Creating backup and reverting...") + "\n"

	case RevertStateSuccess:
		s += RenderSuccess("✓ Reverted!") + "\n\n"
		s += RenderMuted("Your project has been reverted to the selected state.") + "\n"
		s += RenderMuted("Backup created: ") + MutedStyle.Render(m.backupName) + "\n\n"
		s += HelpText("Press any key to continue")

	case RevertStateError:
		s += RenderError("✗ Revert failed") + "\n\n"
		if m.err != nil {
			s += RenderMuted(m.err.Error()) + "\n\n"
		}
		s += HelpText("Press any key to go back")
	}

	return BoxStyle.Render(s)
}

// IsDone returns true if the revert flow is complete
func (m RevertModel) IsDone() bool {
	return m.state == RevertStateSuccess || m.state == RevertStateError || m.state == RevertStateEmpty
}
