package config

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"config_handler/git"
	"config_handler/notification"
	"config_handler/ui"

	"github.com/fsnotify/fsnotify"
	"github.com/gobwas/glob"
)

// Manager handles configuration file management
type Manager struct {
	ConfigDir     string
	RepoDir       string
	GitRepo       *git.GitRepo
	FileWatcher   *fsnotify.Watcher
	IncludeGlobs  []glob.Glob
	ExcludeGlobs  []glob.Glob
	SyncInterval  time.Duration
	Verbose       bool
	NotifyManager *notification.Manager
}

// NewManager creates a new configuration manager
func NewManager(configDir, repoDir string, gitRepo *git.GitRepo, includePatterns, excludePatterns []string, syncInterval time.Duration, verbose bool, notifyManager *notification.Manager) *Manager {
	// Compile include/exclude patterns to glob matchers
	includeGlobs := make([]glob.Glob, 0, len(includePatterns))
	for _, pattern := range includePatterns {
		if pattern != "" {
			g, err := glob.Compile(pattern)
			if err == nil {
				includeGlobs = append(includeGlobs, g)
			}
		}
	}

	excludeGlobs := make([]glob.Glob, 0, len(excludePatterns))
	for _, pattern := range excludePatterns {
		if pattern != "" {
			g, err := glob.Compile(pattern)
			if err == nil {
				excludeGlobs = append(excludeGlobs, g)
			}
		}
	}

	return &Manager{
		ConfigDir:     configDir,
		RepoDir:       repoDir,
		GitRepo:       gitRepo,
		IncludeGlobs:  includeGlobs,
		ExcludeGlobs:  excludeGlobs,
		SyncInterval:  syncInterval,
		Verbose:       verbose,
		NotifyManager: notifyManager,
	}
}

// shouldInclude determines if a file/directory should be included based on patterns
func (m *Manager) shouldInclude(relPath string) bool {
	// Skip git metadata
	if strings.Contains(relPath, ".git") {
		return false
	}

	// Check exclude patterns (exclude takes precedence)
	for _, pattern := range m.ExcludeGlobs {
		if pattern.Match(relPath) {
			if m.Verbose {
				ui.PrintInfo(fmt.Sprintf("Excluding %s (matched exclude pattern)", relPath))
			}
			return false
		}
	}

	// If include patterns are specified, path must match at least one
	if len(m.IncludeGlobs) > 0 {
		for _, pattern := range m.IncludeGlobs {
			if pattern.Match(relPath) {
				return true
			}
		}
		// If we have include patterns but none matched, exclude the path
		return false
	}

	// No include patterns specified, include everything not excluded
	return true
}

