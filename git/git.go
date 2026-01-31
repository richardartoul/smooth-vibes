package git

import (
	"fmt"
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

// Run executes a git command and returns the output
func Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
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

// Commit creates a commit with the given message
func Commit(message string) error {
	_, err := Run("commit", "-m", message)
	return err
}

// Push pushes the current branch to origin
func Push() error {
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

