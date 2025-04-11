// Package notification provides functionality to notify users about events in the config_handler
package notification

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"config_handler/ui"
)

// NotificationType represents the type of a notification
type NotificationType string

// Notification types
const (
	TypeInfo     NotificationType = "INFO"
	TypeWarning  NotificationType = "WARNING"
	TypeError    NotificationType = "ERROR"
	TypeSuccess  NotificationType = "SUCCESS"
	TypeFileSync NotificationType = "FILE_SYNC"
)

// NotificationConfig holds configuration for the notification system
type NotificationConfig struct {
	// Whether to show desktop notifications
	EnableDesktopNotifications bool
	// Whether to log notifications to a file
	EnableFileLogging bool
	// Path to the log file
	LogFilePath string
	// Application name for desktop notifications
	AppName string
}

// DefaultConfig creates a default notification configuration
func DefaultConfig() *NotificationConfig {
	homeDir, err := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".config_handler", "logs")

	// Create logs directory if it doesn't exist
	if err == nil {
		os.MkdirAll(logPath, 0755)
	} else {
		logPath = "."
	}

	return &NotificationConfig{
		EnableDesktopNotifications: true,
		EnableFileLogging:          true,
		LogFilePath:                filepath.Join(logPath, "config_handler.log"),
		AppName:                    "Config Handler",
	}
}

// Manager handles notifications
type Manager struct {
	Config *NotificationConfig
	logger *log.Logger
}

// NewManager creates a new notification manager
func NewManager(config *NotificationConfig) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var logger *log.Logger
	if config.EnableFileLogging {
		// Create log file (append mode)
		logFile, err := os.OpenFile(config.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		logger = log.New(logFile, "", log.Ldate|log.Ltime)

		// Log startup
		logger.Printf("[%s] Config handler notification system started", TypeInfo)
	}

	return &Manager{
		Config: config,
		logger: logger,
	}, nil
}

// Notify sends a notification
func (m *Manager) Notify(nType NotificationType, title, message string) {
	// Log to file if enabled
	if m.Config.EnableFileLogging && m.logger != nil {
		m.logger.Printf("[%s] %s: %s", nType, title, message)
	}

	// Show terminal notification
	switch nType {
	case TypeInfo:
		ui.PrintInfo(message)
	case TypeWarning:
		ui.PrintWarning(message)
	case TypeError:
		ui.PrintError(message)
	case TypeSuccess:
		ui.PrintSuccess(message)
	case TypeFileSync:
		ui.PrintFileOperation(title, message) // Title would be "added", "modified", or "deleted"
	}

	// Show desktop notification if enabled
	if m.Config.EnableDesktopNotifications {
		m.sendDesktopNotification(title, message)
	}
}

// sendDesktopNotification sends a desktop notification using platform-specific methods
func (m *Manager) sendDesktopNotification(title, message string) {
	// Different methods based on OS
	switch runtime.GOOS {
	case "linux":
		// Try notify-send for Linux
		exec.Command("notify-send", m.Config.AppName+" - "+title, message).Run()
	case "darwin":
		// Try osascript for macOS
		appleScript := fmt.Sprintf(`display notification "%s" with title "%s"`, message, m.Config.AppName+" - "+title)
		exec.Command("osascript", "-e", appleScript).Run()
	case "windows":
		// Windows notifications are more complex, usually requiring a COM interface
		// For simplicity, we'll just log that we can't show them
		if m.logger != nil {
			m.logger.Printf("Desktop notifications on Windows not yet implemented")
		}
	}
}

// Close performs cleanup for the notification manager
func (m *Manager) Close() {
	if m.Config.EnableFileLogging && m.logger != nil {
		// Log shutdown
		m.logger.Printf("[%s] Config handler notification system stopped", TypeInfo)
	}
}

// NotifyFileChange sends a notification about a file change
func (m *Manager) NotifyFileChange(changeType, path string) {
	var title string
	switch changeType {
	case "added":
		title = "File Added"
	case "modified":
		title = "File Modified"
	case "deleted":
		title = "File Deleted"
	default:
		title = "File Changed"
	}

	m.Notify(TypeFileSync, changeType, path)

	// Also send desktop notification with more context
	if m.Config.EnableDesktopNotifications {
		m.sendDesktopNotification(title, path)
	}
}

// NotifySyncComplete sends a notification about a sync operation
func (m *Manager) NotifySyncComplete(success bool, message string) {
	if success {
		m.Notify(TypeSuccess, "Sync Complete", message)
	} else {
		m.Notify(TypeError, "Sync Failed", message)
	}
}

// FileChangesDetected sends a notification when file changes are detected
func (m *Manager) FileChangesDetected(message string) {
	m.Notify(TypeInfo, "Changes Detected", message)
}

// SyncError sends a notification about a sync error
func (m *Manager) SyncError(message string) {
	m.Notify(TypeError, "Sync Error", message)
}

// SyncSuccess sends a notification about a successful sync
func (m *Manager) SyncSuccess(message string) {
	m.Notify(TypeSuccess, "Sync Complete", message)
}
