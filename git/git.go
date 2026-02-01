package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CommitInfo represents a simplified commit entry
type CommitInfo struct {
	Hash      string
	Message   string
	Timestamp string
	FullHash  string
}

// BranchInfo represents a branch
type BranchInfo struct {
	Name      string
	IsCurrent bool
}

// Run executes a git command and returns the output (trimmed)
func Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// RunRaw executes a git command and returns the raw output (preserves whitespace)
func RunRaw(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// IsRepo checks if the current directory is a git repository
func IsRepo() bool {
	_, err := Run("rev-parse", "--git-dir")
	return err == nil
}

// CurrentBranch returns the current branch name
func CurrentBranch() (string, error) {
	return Run("rev-parse", "--abbrev-ref", "HEAD")
}

// AddAll stages all changes
func AddAll() error {
	_, err := Run("add", "-A")
	return err
}

// AddFiles stages specific files
func AddFiles(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, paths...)
	_, err := Run(args...)
	return err
}

// AddToGitignore adds a pattern to .gitignore
func AddToGitignore(pattern string) error {
	// Read existing gitignore
	f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add newline + pattern
	if _, err := f.WriteString("\n" + pattern); err != nil {
		return err
	}

	return nil
}

// Commit creates a commit with the given message
func Commit(message string) error {
	_, err := Run("commit", "-m", message)
	return err
}

// Push pushes the current branch to origin
// HasRemote checks if a remote (origin) is configured
func HasRemote() bool {
	output, err := Run("remote", "get-url", "origin")
	return err == nil && output != ""
}

// GetRemoteURL returns the origin remote URL if configured
func GetRemoteURL() string {
	output, _ := Run("remote", "get-url", "origin")
	return output
}

// AddRemote adds a remote with the given name and URL
func AddRemote(name, url string) error {
	_, err := Run("remote", "add", name, url)
	return err
}

// AddOrigin adds the origin remote with the given URL
func AddOrigin(url string) error {
	return AddRemote("origin", url)
}

// NoRemoteError is returned when trying to push without a remote configured
type NoRemoteError struct{}

func (e NoRemoteError) Error() string {
	return "No GitHub remote configured. To set one up:\n\n" +
		"1. Create a repository on GitHub\n" +
		"2. Run: git remote add origin https://github.com/USERNAME/REPO.git\n" +
		"3. Try syncing again"
}

func Push() error {
	// Check if remote exists first
	if !HasRemote() {
		return NoRemoteError{}
	}

	branch, err := CurrentBranch()
	if err != nil {
		return err
	}
	_, err = Run("push", "-u", "origin", branch)
	return err
}

// Log returns a list of recent commits
func Log(count int) ([]CommitInfo, error) {
	format := "%h|%s|%cr|%H"
	output, err := Run("log", fmt.Sprintf("-%d", count), fmt.Sprintf("--format=%s", format))
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []CommitInfo{}, nil
	}

	var commits []CommitInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) == 4 {
			commits = append(commits, CommitInfo{
				Hash:      parts[0],
				Message:   parts[1],
				Timestamp: parts[2],
				FullHash:  parts[3],
			})
		}
	}
	return commits, nil
}

// ResetHard resets to the specified commit
func ResetHard(commitHash string) error {
	_, err := Run("reset", "--hard", commitHash)
	return err
}

// HasChanges checks if there are uncommitted changes
func HasChanges() bool {
	output, err := Run("status", "--porcelain")
	if err != nil {
		return false
	}
	return output != ""
}

