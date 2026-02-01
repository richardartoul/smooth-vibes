package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vc/git"
)

// SyncState represents the state of the sync flow
type SyncState int

const (
	SyncStateChecking SyncState = iota
	SyncStateNoRemote
	SyncStateSyncing
	SyncStateSuccess
	SyncStateError
)

// SyncModel is the model for the sync flow
type SyncModel struct {
	spinner   spinner.Model
	textInput textinput.Model
	state     SyncState
	err       error
	branch    string
}

// NewSyncModel creates a new sync model
func NewSyncModel() SyncModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorAccent)

	ti := textinput.New()
	ti.Placeholder = "git@github.com:username/repo.git"
	ti.CharLimit = 200
	ti.Width = 50
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorAccent)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorText)

	branch, _ := git.CurrentBranch()

	// Check if remote exists
	state := SyncStateChecking
	if !git.HasRemote() {
		state = SyncStateNoRemote
		ti.Focus()
	}

	return SyncModel{
		spinner:   s,
		textInput: ti,
		state:     state,
		branch:    branch,
	}
}

// Init initializes the sync model
func (m SyncModel) Init() tea.Cmd {
	if m.state == SyncStateNoRemote {
		return textinput.Blink
	}
	return tea.Batch(m.spinner.Tick, doSync())
}

// SyncMsg is sent when a sync operation completes
type SyncMsg struct {
	Err error
}

// AddRemoteMsg is sent when adding a remote completes
type AddRemoteMsg struct {
	Err error
}

// doSync performs the actual git push
func doSync() tea.Cmd {
	return func() tea.Msg {
		err := git.Push()
		return SyncMsg{Err: err}
	}
}

// doAddRemote adds the origin remote
func doAddRemote(url string) tea.Cmd {
	return func() tea.Msg {
		err := git.AddOrigin(url)
		return AddRemoteMsg{Err: err}
	}
}

// Update handles messages for the sync model
func (m SyncModel) Update(msg tea.Msg) (SyncModel, tea.Cmd) {
	switch msg := msg.(type) {
	case AddRemoteMsg:
		if msg.Err != nil {
			m.state = SyncStateError
			m.err = msg.Err
		} else {
			// Remote added, now sync
			m.state = SyncStateSyncing
			return m, tea.Batch(m.spinner.Tick, doSync())
		}
		return m, nil

	case SyncMsg:
		if msg.Err != nil {
			m.state = SyncStateError
			m.err = msg.Err
		} else {
			m.state = SyncStateSuccess
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == SyncStateSyncing || m.state == SyncStateChecking {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		if m.state == SyncStateNoRemote {
			switch msg.String() {
			case "enter":
				url := strings.TrimSpace(m.textInput.Value())
				if url != "" {
					m.state = SyncStateSyncing
					return m, tea.Batch(m.spinner.Tick, doAddRemote(url))
				}
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}
	}

	return m, nil
}

// View renders the sync flow
func (m SyncModel) View() string {
	var s string

	s += RenderTitle("Sync to GitHub") + "\n\n"

	switch m.state {
	case SyncStateChecking:
		s += m.spinner.View() + " " + RenderHighlight("Checking...") + "\n"

	case SyncStateNoRemote:
		s += RenderSubtitle("No GitHub remote configured") + "\n\n"
		s += RenderMuted("Enter your GitHub repository SSH URL:") + "\n\n"
		s += m.textInput.View() + "\n\n"
		s += RenderMuted("To get this URL:") + "\n"
		s += RenderMuted("  1. Go to github.com and create a new repository") + "\n"
		s += RenderMuted("  2. Click the green 'Code' button") + "\n"
		s += RenderMuted("  3. Select 'SSH' and copy the URL") + "\n"
		s += RenderMuted("     (looks like git@github.com:user/repo.git)") + "\n\n"
		s += HelpBar([][]string{{"enter", "save and sync"}, {"esc", "cancel"}})

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
		if git.HasRemote() {
			s += RenderMuted("Make sure you have an internet connection.") + "\n\n"
		}
		s += HelpText("Press any key to go back")
	}

	return BoxStyle.Render(s)
}

// IsDone returns true if the sync flow is complete
func (m SyncModel) IsDone() bool {
	return m.state == SyncStateSuccess || m.state == SyncStateError
}
