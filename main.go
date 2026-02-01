package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"vc/git"
	"vc/ui"
	"vc/web"
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
	StateSettings
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
	settings    ui.SettingsModel
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
	// Start the menu's tick for periodic refresh
	return m.menu.Init()
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Pass size to menu (always, since we might return to it)
		m.menu.SetSize(msg.Width, msg.Height)
		// Continue processing to let sub-models handle it too

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
				cmd := m.menu.RefreshStatus()
				return m, cmd
			case StateSettings:
				if m.settings.HasUnsavedChanges() {
					m.settings.PromptExit()
					return m, nil
				}
				m.state = StateMenu
				cmd := m.menu.RefreshStatus()
				return m, cmd
			case StateExperiments:
				if m.experiments.WantsBack() {
					m.state = StateMenu
					cmd := m.menu.RefreshStatus()
					return m, cmd
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
			case ui.ActionSettings:
				m.state = StateSettings
				m.settings = ui.NewSettingsModel()
				return m, m.settings.Init()
			case ui.ActionQuit:
				return m, tea.Quit
			}
		}

		// Handle "any key to continue" on done states
		if m.state == StateSave && m.save.IsDone() {
			m.state = StateMenu
			cmd := m.menu.RefreshStatus()
			return m, cmd
		}
		if m.state == StateSync && m.sync.IsDone() {
			m.state = StateMenu
			cmd := m.menu.RefreshStatus()
			return m, cmd
		}
		if m.state == StateRestore && m.restore.IsDone() {
			m.state = StateMenu
			cmd := m.menu.RefreshStatus()
			return m, cmd
		}
		if m.state == StateBackups && m.backups.IsDone() {
			m.state = StateMenu
			cmd := m.menu.RefreshStatus()
			return m, cmd
		}
		if m.state == StateExperiments && m.experiments.IsDone() {
			// After keep/abandon, go back to main menu
			if m.experiments.ShouldReturnToMainMenu() {
				m.state = StateMenu
				cmd := m.menu.RefreshStatus()
				return m, cmd
			}
			// Otherwise stay in experiments menu
			m.experiments = ui.NewExperimentsModel()
			return m, nil
		}
		// Settings doesn't auto-close, handled by esc key above
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
			cmd := m.menu.RefreshStatus()
			return m, cmd
		}
		m.experiments, cmd = m.experiments.Update(msg)
	case StateSettings:
		m.settings, cmd = m.settings.Update(msg)
		// Check if user confirmed exit
		if m.settings.WantsBack() {
			m.state = StateMenu
			return m, m.menu.RefreshStatus()
		}
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
	case StateSettings:
		return m.settings.View()
	default:
		return m.menu.View()
	}
}

var quitKey = key.NewBinding(
	key.WithKeys("q", "ctrl+c"),
	key.WithHelp("q", "quit"),
)

// generateTestData creates hundreds of garbage files for stress testing the UI
func generateTestData() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// File extensions to use
	extensions := []string{
		".txt", ".md", ".go", ".js", ".ts", ".css", ".html", ".json",
		".yaml", ".yml", ".toml", ".xml", ".py", ".rs", ".c", ".h",
		".cpp", ".hpp", ".java", ".rb", ".sh", ".sql", ".log", ".csv",
	}

	// Directories to create files in
	dirs := []string{
		"test-data",
		"test-data/src",
		"test-data/lib",
		"test-data/config",
		"test-data/docs",
		"test-data/assets",
		"test-data/scripts",
		"test-data/components",
		"test-data/utils",
		"test-data/models",
	}

	// Word lists for generating random names and content
	adjectives := []string{
		"quick", "lazy", "happy", "sad", "bright", "dark", "old", "new",
		"fast", "slow", "big", "small", "hot", "cold", "soft", "hard",
		"clean", "dirty", "loud", "quiet", "sharp", "dull", "smooth", "rough",
	}
	nouns := []string{
		"fox", "dog", "cat", "bird", "fish", "tree", "flower", "rock",
		"river", "mountain", "cloud", "star", "moon", "sun", "rain", "snow",
		"wind", "fire", "earth", "water", "light", "shadow", "dream", "song",
	}
	verbs := []string{
		"jumps", "runs", "walks", "flies", "swims", "climbs", "falls", "rises",
		"grows", "shrinks", "opens", "closes", "starts", "stops", "moves", "stays",
	}

	// Create directories
	fmt.Println("Creating test directories...")
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			continue
		}
	}

	// Generate random file content
	generateContent := func(lines int) string {
		content := ""
		for i := 0; i < lines; i++ {
			adj := adjectives[rng.Intn(len(adjectives))]
			noun := nouns[rng.Intn(len(nouns))]
			verb := verbs[rng.Intn(len(verbs))]
			content += fmt.Sprintf("The %s %s %s over the %s %s.\n",
				adj, noun, verb, adjectives[rng.Intn(len(adjectives))], nouns[rng.Intn(len(nouns))])
		}
		return content
	}

	// Generate file names
	generateFileName := func() string {
		adj := adjectives[rng.Intn(len(adjectives))]
		noun := nouns[rng.Intn(len(nouns))]
		num := rng.Intn(1000)
		return fmt.Sprintf("%s_%s_%d", adj, noun, num)
	}

	// Create files
	totalFiles := 500
	fmt.Printf("Generating %d test files...\n", totalFiles)

	for i := 0; i < totalFiles; i++ {
		// Pick random directory and extension
		dir := dirs[rng.Intn(len(dirs))]
		ext := extensions[rng.Intn(len(extensions))]
		fileName := generateFileName() + ext
		filePath := filepath.Join(dir, fileName)

		// Generate random content (5-50 lines)
		lines := 5 + rng.Intn(46)
		content := generateContent(lines)

		// Write file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			fmt.Printf("Error writing file %s: %v\n", filePath, err)
			continue
		}

		// Progress indicator every 50 files
		if (i+1)%50 == 0 {
			fmt.Printf("  Created %d/%d files...\n", i+1, totalFiles)
		}
	}

	fmt.Printf("\n✓ Generated %d test files in test-data/\n", totalFiles)
	fmt.Println("\nTo clean up later, run: rm -rf test-data/")
}

func main() {
	// Check if we're in a git repo
	if !git.IsRepo() {
		fmt.Println(ui.RenderError("Not a git repository!"))
		fmt.Println(ui.RenderMuted("\nRun this command in a folder with git initialized."))
		fmt.Println(ui.RenderMuted("To initialize git, run: git init"))
		os.Exit(1)
	}

	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "web":
			port := 3000
			if err := web.StartServer(port); err != nil {
				fmt.Printf("Error starting web server: %v\n", err)
				os.Exit(1)
			}
			return
		case "gen-test-data":
			generateTestData()
			return
		case "help", "--help", "-h":
			fmt.Println("smooth - Version control for vibe coders")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  vibevc              Start the TUI interface")
			fmt.Println("  vibevc web          Start the web interface (http://localhost:3000)")
			fmt.Println("  vibevc gen-test-data Generate hundreds of garbage files for stress testing")
			fmt.Println("  vibevc help         Show this help message")
			return
		}
	}

	// Default: run TUI
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