// GetDiff returns the current diff output
func GetDiff() string {
	// Get diff of staged and unstaged changes
	output, err := RunRaw("diff", "HEAD", "--stat")
	if err != nil || strings.TrimSpace(output) == "" {
		// Try without HEAD for new repos
		output, _ = RunRaw("diff", "--stat")
	}

	// Always check for untracked files
	status, _ := Run("status", "--short")
	var untrackedFiles []string
	if status != "" {
		for _, line := range strings.Split(status, "\n") {
			if strings.HasPrefix(line, "?? ") {
				untrackedFiles = append(untrackedFiles, strings.TrimPrefix(line, "?? "))
			}
		}
	}

	result := strings.TrimRight(output, " \t\n\r")

	if len(untrackedFiles) > 0 {
		lines := strings.Split(result, "\n")

		// Find the max filename width from existing diff output
		maxWidth := 0
		for _, line := range lines {
			if idx := strings.Index(line, " | "); idx > 0 {
				if idx > maxWidth {
					maxWidth = idx
				}
			}
		}
		// Also consider untracked filenames
		for _, f := range untrackedFiles {
			if len(f)+1 > maxWidth {
				maxWidth = len(f) + 1
			}
		}
		if maxWidth == 0 {
			maxWidth = 20
		}

		// Format untracked files to match alignment, with line count
		var untracked []string
		for _, f := range untrackedFiles {
			lineCount := countFileLines(f)
			untracked = append(untracked, fmt.Sprintf(" %-*s | %d (new file)", maxWidth-1, f, lineCount))
		}

		if result != "" {
			var newLines []string
			summaryIdx := -1
			for i, line := range lines {
				if strings.Contains(line, "files changed") || strings.Contains(line, "file changed") {
					summaryIdx = i
					// Insert untracked files before summary
					newLines = append(newLines, untracked...)
					// Update the summary to include new files
					newCount := len(untrackedFiles)
					// Parse existing count and add new files
					updated := updateSummaryLine(line, newCount)
					newLines = append(newLines, updated)
					continue
				}
				newLines = append(newLines, line)
			}
			if summaryIdx == -1 {
				newLines = append(newLines, untracked...)
			}
			result = strings.Join(newLines, "\n")
		} else {
			result = strings.Join(untracked, "\n")
		}
	}

	if result == "" {
		return "No changes"
	}
	return result
}

// updateSummaryLine updates the "X files changed" summary to include new files
func updateSummaryLine(line string, newFiles int) string {
	if newFiles == 0 {
		return line
	}
	// Parse "N file(s) changed" and add new files to the count
	// Example: " 4 files changed, 51 insertions(+), 16 deletions(-)"
	parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
	if len(parts) < 2 {
		return line
	}
	count := 0
	fmt.Sscanf(parts[0], "%d", &count)
	newCount := count + newFiles

	// Rebuild the line with updated count
	rest := parts[1]
	if strings.HasPrefix(rest, "file ") {
		rest = "files " + strings.TrimPrefix(rest, "file ")
	}
	return fmt.Sprintf(" %d %s", newCount, rest)
}

// countFileLines counts the number of lines in a file
func countFileLines(filepath string) int {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return 0
	}
	if len(data) == 0 {
		return 0
	}
	count := strings.Count(string(data), "\n")
	// If file doesn't end with newline, add 1
	if data[len(data)-1] != '\n' {
		count++
	}
	return count
}

// GetDiffFull returns the full diff output (not just stats)
func GetDiffFull() string {
	output, err := Run("diff", "HEAD", "--color=never")
	if err != nil || output == "" {
		output, _ = Run("diff", "--color=never")
	}
	if output == "" {
		status, _ := Run("status", "--short")
		if status != "" {
			return status
		}
		return "No changes"
	}
	return output
}

// FileChange represents a changed file
type FileChange struct {
	Status string // "added", "modified", "deleted", "renamed"
	Path   string
}

// GetChangeSummary returns a summary of all changed files
func GetChangeSummary() ([]FileChange, error) {
	output, err := Run("status", "--porcelain")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []FileChange{}, nil
	}

	var changes []FileChange
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		statusCode := line[:2]
		path := strings.TrimSpace(line[2:])

		var status string
		switch {
		case statusCode[0] == 'A' || statusCode[1] == 'A' || statusCode == "??":
			status = "added"
		case statusCode[0] == 'D' || statusCode[1] == 'D':
			status = "deleted"
		case statusCode[0] == 'R' || statusCode[1] == 'R':
			status = "renamed"
		default:
			status = "modified"
		}

		changes = append(changes, FileChange{
			Status: status,
			Path:   path,
		})
	}

	return changes, nil
}

// LastCommitMessage returns the message of the last commit
func LastCommitMessage() (string, error) {
	return Run("log", "-1", "--format=%s")
}

// CreateBranch creates and switches to a new branch
func CreateBranch(name string) error {
	_, err := Run("checkout", "-b", name)
	return err
}

// CreateExperiment creates a new experiment branch with timestamp
func CreateExperiment(name string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	branchName := fmt.Sprintf("experiment-%s-%s", name, timestamp)
	err := CreateBranch(branchName)
	return branchName, err
}

// SwitchBranch switches to the specified branch
func SwitchBranch(name string) error {
	_, err := Run("checkout", name)
	return err
}

// MergeBranch merges the specified branch into the current branch
func MergeBranch(name string) error {
	_, err := Run("merge", name)
	return err
}

// DeleteBranch deletes the specified branch
func DeleteBranch(name string) error {
	_, err := Run("branch", "-D", name)
	return err
}

