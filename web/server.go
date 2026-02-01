package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"

	"vc/git"
)

//go:embed static/*
var staticFiles embed.FS

// StartServer starts the web server on the specified port
func StartServer(port int) error {
	// API routes
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/changes", handleChanges)
	http.HandleFunc("/api/save", handleSave)
	http.HandleFunc("/api/sync", handleSync)
	http.HandleFunc("/api/commits", handleCommits)
	http.HandleFunc("/api/restore", handleRestore)
	http.HandleFunc("/api/backups", handleBackups)
	http.HandleFunc("/api/restore-backup", handleRestoreBackup)
	http.HandleFunc("/api/experiments", handleExperiments)
	http.HandleFunc("/api/experiment/create", handleCreateExperiment)
	http.HandleFunc("/api/experiment/keep", handleKeepExperiment)
	http.HandleFunc("/api/experiment/abandon", handleAbandonExperiment)
	http.HandleFunc("/api/experiment/switch", handleSwitchExperiment)
	http.HandleFunc("/api/gitignore", handleGitignore)

	// Static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	http.Handle("/", http.FileServer(http.FS(staticFS)))

	fmt.Printf("Starting web server at http://localhost:%d\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

// Response helpers
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// API Handlers

func handleStatus(w http.ResponseWriter, r *http.Request) {
	branch, _ := git.CurrentBranch()
	hasChanges := git.HasChanges()
	isOnMain := git.IsOnMain()

	jsonResponse(w, map[string]interface{}{
		"branch":     branch,
		"hasChanges": hasChanges,
		"isOnMain":   isOnMain,
	})
}

func handleChanges(w http.ResponseWriter, r *http.Request) {
	changes, err := git.GetChangeSummary()
	if err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}
	jsonResponse(w, changes)
}

func handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errorResponse(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Message string   `json:"message"`
		Files   []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, "Invalid request", 400)
		return
	}

	// Stage files
	if len(req.Files) > 0 {
		if err := git.AddFiles(req.Files); err != nil {
			errorResponse(w, err.Error(), 500)
			return
		}
	}

	// Commit
	if err := git.Commit(req.Message); err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errorResponse(w, "Method not allowed", 405)
		return
	}

	if err := git.Push(); err != nil {
		// Check for no remote error
		if _, ok := err.(git.NoRemoteError); ok {
			errorResponse(w, "No GitHub remote configured. Create a repo on GitHub, then run: git remote add origin https://github.com/USERNAME/REPO.git", 400)
			return
		}
		errorResponse(w, err.Error(), 500)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

func handleCommits(w http.ResponseWriter, r *http.Request) {
	commits, err := git.Log(20)
	if err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}
	jsonResponse(w, commits)
}

func handleRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errorResponse(w, "Method not allowed", 405)
		return
	}

	var req struct {
		CommitHash string `json:"commitHash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, "Invalid request", 400)
		return
	}

	// Create backup first
	branch, _ := git.CurrentBranch()
	backupName, err := git.CreateBackup(branch)
	if err != nil {
		errorResponse(w, "Failed to create backup: "+err.Error(), 500)
		return
	}

	// Reset
	if err := git.ResetHard(req.CommitHash); err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok", "backup": backupName})
}

func handleBackups(w http.ResponseWriter, r *http.Request) {
	branch, _ := git.CurrentBranch()
	backups, err := git.ListBackups(branch)
	if err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}
	jsonResponse(w, backups)
}

func handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errorResponse(w, "Method not allowed", 405)
		return
	}

	var req struct {
		BackupName string `json:"backupName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, "Invalid request", 400)
		return
	}

	if err := git.RestoreBackup(req.BackupName); err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

func handleExperiments(w http.ResponseWriter, r *http.Request) {
	experiments, err := git.ListExperiments()
	if err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}
	jsonResponse(w, experiments)
}

func handleCreateExperiment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errorResponse(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, "Invalid request", 400)
		return
	}

	branchName, err := git.CreateExperiment(req.Name)
	if err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok", "branch": branchName})
}

func handleKeepExperiment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errorResponse(w, "Method not allowed", 405)
		return
	}

	// Check for unsaved changes
	if git.HasChanges() {
		errorResponse(w, "You have unsaved changes. Please save your progress first.", 400)
		return
	}

	if err := git.KeepExperiment(); err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

func handleAbandonExperiment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errorResponse(w, "Method not allowed", 405)
		return
	}

	// Check for unsaved changes
	if git.HasChanges() {
		errorResponse(w, "You have unsaved changes. Please save your progress first.", 400)
		return
	}

	if err := git.AbandonExperiment(); err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

func handleSwitchExperiment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errorResponse(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Branch string `json:"branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, "Invalid request", 400)
		return
	}

	// Stash changes if any
	if git.HasChanges() {
		if err := git.Stash(); err != nil {
			errorResponse(w, err.Error(), 500)
			return
		}
	}

	if err := git.SwitchBranch(req.Branch); err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}

	// Try to pop stash
	git.StashPop()

	jsonResponse(w, map[string]string{"status": "ok"})
}

func handleGitignore(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errorResponse(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Pattern string `json:"pattern"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, "Invalid request", 400)
		return
	}

	if err := git.AddToGitignore(req.Pattern); err != nil {
		errorResponse(w, err.Error(), 500)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

