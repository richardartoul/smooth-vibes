package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vc/config"
	"vc/git"
)

// tickMsg is sent periodically to refresh the menu
type tickMsg time.Time

// refreshInterval is how often the menu refreshes
const refreshInterval = 2 * time.Second

// MenuItem represents a menu option
type MenuItem struct {
	Title       string
	Description string
	Action      MenuAction
}

// MenuAction represents the action to take when a menu item is selected
type MenuAction int

const (
	ActionQuicksave MenuAction = iota
	ActionSync
	ActionRestore
	ActionBackups
	ActionExperiments
	ActionKeepExperiment
	ActionAbandonExperiment
	ActionSettings
	ActionQuit
)

// FileAction represents what to do with a changed file
type FileAction int

const (
	FileActionSave       FileAction = iota // Stage and commit the file
	FileActionRevert                       // Discard changes (restore to HEAD)
	FileActionIgnoreOnce                   // Skip this time, keep local changes
	FileActionIgnore                       // Add to .gitignore
)

// MenuModel is the model for the main menu
type MenuModel struct {
	items            []MenuItem
	cursor           int
	branch           string
	hasChanges       bool
	isOnMain         bool
	diff             string
	width            int
	height           int
	changedFiles     []git.FileChange
	focusRight       bool
	fileCursor       int
	expandedFiles    map[string]bool
	fileDiffs        map[string]string
	diffScrollOffset map[string]int        // Scroll offset per file
	fileActions      map[string]FileAction // Action for each file (Save/Revert/Skip/Ignore)
}

// NewMenuModel creates a new menu model
func NewMenuModel() MenuModel {
	branch, _ := git.CurrentBranch()
	hasChanges := git.HasChanges()
	isOnMain := git.IsOnMain()
	diff := git.GetDiff()
	changedFiles, _ := git.GetChangeSummary()

	// Initialize file actions - all files default to Save
	fileActions := make(map[string]FileAction)
	for _, f := range changedFiles {
		fileActions[f.Path] = FileActionSave
	}

	m := MenuModel{
		cursor:           0,
		branch:           branch,
		hasChanges:       hasChanges,
		isOnMain:         isOnMain,
		diff:             diff,
		width:            120, // Default to wide, will be updated by WindowSizeMsg
		height:           30,
		changedFiles:     changedFiles,
		focusRight:       false,
		fileCursor:       0,
		expandedFiles:    make(map[string]bool),
		fileDiffs:        make(map[string]string),
		diffScrollOffset: make(map[string]int),
		fileActions:      fileActions,
	}
	m.items = m.buildMenuItems()
	return m
}

// buildMenuItems creates the menu items based on current state
func (m MenuModel) buildMenuItems() []MenuItem {
	// Titles and descriptions change based on whether we're on an experiment
	revertTitle := "Revert"
	revertDesc := "Restore your project to an earlier save point"
	if !m.isOnMain {
		revertTitle = "Revert (experiment)"
		revertDesc = "Restore your experiment to an earlier save point"
	}

	items := []MenuItem{
		{
			Title:       "Save",
			Description: "Save your work (use → to configure per-file actions)",
			Action:      ActionQuicksave,
		},
		{
			Title:       revertTitle,
			Description: revertDesc,
			Action:      ActionRestore,
		},
	}

	// Add experiment-specific actions when on an experiment branch
	if !m.isOnMain {
		items = append(items,
			MenuItem{
				Title:       "Keep this experiment",
				Description: "Merge this experiment into your main work",
				Action:      ActionKeepExperiment,
			},
			MenuItem{
				Title:       "Abandon this experiment",
				Description: "Discard this experiment and go back to main",
				Action:      ActionAbandonExperiment,
			},
		)
	}

	// This is a comment to show troy that it works.
	items = append(items,
		MenuItem{
			Title:       "Restore backup",
			Description: "Restore from automatic backups created during reverts",
			Action:      ActionBackups,
		},
	)

	// Only show experiments if enabled in config
	cfg, _ := config.Load()
	if cfg.ExperimentsEnabled {
		items = append(items,
			MenuItem{
				Title:       "Experiments (advanced)",
				Description: "Try new ideas without breaking your main work",
				Action:      ActionExperiments,
			},
		)
	}

	items = append(items,
		MenuItem{
			Title:       "Sync to GitHub",
			Description: "Upload your saves to the cloud",
			Action:      ActionSync,
		},
		MenuItem{
			Title:       "Settings",
			Description: "Configure auto-sync and backup options",
			Action:      ActionSettings,
		},
		MenuItem{
			Title:       "Quit",
			Description: "Exit the application",
			Action:      ActionQuit,
		},
	)

	return items
}