// ListBranches returns all local branches
func ListBranches() ([]BranchInfo, error) {
	output, err := Run("branch", "--format=%(refname:short)|%(HEAD)")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []BranchInfo{}, nil
	}

	var branches []BranchInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			branches = append(branches, BranchInfo{
				Name:      parts[0],
				IsCurrent: parts[1] == "*",
			})
		}
	}
	return branches, nil
}

// ListExperiments returns only experiment branches
func ListExperiments() ([]BranchInfo, error) {
	branches, err := ListBranches()
	if err != nil {
		return nil, err
	}

	var experiments []BranchInfo
	for _, b := range branches {
		if strings.HasPrefix(b.Name, "experiment-") {
			experiments = append(experiments, b)
		}
	}
	return experiments, nil
}

// Stash stashes current changes
func Stash() error {
	_, err := Run("stash")
	return err
}

// StashPop pops the stashed changes
func StashPop() error {
	_, err := Run("stash", "pop")
	return err
}

// IsOnMain checks if we're on the main or master branch
func IsOnMain() bool {
	branch, err := CurrentBranch()
	if err != nil {
		return false
	}
	return branch == "main" || branch == "master"
}

// GetMainBranch returns "main" or "master" depending on what exists
func GetMainBranch() string {
	branches, err := ListBranches()
	if err != nil {
		return "main"
	}
	for _, b := range branches {
		if b.Name == "main" {
			return "main"
		}
		if b.Name == "master" {
			return "master"
		}
	}
	return "main"
}

// KeepExperiment merges current experiment into main and switches to main
func KeepExperiment() error {
	currentBranch, err := CurrentBranch()
	if err != nil {
		return err
	}

	mainBranch := GetMainBranch()

	// Switch to main
	if err := SwitchBranch(mainBranch); err != nil {
		return err
	}

	// Merge the experiment
	if err := MergeBranch(currentBranch); err != nil {
		// Switch back if merge fails
		SwitchBranch(currentBranch)
		return err
	}

	return nil
}

// AbandonExperiment deletes the current experiment and switches to main
func AbandonExperiment() error {
	currentBranch, err := CurrentBranch()
	if err != nil {
		return err
	}

	mainBranch := GetMainBranch()

	// Switch to main first
	if err := SwitchBranch(mainBranch); err != nil {
		return err
	}

	// Delete the experiment branch
	return DeleteBranch(currentBranch)
}

// BackupInfo represents a backup branch
type BackupInfo struct {
	Name       string
	ForBranch  string
	Timestamp  string
	CommitHash string
	Message    string
}

// CreateBackup creates a backup branch for the current state
// Format: backup/<branch-name>/<timestamp>
func CreateBackup(forBranch string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("backup/%s/%s", forBranch, timestamp)

	// Create the backup branch at current HEAD without switching to it
	_, err := Run("branch", backupName)
	if err != nil {
		return "", err
	}

	return backupName, nil
}

// ListBackups returns all backups for a specific branch
func ListBackups(forBranch string) ([]BackupInfo, error) {
	prefix := fmt.Sprintf("backup/%s/", forBranch)

	// Get all branches matching the backup pattern
	output, err := Run("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []BackupInfo{}, nil
	}

	var backups []BackupInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			// Extract timestamp from branch name
			timestamp := strings.TrimPrefix(line, prefix)

			// Get the commit info for this backup
			commitInfo, err := Run("log", "-1", "--format=%h|%s", line)
			if err != nil {
				continue
			}

			parts := strings.SplitN(commitInfo, "|", 2)
			hash := ""
			message := ""
			if len(parts) >= 1 {
				hash = parts[0]
			}
			if len(parts) >= 2 {
				message = parts[1]
			}

			backups = append(backups, BackupInfo{
				Name:       line,
				ForBranch:  forBranch,
				Timestamp:  timestamp,
				CommitHash: hash,
				Message:    message,
			})
		}
	}

	// Reverse to show newest first
	for i, j := 0, len(backups)-1; i < j; i, j = i+1, j-1 {
		backups[i], backups[j] = backups[j], backups[i]
	}

	return backups, nil
}

// RestoreBackup restores from a backup branch
func RestoreBackup(backupBranch string) error {
	return ResetHard(backupBranch)
}

// DeleteBackup deletes a backup branch
func DeleteBackup(backupBranch string) error {
	return DeleteBranch(backupBranch)
}

