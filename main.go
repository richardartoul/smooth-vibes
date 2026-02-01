package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"vc/git"
	"vc/ui"
)

// AppState represents the current state of the application.
type AppState int

const (
	StateMenu AppState = iota
	StateSave
	StateSync
	StateRestore
	StateBackups
	StateExperiments
)

// Model is the main application model
type Model struct {
	state       AppState
	menu        ui.MenuModel
	save        ui.SaveModel
	sync        ui.SyncModel
	restore     ui.RestoreModel
	backups     ui.BackupsModel
	experiments ui.ExperimentsModel
	width       int
	height      int
}

// NewModel creates a new application model
func NewModel() Model {
	return Model{
		state: StateMenu,
		menu:  ui.NewMenuModel(),
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Global quit
		if key.Matches(msg, quitKey) && m.state == StateMenu {
			return m, tea.Quit
		}

		// Handle escape to go back
		if msg.String() == "esc" {
			switch m.state {
			case StateSave, StateSync, StateRestore, StateBackups:
				m.state = StateMenu
				m.menu.RefreshStatus()
				return m, nil
			case StateExperiments:
				if m.experiments.WantsBack() {
					m.state = StateMenu
					m.menu.RefreshStatus()
					return m, nil
				}
			}
		}

		// Handle enter on menu
		if msg.String() == "enter" && m.state == StateMenu {
			switch m.menu.SelectedAction() {
			case ui.ActionSave:
				m.state = StateSave
				m.save = ui.NewSaveModel()
				return m, m.save.Init()
			case ui.ActionSync:
				m.state = StateSync
				m.sync = ui.NewSyncModel()
				return m, m.sync.Init()
			case ui.ActionRestore:
				m.state = StateRestore
				m.restore = ui.NewRestoreModel()
				return m, m.restore.Init()
			case ui.ActionBackups:
				m.state = StateBackups
				m.backups = ui.NewBackupsModel()
				return m, m.backups.Init()
			case ui.ActionExperiments:
				m.state = StateExperiments
				m.experiments = ui.NewExperimentsModel()
				return m, m.experiments.Init()
			case ui.ActionKeepExperiment:
				m.state = StateExperiments
				var cmd tea.Cmd
				m.experiments, cmd = ui.NewKeepExperimentModel()
				return m, cmd
			case ui.ActionAbandonExperiment:
				m.state = StateExperiments
				var cmd tea.Cmd
				m.experiments, cmd = ui.NewAbandonExperimentModel()
				return m, cmd
			case ui.ActionQuit:
				return m, tea.Quit
			}
		}

		// Handle "any key to continue" on done states
		if m.state == StateSave && m.save.IsDone() {
			m.state = StateMenu
			m.menu.RefreshStatus()
			return m, nil
		}
		if m.state == StateSync && m.sync.IsDone() {
			m.state = StateMenu
			m.menu.RefreshStatus()
			return m, nil
		}
		if m.state == StateRestore && m.restore.IsDone() {
			m.state = StateMenu
			m.menu.RefreshStatus()
			return m, nil
		}
		if m.state == StateBackups && m.backups.IsDone() {
			m.state = StateMenu
			m.menu.RefreshStatus()
			return m, nil
		}
		if m.state == StateExperiments && m.experiments.IsDone() {
			// After keep/abandon, go back to main menu
			if m.experiments.ShouldReturnToMainMenu() {
				m.state = StateMenu
				m.menu.RefreshStatus()
				return m, nil
			}
			// Otherwise stay in experiments menu
			m.experiments = ui.NewExperimentsModel()
			return m, nil
		}
	}

	// Delegate to sub-models
	var cmd tea.Cmd
	switch m.state {
	case StateMenu:
		m.menu, cmd = m.menu.Update(msg)
	case StateSave:
		m.save, cmd = m.save.Update(msg)
	case StateSync:
		m.sync, cmd = m.sync.Update(msg)
	case StateRestore:
		m.restore, cmd = m.restore.Update(msg)
	case StateBackups:
		m.backups, cmd = m.backups.Update(msg)
	case StateExperiments:
		// Check if user wants to go back
		if m.experiments.WantsBack() {
			m.state = StateMenu
			m.menu.RefreshStatus()
			return m, nil
		}
		m.experiments, cmd = m.experiments.Update(msg)
	}

	return m, cmd
}

// View renders the application
func (m Model) View() string {
	switch m.state {
	case StateSave:
		return m.save.View()
	case StateSync:
		return m.sync.View()
	case StateRestore:
		return m.restore.View()
	case StateBackups:
		return m.backups.View()
	case StateExperiments:
		return m.experiments.View()
	default:
		return m.menu.View()
	}
}

var quitKey = key.NewBinding(
	key.WithKeys("q", "ctrl+c"),
	key.WithHelp("q", "quit"),
)

func main() {
	// Check if we're in a git repo
	if !git.IsRepo() {
		fmt.Println(ui.RenderError("Not a git repository!"))
		fmt.Println(ui.RenderMuted("\nRun this command in a folder with git initialized."))
		fmt.Println(ui.RenderMuted("To initialize git, run: git init"))
		os.Exit(1)
	}

	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
