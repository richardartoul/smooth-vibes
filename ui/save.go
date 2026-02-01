package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vc/config"
	"vc/git"
)

// SaveState represents the state of the save flow
type SaveState int

const (
	SaveStateReview SaveState = iota
	SaveStateGitignorePrompt
	SaveStateInput
	SaveStateSaving
	SaveStateAutoSyncing
	SaveStateSuccess
	SaveStateError
	SaveStateNoChanges
)

// FileSelection represents a file with its selection state
type FileSelection struct {
	Change   git.FileChange
	Selected bool
}

// SaveModel is the model for the save flow
type SaveModel struct {
	textInput         textinput.Model
	state             SaveState
	err               error
	files             []FileSelection
	cursor            int
	gitignoreFile     string // file being considered for gitignore
	gitignoreIdx      int    // index of that file
	gitignoreModified bool   // whether we added something to .gitignore
	autoSynced        bool   // whether auto-sync was performed
	syncErr           error  // error from auto-sync (if any)
}

// NewSaveModel creates a new save model
func NewSaveModel() SaveModel {
	ti := textinput.New()
	ti.Placeholder = "What did you work on?"
	ti.CharLimit = 100
	ti.Width = 50
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorAccent)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorText)

	changes, _ := git.GetChangeSummary()

	state := SaveStateReview
	if len(changes) == 0 {
		state = SaveStateNoChanges
	}

	// Convert to FileSelection with all selected by default
	files := make([]FileSelection, len(changes))
	for i, c := range changes {
		files[i] = FileSelection{
			Change:   c,
			Selected: true,
		}
	}

	return SaveModel{
		textInput: ti,
		state:     state,
		files:     files,
		cursor:    0,
	}
}

// Init initializes the save model
func (m SaveModel) Init() tea.Cmd {
	return nil
}

// SaveMsg is sent when a save operation completes
type SaveMsg struct {
	Err error
}

// AutoSyncMsg is sent when auto-sync completes
type AutoSyncMsg struct {
	Err error
}

// doSave performs the actual git operations
func doSave(message string, files []string) tea.Cmd {
	return func() tea.Msg {
		// Stage selected files
		if err := git.AddFiles(files); err != nil {
			return SaveMsg{Err: err}
		}

		// Commit with message
		if err := git.Commit(message); err != nil {
			return SaveMsg{Err: err}
		}

		return SaveMsg{Err: nil}
	}
}

// doAutoSync performs the auto-sync to GitHub
func doAutoSync() tea.Cmd {
	return func() tea.Msg {
		err := git.Push()
		return AutoSyncMsg{Err: err}
	}
}

// getSelectedFiles returns paths of selected files
func (m SaveModel) getSelectedFiles() []string {
	var paths []string
	for _, f := range m.files {
		if f.Selected {
			paths = append(paths, f.Change.Path)
		}
	}
	// Include .gitignore if we modified it
	if m.gitignoreModified {
		paths = append(paths, ".gitignore")
	}
	return paths
}

// countSelected returns count of selected files
func (m SaveModel) countSelected() int {
	count := 0
	for _, f := range m.files {
		if f.Selected {
			count++
		}
	}
	return count
}