// GetFileDiff returns the diff for a specific file
func GetFileDiff(path string) string {
	// Check if this is a directory (e.g., untracked directories from git status)
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return fmt.Sprintf("new directory: %s\n(contains untracked files)", path)
	}

	// Try diff against HEAD first (for tracked files)
	output, err := Run("diff", "HEAD", "--", path)
	if err != nil || output == "" {
		// Try without HEAD for new repos
		output, _ = Run("diff", "--", path)
	}

	// For untracked files, show the file content as "added"
	if output == "" {
		status, _ := Run("status", "--porcelain", "--", path)
		if strings.HasPrefix(status, "??") {
			// Untracked file - show content as new file
			content, err := os.ReadFile(path)
			if err != nil {
				return "Error reading file"
			}
			lines := strings.Split(string(content), "\n")
			var result strings.Builder
			result.WriteString(fmt.Sprintf("new file: %s\n", path))
			result.WriteString("---\n")
			for i, line := range lines {
				if i == len(lines)-1 && line == "" {
					continue // Skip trailing empty line
				}
				result.WriteString("+ " + line + "\n")
			}
			return result.String()
		}
	}

	if output == "" {
		return "No changes in this file"
	}
	return output
}

// RevertFile discards changes for a specific file, restoring it to HEAD
func RevertFile(path string) error {
	_, err := Run("checkout", "HEAD", "--", path)
	return err
}

// RevertFiles discards changes for multiple files
func RevertFiles(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := append([]string{"checkout", "HEAD", "--"}, paths...)
	_, err := Run(args...)
	return err
}

// DiffStat represents the diff statistics for a file
type DiffStat struct {
	Path      string
	Additions int
	Deletions int
	IsBinary  bool
}

// CommitDiffSummary represents the summary of changes between commits
type CommitDiffSummary struct {
	Files        []DiffStat
	TotalAdded   int
	TotalDeleted int
}

// GetDiffStatBetweenCommits returns the diff stats between two commits
// If toHash is empty, compares fromHash to HEAD
func GetDiffStatBetweenCommits(fromHash, toHash string) (CommitDiffSummary, error) {
	var summary CommitDiffSummary

	// Build the diff command
	args := []string{"diff", "--numstat"}
	if toHash == "" {
		args = append(args, fromHash)
	} else {
		args = append(args, fromHash, toHash)
	}

	output, err := Run(args...)
	if err != nil {
		return summary, err
	}

	if output == "" {
		return summary, nil
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		stat := DiffStat{
			Path: parts[2],
		}

		// Binary files show "-" for additions/deletions
		if parts[0] == "-" {
			stat.IsBinary = true
		} else {
			fmt.Sscanf(parts[0], "%d", &stat.Additions)
			fmt.Sscanf(parts[1], "%d", &stat.Deletions)
			summary.TotalAdded += stat.Additions
			summary.TotalDeleted += stat.Deletions
		}

		summary.Files = append(summary.Files, stat)
	}

	return summary, nil
}

// GetUncommittedDiffStat returns the diff stats for uncommitted changes
func GetUncommittedDiffStat() (CommitDiffSummary, error) {
	var summary CommitDiffSummary

	// Get diff stats for tracked files
	output, err := Run("diff", "--numstat", "HEAD")
	if err != nil {
		// Try without HEAD for new repos
		output, _ = Run("diff", "--numstat")
	}

	if output != "" {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}

			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}

			stat := DiffStat{
				Path: parts[2],
			}

			if parts[0] == "-" {
				stat.IsBinary = true
			} else {
				fmt.Sscanf(parts[0], "%d", &stat.Additions)
				fmt.Sscanf(parts[1], "%d", &stat.Deletions)
				summary.TotalAdded += stat.Additions
				summary.TotalDeleted += stat.Deletions
			}

			summary.Files = append(summary.Files, stat)
		}
	}

	// Also get untracked files
	status, _ := Run("status", "--porcelain")
	if status != "" {
		for _, line := range strings.Split(status, "\n") {
			if strings.HasPrefix(line, "?? ") {
				path := strings.TrimPrefix(line, "?? ")
				lineCount := countFileLines(path)
				summary.Files = append(summary.Files, DiffStat{
					Path:      path + " (new)",
					Additions: lineCount,
				})
				summary.TotalAdded += lineCount
			}
		}
	}

	return summary, nil
}

// TrimBackups removes old backups beyond the maxCount limit for a branch
// Keeps the newest backups and deletes the oldest ones
func TrimBackups(forBranch string, maxCount int) error {
	if maxCount < 1 {
		maxCount = 1
	}

	backups, err := ListBackups(forBranch)
	if err != nil {
		return err
	}

	// ListBackups returns newest first, so we keep the first maxCount
	if len(backups) <= maxCount {
		return nil
	}

	// Delete backups beyond the limit (oldest ones)
	for i := maxCount; i < len(backups); i++ {
		if err := DeleteBackup(backups[i].Name); err != nil {
			// Continue trying to delete others even if one fails
			continue
		}
	}

	return nil
}
