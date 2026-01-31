package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vc/git"
)

// ExperimentsState represents the state of the experiments flow
type ExperimentsState int

const (
	ExperimentsStateMenu ExperimentsState = iota
	ExperimentsStateNameInput
	ExperimentsStateCreating
	ExperimentsStateKeeping
	ExperimentsStateAbandoning
	ExperimentsStateSwitchList
	ExperimentsStateSwitching
	ExperimentsStateSuccess
	ExperimentsStateError
)

// ExperimentsAction represents the selected action
type ExperimentsAction int

const (
	ExpActionStart ExperimentsAction = iota
	ExpActionKeep
	ExpActionAbandon
	ExpActionSwitch
	ExpActionBack
)

// ExperimentsModel is the model for the experiments flow
type ExperimentsModel struct {
	state       ExperimentsState
	cursor      int
	textInput   textinput.Model
	experiments []git.BranchInfo
	expCursor   int
	currentBranch string
	isOnMain    bool
	err         error
	message     string
}

type experimentsMenuItem struct {
	Title       string
	Description string
	Action      ExperimentsAction
	Disabled    bool
}

// NewExperimentsModel creates a new experiments model
func NewExperimentsModel() ExperimentsModel {
	ti := textinput.New()
	ti.Placeholder = "my-cool-idea"
	ti.CharLimit = 30
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorAccent)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorText)

	branch, _ := git.CurrentBranch()
	isOnMain := git.IsOnMain()
	experiments, _ := git.ListExperiments()

	return ExperimentsModel{
		state:         ExperimentsStateMenu,
		cursor:        0,
		textInput:     ti,
		experiments:   experiments,
		currentBranch: branch,
		isOnMain:      isOnMain,
	}
}

func (m ExperimentsModel) getMenuItems() []experimentsMenuItem {
	return []experimentsMenuItem{
		{
			Title:       "Start a new experiment",
			Description: "Create a safe space to try something new",
			Action:      ExpActionStart,
		},
		{
			Title:       "Keep this experiment",
			Description: "Merge your experiment into your main work",
			Action:      ExpActionKeep,
			Disabled:    m.isOnMain,
		},
		{
			Title:       "Abandon this experiment",
			Description: "Discard this experiment and go back to main",
			Action:      ExpActionAbandon,
			Disabled:    m.isOnMain,
		},
		{
			Title:       "Switch experiment",
			Description: "Switch to a different experiment",
			Action:      ExpActionSwitch,
			Disabled:    len(m.experiments) == 0,
		},
		{
			Title:       "Back to main menu",
			Description: "",
			Action:      ExpActionBack,
		},
	}
}

// Init initializes the experiments model
func (m ExperimentsModel) Init() tea.Cmd {
	return nil
}

// ExperimentsMsg is sent when an experiments operation completes
type ExperimentsMsg struct {
	Err     error
	Message string
}

// doCreateExperiment creates a new experiment branch
func doCreateExperiment(name string) tea.Cmd {
	return func() tea.Msg {
		branchName, err := git.CreateExperiment(name)
		if err != nil {
			return ExperimentsMsg{Err: err}
		}
		return ExperimentsMsg{Message: fmt.Sprintf("Created experiment: %s", branchName)}
	}
}

// doKeepExperiment merges the current experiment into main
func doKeepExperiment() tea.Cmd {
	return func() tea.Msg {
		err := git.KeepExperiment()
		if err != nil {
			return ExperimentsMsg{Err: err}
		}
		return ExperimentsMsg{Message: "Experiment merged into main!"}
	}
}

// doAbandonExperiment deletes the current experiment
func doAbandonExperiment() tea.Cmd {
	return func() tea.Msg {
		err := git.AbandonExperiment()
		if err != nil {
			return ExperimentsMsg{Err: err}
		}
		return ExperimentsMsg{Message: "Experiment abandoned. Back on main."}
	}
}

// doSwitchExperiment switches to a different experiment
func doSwitchExperiment(branchName string) tea.Cmd {
	return func() tea.Msg {
		// Stash any current changes
		if git.HasChanges() {
			if err := git.Stash(); err != nil {
				return ExperimentsMsg{Err: err}
			}
		}
		
		if err := git.SwitchBranch(branchName); err != nil {
			return ExperimentsMsg{Err: err}
		}

		// Try to pop stash, but don't fail if there's nothing
		git.StashPop()

		return ExperimentsMsg{Message: fmt.Sprintf("Switched to: %s", branchName)}
	}
}

