package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"config_handler/ui"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// File status constants for git worktree status
const (
	// Unmerged represents a file with merge conflicts
	Unmerged git.StatusCode = 'U'
)

type GitRepo struct {
	Repository *git.Repository
	Path       string
	RemoteURL  string
	Auth       *http.BasicAuth
}

// InitOrOpenRepo initializes a new git repository or opens an existing one
func InitOrOpenRepo(path string) (*GitRepo, error) {
	// Check if the repository directory exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create the directory
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create repository directory: %w", err)
		}

		// Initialize a new repository
		ui.PrintInfo("Creating new Git repository at " + path)
		repo, err := git.PlainInit(path, false)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize repository: %w", err)
		}

		return &GitRepo{
			Repository: repo,
			Path:       path,
		}, nil
	}

	// Open existing repository
	ui.PrintInfo("Opening existing Git repository at " + path)
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	return &GitRepo{
		Repository: repo,
		Path:       path,
	}, nil
}

// SetRemote sets the remote URL for the repository
func (g *GitRepo) SetRemote(remoteURL string) error {
	// Check if remote already exists
	remote, err := g.Repository.Remote("origin")
	if err == nil && remote != nil {
		// Remote already exists, update it
		ui.PrintInfo("Updating existing remote URL")
		err = g.Repository.DeleteRemote("origin")
		if err != nil {
			return fmt.Errorf("failed to delete existing remote: %w", err)
		}
	} else {
		ui.PrintInfo("Setting up new remote URL")
	}

	// Create new remote
	_, err = g.Repository.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteURL},
	})
	if err != nil {
		return fmt.Errorf("failed to create remote: %w", err)
	}

	g.RemoteURL = remoteURL
	return nil
}

// SetCredentials sets the authentication credentials for the repository
func (g *GitRepo) SetCredentials(username, token string) {
	g.Auth = &http.BasicAuth{
		Username: username,
		Password: token,
	}
}

// Add stages files to the repository
func (g *GitRepo) Add(paths ...string) error {
	w, err := g.Repository.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	for _, path := range paths {
		_, err = w.Add(path)
		if err != nil {
			return fmt.Errorf("failed to add file %s: %w", path, err)
		}
	}

	return nil
}

