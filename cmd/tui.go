package cmd

import (
	"database/sql"
	"github.com/spf13/cobra"
	"github.com/ramanasai/pulse/internal/db"
	"github.com/ramanasai/pulse/internal/ui"
)

// tuiCmd launches the Bubble Tea TUI.
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbh, err := db.Open()
		if err != nil {
			return err
		}
		defer dbh.Close()

		rows, err := dbh.Query(`
			SELECT '['||substr(ts,12,5)||'] '||COALESCE(project,'')||
			       CASE WHEN project IS NULL OR project='' THEN '' ELSE ' ' END || text
			FROM entries
			ORDER BY ts DESC
			LIMIT 200
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		var list []string
		for rows.Next() {
			var s sql.NullString
			if err := rows.Scan(&s); err != nil {
				return err
			}
			list = append(list, s.String)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return ui.Run(list)
	},
}
