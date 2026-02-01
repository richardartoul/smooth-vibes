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
	state      QuicksaveState
	err        error
	fileCount  int
	synced     bool
	syncErr    error
	commitHash string
}

// NewQuicksaveModel creates a new quicksave model
func NewQuicksaveModel() QuicksaveModel {
	if !git.HasChanges() {
		return QuicksaveModel{
			state: QuicksaveStateNoChanges,
		}
	}

	return QuicksaveModel{
		state: QuicksaveStateSaving,
	}
}

// Init initializes the quicksave model and starts saving
func (m QuicksaveModel) Init() tea.Cmd {
	if m.state == QuicksaveStateNoChanges {
		return nil
	}
	return doQuicksave()
}

// QuicksaveMsg is sent when quicksave completes
type QuicksaveMsg struct {
	Err       error
	FileCount int
	Hash      string
}

// QuicksaveSyncMsg is sent when sync completes
type QuicksaveSyncMsg struct {
	Err error
}

// doQuicksave performs the quicksave operation
func doQuicksave() tea.Cmd {
	return func() tea.Msg {
		// Get all changed files
		changes, err := git.GetChangeSummary()
		if err != nil {
			return QuicksaveMsg{Err: err}
		}

		if len(changes) == 0 {
			return QuicksaveMsg{Err: fmt.Errorf("no changes to save")}
		}

		// Stage all files
		if err := git.AddAll(); err != nil {
			return QuicksaveMsg{Err: fmt.Errorf("failed to stage files: %w", err)}
		}

		// Generate a simple commit message with timestamp
		message := fmt.Sprintf("Quicksave %s", time.Now().Format("Jan 2, 3:04 PM"))

		// Commit
		if err := git.Commit(message); err != nil {
			return QuicksaveMsg{Err: fmt.Errorf("failed to commit: %w", err)}
		}

		// Get the commit hash for display
		hash, _ := git.Run("rev-parse", "--short", "HEAD")

		return QuicksaveMsg{
			FileCount: len(changes),
			Hash:      hash,
		}
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

		m.fileCount = msg.FileCount
		m.commitHash = msg.Hash

		// Check if auto-sync is enabled
		cfg, _ := config.Load()
		if cfg.AutoSyncEnabled && git.HasRemote() {
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

	s += RenderTitle("Quicksave") + "\n\n"

	switch m.state {
	case QuicksaveStateNoChanges:
		s += RenderMuted("No changes to save!") + "\n\n"
		s += RenderMuted("Your work is already saved.") + "\n\n"
		s += HelpText("Press any key to go back")

	case QuicksaveStateSaving:
		s += RenderHighlight("⟳ Saving all changes...") + "\n"

	case QuicksaveStateSyncing:
		s += RenderSuccess("✓ Saved!") + "\n\n"
		s += RenderHighlight("⟳ Syncing to GitHub...") + "\n"

	case QuicksaveStateSuccess:
		s += RenderSuccess("✓ Quicksave complete!") + "\n\n"
		s += fmt.Sprintf("  Saved %s across %d file(s)\n",
			HighlightStyle.Render(m.commitHash),
			m.fileCount)
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
		s += RenderError("✗ Quicksave failed") + "\n\n"
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