// Commit commits staged changes to the repository
func (g *GitRepo) Commit(message string) error {
	w, err := g.Repository.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Check if there are changes to commit
	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if status.IsClean() {
		return errors.New("no changes to commit")
	}

	ui.PrintInfo("Committing changes: " + ui.FormatCommitMessage(message))
	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Config Handler",
			Email: "config-handler@automatic.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// Push pushes changes to the remote repository
func (g *GitRepo) Push() error {
	if g.RemoteURL == "" {
		return errors.New("remote URL not set")
	}

	if g.Auth == nil {
		return errors.New("authentication credentials not set")
	}

	ui.PrintInfo("Pushing changes to remote repository...")
	err := g.Repository.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       g.Auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// Pull pulls changes from the remote repository
func (g *GitRepo) Pull() error {
	if g.RemoteURL == "" {
		return errors.New("remote URL not set")
	}

	w, err := g.Repository.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	ui.PrintInfo("Pulling latest changes from remote repository...")
	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       g.Auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull: %w", err)
	}

	return nil
}

// SyncWithRemote syncs local changes with the remote repository
func (g *GitRepo) SyncWithRemote(message string) error {
	// Try to pull latest changes first
	err := g.Pull()
	if err != nil && err != git.NoErrAlreadyUpToDate {
		ui.PrintWarning("Pull warning: " + err.Error())

		// Check if there are merge conflicts to resolve
		if strings.Contains(err.Error(), "conflict") {
			ui.PrintInfo("Detected potential merge conflicts. Attempting to resolve...")

			// Create a conflict resolver
			resolver := NewConflictResolver(g)

			// Detect conflicts
			hasConflicts, err := resolver.DetectConflicts()
			if err != nil {
				return fmt.Errorf("failed to detect conflicts: %w", err)
			}

			if hasConflicts {
				// Show conflicts to user
				conflicts := resolver.GetConflicts()
				ui.PrintWarning(fmt.Sprintf("Found %d file(s) with conflicts:", len(conflicts)))

				for i, conflict := range conflicts {
					ui.PrintInfo(fmt.Sprintf("%d. %s", i+1, conflict.Path))
				}

				// Ask user how they want to resolve conflicts
				ui.PrintInfo("Do you want to use the same strategy for all conflicts?")
				useSameStrategy := ui.PromptYesNo("Use same strategy for all files?", true)

				if useSameStrategy {
					// Use the same strategy for all conflicts
					strategy := promptForResolutionStrategy()

					// Apply resolution strategy to all conflicts
					err = resolver.ResolveAllConflicts(strategy)
					if err != nil {
						return fmt.Errorf("failed to resolve conflicts: %w", err)
					}
				} else {
					// Resolve each conflict with potentially different strategies
					for _, conflict := range conflicts {
						ui.PrintInfo(fmt.Sprintf("Resolving conflict for: %s", conflict.Path))
						strategy := promptForResolutionStrategy()

						err = resolver.ResolveConflict(conflict.Path, strategy)
						if err != nil {
							return fmt.Errorf("failed to resolve conflict for %s: %w", conflict.Path, err)
						}
					}
				}

				ui.PrintSuccess("All conflicts resolved successfully")
			}
		} else {
			// Other pull error, not conflict related
			return fmt.Errorf("failed to pull from remote: %w", err)
		}
	}

	// Add all changes
	err = g.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Commit changes
	err = g.Commit(message)
	if err != nil && err.Error() != "no changes to commit" {
		return fmt.Errorf("failed to commit changes: %w", err)
	} else if err != nil && err.Error() == "no changes to commit" {
		ui.PrintInfo("No changes to commit")
		return nil
	}

	// Push changes
	err = g.Push()
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	ui.PrintSuccess("Changes pushed to remote repository successfully")
	return nil
}

// ResolutionStrategy defines how to resolve conflicts
type ResolutionStrategy string

const (
	// KeepLocal keeps the local version of a file
	KeepLocal ResolutionStrategy = "local"
	// KeepRemote keeps the remote version of a file
	KeepRemote ResolutionStrategy = "remote"
	// Merge attempts to merge both versions
	Merge ResolutionStrategy = "merge"
)

// ConflictFile represents a file with conflicts
type ConflictFile struct {
	Path      string
	LocalSHA  string
	RemoteSHA string
}

// ConflictResolver handles git merge conflicts
type ConflictResolver struct {
	Repo      *GitRepo
	Conflicts []ConflictFile
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(repo *GitRepo) *ConflictResolver {
	return &ConflictResolver{
		Repo:      repo,
		Conflicts: []ConflictFile{},
	}
}

// DetectConflicts checks for conflicts after a failed pull
func (cr *ConflictResolver) DetectConflicts() (bool, error) {
	cr.Conflicts = []ConflictFile{} // Reset conflicts list

	// Get repository worktree
	w, err := cr.Repo.Repository.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get status to check for conflicts
	status, err := w.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}

	// Look for conflicts in status
	for path, fileStatus := range status {
		if fileStatus.Staging == Unmerged || fileStatus.Worktree == Unmerged {
			// This file has conflicts
			conflict := ConflictFile{
				Path: path,
			}

			// Try to get the SHAs for the different versions
			idx, err := cr.Repo.Repository.Storer.Index()
			if err == nil {
				for _, entry := range idx.Entries {
					if entry.Name == path {
						if entry.Stage == 2 { // "our" version (local)
							conflict.LocalSHA = entry.Hash.String()
						} else if entry.Stage == 3 { // "their" version (remote)
							conflict.RemoteSHA = entry.Hash.String()
						}
					}
				}
			}

			cr.Conflicts = append(cr.Conflicts, conflict)
		}
	}

	return len(cr.Conflicts) > 0, nil
}

// GetConflicts returns the list of conflicting files
func (cr *ConflictResolver) GetConflicts() []ConflictFile {
	return cr.Conflicts
}

// ResolveConflict resolves a conflict for a specific file
func (cr *ConflictResolver) ResolveConflict(path string, strategy ResolutionStrategy) error {
	// Get repository worktree
	w, err := cr.Repo.Repository.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	fullPath := filepath.Join(cr.Repo.Path, path)

	switch strategy {
	case KeepLocal:
		// Resolve by keeping the local version
		ui.PrintInfo("Keeping local version of " + path)

		// Use git checkout --ours
		err = cr.Repo.execGitCheckout(path, "--ours")
		if err != nil {
			return fmt.Errorf("failed to checkout local version: %w", err)
		}

		// Mark as resolved
		_, err = w.Add(path)
		if err != nil {
			return fmt.Errorf("failed to mark as resolved: %w", err)
		}

	case KeepRemote:
		// Resolve by keeping the remote version
		ui.PrintInfo("Keeping remote version of " + path)

		// Use git checkout --theirs
		err = cr.Repo.execGitCheckout(path, "--theirs")
		if err != nil {
			return fmt.Errorf("failed to checkout remote version: %w", err)
		}

		// Mark as resolved
		_, err = w.Add(path)
		if err != nil {
			return fmt.Errorf("failed to mark as resolved: %w", err)
		}

	case Merge:
		// For now, just create a merged file with both versions
		ui.PrintInfo("Creating merged version of " + path)

		// Read the conflict markers file
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read conflict file: %w", err)
		}

		// Create a simple merged version (keeping both with markers)
		// In a real implementation, this would use a proper merge tool
		mergedContent := "# MERGED FILE\n# Contains both versions due to conflict\n\n" +
			string(content)

		// Write back the merged content
		err = os.WriteFile(fullPath, []byte(mergedContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write merged file: %w", err)
		}

		// Mark as resolved
		_, err = w.Add(path)
		if err != nil {
			return fmt.Errorf("failed to mark as resolved: %w", err)
		}
	}

	return nil
}

// ResolveAllConflicts resolves all conflicts using the same strategy
func (cr *ConflictResolver) ResolveAllConflicts(strategy ResolutionStrategy) error {
	for _, conflict := range cr.Conflicts {
		err := cr.ResolveConflict(conflict.Path, strategy)
		if err != nil {
			return fmt.Errorf("failed to resolve conflict for %s: %w", conflict.Path, err)
		}
	}
	return nil
}

// Helper method to execute git checkout commands
func (g *GitRepo) execGitCheckout(path string, option string) error {
	// This is a helper that allows us to run git checkout with options
	// Using the go-git library directly would be better, but it doesn't expose
	// all the checkout options we need, so we're using this workaround

	// Create a temporary script to execute the checkout command
	scriptContent := fmt.Sprintf("cd %s && git checkout %s -- %s",
		g.Path, option, path)

	tempFile := filepath.Join(os.TempDir(), "git_checkout.sh")
	err := os.WriteFile(tempFile, []byte(scriptContent), 0700)
	if err != nil {
		return fmt.Errorf("failed to create checkout script: %w", err)
	}

	// Execute the script
	output, err := ui.ExecuteCommand("/bin/sh", tempFile)
	if err != nil {
		return fmt.Errorf("checkout failed: %s: %w", output, err)
	}

	// Clean up
	os.Remove(tempFile)

	return nil
}

// promptForResolutionStrategy asks the user which strategy to use for conflict resolution
func promptForResolutionStrategy() ResolutionStrategy {
	ui.PrintInfo("How would you like to resolve these conflicts?")
	ui.PrintInfo("1. Keep local changes (your version)")
	ui.PrintInfo("2. Keep remote changes (remote version)")
	ui.PrintInfo("3. Create merged files (with both versions)")

	for {
		choice := ui.PromptInput("Enter choice (1-3)", "3")

		switch choice {
		case "1":
			return KeepLocal
		case "2":
			return KeepRemote
		case "3":
			return Merge
		default:
			ui.PrintWarning("Invalid choice. Please enter 1, 2, or 3.")
		}
	}
}