// Init initializes the menu model
func (m MenuModel) Init() tea.Cmd {
	return tickCmd()
}

// tickCmd returns a command that sends a tick after the refresh interval
func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles messages for the menu model
func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		// Refresh data from git
		m.branch, _ = git.CurrentBranch()
		m.hasChanges = git.HasChanges()
		m.isOnMain = git.IsOnMain()
		m.diff = git.GetDiff()
		m.changedFiles, _ = git.GetChangeSummary()
		m.items = m.buildMenuItems()
		// Reset file cursor if out of bounds
		if m.fileCursor >= len(m.changedFiles) {
			m.fileCursor = max(0, len(m.changedFiles)-1)
		}
		// Update file actions - keep existing actions, add new files with Save
		for _, f := range m.changedFiles {
			if _, exists := m.fileActions[f.Path]; !exists {
				m.fileActions[f.Path] = FileActionSave
			}
		}
		// Schedule next tick
		return m, tickCmd()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		// Check if we should show the diff panel (determines if right navigation is available)
		showDiffPanel := m.width >= 90 && len(m.changedFiles) > 0

		switch {
		case key.Matches(msg, keys.Left):
			if m.focusRight {
				m.focusRight = false
			}
		case key.Matches(msg, keys.Right):
			if showDiffPanel && !m.focusRight {
				m.focusRight = true
			}
		case key.Matches(msg, keys.Up):
			if m.focusRight {
				// Check if current file is expanded - if so, scroll the diff
				if len(m.changedFiles) > 0 {
					filePath := m.changedFiles[m.fileCursor].Path
					if m.expandedFiles[filePath] {
						// Scroll up in diff
						if m.diffScrollOffset[filePath] > 0 {
							m.diffScrollOffset[filePath]--
						}
					} else {
						// Move to previous file
						if m.fileCursor > 0 {
							m.fileCursor--
						}
					}
				}
			} else {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case key.Matches(msg, keys.Down):
			if m.focusRight {
				// Check if current file is expanded - if so, scroll the diff
				if len(m.changedFiles) > 0 {
					filePath := m.changedFiles[m.fileCursor].Path
					if m.expandedFiles[filePath] {
						// Scroll down in diff
						diff := m.fileDiffs[filePath]
						diffLines := strings.Split(diff, "\n")
						maxScroll := len(diffLines) - m.getMaxDiffLines()
						if maxScroll < 0 {
							maxScroll = 0
						}
						if m.diffScrollOffset[filePath] < maxScroll {
							m.diffScrollOffset[filePath]++
						}
					} else {
						// Move to next file
						if m.fileCursor < len(m.changedFiles)-1 {
							m.fileCursor++
						}
					}
				}
			} else {
				if m.cursor < len(m.items)-1 {
					m.cursor++
				}
			}
		case key.Matches(msg, keys.Enter):
			if m.focusRight && len(m.changedFiles) > 0 {
				// Toggle diff for the selected file
				filePath := m.changedFiles[m.fileCursor].Path
				if m.expandedFiles[filePath] {
					m.expandedFiles[filePath] = false
				} else {
					// Load diff if not cached
					if _, ok := m.fileDiffs[filePath]; !ok {
						m.fileDiffs[filePath] = git.GetFileDiff(filePath)
					}
					m.expandedFiles[filePath] = true
				}
			}
		case key.Matches(msg, keys.Space):
			if m.focusRight && len(m.changedFiles) > 0 {
				// Cycle file action
				filePath := m.changedFiles[m.fileCursor].Path
				current := m.fileActions[filePath]
				m.fileActions[filePath] = cycleFileAction(current)
			}
		case msg.String() == "1":
			if m.focusRight && len(m.changedFiles) > 0 {
				filePath := m.changedFiles[m.fileCursor].Path
				m.fileActions[filePath] = FileActionSave
			}
		case msg.String() == "2":
			if m.focusRight && len(m.changedFiles) > 0 {
				filePath := m.changedFiles[m.fileCursor].Path
				m.fileActions[filePath] = FileActionRevert
			}
		case msg.String() == "3":
			if m.focusRight && len(m.changedFiles) > 0 {
				filePath := m.changedFiles[m.fileCursor].Path
				m.fileActions[filePath] = FileActionIgnoreOnce
			}
		case msg.String() == "4":
			if m.focusRight && len(m.changedFiles) > 0 {
				filePath := m.changedFiles[m.fileCursor].Path
				m.fileActions[filePath] = FileActionIgnore
			}
		}
	}
	return m, nil
}

