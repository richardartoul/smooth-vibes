package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"vc/config"
	"vc/git"
)

// QuicksaveState represents the state of the quicksave flow
type QuicksaveState int

const (
	QuicksaveStateSaving QuicksaveState = iota
	QuicksaveStateSyncing
	QuicksaveStateSuccess
	QuicksaveStateError
	QuicksaveStateNoChanges
)

// QuicksaveModel is the model for the quicksave flow
type QuicksaveModel struct {
	state         QuicksaveState
	err           error
	fileActions   map[string]FileAction
	synced        bool
	syncErr       error
	commitHash    string
	savedCount    int
	revertedCount int
	ignoredCount  int
	skippedCount  int
}

// NewQuicksaveModel creates a new quicksave model with file actions
func NewQuicksaveModel(fileActions map[string]FileAction) QuicksaveModel {
	if !git.HasChanges() {
		return QuicksaveModel{
			state: QuicksaveStateNoChanges,
		}
	}

	return QuicksaveModel{
		state:       QuicksaveStateSaving,
		fileActions: fileActions,
	}
}

// Init initializes the quicksave model and starts saving
func (m QuicksaveModel) Init() tea.Cmd {
	if m.state == QuicksaveStateNoChanges {
		return nil
	}
	return doQuicksave(m.fileActions)
}

// QuicksaveMsg is sent when quicksave completes
type QuicksaveMsg struct {
	Err           error
	Hash          string
	SavedCount    int
	RevertedCount int
	IgnoredCount  int
	SkippedCount  int
}

// QuicksaveSyncMsg is sent when sync completes
type QuicksaveSyncMsg struct {
	Err error
}

// doQuicksave performs the quicksave operation with file actions
func doQuicksave(fileActions map[string]FileAction) tea.Cmd {
	return func() tea.Msg {
		// Get all changed files
		changes, err := git.GetChangeSummary()
		if err != nil {
			return QuicksaveMsg{Err: err}
		}

		if len(changes) == 0 {
			return QuicksaveMsg{Err: fmt.Errorf("no changes to save")}
		}

		var toSave []string
		var toRevert []string
		var toIgnore []string
		skipped := 0

		for _, change := range changes {
			action, ok := fileActions[change.Path]
			if !ok {
				action = FileActionSave // Default to save
			}

			switch action {
			case FileActionSave:
				toSave = append(toSave, change.Path)
			case FileActionRevert:
				toRevert = append(toRevert, change.Path)
			case FileActionIgnore:
				toIgnore = append(toIgnore, change.Path)
			case FileActionIgnoreOnce:
				skipped++
			}
		}

		result := QuicksaveMsg{
			SavedCount:    len(toSave),
			RevertedCount: len(toRevert),
			IgnoredCount:  len(toIgnore),
			SkippedCount:  skipped,
		}

		// 1. Revert files first
		if len(toRevert) > 0 {
			if err := git.RevertFiles(toRevert); err != nil {
				result.Err = fmt.Errorf("failed to revert files: %w", err)
				return result
			}
		}

		// 2. Add files to gitignore
		for _, path := range toIgnore {
			if err := git.AddToGitignore(path); err != nil {
				result.Err = fmt.Errorf("failed to add %s to .gitignore: %w", path, err)
				return result
			}
		}

		// 3. Stage and commit if there are files to save
		if len(toSave) > 0 {
			// Include .gitignore if we modified it
			if len(toIgnore) > 0 {
				toSave = append(toSave, ".gitignore")
			}

			if err := git.AddFiles(toSave); err != nil {
				result.Err = fmt.Errorf("failed to stage files: %w", err)
				return result
			}

			// Generate commit message with timestamp
			message := fmt.Sprintf("Save %s", time.Now().Format("Jan 2, 3:04 PM"))

			if err := git.Commit(message); err != nil {
				result.Err = fmt.Errorf("failed to commit: %w", err)
				return result
			}

			// Get the commit hash for display
			result.Hash, _ = git.Run("rev-parse", "--short", "HEAD")
		}

		return result
	}
}

