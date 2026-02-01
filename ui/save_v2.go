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

// FileAction represents what to do with a file
type FileAction int

const (
	FileActionSave       FileAction = iota // Stage and commit the file
	FileActionRevert                       // Discard changes (restore to HEAD)
	FileActionIgnoreOnce                   // Skip this time, keep local changes
	FileActionIgnore                       // Add to .gitignore
)

func (a FileAction) String() string {
	switch a {
	case FileActionSave:
		return "Save"
	case FileActionRevert:
		return "Revert"
	case FileActionIgnoreOnce:
		return "Skip"
	case FileActionIgnore:
		return "Ignore"
	default:
		return "?"
	}
}

// SaveV2State represents the state of the save flow
type SaveV2State int

const (
	SaveV2StateReview SaveV2State = iota
	SaveV2StateInput
	SaveV2StateExecuting
	SaveV2StateAutoSyncing
	SaveV2StateSuccess
	SaveV2StateError
	SaveV2StateNoChanges
)

// FileItem represents a file with its action
type FileItem struct {
	Change git.FileChange
	Action FileAction
}

// SaveV2Model is the model for the experimental save flow
type SaveV2Model struct {
	textInput         textinput.Model
	state             SaveV2State
	err               error
	files             []FileItem
	cursor            int
	gitignoreModified bool
	autoSynced        bool
	syncErr           error
	revertedCount     int
	savedCount        int
	ignoredCount      int
	skippedCount      int
}

// NewSaveV2Model creates a new experimental save model
func NewSaveV2Model() SaveV2Model {
	ti := textinput.New()
	ti.Placeholder = "What did you work on?"
	ti.CharLimit = 100
	ti.Width = 50
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorAccent)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorText)

	changes, _ := git.GetChangeSummary()

	state := SaveV2StateReview
	if len(changes) == 0 {
		state = SaveV2StateNoChanges
	}

	// Convert to FileItem with Save as default
	files := make([]FileItem, len(changes))
	for i, c := range changes {
		files[i] = FileItem{
			Change: c,
			Action: FileActionSave,
		}
	}

	return SaveV2Model{
		textInput: ti,
		state:     state,
		files:     files,
		cursor:    0,
	}
}

// Init initializes the model
func (m SaveV2Model) Init() tea.Cmd {
	return nil
}

// SaveV2Msg is sent when operations complete
type SaveV2Msg struct {
	Err           error
	RevertedCount int
	SavedCount    int
	IgnoredCount  int
}

// AutoSyncV2Msg is sent when auto-sync completes
type AutoSyncV2Msg struct {
	Err error
}

// doSaveV2 performs all the git operations based on file actions
func doSaveV2(message string, files []FileItem) tea.Cmd {
	return func() tea.Msg {
		var toSave []string
		var toRevert []string
		var toIgnore []string

		for _, f := range files {
			switch f.Action {
			case FileActionSave:
				toSave = append(toSave, f.Change.Path)
			case FileActionRevert:
				toRevert = append(toRevert, f.Change.Path)
			case FileActionIgnore:
				toIgnore = append(toIgnore, f.Change.Path)
			// FileActionIgnoreOnce: do nothing, leave file as-is
			}
		}

		result := SaveV2Msg{
			RevertedCount: len(toRevert),
			SavedCount:    len(toSave),
			IgnoredCount:  len(toIgnore),
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

			if err := git.Commit(message); err != nil {
				result.Err = fmt.Errorf("failed to commit: %w", err)
				return result
			}
		}

		return result
	}
}

// doAutoSyncV2 performs auto-sync
func doAutoSyncV2() tea.Cmd {
	return func() tea.Msg {
		err := git.Push()
		return AutoSyncV2Msg{Err: err}
	}
}

// cycleAction moves to the next action state
func cycleAction(current FileAction) FileAction {
	switch current {
	case FileActionSave:
		return FileActionRevert
	case FileActionRevert:
		return FileActionIgnoreOnce
	case FileActionIgnoreOnce:
		return FileActionIgnore
	case FileActionIgnore:
		return FileActionSave
	default:
		return FileActionSave
	}
}

// cycleActionReverse moves to the previous action state
func cycleActionReverse(current FileAction) FileAction {
	switch current {
	case FileActionSave:
		return FileActionIgnore
	case FileActionRevert:
		return FileActionSave
	case FileActionIgnoreOnce:
		return FileActionRevert
	case FileActionIgnore:
		return FileActionIgnoreOnce
	default:
		return FileActionSave
	}
}

// countByAction returns counts for each action type
func (m SaveV2Model) countByAction() (save, revert, skip, ignore int) {
	for _, f := range m.files {
		switch f.Action {
		case FileActionSave:
			save++
		case FileActionRevert:
			revert++
		case FileActionIgnoreOnce:
			skip++
		case FileActionIgnore:
			ignore++
		}
	}
	return
}