// View renders the menu
func (m MenuModel) View() string {
	// Determine if we should show split view (need at least 90 chars wide)
	showDiffPanel := m.width >= 90

	// === LEFT PANEL: Menu ===
	var leftContent string

	// Banner (skip if narrow or short terminal)
	if m.width >= 60 && m.height >= 30 {
		leftContent += Banner() + "\n\n"
	} else if m.height >= 20 {
		leftContent += TitleStyle.Render("SMOOTH") + "\n\n"
	}
	// Skip title entirely if very short

	// Status bar
	branchDisplay := m.branch
	if !m.isOnMain {
		branchDisplay = HighlightStyle.Render(m.branch) + " " + MutedStyle.Render("(experiment)")
	}
	statusText := fmt.Sprintf("Branch: %s", branchDisplay)
	if m.hasChanges {
		statusText += " " + SuccessStyle.Render("(unsaved changes)")
	}
	leftContent += HeaderBoxStyle.Render(statusText) + "\n\n"

	// Title - show focus indicator
	menuTitle := "What would you like to do?"
	if showDiffPanel && !m.focusRight {
		menuTitle = "▸ " + menuTitle
	}
	leftContent += RenderTitle(menuTitle) + "\n\n"

	// Menu items
	// Hide descriptions if narrow OR short terminal
	showDescriptions := m.width >= 60 && m.height >= 25

	// Calculate visible range for scrolling
	maxVisible := 12
	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	for i := start; i < len(m.items) && i < start+maxVisible; i++ {
		item := m.items[i]
		cursor := "  "
		style := MenuItemStyle

		if m.cursor == i && !m.focusRight {
			cursor = MenuCursorStyle.Render("> ")
			style = MenuItemSelectedStyle
		} else if m.cursor == i && m.focusRight {
			// Show dimmed selection when right panel is focused
			cursor = MutedStyle.Render("> ")
		}

		title := style.Render(item.Title)
		leftContent += cursor + title + "\n"
	}

	if len(m.items) > maxVisible {
		leftContent += MutedStyle.Render(fmt.Sprintf("  ... %d total items\n", len(m.items)))
	}

	// Show selected item's description in a fixed location below the menu
	if showDescriptions {
		selectedDesc := m.items[m.cursor].Description
		leftContent += "\n" + MutedStyle.Render("  "+selectedDesc) + "\n"
	}

	// Help bar - changes based on focus
	var helpBar string
	// Check if we're viewing an expanded diff
	viewingExpandedDiff := false
	if m.focusRight && len(m.changedFiles) > 0 {
		filePath := m.changedFiles[m.fileCursor].Path
		viewingExpandedDiff = m.expandedFiles[filePath]
	}

	if m.focusRight && viewingExpandedDiff {
		helpBar = HelpBar([][]string{
			{"↑↓", "scroll"},
			{"⏎", "collapse"},
			{"space", "action"},
			{"1-4", "set"},
			{"←", "menu"},
		})
	} else if m.focusRight {
		helpBar = HelpBar([][]string{
			{"↑↓", "navigate"},
			{"⏎", "diff"},
			{"space", "action"},
			{"1-4", "set"},
			{"←", "menu"},
		})
	} else if showDiffPanel && len(m.changedFiles) > 0 {
		helpBar = HelpBar([][]string{
			{"↑↓", "navigate"},
			{"enter", "select"},
			{"→", "changes"},
			{"q", "quit"},
		})
	} else {
		helpBar = HelpBar([][]string{
			{"↑↓", "navigate"},
			{"enter", "select"},
			{"q", "quit"},
		})
	}

	// If no split view, just return the menu
	if !showDiffPanel {
		content := lipgloss.NewStyle().
			Padding(1, 2).
			Render(leftContent)
		// Place content at top, help bar at bottom center
		return placeWithBottomHelp(content, helpBar, m.width, m.height)
	}

	// Calculate widths for split layout
	leftWidth := m.width / 2
	if leftWidth < 50 {
		leftWidth = 50
	}
	rightWidth := m.width - leftWidth - 4

	// Use available height minus some margin
	panelHeight := m.height - 2
	if panelHeight < 10 {
		panelHeight = 10
	}

	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Height(panelHeight).
		Padding(1, 2).
		Render(leftContent)

	// === RIGHT PANEL: Changed Files ===
	var rightContent string

	// Title with focus indicator
	changesTitle := "Current Changes"
	if m.focusRight {
		changesTitle = "▸ " + changesTitle
	}
	rightContent += RenderSubtitle(changesTitle) + "\n\n"

	if len(m.changedFiles) == 0 {
		rightContent += MutedStyle.Render("No uncommitted changes") + "\n"
	} else {
		// Calculate available lines for files (reserve space for scroll indicators)
		maxVisibleFiles := panelHeight - 14
		if maxVisibleFiles < 3 {
			maxVisibleFiles = 3
		}

		totalFiles := len(m.changedFiles)

		// Calculate visible window around cursor
		startFileIdx := 0
		if m.fileCursor >= maxVisibleFiles {
			startFileIdx = m.fileCursor - maxVisibleFiles + 1
		}
		// Adjust if we're near the end
		if startFileIdx+maxVisibleFiles > totalFiles {
			startFileIdx = totalFiles - maxVisibleFiles
			if startFileIdx < 0 {
				startFileIdx = 0
			}
		}

		endFileIdx := startFileIdx + maxVisibleFiles
		if endFileIdx > totalFiles {
			endFileIdx = totalFiles
		}

		// Show scroll indicator if there are files above
		if startFileIdx > 0 {
			rightContent += MutedStyle.Render(fmt.Sprintf("  ▲ %d more files above", startFileIdx)) + "\n"
		}

		lineCount := 0
		maxFileLines := panelHeight - 12
		if maxFileLines < 5 {
			maxFileLines = 5
		}

		for i := startFileIdx; i < endFileIdx; i++ {
			file := m.changedFiles[i]

			if lineCount >= maxFileLines {
				break
			}

			// Status icon
			var statusIcon string
			switch file.Status {
			case "added":
				statusIcon = SuccessStyle.Render("+")
			case "deleted":
				statusIcon = ErrorStyle.Render("-")
			case "modified":
				statusIcon = HighlightStyle.Render("~")
			case "renamed":
				statusIcon = HighlightStyle.Render("→")
			default:
				statusIcon = MutedStyle.Render("?")
			}

			// Cursor and selection
			cursor := "  "
			fileStyle := MutedStyle
			if i == m.fileCursor && m.focusRight {
				cursor = MenuCursorStyle.Render("> ")
				fileStyle = lipgloss.NewStyle().Bold(true)
			} else if i == m.fileCursor && !m.focusRight {
				cursor = MutedStyle.Render("> ")
			}

			// Action badge
			action := m.fileActions[file.Path]
			actionBadge := m.renderFileActionBadge(action)

			// Expand/collapse indicator
			expandIcon := "▶"
			if m.expandedFiles[file.Path] {
				expandIcon = "▼"
			}

			// Truncate filename if needed (account for badge width)
			displayPath := truncateLine(file.Path, rightWidth-20)
			rightContent += cursor + actionBadge + " " + MutedStyle.Render(expandIcon) + " " + statusIcon + " " + fileStyle.Render(displayPath) + "\n"
			lineCount++

			// Show diff if expanded
			if m.expandedFiles[file.Path] {
				diff := m.fileDiffs[file.Path]
				diffLines := strings.Split(diff, "\n")

				// Filter out leading empty lines
				startIdx := 0
				for startIdx < len(diffLines) && diffLines[startIdx] == "" {
					startIdx++
				}
				diffLines = diffLines[startIdx:]

				maxDiffLines := m.getMaxDiffLines()
				scrollOffset := m.diffScrollOffset[file.Path]
				totalLines := len(diffLines)

				// Show scroll indicator if there's content above
				if scrollOffset > 0 {
					rightContent += MutedStyle.Render("    ▲ scroll up for more") + "\n"
					lineCount++
				}

				// Calculate visible window
				endIdx := scrollOffset + maxDiffLines
				if endIdx > totalLines {
					endIdx = totalLines
				}

				visibleLines := diffLines[scrollOffset:endIdx]
				for _, line := range visibleLines {
					if lineCount >= maxFileLines {
						break
					}
					// Color-code diff lines
					displayLine := truncateLine(line, rightWidth-10)
					prefix := "    "
					if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
						rightContent += prefix + SuccessStyle.Render(displayLine) + "\n"
					} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
						rightContent += prefix + ErrorStyle.Render(displayLine) + "\n"
					} else if strings.HasPrefix(line, "@@") {
						rightContent += prefix + HighlightStyle.Render(displayLine) + "\n"
					} else if strings.HasPrefix(line, "new file:") || strings.HasPrefix(line, "---") {
						rightContent += prefix + MutedStyle.Render(displayLine) + "\n"
					} else {
						rightContent += prefix + MutedStyle.Render(displayLine) + "\n"
					}
					lineCount++
				}

				// Show scroll indicator if there's content below
				if endIdx < totalLines {
					remaining := totalLines - endIdx
					rightContent += MutedStyle.Render(fmt.Sprintf("    ▼ %d more lines below", remaining)) + "\n"
					lineCount++
				}
			}
		}

		// Show scroll indicator if there are files below
		if endFileIdx < totalFiles {
			remaining := totalFiles - endFileIdx
			rightContent += MutedStyle.Render(fmt.Sprintf("  ▼ %d more files below", remaining)) + "\n"
		}
	}

	// Border color changes based on focus
	borderColor := ColorSecondary
	if m.focusRight {
		borderColor = ColorPrimary
	}

	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Height(panelHeight-6). // Account for border and bottom help bar
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Render(rightContent)

	// Join panels horizontally
	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Place content at top, help bar at bottom center
	return placeWithBottomHelp(combined, helpBar, m.width, m.height)
}

