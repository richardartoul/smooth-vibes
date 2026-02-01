package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vc/git"
)

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
	ActionExperiments
	ActionKeepExperiment
	ActionAbandonExperiment
	ActionQuit
)

// MenuModel is the model for the main menu
type MenuModel struct {
	items      []MenuItem
	cursor     int
	branch     string
	hasChanges bool
	isOnMain   bool
}

// NewMenuModel creates a new menu model
func NewMenuModel() MenuModel {
	branch, _ := git.CurrentBranch()
	hasChanges := git.HasChanges()
	isOnMain := git.IsOnMain()

	m := MenuModel{
		cursor:     0,
		branch:     branch,
		hasChanges: hasChanges,
		isOnMain:   isOnMain,
	}
	m.items = m.buildMenuItems()
	return m
}

// buildMenuItems creates the menu items based on current state
func (m MenuModel) buildMenuItems() []MenuItem {
	// Titles and descriptions change based on whether we're on an experiment
	saveTitle := "Save my current progress"
	saveDesc := "Create a save point of your current work"
	restoreTitle := "Go back to a previous state"
	restoreDesc := "Restore your project to an earlier save point"
	if !m.isOnMain {
		saveTitle = "Save my current progress (experiment)"
		saveDesc = "Create a save point of your current work for this experiment"
		restoreTitle = "Go back to a previous state (experiment)"
		restoreDesc = "Restore your experiment to an earlier save point"
	}

	items := []MenuItem{
		{
			Title:       saveTitle,
			Description: saveDesc,
			Action:      ActionSave,
		},
		{
			Title:       restoreTitle,
			Description: restoreDesc,
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

	items = append(items,
		MenuItem{
			Title:       "Experiments",
			Description: "Try new ideas without breaking your main work",
			Action:      ActionExperiments,
		},
		MenuItem{
			Title:       "Sync to GitHub",
			Description: "Upload your saves to the cloud",
			Action:      ActionSync,
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
	return nil
}

// Update handles messages for the menu model
func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
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
	var s string

	// Banner
	s += Banner() + "\n\n"

	// Status bar
	branchDisplay := m.branch
	if !m.isOnMain {
		branchDisplay = HighlightStyle.Render(m.branch) + " " + MutedStyle.Render("(experiment)")
	}
	statusText := fmt.Sprintf("Branch: %s", branchDisplay)
	if m.hasChanges {
		statusText += " " + SuccessStyle.Render("(unsaved changes)")
	}
	s += HeaderBoxStyle.Render(statusText) + "\n\n"

	// Title
	s += RenderTitle("What would you like to do?") + "\n\n"

	// Menu items
	for i, item := range m.items {
		cursor := "  "
		style := MenuItemStyle

		if m.cursor == i {
			cursor = MenuCursorStyle.Render("> ")
			style = MenuItemSelectedStyle
		}

		title := style.Render(item.Title)
		desc := MutedStyle.Render("  " + item.Description)

		s += cursor + title + "\n" + desc + "\n\n"
	}

	// Help
	s += HelpText("↑/↓: navigate • enter: select • q: quit")

	return lipgloss.NewStyle().Padding(1, 2).Render(s)
}

// SelectedAction returns the currently selected action
func (m MenuModel) SelectedAction() MenuAction {
	return m.items[m.cursor].Action
}

// RefreshStatus updates the branch and changes status
func (m *MenuModel) RefreshStatus() {
	m.branch, _ = git.CurrentBranch()
	m.hasChanges = git.HasChanges()
	m.isOnMain = git.IsOnMain()
	m.items = m.buildMenuItems()
	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
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