// hasFilesToSave returns true if any files are marked for saving
func (m SaveV2Model) hasFilesToSave() bool {
	for _, f := range m.files {
		if f.Action == FileActionSave {
			return true
		}
	}
	return false
}

// hasAnyAction returns true if any files have non-skip actions
func (m SaveV2Model) hasAnyAction() bool {
	for _, f := range m.files {
		if f.Action != FileActionIgnoreOnce {
			return true
		}
	}
	return false
}

// Update handles messages
func (m SaveV2Model) Update(msg tea.Msg) (SaveV2Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SaveV2Msg:
		m.revertedCount = msg.RevertedCount
		m.savedCount = msg.SavedCount
		m.ignoredCount = msg.IgnoredCount

		if msg.Err != nil {
			m.state = SaveV2StateError
			m.err = msg.Err
		} else {
			// Check if auto-sync is enabled and we saved files
			cfg, _ := config.Load()
			if cfg.AutoSyncEnabled && git.HasRemote() && m.savedCount > 0 {
				m.state = SaveV2StateAutoSyncing
				m.autoSynced = true
				return m, doAutoSyncV2()
			}
			m.state = SaveV2StateSuccess
		}
		return m, nil

	case AutoSyncV2Msg:
		m.syncErr = msg.Err
		m.state = SaveV2StateSuccess
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case SaveV2StateReview:
			switch {
			case key.Matches(msg, keys.Up):
				if m.cursor > 0 {
					m.cursor--
				}
			case key.Matches(msg, keys.Down):
				if m.cursor < len(m.files)-1 {
					m.cursor++
				}
			case key.Matches(msg, keys.Left):
				// Move to previous column (action)
				m.files[m.cursor].Action = cycleActionReverse(m.files[m.cursor].Action)
			case key.Matches(msg, keys.Right):
				// Move to next column (action)
				m.files[m.cursor].Action = cycleAction(m.files[m.cursor].Action)
			case msg.String() == " " || msg.String() == "tab":
				// Cycle forward through actions
				m.files[m.cursor].Action = cycleAction(m.files[m.cursor].Action)
			case msg.String() == "shift+tab":
				// Cycle backward through actions
				m.files[m.cursor].Action = cycleActionReverse(m.files[m.cursor].Action)
			case msg.String() == "1":
				m.files[m.cursor].Action = FileActionSave
			case msg.String() == "2":
				m.files[m.cursor].Action = FileActionRevert
			case msg.String() == "3":
				m.files[m.cursor].Action = FileActionIgnoreOnce
			case msg.String() == "4":
				m.files[m.cursor].Action = FileActionIgnore
			case key.Matches(msg, keys.Enter):
				if m.hasFilesToSave() {
					m.state = SaveV2StateInput
					m.textInput.Focus()
					return m, textinput.Blink
				} else if m.hasAnyAction() {
					// No files to save but has reverts/ignores - execute directly
					m.state = SaveV2StateExecuting
					return m, doSaveV2("", m.files)
				}
			}

		case SaveV2StateInput:
			switch msg.String() {
			case "enter":
				if m.textInput.Value() != "" {
					m.state = SaveV2StateExecuting
					return m, doSaveV2(m.textInput.Value(), m.files)
				}
			case "esc":
				m.state = SaveV2StateReview
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
func (m SaveV2Model) View() string {
	var s string

	s += RenderTitle("Save Progress") + " " + MutedStyle.Render("(v2)") + "\n\n"

	switch m.state {
	case SaveV2StateNoChanges:
		s += RenderMuted("No changes to save!") + "\n\n"
		s += RenderMuted("Your work is already saved.") + "\n\n"
		s += HelpText("Press any key to go back")

	case SaveV2StateReview:
		s += RenderSubtitle("Choose action for each file:") + "\n\n"
		s += m.renderFileList() + "\n"
		s += m.renderSummary() + "\n"
		s += m.renderLegend() + "\n"
		s += HelpBar([][]string{
			{"↑↓", "navigate"},
			{"←→", "change action"},
			{"1-4", "set action"},
			{"enter", "continue"},
			{"esc", "cancel"},
		})

	case SaveV2StateInput:
		s += m.renderPreview() + "\n"
		s += RenderSubtitle("Describe what you worked on:") + "\n\n"
		s += m.textInput.View() + "\n\n"
		s += HelpBar([][]string{{"enter", "save"}, {"esc", "go back"}})

	case SaveV2StateExecuting:
		s += RenderHighlight("Executing actions...") + "\n"

	case SaveV2StateAutoSyncing:
		s += RenderSuccess("✓ Actions completed!") + "\n\n"
		s += RenderHighlight("Syncing to GitHub...") + "\n"

	case SaveV2StateSuccess:
		s += RenderSuccess("✓ Done!") + "\n\n"
		s += m.renderResults() + "\n"
		if m.autoSynced {
			if m.syncErr != nil {
				s += RenderError("✗ Auto-sync failed: ") + RenderMuted(m.syncErr.Error()) + "\n"
			} else {
				s += RenderSuccess("✓ Synced to GitHub!") + "\n"
			}
			s += "\n"
		}
		s += HelpText("Press any key to continue")

	case SaveV2StateError:
		s += RenderError("✗ Operation failed") + "\n\n"
		if m.err != nil {
			s += RenderMuted(m.err.Error()) + "\n\n"
		}
		s += HelpText("Press any key to go back")
	}

	return BoxStyle.Render(s)
}

// renderFileList renders the interactive file list with action badges
func (m SaveV2Model) renderFileList() string {
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

		// Action badge
		badge := m.renderActionBadge(f.Action)

		// File status icon
		var icon string
		var statusStyle lipgloss.Style
		switch f.Change.Status {
		case "added":
			icon = "+"
			statusStyle = SuccessStyle
		case "deleted":
			icon = "-"
			statusStyle = ErrorStyle
		default:
			icon = "~"
			statusStyle = HighlightStyle
		}

		// File path
		path := f.Change.Path
		if len(path) > 35 {
			path = "..." + path[len(path)-32:]
		}

		// Dim the path if action is not Save
		pathStyle := NormalStyle
		if f.Action != FileActionSave {
			pathStyle = MutedStyle
		}

		s += fmt.Sprintf("%s%s %s %s\n", cursor, badge, statusStyle.Render(icon), pathStyle.Render(path))
	}

	if len(m.files) > maxVisible {
		s += MutedStyle.Render(fmt.Sprintf("\n  ... %d total files", len(m.files)))
	}

	return s
}

// renderActionBadge renders a colored badge for the action
func (m SaveV2Model) renderActionBadge(action FileAction) string {
	var style lipgloss.Style
	var text string

	switch action {
	case FileActionSave:
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(ColorSuccess).
			Padding(0, 1).
			Bold(true)
		text = "SAVE"
	case FileActionRevert:
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(ColorDanger).
			Padding(0, 1).
			Bold(true)
		text = "RVRT"
	case FileActionIgnoreOnce:
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(ColorMuted).
			Padding(0, 1)
		text = "SKIP"
	case FileActionIgnore:
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(ColorHighlight).
			Padding(0, 1).
			Bold(true)
		text = "IGNR"
	}

	return style.Render(text)
}

// renderLegend shows what each action means
func (m SaveV2Model) renderLegend() string {
	legend := lipgloss.NewStyle().Foreground(ColorMuted).Render(
		"1=Save  2=Revert  3=Skip  4=Ignore forever",
	)
	return "\n" + legend
}

// renderSummary shows a summary of planned actions
func (m SaveV2Model) renderSummary() string {
	save, revert, skip, ignore := m.countByAction()

	var parts []string
	if save > 0 {
		parts = append(parts, SuccessStyle.Render(fmt.Sprintf("%d save", save)))
	}
	if revert > 0 {
		parts = append(parts, ErrorStyle.Render(fmt.Sprintf("%d revert", revert)))
	}
	if skip > 0 {
		parts = append(parts, MutedStyle.Render(fmt.Sprintf("%d skip", skip)))
	}
	if ignore > 0 {
		parts = append(parts, HighlightStyle.Render(fmt.Sprintf("%d ignore", ignore)))
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "  "
		}
		result += part
	}

	return result
}

