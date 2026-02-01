package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"smooth/config"
	"smooth/git"
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
	commits       []git.CommitInfo
	cursor        int
	state         RestoreState
	err           error
	selected      git.CommitInfo
	branch        string
	backupName    string
	width         int
	height        int
	diffPreview   git.CommitDiffSummary // Preview of file changes
	uncommitted   git.CommitDiffSummary // Current uncommitted changes
	hasUncommit   bool                  // Whether there are uncommitted changes
	prevCursor    int                   // Track cursor changes for preview updates
}

// NewRestoreModel creates a new restore model
func NewRestoreModel() RestoreModel {
	commits, err := git.Log(20)
	branch, _ := git.CurrentBranch()

	state := RestoreStateList
	if err != nil || len(commits) == 0 {
		state = RestoreStateEmpty
	}

	// Get uncommitted changes
	uncommitted, _ := git.GetUncommittedDiffStat()
	hasUncommit := len(uncommitted.Files) > 0

	// Get initial diff preview (comparing HEAD to first commit if any)
	var diffPreview git.CommitDiffSummary
	if len(commits) > 0 {
		diffPreview, _ = git.GetDiffStatBetweenCommits(commits[0].FullHash, "HEAD")
	}

	return RestoreModel{
		commits:     commits,
		cursor:      0,
		state:       state,
		branch:      branch,
		diffPreview: diffPreview,
		uncommitted: uncommitted,
		hasUncommit: hasUncommit,
		prevCursor:  -1, // Force initial update
	}
}

// Init initializes the restore model
func (m RestoreModel) Init() tea.Cmd {
	return nil
}

// RestoreMsg is sent when a restore operation completes
type RestoreMsg struct {
	Err        error
	BackupName string
}

// doRestore creates a backup then performs the git reset
func doRestore(commitHash string, branch string) tea.Cmd {
	return func() tea.Msg {
		// Create a backup first
		backupName, err := git.CreateBackup(branch)
		if err != nil {
			return RestoreMsg{Err: fmt.Errorf("failed to create backup: %w", err)}
		}

		// Trim old backups based on config
		cfg, _ := config.Load()
		git.TrimBackups(branch, cfg.MaxBackups)

		// Now do the reset
		err = git.ResetHard(commitHash)
		if err != nil {
			return RestoreMsg{Err: err, BackupName: backupName}
		}

		return RestoreMsg{Err: nil, BackupName: backupName}
	}
}

// Update handles messages for the restore model
func (m RestoreModel) Update(msg tea.Msg) (RestoreModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case RestoreMsg:
		m.backupName = msg.BackupName
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
				return m, doRestore(m.selected.FullHash, m.branch)
			case "n", "N", "esc":
				m.state = RestoreStateList
			}
		}
	}

	// Update diff preview when cursor changes
	if m.state == RestoreStateList && m.cursor != m.prevCursor && len(m.commits) > 0 {
		m.prevCursor = m.cursor
		// Get diff between selected commit and HEAD
		m.diffPreview, _ = git.GetDiffStatBetweenCommits(m.commits[m.cursor].FullHash, "HEAD")
	}

	return m, nil
}

