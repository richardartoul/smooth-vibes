package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"smooth/git"
)

// BranchChoice represents the user's choice for handling wrong branch
type BranchChoice int

const (
	BranchChoiceNone BranchChoice = iota
	BranchChoiceSwitch
	BranchChoiceExit
)

// BranchModel is the model for the "wrong branch" prompt
type BranchModel struct {
	cursor        int
	currentBranch string
	mainBranch    string
	width         int
	height        int
	done          bool
	choice        BranchChoice
	switchError   string
}

// NewBranchModel creates a new branch model
func NewBranchModel(currentBranch string) BranchModel {
	return BranchModel{
		cursor:        0,
		currentBranch: currentBranch,
		mainBranch:    git.GetMainBranch(),
		width:         80,
		height:        24,
	}
}

// Init initializes the model
func (m BranchModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m BranchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.done {
			return m, tea.Quit
		}

		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < 1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Enter):
			if m.cursor == 0 {
				// Switch to main branch
				err := git.SwitchBranch(m.mainBranch)
				if err != nil {
					m.switchError = err.Error()
					m.done = true
					m.choice = BranchChoiceExit
				} else {
					m.done = true
					m.choice = BranchChoiceSwitch
				}
				return m, nil
			} else {
				// Exit immediately
				m.done = true
				m.choice = BranchChoiceExit
				return m, tea.Quit
			}
		case msg.String() == "q" || msg.String() == "ctrl+c" || msg.String() == "esc":
			m.done = true
			m.choice = BranchChoiceExit
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the prompt
func (m BranchModel) View() string {
	var content string

	// Show a smaller banner for this screen
	content += TitleStyle.Render("SMOOTH") + "\n\n"

	// Error state
	if m.done && m.switchError != "" {
		content += ErrorStyle.Render("✗ Failed to switch branches") + "\n\n"
		content += MutedStyle.Render("Error: "+m.switchError) + "\n\n"
		content += MutedStyle.Render("You may have uncommitted changes. Commit or stash them first.") + "\n\n"
		content += MutedStyle.Render("Press any key to exit...") + "\n"

		return lipgloss.NewStyle().
			Padding(2, 4).
			Width(m.width).
			Height(m.height).
			Render(content)
	}

	// Success state
	if m.done && m.choice == BranchChoiceSwitch {
		content += SuccessStyle.Render(fmt.Sprintf("✓ Switched to %s!", m.mainBranch)) + "\n\n"
		content += MutedStyle.Render("Press any key to continue...") + "\n"

		return lipgloss.NewStyle().
			Padding(2, 4).
			Width(m.width).
			Height(m.height).
			Render(content)
	}

	// Main prompt
	warningBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDanger).
		Padding(1, 2).
		Render(ErrorStyle.Render("⚠ Wrong branch: ") + HighlightStyle.Render(m.currentBranch))

	content += warningBox + "\n\n"

	// Explanation
	explanationStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Width(60)

	explanation := fmt.Sprintf(`Smooth needs to run from your main branch (%s).

You're currently on a different branch. This might be an 
experiment branch or a feature branch.

You can switch to %s to continue, or exit if you want 
to stay on this branch.`, m.mainBranch, m.mainBranch)

	content += explanationStyle.Render(explanation) + "\n\n"

	// Menu options
	content += RenderTitle("What would you like to do?") + "\n\n"

	options := []struct {
		title string
		desc  string
	}{
		{fmt.Sprintf("Switch to %s", m.mainBranch), fmt.Sprintf("Checkout the %s branch and continue", m.mainBranch)},
		{"Exit", "Stay on current branch"},
	}

	for i, opt := range options {
		cursor := "  "
		style := MenuItemStyle
		if m.cursor == i {
			cursor = MenuCursorStyle.Render("> ")
			style = MenuItemSelectedStyle
		}
		content += cursor + style.Render(opt.title) + "\n"
		content += "    " + MutedStyle.Render(opt.desc) + "\n"
		if i < len(options)-1 {
			content += "\n"
		}
	}

	// Help bar
	helpBar := HelpBar([][]string{
		{"↑↓", "navigate"},
		{"enter", "select"},
		{"q", "quit"},
	})

	// Center help bar
	centeredHelp := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, helpBar)

	// Layout
	mainContent := lipgloss.NewStyle().
		Padding(2, 4).
		Render(content)

	// Calculate content height
	contentHeight := m.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}

	placedContent := lipgloss.Place(m.width, contentHeight, lipgloss.Left, lipgloss.Top, mainContent)

	return lipgloss.JoinVertical(lipgloss.Left, placedContent, centeredHelp)
}

// IsDone returns true if the user has made a choice
func (m BranchModel) IsDone() bool {
	return m.done
}

// Choice returns the user's choice
func (m BranchModel) Choice() BranchChoice {
	return m.choice
}

// ShouldContinue returns true if branch was switched and the app should continue
func (m BranchModel) ShouldContinue() bool {
	return m.done && m.choice == BranchChoiceSwitch && m.switchError == ""
}

