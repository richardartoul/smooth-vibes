package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vc/git"
)

// SyncState represents the state of the sync flow
type SyncState int

const (
	SyncStateSyncing SyncState = iota
	SyncStateSuccess
	SyncStateError
)

// SyncModel is the model for the sync flow
type SyncModel struct {
	spinner spinner.Model
	state   SyncState
	err     error
	branch  string
}

// NewSyncModel creates a new sync model
func NewSyncModel() SyncModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorAccent)

	branch, _ := git.CurrentBranch()

	return SyncModel{
		spinner: s,
		state:   SyncStateSyncing,
		branch:  branch,
	}
}

// Init initializes the sync model
func (m SyncModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, doSync())
}

// SyncMsg is sent when a sync operation completes
type SyncMsg struct {
	Err error
}

// doSync performs the actual git push
func doSync() tea.Cmd {
	return func() tea.Msg {
		err := git.Push()
		return SyncMsg{Err: err}
	}
}

// Update handles messages for the sync model
func (m SyncModel) Update(msg tea.Msg) (SyncModel, tea.Cmd) {
	switch msg := msg.(type) {
	case SyncMsg:
		if msg.Err != nil {
			m.state = SyncStateError
			m.err = msg.Err
		} else {
			m.state = SyncStateSuccess
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == SyncStateSyncing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View renders the sync flow
func (m SyncModel) View() string {
	var s string

	s += RenderTitle("Sync to GitHub") + "\n\n"

	switch m.state {
	case SyncStateSyncing:
		s += m.spinner.View() + " " + RenderHighlight("Syncing...") + "\n\n"
		s += RenderMuted("Uploading your saves to GitHub...") + "\n"

	case SyncStateSuccess:
		s += RenderSuccess("✓ Synced!") + "\n\n"
		s += RenderMuted("Your work is now on GitHub.") + "\n\n"
		s += HelpText("Press any key to continue")

	case SyncStateError:
		s += RenderError("✗ Sync failed") + "\n\n"
		if m.err != nil {
			s += RenderMuted(m.err.Error()) + "\n\n"
		}
		s += RenderMuted("Make sure you have an internet connection") + "\n"
		s += RenderMuted("and have set up your GitHub remote.") + "\n\n"
		s += HelpText("Press any key to go back")
	}

	return BoxStyle.Render(s)
}

// IsDone returns true if the sync flow is complete
func (m SyncModel) IsDone() bool {
	return m.state == SyncStateSuccess || m.state == SyncStateError
}

