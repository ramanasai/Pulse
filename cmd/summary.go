package cmd

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ramanasai/pulse/internal/db"
	"github.com/spf13/cobra"
)

// summaryCmd prints a per-category breakdown for today and totals.
var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Daily summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbh, err := db.Open()
		if err != nil {
			return err
		}
		defer dbh.Close()

		start := time.Now().Truncate(24 * time.Hour)
		rows, err := dbh.Query(`
			SELECT category, COUNT(*), COALESCE(SUM(duration_minutes),0)
			FROM entries
			WHERE ts >= ?
			GROUP BY category
			ORDER BY category ASC
		`, start.Format(time.RFC3339))
		if err != nil {
			return err
		}
		defer rows.Close()

	fmt.Printf("Today (%s):\n", start.Format("2006-01-02"))
		var totalCount int64
		var totalMins int64
		for rows.Next() {
			var cat string
			var n, mins sql.NullInt64
			if err := rows.Scan(&cat, &n, &mins); err != nil {
				return err
			}
	fmt.Printf("  %-10s %3d items, %4d mins\n", cat, n.Int64, mins.Int64)
			totalCount += n.Int64
			totalMins += mins.Int64
		}
		if err := rows.Err(); err != nil {
			return err
		}
		fmt.Printf("  %-10s %3d items, %4d mins\n", "TOTAL", totalCount, totalMins)
		return nil
	},
}