// doQuicksaveSync performs the sync operation
func doQuicksaveSync() tea.Cmd {
	return func() tea.Msg {
		err := git.Push()
		return QuicksaveSyncMsg{Err: err}
	}
}

// Update handles messages for the quicksave model
func (m QuicksaveModel) Update(msg tea.Msg) (QuicksaveModel, tea.Cmd) {
	switch msg := msg.(type) {
	case QuicksaveMsg:
		if msg.Err != nil {
			m.state = QuicksaveStateError
			m.err = msg.Err
			return m, nil
		}

		m.savedCount = msg.SavedCount
		m.revertedCount = msg.RevertedCount
		m.ignoredCount = msg.IgnoredCount
		m.skippedCount = msg.SkippedCount
		m.commitHash = msg.Hash

		// Check if auto-sync is enabled and we saved files
		cfg, _ := config.Load()
		if cfg.AutoSyncEnabled && git.HasRemote() && m.savedCount > 0 {
			m.state = QuicksaveStateSyncing
			m.synced = true
			return m, doQuicksaveSync()
		}

		m.state = QuicksaveStateSuccess
		return m, nil

	case QuicksaveSyncMsg:
		m.syncErr = msg.Err
		m.state = QuicksaveStateSuccess
		return m, nil
	}

	return m, nil
}

// View renders the quicksave flow
func (m QuicksaveModel) View() string {
	var s string

	s += RenderTitle("Save") + "\n\n"

	switch m.state {
	case QuicksaveStateNoChanges:
		s += RenderMuted("No changes to save!") + "\n\n"
		s += RenderMuted("Your work is already saved.") + "\n\n"
		s += HelpText("Press any key to go back")

	case QuicksaveStateSaving:
		s += RenderHighlight("⟳ Processing changes...") + "\n"

	case QuicksaveStateSyncing:
		s += RenderSuccess("✓ Done!") + "\n\n"
		s += RenderHighlight("⟳ Syncing to GitHub...") + "\n"

	case QuicksaveStateSuccess:
		s += RenderSuccess("✓ Complete!") + "\n\n"

		if m.savedCount > 0 {
			s += fmt.Sprintf("  %s Saved %d file(s)",
				SuccessStyle.Render("✓"), m.savedCount)
			if m.commitHash != "" {
				s += " " + MutedStyle.Render("["+m.commitHash+"]")
			}
			s += "\n"
		}
		if m.revertedCount > 0 {
			s += fmt.Sprintf("  %s Reverted %d file(s)\n", SuccessStyle.Render("✓"), m.revertedCount)
		}
		if m.ignoredCount > 0 {
			s += fmt.Sprintf("  %s Added %d file(s) to .gitignore\n", SuccessStyle.Render("✓"), m.ignoredCount)
		}
		if m.skippedCount > 0 {
			s += fmt.Sprintf("  %s Skipped %d file(s)\n", MutedStyle.Render("○"), m.skippedCount)
		}

		if m.synced {
			s += "\n"
			if m.syncErr != nil {
				s += RenderError("✗ Sync failed: ") + RenderMuted(m.syncErr.Error()) + "\n"
			} else {
				s += RenderSuccess("✓ Synced to GitHub!") + "\n"
			}
		}
		s += "\n" + HelpText("Press any key to continue")

	case QuicksaveStateError:
		s += RenderError("✗ Save failed") + "\n\n"
		if m.err != nil {
			s += RenderMuted(m.err.Error()) + "\n\n"
		}
		s += HelpText("Press any key to go back")
	}

	return BoxStyle.Render(s)
}

// IsDone returns true if the quicksave flow is complete
func (m QuicksaveModel) IsDone() bool {
	return m.state == QuicksaveStateSuccess || m.state == QuicksaveStateError || m.state == QuicksaveStateNoChanges
}