// renderPreview shows what will happen
func (m SaveV2Model) renderPreview() string {
	save, revert, skip, ignore := m.countByAction()

	var s string
	s += RenderSubtitle("Actions:") + "\n"

	if save > 0 {
		s += fmt.Sprintf("  %s %d file(s) will be saved\n", SuccessStyle.Render("●"), save)
	}
	if revert > 0 {
		s += fmt.Sprintf("  %s %d file(s) will be reverted\n", ErrorStyle.Render("●"), revert)
	}
	if skip > 0 {
		s += fmt.Sprintf("  %s %d file(s) will be skipped\n", MutedStyle.Render("●"), skip)
	}
	if ignore > 0 {
		s += fmt.Sprintf("  %s %d file(s) will be added to .gitignore\n", HighlightStyle.Render("●"), ignore)
	}

	return s
}

// renderResults shows what was done
func (m SaveV2Model) renderResults() string {
	var s string

	if m.savedCount > 0 {
		s += fmt.Sprintf("%s Saved %d file(s)\n", SuccessStyle.Render("✓"), m.savedCount)
	}
	if m.revertedCount > 0 {
		s += fmt.Sprintf("%s Reverted %d file(s)\n", SuccessStyle.Render("✓"), m.revertedCount)
	}
	if m.ignoredCount > 0 {
		s += fmt.Sprintf("%s Added %d file(s) to .gitignore\n", SuccessStyle.Render("✓"), m.ignoredCount)
	}

	return s
}

// IsDone returns true if the save flow is complete
func (m SaveV2Model) IsDone() bool {
	return m.state == SaveV2StateSuccess || m.state == SaveV2StateError || m.state == SaveV2StateNoChanges
}

