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
	SaveStateExecuting
	SaveStateAutoSyncing
	SaveStateSuccess
	SaveStateError
	SaveStateNoChanges
)

// SaveFileItem represents a file with its action
type SaveFileItem struct {
	Change git.FileChange
	Action FileAction
}

// SaveModel is the model for the save flow
type SaveModel struct {
	textInput     textinput.Model
	state         SaveState
	err           error
	files         []SaveFileItem
	cursor        int
	focusOnFiles  bool // true = file list focused, false = text input focused
	synced        bool
	syncErr       error
	commitHash    string
	savedCount    int
	revertedCount int
	ignoredCount  int
	skippedCount  int
	width         int
	height        int
}

// NewSaveModel creates a new save model
func NewSaveModel() SaveModel {
	ti := textinput.New()
	ti.Placeholder = "What did you work on?"
	ti.CharLimit = 100
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorAccent)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorText)
	ti.Focus()

	changes, _ := git.GetChangeSummary()

	state := SaveStateReview
	if len(changes) == 0 {
		state = SaveStateNoChanges
	}

	// Convert to SaveFileItem with Save as default
	files := make([]SaveFileItem, len(changes))
	for i, c := range changes {
		files[i] = SaveFileItem{
			Change: c,
			Action: FileActionSave,
		}
	}

	return SaveModel{
		textInput:    ti,
		state:        state,
		files:        files,
		cursor:       0,
		focusOnFiles: false, // Start with text input focused
	}
}

// Init initializes the model
func (m SaveModel) Init() tea.Cmd {
	return textinput.Blink
}

// SaveMsg is sent when save completes
type SaveMsg struct {
	Err           error
	Hash          string
	SavedCount    int
	RevertedCount int
	IgnoredCount  int
	SkippedCount  int
}

// SaveSyncMsg is sent when sync completes
type SaveSyncMsg struct {
	Err error
}

