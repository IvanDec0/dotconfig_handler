// Package notification provides desktop notification functionality
package notification

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// Icon types for desktop notifications
const (
	IconInfo     = "info"
	IconWarning  = "warning"
	IconError    = "error"
	IconQuestion = "question"
)

// DesktopNotification represents a desktop notification with additional options
type DesktopNotification struct {
	Title      string
	Message    string
	Icon       string
	Urgency    string
	TimeoutSec int
}

// NewDesktopNotification creates a new desktop notification with default values
func NewDesktopNotification(title, message string) *DesktopNotification {
	return &DesktopNotification{
		Title:      title,
		Message:    message,
		Icon:       IconInfo,
		Urgency:    "normal",
		TimeoutSec: 5,
	}
}

// Send sends the desktop notification
func (n *DesktopNotification) Send(appName string) error {
	switch runtime.GOOS {
	case "linux":
		return n.sendLinux(appName)
	case "darwin":
		return n.sendMacOS(appName)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// sendLinux sends a notification on Linux using notify-send
func (n *DesktopNotification) sendLinux(appName string) error {
	args := []string{
		"--app-name=" + appName,
		"--icon=" + n.Icon,
		"--urgency=" + n.Urgency,
		"--expire-time=" + fmt.Sprintf("%d", n.TimeoutSec*1000),
		n.Title,
		n.Message,
	}

	cmd := exec.Command("notify-send", args...)
	return cmd.Run()
}

// sendMacOS sends a notification on macOS using osascript
func (n *DesktopNotification) sendMacOS(appName string) error {
	fullTitle := appName
	if n.Title != "" {
		fullTitle += " - " + n.Title
	}

	// Basic AppleScript for notification
	script := fmt.Sprintf(`display notification "%s" with title "%s"`,
		n.Message, fullTitle)

	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// SendWithTimeout sends a notification with a custom timeout and waits for completion
func (n *DesktopNotification) SendWithTimeout(appName string, timeoutSec int) error {
	n.TimeoutSec = timeoutSec
	err := n.Send(appName)

	// Wait for the notification to be displayed
	if err == nil {
		time.Sleep(time.Duration(timeoutSec) * time.Second)
	}

	return err
}

// SendInfoNotification sends an info notification
func SendInfoNotification(appName, title, message string) error {
	notification := NewDesktopNotification(title, message)
	notification.Icon = IconInfo
	return notification.Send(appName)
}

// SendWarningNotification sends a warning notification
func SendWarningNotification(appName, title, message string) error {
	notification := NewDesktopNotification(title, message)
	notification.Icon = IconWarning
	notification.Urgency = "critical"
	return notification.Send(appName)
}

// SendErrorNotification sends an error notification
func SendErrorNotification(appName, title, message string) error {
	notification := NewDesktopNotification(title, message)
	notification.Icon = IconError
	notification.Urgency = "critical"
	return notification.Send(appName)
}