// placeWithBottomHelp renders content with a help bar fixed at the bottom center
func placeWithBottomHelp(content, helpBar string, width, height int) string {
	// Center the help bar horizontally
	centeredHelp := lipgloss.PlaceHorizontal(width, lipgloss.Center, helpBar)

	// Calculate content height (leave room for help bar)
	contentHeight := height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Place content in the available space
	placedContent := lipgloss.Place(width, contentHeight, lipgloss.Left, lipgloss.Top, content)

	// Join content and help bar vertically
	return lipgloss.JoinVertical(lipgloss.Left, placedContent, centeredHelp)
}

// truncateLine truncates a line to fit within maxWidth
func truncateLine(line string, maxWidth int) string {
	if maxWidth < 10 {
		maxWidth = 10
	}
	if len(line) > maxWidth {
		return line[:maxWidth-3] + "..."
	}
	return line
}

// getMaxDiffLines returns the max number of diff lines that can be displayed
func (m MenuModel) getMaxDiffLines() int {
	panelHeight := m.height - 2
	if panelHeight < 10 {
		panelHeight = 10
	}
	// Available space minus header, file list overhead, and some padding
	maxLines := panelHeight - 10
	if maxLines < 5 {
		maxLines = 5
	}
	return maxLines
}

// SelectedAction returns the currently selected action
func (m MenuModel) SelectedAction() MenuAction {
	return m.items[m.cursor].Action
}

