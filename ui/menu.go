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
	ActionSave MenuAction = iota
	ActionSync
	ActionRestore
	ActionBackups
	ActionExperiments
	ActionKeepExperiment
	ActionAbandonExperiment
	ActionSettings
	ActionQuit
)

// MenuModel is the model for the main menu
type MenuModel struct {
	items      []MenuItem
	cursor     int
	branch     string
	hasChanges bool
	isOnMain   bool
	diff       string
	width      int
	height     int
}

// NewMenuModel creates a new menu model
func NewMenuModel() MenuModel {
	branch, _ := git.CurrentBranch()
	hasChanges := git.HasChanges()
	isOnMain := git.IsOnMain()
	diff := git.GetDiff()

	m := MenuModel{
		cursor:     0,
		branch:     branch,
		hasChanges: hasChanges,
		isOnMain:   isOnMain,
		diff:       diff,
		width:      120, // Default to wide, will be updated by WindowSizeMsg
		height:     30,
	}
	m.items = m.buildMenuItems()
	return m
}

// buildMenuItems creates the menu items based on current state
func (m MenuModel) buildMenuItems() []MenuItem {
	// Titles and descriptions change based on whether we're on an experiment
	saveTitle := "Save"
	saveDesc := "Create a save point of your current work"
	undoTitle := "Undo"
	undoDesc := "Restore your project to an earlier save point"
	if !m.isOnMain {
		saveTitle = "Save (experiment)"
		saveDesc = "Create a save point of your current work for this experiment"
		undoTitle = "Undo (experiment)"
		undoDesc = "Restore your experiment to an earlier save point"
	}

	items := []MenuItem{
		{
			Title:       saveTitle,
			Description: saveDesc,
			Action:      ActionSave,
		},
		{
			Title:       undoTitle,
			Description: undoDesc,
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
		m.items = m.buildMenuItems()
		// Schedule next tick
		return m, tickCmd()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
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

	// Title
	leftContent += RenderTitle("What would you like to do?") + "\n\n"

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

		if m.cursor == i {
			cursor = MenuCursorStyle.Render("> ")
			style = MenuItemSelectedStyle
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

	// Help bar - rendered separately and positioned at bottom center
	helpBar := HelpBar([][]string{
		{"↑↓", "navigate"},
		{"enter", "select"},
		{"q", "quit"},
	})

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

	// === RIGHT PANEL: Diff ===
	var rightContent string
	rightContent += RenderSubtitle("Current Changes") + "\n\n"

	// Format and truncate diff for display
	diffLines := strings.Split(m.diff, "\n")
	maxLines := panelHeight - 10
	if maxLines < 10 {
		maxLines = 10
	}
	if maxLines > 30 {
		maxLines = 30
	}

	lineCount := 0
	for _, line := range diffLines {
		if lineCount >= maxLines {
			remaining := len(diffLines) - lineCount
			if remaining > 0 {
				rightContent += MutedStyle.Render(fmt.Sprintf("... and %d more lines", remaining)) + "\n"
			}
			break
		}
		// Skip empty lines at start
		if lineCount == 0 && line == "" {
			continue
		}
		// Color-code diff lines
		displayLine := truncateLine(line, rightWidth-6)
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			rightContent += SuccessStyle.Render(displayLine) + "\n"
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			rightContent += ErrorStyle.Render(displayLine) + "\n"
		} else if strings.HasPrefix(line, "@@") {
			rightContent += HighlightStyle.Render(displayLine) + "\n"
		} else if strings.HasPrefix(line, " M ") || strings.HasPrefix(line, "?? ") || strings.HasPrefix(line, " A ") || strings.HasPrefix(line, " D ") || strings.HasPrefix(line, "M ") || strings.HasPrefix(line, "A ") {
			// Git status short format
			if strings.HasPrefix(line, "?? ") || strings.HasPrefix(line, " A ") || strings.HasPrefix(line, "A ") {
				rightContent += SuccessStyle.Render(displayLine) + "\n"
			} else if strings.HasPrefix(line, " D ") || strings.HasPrefix(line, "D ") {
				rightContent += ErrorStyle.Render(displayLine) + "\n"
			} else {
				rightContent += HighlightStyle.Render(displayLine) + "\n"
			}
		} else {
			rightContent += MutedStyle.Render(displayLine) + "\n"
		}
		lineCount++
	}

	if m.diff == "" || m.diff == "No changes" {
		rightContent += MutedStyle.Render("No uncommitted changes") + "\n"
	}

	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Height(panelHeight-6). // Account for border and bottom help bar
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
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

// SelectedAction returns the currently selected action
func (m MenuModel) SelectedAction() MenuAction {
	return m.items[m.cursor].Action
}

// RefreshStatus updates the branch and changes status and returns a tick command
func (m *MenuModel) RefreshStatus() tea.Cmd {
	m.branch, _ = git.CurrentBranch()
	m.hasChanges = git.HasChanges()
	m.isOnMain = git.IsOnMain()
	m.diff = git.GetDiff()
	m.items = m.buildMenuItems()
	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
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
	Enter key.Binding
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
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
