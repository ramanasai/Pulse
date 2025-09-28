package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ramanasai/pulse/internal/config"
	"github.com/ramanasai/pulse/internal/db"
	"github.com/spf13/cobra"
)

var (
	since string
	limit int
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent entries (timeline view)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config to get timezone
		cfg, _ := config.Load()
		loc := cfg.Location()

		dbh, err := db.Open()
		if err != nil {
			return err
		}
		defer dbh.Close()

		// Compute 'since' for query (UTC) and for display (local)
		var sinceLocal time.Time
		if strings.TrimSpace(since) == "" {
			sinceLocal = time.Now().In(loc).Add(-24 * time.Hour)
		} else {
			// try parse as RFC3339 or RFC3339Nano and then localize
			if t, err := time.Parse(time.RFC3339Nano, since); err == nil {
				sinceLocal = t.In(loc)
			} else if t2, err2 := time.Parse(time.RFC3339, since); err2 == nil {
				sinceLocal = t2.In(loc)
			} else {
				sinceLocal = time.Now().In(loc).Add(-24 * time.Hour)
			}
		}
		sinceForQuery := sinceLocal.UTC().Format(time.RFC3339)

		if limit <= 0 || limit > 1000 {
			limit = 200
		}

		q := `
SELECT id, ts, category, COALESCE(project,''), COALESCE(tags,''), text
FROM entries
WHERE ts >= ?
ORDER BY ts DESC
LIMIT ?
`
		rows, err := dbh.Query(q, sinceForQuery, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		// ---- styles ----
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1"))
		sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
		idStyle := lipgloss.NewStyle().Faint(true)
		timeStyle := lipgloss.NewStyle().Faint(true)
		sidebarStyle := lipgloss.NewStyle().PaddingRight(2)
		projectStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA"))
		tagsStyle := lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#CBA6F7"))
		textStyle := lipgloss.NewStyle()

		// terminal width (best effort)
		colEnv := os.Getenv("COLUMNS")
		termWidth := 100
		if colEnv != "" {
			if v, err := strconv.Atoi(colEnv); err == nil && v > 40 {
				termWidth = v
			}
		}
		leftWidth := 14 // "● 12:34 PM" + padding
		rightWidth := termWidth - leftWidth
		if rightWidth < 30 {
			rightWidth = 30
		}

		header := titleStyle.Render("Pulse — Recent Entries") + "  " +
			sepStyle.Render("since ") +
			timeStyle.Render(sinceLocal.Format("2006-01-02 03:04 PM MST"))
		fmt.Println(header)
		fmt.Println(sepStyle.Render(strings.Repeat("─", min(termWidth, 120))))

		for rows.Next() {
			var (
				id                        int
				ts, cat, proj, tags, text string
			)
			if err := rows.Scan(&id, &ts, &cat, &proj, &tags, &text); err != nil {
				return err
			}

			// parse DB timestamp (UTC) and render in configured timezone, 12-hour
			var parsed time.Time
			if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				parsed = t
			} else if t2, err2 := time.Parse(time.RFC3339, ts); err2 == nil {
				parsed = t2
			}
			tstr := parsed.In(loc).Format("03:04 PM")

			dot := lipgloss.NewStyle().
				Foreground(colorForCategory(cat)).
				Render("●")

			left := sidebarStyle.
				Width(leftWidth).
				Render(dot + " " + timeStyle.Render(tstr))

			// line one: [id] category project tags
			meta := idStyle.Render(fmt.Sprintf("[%d]", id)) + "  " +
				lipgloss.NewStyle().Bold(true).Foreground(colorForCategory(cat)).Render(cat)

			if proj != "" {
				meta += "  " + projectStyle.Render("["+proj+"]")
			}
			if tags = strings.TrimSpace(tags); tags != "" {
				meta += "  " + tagsStyle.Render("#"+strings.ReplaceAll(tags, ",", " #"))
			}

			// text body
			body := textStyle.Width(rightWidth).Render(strings.TrimSpace(text))

			right := lipgloss.JoinVertical(lipgloss.Left, meta, body)
			line := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

			fmt.Println(line)
			fmt.Println(sepStyle.Render(strings.Repeat("─", min(termWidth, 120))))
		}
		return rows.Err()
	},
}

func init() {
	listCmd.Flags().StringVar(&since, "since", "", "RFC3339 timestamp or empty for last 24h (interpreted in your configured timezone)")
	listCmd.Flags().IntVar(&limit, "limit", 200, "Max entries to show (default 200)")
}

func colorForCategory(cat string) lipgloss.Color {
	switch strings.ToLower(cat) {
	case "task":
		return lipgloss.Color("#F9E2AF") // yellow
	case "meeting":
		return lipgloss.Color("#F5C2E7") // pink
	case "timer":
		return lipgloss.Color("#A6E3A1") // green
	case "note":
		return lipgloss.Color("#89B4FA") // blue
	default:
		return lipgloss.Color("#94E2D5") // teal
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
