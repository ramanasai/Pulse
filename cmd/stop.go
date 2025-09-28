package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ramanasai/pulse/internal/db"
	"github.com/ramanasai/pulse/internal/notify"
	"github.com/spf13/cobra"
)

var (
	stopID   int64
	stopNote string
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop an active timer",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbh, err := db.Open()
		if err != nil {
			return err
		}
		defer dbh.Close()

		// Find target timer
		var id int64
		var ts string
		var txt, tags string
		if stopID > 0 {
			row := dbh.QueryRow(`SELECT id, ts, text, coalesce(tags,'') FROM entries WHERE id=? AND category='timer'`, stopID)
			if err := row.Scan(&id, &ts, &txt, &tags); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("timer #%d not found", stopID)
				}
				return err
			}
			if !strings.Contains(tags, "active") {
				return fmt.Errorf("timer #%d is not active", stopID)
			}
		} else {
			row := dbh.QueryRow(`SELECT id, ts, text, coalesce(tags,'') FROM entries WHERE category='timer' AND instr(tags,'active')>0 ORDER BY ts DESC LIMIT 1`)
			if err := row.Scan(&id, &ts, &txt, &tags); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("no active timers")
				}
				return err
			}
		}

		start, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			// fallback for RFC3339 without nanos
			start, err = time.Parse(time.RFC3339, ts)
		}
		if err != nil {
			return fmt.Errorf("bad start time in DB: %w", err)
		}

		durMin := int(time.Since(start).Minutes())
		if durMin < 0 {
			durMin = 0
		}

		// Update: remove 'active', append optional stop note
		newTags := strings.ReplaceAll(tags, "active", "")
		newTags = strings.Trim(strings.ReplaceAll(newTags, ",,", ","), ", ")
		newText := txt
		if strings.TrimSpace(stopNote) != "" {
			sep := "\n"
			if strings.Contains(newText, "\n") {
				sep = "\n\n"
			}
			newText = newText + sep + "Stop note: " + stopNote
		}

		_, err = dbh.Exec(`UPDATE entries SET duration_minutes=?, tags=?, text=? WHERE id=?`, durMin, newTags, newText, id)
		if err != nil {
			return err
		}

		msg := fmt.Sprintf("Timer #%d stopped: %d minutes", id, durMin)
		fmt.Println(msg)
		_ = notify.Done(msg)
		return nil
	},
}

func init() {
	stopCmd.Flags().Int64VarP(&stopID, "id", "i", 0, "Specific timer id to stop")
	stopCmd.Flags().StringVarP(&stopNote, "note", "n", "", "Optional note to append when stopping")
}
