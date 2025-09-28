package notify

import (
	"fmt"

	"github.com/gen2brain/beeep"
)

func Info(title, message string) error {
	return beeep.Notify(title, message, "")
}

func Done(message string) error {
	return beeep.Alert("Pulse", message, "")
}

func FormatDailyPrompt(pending int) (string, string) {
	title := "Daily log reminder"
	msg := fmt.Sprintf("You have %d pending logs. Jot down today's notes?", pending)
	return title, msg
}
