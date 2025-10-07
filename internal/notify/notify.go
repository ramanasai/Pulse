package notify

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/esiqveland/notify"
	"github.com/godbus/dbus/v5"
	"github.com/ramanasai/pulse/internal/config"
)

// Notification types
type NotificationType int

const (
	NotificationDailyReminder NotificationType = iota
	NotificationPomodoroWork
	NotificationPomodoroBreak
	NotificationEntryCreated
	NotificationGeneral
)

// SendNotification sends a desktop notification if enabled in config
func SendNotification(cfg config.NotificationConfig, notificationType NotificationType, title, message string) error {
	if !cfg.Enabled {
		return nil // Notifications disabled
	}

	// Check if this notification type is enabled
	switch notificationType {
	case NotificationDailyReminder:
		if !cfg.DailyReminders {
			return nil
		}
	case NotificationPomodoroWork, NotificationPomodoroBreak:
		if !cfg.PomodoroSessions {
			return nil
		}
	case NotificationEntryCreated:
		if !cfg.EntryCreated {
			return nil
		}
	}

	// Send the notification
	return Info(title, message)
}

func Info(title, message string) error {
	return sendDesktopNotification(title, message)
}

func Done(message string) error {
	return sendDesktopNotification("Pulse", message)
}

// sendDesktopNotification sends a notification using the modern notify library
func sendDesktopNotification(title, message string) error {
	// Try desktop notifications first
	if err := tryDesktopNotification(title, message); err == nil {
		return nil // Success
	}

	// Fallback to platform-specific alternatives
	return tryNotificationFallback(title, message)
}

// tryDesktopNotification attempts to send a desktop notification
func tryDesktopNotification(title, message string) error {
	conn, err := dbus.SessionBusPrivate()
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.Auth(nil); err != nil {
		return err
	}

	if err := conn.Hello(); err != nil {
		return err
	}

	notifyClient, err := notify.New(conn)
	if err != nil {
		return err
	}

	n := notify.Notification{
		AppName:       "Pulse",
		Summary:       title,
		Body:          message,
		ExpireTimeout: 5000 * time.Millisecond,
	}

	_, err = notifyClient.SendNotification(n)
	return err
}

// tryNotificationFallback provides platform-specific fallbacks
func tryNotificationFallback(title, message string) error {
	// Suppress fallback in terminal/CI environments
	if isTerminalEnvironment() {
		return nil
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS: Use osascript for notifications
		return tryMacOSNotification(title, message)
	case "linux":
		// Linux: Try notify-send as fallback
		return tryLinuxNotification(title, message)
	case "windows":
		// Windows: Could add PowerShell toast notifications here
		fmt.Printf("🔕 Desktop notifications unavailable on Windows - Pulse will continue without notifications\n")
	default:
		fmt.Printf("🔕 Desktop notifications unavailable on %s - Pulse will continue without notifications\n", runtime.GOOS)
	}
	return nil
}

// isTerminalEnvironment checks if we're running in a terminal/CI environment
func isTerminalEnvironment() bool {
	// Check for CI environment variables
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("TERM_PROGRAM") != "" {
		return true
	}

	// Check if we're in a non-interactive shell
	if os.Getenv("TERM") == "dumb" || os.Getenv("TERM") == "" {
		return true
	}

	return false
}

func FormatDailyPrompt(pending int) (string, string) {
	title := "📅 Daily log reminder"
	msg := fmt.Sprintf("You have %d pending logs. Jot down today's notes?", pending)
	return title, msg
}

// Pomodoro notifications
func FormatPomodoroWorkComplete(sessionNumber int, totalSessions int) (string, string) {
	title := "🍅 Pomodoro Work Session Complete"
	msg := fmt.Sprintf("Session #%d completed. Total sessions today: %d", sessionNumber, totalSessions)
	return title, msg
}

func FormatPomodoroBreakComplete() (string, string) {
	title := "☕ Pomodoro Break Complete"
	msg := "Break completed! Back to work 💪"
	return title, msg
}

func FormatPomodoroLongBreak() (string, string) {
	title := "🎉 Long Break Time!"
	msg := "4 sessions completed! Time for a 15-minute break"
	return title, msg
}

// Entry notifications
func FormatEntryCreated(entryType string) (string, string) {
	title := "📝 Entry Created"
	msg := fmt.Sprintf("New %s entry saved", entryType)
	return title, msg
}

// Platform-specific notification fallbacks

// tryMacOSNotification uses osascript to send a notification on macOS
func tryMacOSNotification(title, message string) error {
	// Escape quotes for shell command
	escapedTitle := strings.ReplaceAll(title, `"`, `\"`)
	escapedMessage := strings.ReplaceAll(message, `"`, `\"`)

	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`display notification "%s" with title "%s" subtitle "Pulse"`, escapedMessage, escapedTitle))
	if err := cmd.Run(); err != nil {
		fmt.Printf("🔕 macOS notifications unavailable - Pulse will continue without notifications\n")
		return err
	}
	return nil
}

// tryLinuxNotification uses notify-send as fallback on Linux
func tryLinuxNotification(title, message string) error {
	// Check if notify-send is available
	if _, err := exec.LookPath("notify-send"); err != nil {
		fmt.Printf("🔕 Linux notifications unavailable (notify-send not found) - Pulse will continue without notifications\n")
		return err
	}

	cmd := exec.Command("notify-send", "-i", "dialog-information", "-t", "5000", "Pulse - "+title, message)
	if err := cmd.Run(); err != nil {
		fmt.Printf("🔕 Linux notifications failed - Pulse will continue without notifications\n")
		return err
	}
	return nil
}