// IsFocusedOnChanges returns true if the right panel (changes) is focused
func (m MenuModel) IsFocusedOnChanges() bool {
	return m.focusRight
}

// GetFileActions returns the current file actions map
func (m MenuModel) GetFileActions() map[string]FileAction {
	return m.fileActions
}

// cycleFileAction cycles through file actions
func cycleFileAction(current FileAction) FileAction {
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

// renderFileActionBadge renders a compact badge for the file action
func (m MenuModel) renderFileActionBadge(action FileAction) string {
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

// RefreshStatus updates the branch and changes status and returns a tick command
func (m *MenuModel) RefreshStatus() tea.Cmd {
	m.branch, _ = git.CurrentBranch()
	m.hasChanges = git.HasChanges()
	m.isOnMain = git.IsOnMain()
	m.diff = git.GetDiff()
	m.changedFiles, _ = git.GetChangeSummary()
	m.items = m.buildMenuItems()
	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	// Reset file cursor if out of bounds
	if m.fileCursor >= len(m.changedFiles) {
		m.fileCursor = max(0, len(m.changedFiles)-1)
	}
	// Clear cached diffs and expanded state on refresh
	m.expandedFiles = make(map[string]bool)
	m.fileDiffs = make(map[string]string)
	m.diffScrollOffset = make(map[string]int)
	// Reset file actions - new files get Save action
	m.fileActions = make(map[string]FileAction)
	for _, f := range m.changedFiles {
		m.fileActions[f.Path] = FileActionSave
	}
	// Return tick command to restart periodic refresh
	return tickCmd()
}

// SetSize updates the terminal dimensions
func (m *MenuModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Key bindings
type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Enter key.Binding
	Space key.Binding
	Quit  key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "right"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle diff"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