// InitialSync copies all configuration files to the repo
func (m *Manager) InitialSync() error {
	// Create the repository directory if it doesn't exist
	if _, err := os.Stat(m.RepoDir); os.IsNotExist(err) {
		err = os.MkdirAll(m.RepoDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create repository directory: %w", err)
		}
	}

	fileCount := 0
	dirCount := 0

	// Walk the config directory and copy files to the repo
	err := filepath.Walk(m.ConfigDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from config directory
		relPath, err := filepath.Rel(m.ConfigDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip the root directory
		if relPath == "." {
			return nil
		}

		// Check if this path should be included
		if !m.shouldInclude(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Target path in the repo
		targetPath := filepath.Join(m.RepoDir, relPath)

		if info.IsDir() {
			// Create directory if it doesn't exist
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				err = os.MkdirAll(targetPath, info.Mode())
				if err != nil {
					return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
				}
				dirCount++
				if dirCount <= 5 || m.Verbose {
					ui.PrintFileOperation("added", "dir: "+relPath)
				}
			}
		} else {
			// Make sure the target directory exists
			err = os.MkdirAll(filepath.Dir(targetPath), 0755)
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(targetPath), err)
			}

			// Copy the file
			err = copyFile(path, targetPath)
			if err != nil {
				return fmt.Errorf("failed to copy file %s to %s: %w", path, targetPath, err)
			}
			fileCount++
			if fileCount <= 10 || m.Verbose {
				ui.PrintFileOperation("added", relPath)
			} else if fileCount == 11 && !m.Verbose {
				ui.PrintInfo("... and more files")
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to sync config files: %w", err)
	}

	// Commit and push changes
	ui.PrintInfo(fmt.Sprintf("Synchronized %d files and %d directories", fileCount, dirCount))
	ui.PrintInfo("Committing changes to repository...")

	err = m.GitRepo.SyncWithRemote("Initial sync of configuration files")
	if err != nil {
		return fmt.Errorf("failed to sync with remote: %w", err)
	}

	return nil
}

// StartWatcher initializes and starts a file watcher to detect changes
func (m *Manager) StartWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	m.FileWatcher = watcher

	// Start watching the config directory recursively
	err = filepath.Walk(m.ConfigDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from config directory
		relPath, err := filepath.Rel(m.ConfigDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip the root directory
		if relPath == "." {
			return nil
		}

		// Check if this path should be included
		if !m.shouldInclude(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			err = watcher.Add(path)
			if err != nil {
				log.Printf("Error watching directory %s: %v", path, err)
			} else if m.Verbose {
				ui.PrintInfo(fmt.Sprintf("Watching directory: %s", relPath))
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to set up file watching: %w", err)
	}

	// Start the watcher goroutine
	go m.watcherLoop()

	return nil
}

// watcherLoop handles file system events
func (m *Manager) watcherLoop() {
	debounceEvents := make(map[string]time.Time)
	syncTicker := time.NewTicker(m.SyncInterval)

	for {
		select {
		case event, ok := <-m.FileWatcher.Events:
			if !ok {
				return
			}

			// Calculate relative path
			relPath, err := filepath.Rel(m.ConfigDir, event.Name)
			if err != nil {
				continue
			}

			// Check if this path should be included
			if !m.shouldInclude(relPath) {
				continue
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				// Debounce by recording the event time
				debounceEvents[event.Name] = time.Now()

				// If it's a new directory, add it to the watcher
				if event.Op&fsnotify.Create != 0 {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						m.FileWatcher.Add(event.Name)
						if m.Verbose {
							ui.PrintInfo(fmt.Sprintf("Added new directory to watch: %s", relPath))
						}
					}
				}
			}

		case err, ok := <-m.FileWatcher.Errors:
			if !ok {
				return
			}
			ui.PrintError("Watcher error: " + err.Error())

		case <-syncTicker.C:
			// Process debounced events
			now := time.Now()
			filesToSync := make(map[string]bool)

			for eventPath, eventTime := range debounceEvents {
				// Only process events older than 1 second
				if now.Sub(eventTime) > 1*time.Second {
					// Get the relative path for syncing
					relPath, err := filepath.Rel(m.ConfigDir, eventPath)
					if err == nil {
						filesToSync[relPath] = true
					}
					delete(debounceEvents, eventPath)
				}
			}

			if len(filesToSync) > 0 {
				// Sync changed files
				m.syncChangedFiles(filesToSync)
			}
		}
	}
}

// syncChangedFiles synchronizes changed files with the repository
func (m *Manager) syncChangedFiles(changedFiles map[string]bool) {
	fileChangeSummary := make(map[string]string) // Track types of changes for commit message
	fileChanges := make(map[string][]string)     // Store file paths by operation for UI display

	for relPath := range changedFiles {
		sourcePath := filepath.Join(m.ConfigDir, relPath)
		targetPath := filepath.Join(m.RepoDir, relPath)

		info, err := os.Stat(sourcePath)
		if os.IsNotExist(err) {
			// File or directory was deleted
			targetInfo, targetErr := os.Stat(targetPath)
			if targetErr == nil {
				if targetInfo.IsDir() {
					// Use the helper function to recursively remove directory
					err = removeDirectory(targetPath)
					if err != nil {
						ui.PrintError("Failed to remove directory " + relPath + ": " + err.Error())
					} else {
						if fileChanges["deleted"] == nil {
							fileChanges["deleted"] = []string{}
						}
						fileChanges["deleted"] = append(fileChanges["deleted"], "dir: "+relPath)
						fileChangeSummary["deleted"] = fileChangeSummary["deleted"] + relPath + ", "
					}
				} else {
					// It's a file, just remove it
					err = os.Remove(targetPath)
					if err != nil && !os.IsNotExist(err) {
						ui.PrintError("Failed to remove file " + relPath + ": " + err.Error())
					} else {
						if fileChanges["deleted"] == nil {
							fileChanges["deleted"] = []string{}
						}
						fileChanges["deleted"] = append(fileChanges["deleted"], relPath)
						fileChangeSummary["deleted"] = fileChangeSummary["deleted"] + relPath + ", "
					}
				}
			}
		} else if err == nil {
			if info.IsDir() {
				// Make sure the directory exists in the repo
				err = os.MkdirAll(targetPath, info.Mode())
				if err != nil {
					ui.PrintError("Failed to create directory " + relPath + ": " + err.Error())
				} else {
					// Check if this is a new directory
					if _, statErr := os.Stat(targetPath); os.IsNotExist(statErr) {
						if fileChanges["added"] == nil {
							fileChanges["added"] = []string{}
						}
						fileChanges["added"] = append(fileChanges["added"], "dir: "+relPath)
						fileChangeSummary["added"] = fileChangeSummary["added"] + relPath + ", "
					}
				}
			} else {
				// Make sure the target directory exists
				err = os.MkdirAll(filepath.Dir(targetPath), 0755)
				if err != nil {
					ui.PrintError("Failed to create directory " + filepath.Dir(relPath) + ": " + err.Error())
					continue
				}

				// Check if this is a new file or modified file
				fileOperation := "modified"
				if _, statErr := os.Stat(targetPath); os.IsNotExist(statErr) {
					fileOperation = "added"
				}

				// Copy the file
				err = copyFile(sourcePath, targetPath)
				if err != nil {
					ui.PrintError("Failed to copy file " + relPath + ": " + err.Error())
					continue
				}

				if fileChanges[fileOperation] == nil {
					fileChanges[fileOperation] = []string{}
				}
				fileChanges[fileOperation] = append(fileChanges[fileOperation], relPath)
				fileChangeSummary[fileOperation] = fileChangeSummary[fileOperation] + relPath + ", "
			}
		}
	}

	// Display changes with beautiful formatting
	ui.PrintSection("Changes Detected")

	// Show added files
	if len(fileChanges["added"]) > 0 {
		ui.PrintInfo("Added files:")
		for i, file := range fileChanges["added"] {
			if i < 5 {
				ui.PrintFileOperation("added", file)
			} else {
				ui.PrintInfo(fmt.Sprintf("... and %d more added files", len(fileChanges["added"])-5))
				break
			}
		}
	}

	// Show modified files
	if len(fileChanges["modified"]) > 0 {
		ui.PrintInfo("Modified files:")
		for i, file := range fileChanges["modified"] {
			if i < 5 {
				ui.PrintFileOperation("modified", file)
			} else {
				ui.PrintInfo(fmt.Sprintf("... and %d more modified files", len(fileChanges["modified"])-5))
				break
			}
		}
	}

	// Show deleted files
	if len(fileChanges["deleted"]) > 0 {
		ui.PrintInfo("Deleted files:")
		for i, file := range fileChanges["deleted"] {
			if i < 5 {
				ui.PrintFileOperation("deleted", file)
			} else {
				ui.PrintInfo(fmt.Sprintf("... and %d more deleted files", len(fileChanges["deleted"])-5))
				break
			}
		}
	}

	// Send desktop notification about file changes
	if m.NotifyManager != nil {
		changeText := ""

		if len(fileChanges["added"]) > 0 {
			changeText += fmt.Sprintf("Added: %d files, ", len(fileChanges["added"]))
		}
		if len(fileChanges["modified"]) > 0 {
			changeText += fmt.Sprintf("Modified: %d files, ", len(fileChanges["modified"]))
		}
		if len(fileChanges["deleted"]) > 0 {
			changeText += fmt.Sprintf("Deleted: %d files, ", len(fileChanges["deleted"]))
		}

		changeText = strings.TrimSuffix(changeText, ", ")

		// Send notification about the file changes
		m.NotifyManager.FileChangesDetected(changeText)
	}

	// Create a detailed commit message
	commitMsg := buildCommitMessage(fileChangeSummary, len(changedFiles))

	// Sync with remote
	ui.PrintInfo("Syncing changes with remote repository...")
	err := m.GitRepo.SyncWithRemote(commitMsg)
	if err != nil {
		ui.PrintError("Failed to sync with remote: " + err.Error())

		// Send error notification
		if m.NotifyManager != nil {
			m.NotifyManager.SyncError("Failed to sync with remote: " + err.Error())
		}
	} else {
		ui.PrintSuccess(fmt.Sprintf("Successfully synchronized %d changed files", len(changedFiles)))

		// Send success notification
		if m.NotifyManager != nil {
			m.NotifyManager.SyncSuccess(fmt.Sprintf("Successfully synchronized %d files", len(changedFiles)))
		}
	}
	ui.PrintSeparator()
}

// buildCommitMessage creates a descriptive commit message based on file changes
func buildCommitMessage(changes map[string]string, totalChanges int) string {
	var parts []string

	// Remove trailing comma and space from each change type
	for changeType, files := range changes {
		if files != "" {
			// Trim trailing comma and space
			files = strings.TrimSuffix(files, ", ")

			// Limit long lists to 3 files + count of remaining
			fileList := strings.Split(files, ", ")
			if len(fileList) > 3 {
				remaining := len(fileList) - 3
				files = strings.Join(fileList[:3], ", ") + fmt.Sprintf(" and %d more", remaining)
			}

			parts = append(parts, fmt.Sprintf("%s: %s", changeType, files))
		}
	}

	// If we have specific changes, format them nicely
	if len(parts) > 0 {
		return fmt.Sprintf("Configuration update: %s", strings.Join(parts, "; "))
	}

	// Default message if no specific changes were tracked
	return fmt.Sprintf("Updated configuration files: %d files changed", totalChanges)
}

// removeDirectory recursively removes a directory and all its contents
func removeDirectory(path string) error {
	// First, remove all contents of the directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			// Recursively remove subdirectories
			err = removeDirectory(entryPath)
		} else {
			// Remove files
			err = os.Remove(entryPath)
		}

		if err != nil {
			return err
		}
	}

	// Finally, remove the directory itself
	return os.Remove(path)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the contents
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy the permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
