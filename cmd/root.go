package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/ramanasai/pulse/internal/config"
	"github.com/ramanasai/pulse/internal/notify"
	"github.com/ramanasai/pulse/internal/schedule"
)

var rootCmd = &cobra.Command{
	Use:   "pulse",
	Short: "Personal logging & tracking",
}

func Execute() error { return rootCmd.Execute() }

func init() {
	// Load config and start reminder if enabled
	cfg, _ := config.Load()

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cfg.Reminder.Enabled && os.Getenv("PULSE_NO_REMINDER") != "1" {
			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			go func() {
				schedule.RunConfigured(ctx, cfg, func() {
					title, msg := notify.FormatDailyPrompt(0) // TODO: compute pending
					_ = notify.Info(title, msg)
				})
			}()
			// We intentionally don't store cancel globally; on process exit, signal cancels
			_ = cancel // avoid unused if we change logic
		}
		return nil
	}

	// Add commands; other files define these vars
	rootCmd.AddCommand(logCmd, listCmd, startCmd, stopCmd, summaryCmd, searchCmd, editCmd)
}
