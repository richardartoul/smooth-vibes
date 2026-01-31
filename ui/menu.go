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
	ActionQuit
)

// MenuModel is the model for the main menu
type MenuModel struct {
	items    []MenuItem
	cursor   int
	branch   string
	hasChanges bool
}

// NewMenuModel creates a new menu model
func NewMenuModel() MenuModel {
	items := []MenuItem{
		{
			Title:       "Save my current progress",
			Description: "Create a save point of your current work",
			Action:      ActionSave,
		},
		{
			Title:       "Sync to GitHub",
			Description: "Upload your saves to the cloud",
			Action:      ActionSync,
		},
		{
			Title:       "Go back to a previous state",
			Description: "Restore your project to an earlier save point",
			Action:      ActionRestore,
		},
		{
			Title:       "Experiments",
			Description: "Try new ideas without breaking your main work",
			Action:      ActionExperiments,
		},
		{
			Title:       "Quit",
			Description: "Exit the application",
			Action:      ActionQuit,
		},
	}

	branch, _ := git.CurrentBranch()
	hasChanges := git.HasChanges()

	return MenuModel{
		items:      items,
		cursor:     0,
		branch:     branch,
		hasChanges: hasChanges,
	}
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
	statusText := fmt.Sprintf("Branch: %s", m.branch)
	if m.hasChanges {
		statusText += " " + HighlightStyle.Render("(unsaved changes)")
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