// doSave performs the save operation
func doSave(message string, files []SaveFileItem) tea.Cmd {
	return func() tea.Msg {
		var toSave []string
		var toRevert []string
		var toIgnore []string
		skipped := 0

		for _, f := range files {
			switch f.Action {
			case FileActionSave:
				toSave = append(toSave, f.Change.Path)
			case FileActionRevert:
				toRevert = append(toRevert, f.Change.Path)
			case FileActionIgnore:
				toIgnore = append(toIgnore, f.Change.Path)
			case FileActionIgnoreOnce:
				skipped++
			}
		}

		result := SaveMsg{
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

// doSaveSync performs the sync operation
func doSaveSync() tea.Cmd {
	return func() tea.Msg {
		err := git.Push()
		return SaveSyncMsg{Err: err}
	}
}

// countByAction returns counts for each action type
func (m SaveModel) countByAction() (save, revert, skip, ignore int) {
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
func (m SaveModel) hasFilesToSave() bool {
	for _, f := range m.files {
		if f.Action == FileActionSave {
			return true
		}
	}
	return false
}

// cycleAction moves to the next action state
func (m SaveModel) cycleAction(current FileAction) FileAction {
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

// Update handles messages
func (m SaveModel) Update(msg tea.Msg) (SaveModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case SaveMsg:
		if msg.Err != nil {
			m.state = SaveStateError
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
			m.state = SaveStateAutoSyncing
			m.synced = true
			return m, doSaveSync()
		}

		m.state = SaveStateSuccess
		return m, nil

	case SaveSyncMsg:
		m.syncErr = msg.Err
		m.state = SaveStateSuccess
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case SaveStateReview:
			// Left/Right arrows switch focus between panels
			if key.Matches(msg, keys.Right) && !m.focusOnFiles {
				m.focusOnFiles = true
				m.textInput.Blur()
				return m, nil
			}
			if key.Matches(msg, keys.Left) && m.focusOnFiles {
				m.focusOnFiles = false
				m.textInput.Focus()
				return m, textinput.Blink
			}

			// Enter executes save from either focus
			if key.Matches(msg, keys.Enter) {
				if m.textInput.Value() != "" {
					m.state = SaveStateExecuting
					return m, doSave(m.textInput.Value(), m.files)
				}
				return m, nil
			}

			if m.focusOnFiles {
				// File list is focused - handle file navigation and actions
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
					// Cycle file action
					m.files[m.cursor].Action = m.cycleAction(m.files[m.cursor].Action)
				case msg.String() == "1":
					m.files[m.cursor].Action = FileActionSave
				case msg.String() == "2":
					m.files[m.cursor].Action = FileActionRevert
				case msg.String() == "3":
					m.files[m.cursor].Action = FileActionIgnoreOnce
				case msg.String() == "4":
					m.files[m.cursor].Action = FileActionIgnore
				}
			} else {
				// Text input is focused - pass keys to text input
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
	switch m.state {
	case SaveStateNoChanges:
		s := RenderTitle("Save") + "\n\n"
		s += RenderMuted("No changes to save!") + "\n\n"
		s += RenderMuted("Your work is already saved.") + "\n\n"
		s += HelpText("Press any key to go back")
		return BoxStyle.Render(s)

	case SaveStateReview:
		return m.renderTwoPanelView()

	case SaveStateExecuting:
		s := RenderTitle("Save") + "\n\n"
		s += RenderHighlight("⟳ Processing changes...") + "\n"
		return BoxStyle.Render(s)

	case SaveStateAutoSyncing:
		s := RenderTitle("Save") + "\n\n"
		s += RenderSuccess("✓ Done!") + "\n\n"
		s += RenderHighlight("⟳ Syncing to GitHub...") + "\n"
		return BoxStyle.Render(s)

	case SaveStateSuccess:
		s := RenderTitle("Save") + "\n\n"
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
		return BoxStyle.Render(s)

	case SaveStateError:
		s := RenderTitle("Save") + "\n\n"
		s += RenderError("✗ Save failed") + "\n\n"
		if m.err != nil {
			s += RenderMuted(m.err.Error()) + "\n\n"
		}
		s += HelpText("Press any key to go back")
		return BoxStyle.Render(s)
	}

	return ""
}

// renderTwoPanelView renders the two-panel save review layout
func (m SaveModel) renderTwoPanelView() string {
	width := m.width
	if width < 80 {
		width = 100
	}

	// Calculate panel widths (35% left for message, 65% right for files)
	leftWidth := width*35/100 - 2
	rightWidth := width*65/100 - 2

	// Build panel contents
	leftContent := m.renderLeftPanel(leftWidth)
	rightContent := m.renderRightPanel(rightWidth)

	// Style the panels
	leftBorderColor := ColorMuted
	rightBorderColor := ColorMuted
	if !m.focusOnFiles {
		leftBorderColor = ColorAccent
	} else {
		rightBorderColor = ColorAccent
	}

	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(leftBorderColor).
		Render(leftContent)

	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(rightBorderColor).
		Render(rightContent)

	// Join panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Build full view
	var s string
	s += RenderTitle("Save") + "\n\n"
	s += panels + "\n\n"

	// Help bar at bottom
	if m.focusOnFiles {
		s += HelpBar([][]string{
			{"←", "message"},
			{"↑↓", "navigate"},
			{"space", "cycle"},
			{"1-4", "set action"},
			{"enter", "save"},
			{"esc", "cancel"},
		})
	} else {
		s += HelpBar([][]string{
			{"→", "files"},
			{"enter", "save"},
			{"esc", "cancel"},
		})
	}

	return s
}

// renderLeftPanel renders the instructions and save message input
func (m SaveModel) renderLeftPanel(width int) string {
	var s string

	// Title
	titleStyle := MutedStyle
	if !m.focusOnFiles {
		titleStyle = HighlightStyle
	}
	s += titleStyle.Render("Save Message") + "\n\n"

	// Text input
	s += m.textInput.View() + "\n\n"

	// Summary of actions
	s += m.renderSummary()

	return s
}

// renderRightPanel renders the file list with actions
func (m SaveModel) renderRightPanel(width int) string {
	var s string

	// Title
	titleStyle := MutedStyle
	if m.focusOnFiles {
		titleStyle = HighlightStyle
	}
	s += titleStyle.Render("Files") + "\n\n"

	// File list
	maxVisible := 10
	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	for i := start; i < len(m.files) && i < start+maxVisible; i++ {
		f := m.files[i]

		// Cursor
		cursor := "  "
		if m.focusOnFiles && i == m.cursor {
			cursor = HighlightStyle.Render("▸ ")
		}

		// Action badge
		badge := m.renderActionBadge(f.Action)

		// Filename (truncate if needed)
		name := f.Change.Path
		maxNameLen := width - 15
		if maxNameLen < 10 {
			maxNameLen = 10
		}
		if len(name) > maxNameLen {
			name = "..." + name[len(name)-maxNameLen+3:]
		}

		// Status indicator
		status := ""
		switch f.Change.Status {
		case "added":
			status = SuccessStyle.Render("+")
		case "deleted":
			status = ErrorStyle.Render("-")
		default:
			status = HighlightStyle.Render("~")
		}

		// Dim filename if not saving
		nameStyle := NormalStyle
		if f.Action != FileActionSave {
			nameStyle = MutedStyle
		}

		s += fmt.Sprintf("%s%s %s %s\n", cursor, badge, status, nameStyle.Render(name))
	}

	if len(m.files) > maxVisible {
		s += MutedStyle.Render(fmt.Sprintf("\n  ... %d total files", len(m.files)))
	}

	// Legend (only when focused)
	if m.focusOnFiles {
		s += "\n\n" + MutedStyle.Render("1=Save 2=Revert 3=Skip 4=Ignore")
	}

	return s
}


// renderActionBadge renders a colored badge for the action
func (m SaveModel) renderActionBadge(action FileAction) string {
	var style lipgloss.Style
	var text string

	switch action {
	case FileActionSave:
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(ColorSuccess).
			Bold(true)
		text = "SAVE"
	case FileActionRevert:
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(ColorDanger).
			Bold(true)
		text = "RVRT"
	case FileActionIgnoreOnce:
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(ColorMuted)
		text = "SKIP"
	case FileActionIgnore:
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(ColorHighlight).
			Bold(true)
		text = "IGNR"
	default:
		style = lipgloss.NewStyle().Background(ColorMuted)
		text = "????"
	}

	return style.Render(text)
}

// renderSummary shows a summary of planned actions
func (m SaveModel) renderSummary() string {
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

	return result + "\n"
}

// IsDone returns true if the save flow is complete
func (m SaveModel) IsDone() bool {
	return m.state == SaveStateSuccess || m.state == SaveStateError || m.state == SaveStateNoChanges
}

