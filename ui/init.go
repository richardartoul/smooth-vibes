package ui

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"smooth/git"
)

// InitChoice represents the user's choice for handling missing git repo
type InitChoice int

const (
	InitChoiceNone InitChoice = iota
	InitChoiceInit
	InitChoiceExit
)

// InitModel is the model for the "not a git repository" prompt
type InitModel struct {
	cursor    int
	cwd       string
	width     int
	height    int
	done      bool
	choice    InitChoice
	initError string
}

// NewInitModel creates a new init model
func NewInitModel() InitModel {
	cwd, _ := os.Getwd()
	return InitModel{
		cursor: 0,
		cwd:    cwd,
		width:  80,
		height: 24,
	}
}

// Init initializes the model
func (m InitModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m InitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				// Initialize git
				_, err := git.Run("init")
				if err != nil {
					m.initError = err.Error()
					m.done = true
					m.choice = InitChoiceExit
				} else {
					m.done = true
					m.choice = InitChoiceInit
				}
				return m, nil
			} else {
				// Exit immediately
				m.done = true
				m.choice = InitChoiceExit
				return m, tea.Quit
			}
		case msg.String() == "q" || msg.String() == "ctrl+c" || msg.String() == "esc":
			m.done = true
			m.choice = InitChoiceExit
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the prompt
func (m InitModel) View() string {
	var content string

	// Show a smaller banner for this screen
	content += TitleStyle.Render("SMOOTH") + "\n\n"

	// Error state
	if m.done && m.initError != "" {
		content += ErrorStyle.Render("✗ Failed to initialize git") + "\n\n"
		content += MutedStyle.Render("Error: "+m.initError) + "\n\n"
		content += MutedStyle.Render("Press any key to exit...") + "\n"

		return lipgloss.NewStyle().
			Padding(2, 4).
			Width(m.width).
			Height(m.height).
			Render(content)
	}

	// Success state
	if m.done && m.choice == InitChoiceInit {
		content += SuccessStyle.Render("✓ Git initialized successfully!") + "\n\n"
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
		Render(ErrorStyle.Render("⚠ Not a git repository"))

	content += warningBox + "\n\n"

	// Current directory info
	dirName := filepath.Base(m.cwd)
	content += MutedStyle.Render("Current folder: ") + HighlightStyle.Render(dirName) + "\n"
	content += MutedStyle.Render(m.cwd) + "\n\n"

	// Explanation
	explanationStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Width(60)

	explanation := `Smooth needs to run in the root folder of your project 
where git is initialized.

If this is your project folder, you can initialize git here.
If you're in the wrong place, exit and navigate to your project.`

	content += explanationStyle.Render(explanation) + "\n\n"

	// Menu options
	content += RenderTitle("What would you like to do?") + "\n\n"

	options := []struct {
		title string
		desc  string
	}{
		{"Initialize git here", "Run 'git init' to start tracking this folder"},
		{"Exit", "I'm in the wrong folder"},
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
func (m InitModel) IsDone() bool {
	return m.done
}

// Choice returns the user's choice
func (m InitModel) Choice() InitChoice {
	return m.choice
}

// ShouldContinue returns true if git was initialized and the app should continue
func (m InitModel) ShouldContinue() bool {
	return m.done && m.choice == InitChoiceInit && m.initError == ""
}

