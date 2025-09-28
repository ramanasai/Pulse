package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/ramanasai/pulse/internal/db"
	"github.com/spf13/cobra"
)

var (
	startProject string
	startTags    string
	allowMulti   bool
)

// startCmd begins a new active timer entry. By default it enforces a single active timer.
var startCmd = &cobra.Command{
	Use:   "start [text]",
	Short: "Start a timer",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbh, err := db.Open()
		if err != nil {
			return err
		}
		defer dbh.Close()

		if !allowMulti {
			var n int
			if err := dbh.QueryRow(`SELECT count(1) FROM entries WHERE category='timer' AND instr(tags,'active')>0`).Scan(&n); err != nil {
				return err
			}
			if n > 0 {
				return fmt.Errorf("an active timer already exists (use --allow-multiple to override)")
			}
		}

		text := strings.Join(args, " ")
		// Ensure "active" tag is present only once
		tags := strings.Trim(strings.ReplaceAll(startTags+",active", ",,", ","), ", ")
		res, err := dbh.Exec(`INSERT INTO entries(category, text, project, tags) VALUES('timer', ?, ?, ?)`, text, startProject, tags)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		fmt.Printf("Timer #%d started at %s\n", id, time.Now().Format(time.Kitchen))
		return nil
	},
}

func init() {
	startCmd.Flags().StringVarP(&startProject, "project", "p", "", "Project name")
	startCmd.Flags().StringVarP(&startTags, "tags", "t", "", "Additional comma separated tags")
	startCmd.Flags().BoolVar(&allowMulti, "allow-multiple", false, "Allow multiple concurrent active timers")
}