// View renders the restore flow
func (m RestoreModel) View() string {
	var s string

	s += RenderTitle("Revert to Previous State") + "\n\n"

	switch m.state {
	case RestoreStateEmpty:
		s += RenderMuted("No save points found!") + "\n\n"
		s += RenderMuted("Save your progress first before you can restore.") + "\n\n"
		s += HelpText("Press any key to go back")

	case RestoreStateList:
		s += RenderSubtitle("Select a save point to revert back to:") + "\n\n"

		// Build left panel (commit list)
		leftPanel := m.renderCommitList()

		// Build right panel (preview)
		rightPanel := m.renderPreviewPanel()

		// Join panels side by side
		content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
		s += content + "\n\n"

		s += HelpBar([][]string{{"↑↓", "navigate"}, {"enter", "select"}, {"esc", "cancel"}})

	case RestoreStateConfirm:
		s += RenderError("⚠ Warning: This will discard current changes!") + "\n\n"
		s += "Restore to: " + HighlightStyle.Render(m.selected.Hash) + "\n"
		s += RenderMuted(m.selected.Message) + "\n\n"
		s += RenderMuted("A backup will be created before restoring.") + "\n\n"
		s += RenderSubtitle("Are you sure? (y/n)") + "\n"

	case RestoreStateRestoring:
		s += RenderHighlight("Creating backup and restoring...") + "\n"

	case RestoreStateSuccess:
		s += RenderSuccess("✓ Restored!") + "\n\n"
		s += RenderMuted("Your project has been restored to the selected state.") + "\n"
		s += RenderMuted("Backup created: ") + MutedStyle.Render(m.backupName) + "\n\n"
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

// renderCommitList renders the left panel with the commit list
func (m RestoreModel) renderCommitList() string {
	var lines []string

	// Calculate maxVisible based on terminal height
	maxVisible := 8
	if m.height > 0 {
		available := m.height - 14 // Reserve space for chrome
		maxVisible = available / 3 // Each item is ~3 lines
		if maxVisible < 2 {
			maxVisible = 2
		}
		if maxVisible > 10 {
			maxVisible = 10
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
		if len(line) > 45 {
			line = line[:42] + "..."
		}

		lines = append(lines, cursor+style.Render(line))
		lines = append(lines, "    "+MutedStyle.Render(commit.Timestamp))
		lines = append(lines, "")
	}

	if len(m.commits) > maxVisible {
		lines = append(lines, MutedStyle.Render(fmt.Sprintf("  ... %d total saves", len(m.commits))))
	}

	// Set a fixed width for the left panel
	leftStyle := lipgloss.NewStyle().Width(50)
	return leftStyle.Render(strings.Join(lines, "\n"))
}

// renderPreviewPanel renders the right panel with the preview of what will happen
func (m RestoreModel) renderPreviewPanel() string {
	var lines []string

	// Panel styling
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(0, 1).
		Width(40)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	addStyle := lipgloss.NewStyle().Foreground(ColorSuccess)
	delStyle := lipgloss.NewStyle().Foreground(ColorDanger)

	lines = append(lines, titleStyle.Render("Preview"))
	lines = append(lines, "")

	// If on the most recent commit (cursor == 0), just show uncommitted changes
	if m.cursor == 0 {
		if m.hasUncommit {
			lines = append(lines, MutedStyle.Render("Uncommitted changes will be lost:"))
			lines = append(lines, "")
			lines = append(lines, m.renderFileStats(m.uncommitted, 6)...)
		} else {
			lines = append(lines, MutedStyle.Render("No uncommitted changes."))
			lines = append(lines, "")
			lines = append(lines, MutedStyle.Render("Reverting here has no effect."))
		}
	} else {
		// Show commits that will be lost
		lines = append(lines, ErrorStyle.Render("Saves that will be reverted:"))
		lines = append(lines, "")

		// Show commits between current position and selected
		maxCommitsToShow := 4
		for i := 0; i < m.cursor && i < maxCommitsToShow; i++ {
			c := m.commits[i]
			msg := c.Message
			if len(msg) > 30 {
				msg = msg[:27] + "..."
			}
			lines = append(lines, MutedStyle.Render(fmt.Sprintf("  • %s", msg)))
		}
		if m.cursor > maxCommitsToShow {
			lines = append(lines, MutedStyle.Render(fmt.Sprintf("  ... and %d more", m.cursor-maxCommitsToShow)))
		}
		lines = append(lines, "")

		// Show file changes
		if len(m.diffPreview.Files) > 0 {
			lines = append(lines, MutedStyle.Render("File changes:"))
			lines = append(lines, "")
			lines = append(lines, m.renderFileStats(m.diffPreview, 5)...)

			// Summary
			lines = append(lines, "")
			summary := fmt.Sprintf("%s / %s",
				addStyle.Render(fmt.Sprintf("+%d", m.diffPreview.TotalAdded)),
				delStyle.Render(fmt.Sprintf("-%d", m.diffPreview.TotalDeleted)))
			lines = append(lines, MutedStyle.Render("Total: ")+summary)
		} else {
			lines = append(lines, MutedStyle.Render("No file changes."))
		}

		// Also note uncommitted changes if any
		if m.hasUncommit {
			lines = append(lines, "")
			lines = append(lines, ErrorStyle.Render("+ uncommitted changes"))
		}
	}

	return panelStyle.Render(strings.Join(lines, "\n"))
}

// renderFileStats renders file statistics with +/- numbers
func (m RestoreModel) renderFileStats(summary git.CommitDiffSummary, maxFiles int) []string {
	var lines []string

	addStyle := lipgloss.NewStyle().Foreground(ColorSuccess)
	delStyle := lipgloss.NewStyle().Foreground(ColorDanger)

	for i, f := range summary.Files {
		if i >= maxFiles {
			lines = append(lines, MutedStyle.Render(fmt.Sprintf("  ... and %d more files", len(summary.Files)-maxFiles)))
			break
		}

		path := f.Path
		if len(path) > 25 {
			path = "..." + path[len(path)-22:]
		}

		var stat string
		if f.IsBinary {
			stat = MutedStyle.Render("(binary)")
		} else {
			stat = fmt.Sprintf("%s %s",
				addStyle.Render(fmt.Sprintf("+%d", f.Additions)),
				delStyle.Render(fmt.Sprintf("-%d", f.Deletions)))
		}

		lines = append(lines, fmt.Sprintf("  %-25s %s", path, stat))
	}

	return lines
}

// IsDone returns true if the restore flow is complete
func (m RestoreModel) IsDone() bool {
	return m.state == RestoreStateSuccess || m.state == RestoreStateError || m.state == RestoreStateEmpty
}