// Update handles messages for the save model
func (m SaveModel) Update(msg tea.Msg) (SaveModel, tea.Cmd) {
	switch msg := msg.(type) {
	case SaveMsg:
		if msg.Err != nil {
			m.state = SaveStateError
			m.err = msg.Err
		} else {
			// Check if auto-sync is enabled
			cfg, _ := config.Load()
			if cfg.AutoSyncEnabled && git.HasRemote() {
				m.state = SaveStateAutoSyncing
				m.autoSynced = true
				return m, doAutoSync()
			}
			m.state = SaveStateSuccess
		}
		return m, nil

	case AutoSyncMsg:
		m.syncErr = msg.Err
		m.state = SaveStateSuccess
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case SaveStateReview:
			switch {
			case key.Matches(msg, keys.Up):
				if m.cursor > 0 {
					m.cursor--
				}
			case key.Matches(msg, keys.Down):
				if m.cursor < len(m.files)-1 {
					m.cursor++
				}
			case msg.String() == " ":
				// Toggle selection
				wasSelected := m.files[m.cursor].Selected
				m.files[m.cursor].Selected = !wasSelected

				// If deselecting, prompt for gitignore
				if wasSelected {
					m.gitignoreFile = m.files[m.cursor].Change.Path
					m.gitignoreIdx = m.cursor
					m.state = SaveStateGitignorePrompt
				}
			case key.Matches(msg, keys.Enter):
				if m.countSelected() > 0 {
					m.state = SaveStateInput
					m.textInput.Focus()
					return m, textinput.Blink
				}
			}

		case SaveStateGitignorePrompt:
			switch msg.String() {
			case "y", "Y":
				// Add to gitignore
				git.AddToGitignore(m.gitignoreFile)
				m.gitignoreModified = true
				m.state = SaveStateReview
			case "n", "N", "esc":
				m.state = SaveStateReview
			}

		case SaveStateInput:
			switch msg.String() {
			case "enter":
				if m.textInput.Value() != "" {
					m.state = SaveStateSaving
					return m, doSave(m.textInput.Value(), m.getSelectedFiles())
				}
			case "esc":
				// Go back to review
				m.state = SaveStateReview
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}
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

	case SaveStateReview:
		s += RenderSubtitle("Select files to save:") + "\n\n"
		s += m.renderFileList() + "\n"

		selected := m.countSelected()
		total := len(m.files)
		s += MutedStyle.Render(fmt.Sprintf("%d of %d files selected", selected, total)) + "\n\n"

		s += HelpBar([][]string{{"↑↓", "navigate"}, {"space", "toggle"}, {"enter", "continue"}, {"esc", "cancel"}})

	case SaveStateGitignorePrompt:
		s += RenderSubtitle("Add to .gitignore?") + "\n\n"
		s += "You deselected: " + HighlightStyle.Render(m.gitignoreFile) + "\n\n"
		s += RenderMuted("Would you like to add this file to .gitignore") + "\n"
		s += RenderMuted("so it's never tracked?") + "\n\n"
		s += HelpBar([][]string{{"y", "add to .gitignore"}, {"n", "skip this time"}})

	case SaveStateInput:
		// Show summary of what will be saved
		s += m.renderSummary() + "\n"
		s += RenderSubtitle("Describe what you worked on:") + "\n\n"
		s += m.textInput.View() + "\n\n"
		s += HelpBar([][]string{{"enter", "save"}, {"esc", "go back"}})

	case SaveStateSaving:
		s += RenderHighlight("Saving your progress...") + "\n"

	case SaveStateAutoSyncing:
		s += RenderSuccess("✓ Progress saved!") + "\n\n"
		s += RenderHighlight("Syncing to GitHub...") + "\n"

	case SaveStateSuccess:
		s += RenderSuccess("✓ Progress saved!") + "\n\n"
		if m.autoSynced {
			if m.syncErr != nil {
				s += RenderError("✗ Auto-sync failed: ") + RenderMuted(m.syncErr.Error()) + "\n"
			} else {
				s += RenderSuccess("✓ Synced to GitHub!") + "\n"
			}
			s += "\n"
		}
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

// renderFileList renders the interactive file list
func (m SaveModel) renderFileList() string {
	var s string

	maxVisible := 10
	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	for i := start; i < len(m.files) && i < start+maxVisible; i++ {
		f := m.files[i]

		// Cursor
		cursor := "  "
		if i == m.cursor {
			cursor = MenuCursorStyle.Render("> ")
		}

		// Checkbox
		checkbox := "[ ]"
		if f.Selected {
			checkbox = SuccessStyle.Render("[✓]")
		}

		// Status icon
		var icon string
		var style lipgloss.Style
		switch f.Change.Status {
		case "added":
			icon = "+"
			style = SuccessStyle
		case "deleted":
			icon = "-"
			style = ErrorStyle
		default:
			icon = "~"
			style = HighlightStyle
		}

		// File path
		path := f.Change.Path
		if len(path) > 45 {
			path = "..." + path[len(path)-42:]
		}

		pathStyle := NormalStyle
		if !f.Selected {
			pathStyle = MutedStyle
		}

		s += fmt.Sprintf("%s%s %s %s\n", cursor, checkbox, style.Render(icon), pathStyle.Render(path))
	}

	if len(m.files) > maxVisible {
		s += MutedStyle.Render(fmt.Sprintf("\n  ... %d total files", len(m.files)))
	}

	return s
}

// renderSummary renders the summary of selected changes
func (m SaveModel) renderSummary() string {
	var s string

	// Count by type (only selected)
	added, modified, deleted := 0, 0, 0
	for _, f := range m.files {
		if !f.Selected {
			continue
		}
		switch f.Change.Status {
		case "added":
			added++
		case "deleted":
			deleted++
		default:
			modified++
		}
	}

	s += RenderSubtitle("Saving:") + " "

	var parts []string
	if added > 0 {
		parts = append(parts, SuccessStyle.Render(fmt.Sprintf("+%d", added)))
	}
	if modified > 0 {
		parts = append(parts, HighlightStyle.Render(fmt.Sprintf("~%d", modified)))
	}
	if deleted > 0 {
		parts = append(parts, ErrorStyle.Render(fmt.Sprintf("-%d", deleted)))
	}

	for i, part := range parts {
		if i > 0 {
			s += " "
		}
		s += part
	}
	s += "\n\n"

	return s
}

// IsDone returns true if the save flow is complete
func (m SaveModel) IsDone() bool {
	return m.state == SaveStateSuccess || m.state == SaveStateError || m.state == SaveStateNoChanges
}
