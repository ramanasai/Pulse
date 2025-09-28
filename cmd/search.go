package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ramanasai/pulse/internal/db"
	"github.com/spf13/cobra"
)

var (
	searchSince string
	searchUntil string
	searchLimit int
	searchProj  string
	searchTags  string
)

// searchCmd performs an FTS5 search with highlighted snippets.
var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text search with highlights",
	Long: `Examples:
			pulse search "deploy failed"              # free text
			pulse search 'text:started tags:devops'   # target fields
			pulse search 'incid*'                     # prefix search
			pulse search "retro" --project devops     # combine filters
			pulse search "error" --since 2025-09-01 --until 2025-09-28`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		dbh, err := db.Open()
		if err != nil {
			return err
		}
		defer dbh.Close()

		if searchUntil == "" {
			searchUntil = time.Now().Format(time.RFC3339)
		}
		if searchSince == "" {
			// default: last 90 days for search
			searchSince = time.Now().Add(-90 * 24 * time.Hour).Format(time.RFC3339)
		}
		if searchLimit <= 0 || searchLimit > 1000 {
			searchLimit = 200
		}

		// Build filters
		conds := []string{"e.ts BETWEEN ? AND ?"}
		argsQ := []any{searchSince, searchUntil}

		if strings.TrimSpace(searchProj) != "" {
			conds = append(conds, "e.project = ?")
			argsQ = append(argsQ, searchProj)
		}
		if t := strings.TrimSpace(searchTags); t != "" {
			// simple contains-any: for each tag, require instr(tags,tag)>0
			for _, tg := range strings.Split(t, ",") {
				tg = strings.TrimSpace(tg)
				if tg == "" {
					continue
				}
				conds = append(conds, "instr(e.tags, ?) > 0")
				argsQ = append(argsQ, tg)
			}
		}

		where := strings.Join(conds, " AND ")
		sqlStr := `
			SELECT e.id, e.ts, e.category, COALESCE(e.project,''), COALESCE(e.tags,''),
				snippet(entries_fts, 0, '[', ']', '…', 8) AS snip,
				bm25(entries_fts) AS rank
			FROM entries_fts
			JOIN entries e ON e.id = entries_fts.rowid
			WHERE entries_fts MATCH ? AND ` + where + `
			ORDER BY rank ASC, e.ts DESC
			LIMIT ?
			`
		argsQ = append([]any{query}, argsQ...)
		argsQ = append(argsQ, searchLimit)

		rows, err := dbh.Query(sqlStr, argsQ...)
		if err != nil {
			return err
		}
		defer rows.Close()

		// styles
		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1"))
		sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
		meta := lipgloss.NewStyle().Faint(true)
		cat := lipgloss.NewStyle().Bold(true)
		proj := lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA"))
		tags := lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#CBA6F7"))
		snip := lipgloss.NewStyle()

		colEnv := os.Getenv("COLUMNS")
		w := 100
		if colEnv != "" {
			if v, err := strconv.Atoi(colEnv); err == nil && v > 40 {
				w = v
			}
		}

		fmt.Println(title.Render("Search") + "  " + sep.Render("query: ") + query)
		fmt.Println(sep.Render(strings.Repeat("─", min(w, 120))))

		var count int
		for rows.Next() {
			var id int
			var ts, c, p, tg, s sql.NullString
			var rank float64
			if err := rows.Scan(&id, &ts, &c, &p, &tg, &s, &rank); err != nil {
				return err
			}
			tm := ts.String
			if t, err := time.Parse(time.RFC3339Nano, tm); err == nil {
				tm = t.Format("2006-01-02 15:04")
			}
			line := meta.Render(fmt.Sprintf("[%d] %s", id, tm)) + "  " +
				cat.Foreground(colorForCategory(c.String)).Render(c.String)
			if p.String != "" {
				line += "  " + proj.Render("["+p.String+"]")
			}
			if strings.TrimSpace(tg.String) != "" {
				line += "  " + tags.Render("#"+strings.ReplaceAll(tg.String, ",", " #"))
			}
			fmt.Println(line)

			// Bold the matched segments (we mark [ ... ] around matches in snippet())
			highlight := s.String
			highlight = strings.ReplaceAll(highlight, "[", "\x1b[1m")
			highlight = strings.ReplaceAll(highlight, "]", "\x1b[0m")

			fmt.Println(snip.Render("  " + highlight))
			fmt.Println(sep.Render(strings.Repeat("─", min(w, 120))))
			count++
		}
		if count == 0 {
			fmt.Println(meta.Render("no results"))
		}
		return rows.Err()
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchSince, "since", "", "RFC3339 start time (default: 90 days ago)")
	searchCmd.Flags().StringVar(&searchUntil, "until", "", "RFC3339 end time (default: now)")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 200, "Max results (default 200)")
	searchCmd.Flags().StringVar(&searchProj, "project", "", "Filter by project")
	searchCmd.Flags().StringVar(&searchTags, "tags", "", "Comma separated tags to require")
}