// Update handles messages for the experiments model
func (m ExperimentsModel) Update(msg tea.Msg) (ExperimentsModel, tea.Cmd) {
	menuItems := m.getMenuItems()

	switch msg := msg.(type) {
	case ExperimentsMsg:
		if msg.Err != nil {
			m.state = ExperimentsStateError
			m.err = msg.Err
		} else {
			m.state = ExperimentsStateSuccess
			m.message = msg.Message
		}
		// Refresh state
		m.currentBranch, _ = git.CurrentBranch()
		m.isOnMain = git.IsOnMain()
		m.experiments, _ = git.ListExperiments()
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case ExperimentsStateMenu:
			switch {
			case key.Matches(msg, keys.Up):
				for {
					if m.cursor > 0 {
						m.cursor--
					} else {
						break
					}
					if !menuItems[m.cursor].Disabled {
						break
					}
				}
			case key.Matches(msg, keys.Down):
				for {
					if m.cursor < len(menuItems)-1 {
						m.cursor++
					} else {
						break
					}
					if !menuItems[m.cursor].Disabled {
						break
					}
				}
			case key.Matches(msg, keys.Enter):
				item := menuItems[m.cursor]
				if item.Disabled {
					return m, nil
				}
				switch item.Action {
				case ExpActionStart:
					m.state = ExperimentsStateNameInput
					m.textInput.Focus()
					return m, textinput.Blink
				case ExpActionKeep:
					m.state = ExperimentsStateKeeping
					return m, doKeepExperiment()
				case ExpActionAbandon:
					m.state = ExperimentsStateAbandoning
					return m, doAbandonExperiment()
				case ExpActionSwitch:
					m.state = ExperimentsStateSwitchList
					m.expCursor = 0
				case ExpActionBack:
					// Signal to return to main menu - handled in main model
				}
			}

		case ExperimentsStateNameInput:
			switch msg.String() {
			case "enter":
				if m.textInput.Value() != "" {
					m.state = ExperimentsStateCreating
					return m, doCreateExperiment(m.textInput.Value())
				}
			case "esc":
				m.state = ExperimentsStateMenu
				m.textInput.SetValue("")
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}

		case ExperimentsStateSwitchList:
			switch {
			case key.Matches(msg, keys.Up):
				if m.expCursor > 0 {
					m.expCursor--
				}
			case key.Matches(msg, keys.Down):
				if m.expCursor < len(m.experiments)-1 {
					m.expCursor++
				}
			case key.Matches(msg, keys.Enter):
				if len(m.experiments) > 0 {
					m.state = ExperimentsStateSwitching
					return m, doSwitchExperiment(m.experiments[m.expCursor].Name)
				}
			case msg.String() == "esc":
				m.state = ExperimentsStateMenu
			}
		}
	}

	return m, nil
}

// View renders the experiments flow
func (m ExperimentsModel) View() string {
	var s string

	s += RenderTitle("Experiments") + "\n\n"

	// Show current branch status
	branchLabel := "main"
	if !m.isOnMain {
		branchLabel = HighlightStyle.Render(m.currentBranch)
	}
	s += RenderMuted("Currently on: ") + branchLabel + "\n\n"

	switch m.state {
	case ExperimentsStateMenu:
		menuItems := m.getMenuItems()
		for i, item := range menuItems {
			cursor := "  "
			style := MenuItemStyle

			if item.Disabled {
				style = MutedStyle
			} else if m.cursor == i {
				cursor = MenuCursorStyle.Render("> ")
				style = MenuItemSelectedStyle
			}

			title := style.Render(item.Title)
			s += cursor + title + "\n"
			if item.Description != "" {
				descStyle := MutedStyle
				if item.Disabled {
					descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#444"))
				}
				s += "    " + descStyle.Render(item.Description) + "\n"
			}
			s += "\n"
		}

		s += HelpText("↑/↓: navigate • enter: select • esc: back")

	case ExperimentsStateNameInput:
		s += RenderSubtitle("Name your experiment:") + "\n\n"
		s += m.textInput.View() + "\n\n"
		s += RenderMuted("Use a short, descriptive name (no spaces)") + "\n\n"
		s += HelpText("enter: create • esc: cancel")

	case ExperimentsStateCreating:
		s += RenderHighlight("Creating experiment...") + "\n"

	case ExperimentsStateKeeping:
		s += RenderHighlight("Merging experiment into main...") + "\n"

	case ExperimentsStateAbandoning:
		s += RenderHighlight("Abandoning experiment...") + "\n"

	case ExperimentsStateSwitchList:
		s += RenderSubtitle("Select an experiment to switch to:") + "\n\n"

		// Also show main as an option
		allOptions := append([]git.BranchInfo{{Name: git.GetMainBranch(), IsCurrent: m.isOnMain}}, m.experiments...)

		for i, exp := range allOptions {
			cursor := "  "
			style := ListItemStyle

			if m.expCursor == i {
				cursor = MenuCursorStyle.Render("> ")
				style = ListItemSelectedStyle
			}

			label := exp.Name
			if exp.IsCurrent {
				label += " (current)"
			}

			s += cursor + style.Render(label) + "\n\n"
		}

		s += HelpText("↑/↓: navigate • enter: switch • esc: back")

	case ExperimentsStateSwitching:
		s += RenderHighlight("Switching...") + "\n"

	case ExperimentsStateSuccess:
		s += RenderSuccess("✓ " + m.message) + "\n\n"
		s += HelpText("Press any key to continue")

	case ExperimentsStateError:
		s += RenderError("✗ Operation failed") + "\n\n"
		if m.err != nil {
			s += RenderMuted(m.err.Error()) + "\n\n"
		}
		s += HelpText("Press any key to go back")
	}

	return BoxStyle.Render(s)
}

// IsDone returns true if the user wants to go back to main menu
func (m ExperimentsModel) IsDone() bool {
	return m.state == ExperimentsStateSuccess || m.state == ExperimentsStateError
}

// WantsBack returns true if the user selected "Back to main menu"
func (m ExperimentsModel) WantsBack() bool {
	return m.state == ExperimentsStateMenu && m.getMenuItems()[m.cursor].Action == ExpActionBack
}

