package cmd

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ramanasai/pulse/internal/config"
	"github.com/ramanasai/pulse/internal/db"
	"github.com/ramanasai/pulse/internal/utils"
	"github.com/spf13/cobra"
)

var (
	since      string
	limit      int
	page       int
	format     string
	noColor    bool
	groupBy    string
	projects   string
	categories string
	filterTags string
	preset     string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent entries (timeline view)",
	Long: `Examples:
	pulse list                                    # last 24h
	pulse list --since yesterday                  # since yesterday
	pulse list --preset last7days                 # last 7 days
	pulse list --format table --limit 50          # table format
	pulse list --project api --category task      # filter by project and category
	pulse list --tags bug,urgent --page 2         # filter by tags with pagination`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config to get timezone
		cfg, _ := config.Load()
		loc := cfg.Location()

		// Setup renderer
		renderConfig := utils.DefaultRenderConfig()
		if noColor {
			renderConfig.Color = false
		}
		if format != "" {
			renderConfig.Format = utils.OutputFormat(format)
		}
		renderConfig.Location = loc

		// Parse date range
		var sinceTime, untilTime time.Time
		var err error

		if preset != "" {
			sinceTime, untilTime, err = utils.GetDateRange(preset, loc)
			if err != nil {
				return fmt.Errorf("invalid preset %q: %w", preset, err)
			}
		} else if since != "" {
			sinceTime, err = utils.ParseFlexibleDate(since, loc)
			if err != nil {
				return fmt.Errorf("invalid --since date %q: %w", since, err)
			}
		} else {
			sinceTime = time.Now().In(loc).Add(-24 * time.Hour)
		}

		// Default until to now
		if untilTime.IsZero() {
			untilTime = time.Now()
		}

		// Validate pagination
		if limit <= 0 || limit > 1000 {
			limit = 50 // Reduced default for better UX
		}

		// Open database
		dbh, err := db.Open()
		if err != nil {
			return err
		}
		defer dbh.Close()

		// Build query
		query, queryArgs, err := buildListQuery(sinceTime, untilTime)
		if err != nil {
			return err
		}

		// Get total count for pagination
		countQuery, countArgs := buildCountQuery(sinceTime, untilTime)
		var total int
		if err := dbh.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
			return err
		}

		// Handle pagination
		pagination := utils.NewPagination(total, limit, page)
		limitSQL, offsetSQL := pagination.GetSQLLimitOffset()
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", limitSQL, offsetSQL)

		// Execute query
		rows, err := dbh.Query(query, queryArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()

		// Convert to Entry objects
		entries := make([]utils.Entry, 0)
		for rows.Next() {
			var id int
			var ts, cat, proj, tags, text string
			var durationMinutes sql.NullInt64

			if err := rows.Scan(&id, &ts, &cat, &proj, &tags, &text, &durationMinutes); err != nil {
				return err
			}

			// Parse timestamp
			timestamp, err := time.Parse(time.RFC3339Nano, ts)
			if err != nil {
				continue
			}

			entries = append(entries, utils.Entry{
				ID:              int64(id),
				Timestamp:       timestamp,
				Category:        cat,
				Text:            text,
				Project:         proj,
				Tags:            tags,
				DurationMinutes: int(durationMinutes.Int64),
			})
		}

		// Group entries if requested
		if groupBy != "" {
			entries = groupEntries(entries, groupBy, loc)
		}

		// Prepare entry list
		entryList := &utils.EntryList{
			Entries:    entries,
			Total:      total,
			Page:       pagination.Current,
			PerPage:    pagination.PerPage,
			TotalPages: pagination.TotalPages,
			Filters: map[string]string{
				"since": sinceTime.In(loc).Format("2006-01-02 03:04 PM MST"),
			},
		}

		// Render output
		renderer := utils.NewRenderer(renderConfig)
		output, err := renderer.RenderEntryList(entryList)
		if err != nil {
			return err
		}

		fmt.Print(output)

		return nil
	},
}

// buildListQuery builds the SQL query for listing entries
func buildListQuery(since, until time.Time) (string, []interface{}, error) {
	conditions := []string{"ts BETWEEN ? AND ?"}
	args := []interface{}{since.UTC().Format(time.RFC3339), until.UTC().Format(time.RFC3339)}

	// Add filters
	if strings.TrimSpace(projects) != "" {
		for _, proj := range strings.Split(projects, ",") {
			proj = strings.TrimSpace(proj)
			if proj != "" {
				conditions = append(conditions, "project = ?")
				args = append(args, proj)
			}
		}
	}

	if strings.TrimSpace(categories) != "" {
		for _, cat := range strings.Split(categories, ",") {
			cat = strings.TrimSpace(cat)
			if cat != "" {
				conditions = append(conditions, "category = ?")
				args = append(args, cat)
			}
		}
	}

	if strings.TrimSpace(filterTags) != "" {
		for _, tag := range strings.Split(filterTags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				conditions = append(conditions, "instr(tags, ?) > 0")
				args = append(args, tag)
			}
		}
	}

	query := `
		SELECT id, ts, category, COALESCE(project,''), COALESCE(tags,''), text, duration_minutes
		FROM entries
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY ts DESC`

	return query, args, nil
}

// buildCountQuery builds the count query for pagination
func buildCountQuery(since, until time.Time) (string, []interface{}) {
	conditions := []string{"ts BETWEEN ? AND ?"}
	args := []interface{}{since.UTC().Format(time.RFC3339), until.UTC().Format(time.RFC3339)}

	// Add same filters as buildListQuery
	if strings.TrimSpace(projects) != "" {
		for _, proj := range strings.Split(projects, ",") {
			proj = strings.TrimSpace(proj)
			if proj != "" {
				conditions = append(conditions, "project = ?")
				args = append(args, proj)
			}
		}
	}

	if strings.TrimSpace(categories) != "" {
		for _, cat := range strings.Split(categories, ",") {
			cat = strings.TrimSpace(cat)
			if cat != "" {
				conditions = append(conditions, "category = ?")
				args = append(args, cat)
			}
		}
	}

	if strings.TrimSpace(filterTags) != "" {
		for _, tag := range strings.Split(filterTags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				conditions = append(conditions, "instr(tags, ?) > 0")
				args = append(args, tag)
			}
		}
	}

	query := `
		SELECT COUNT(*)
		FROM entries
		WHERE ` + strings.Join(conditions, " AND ")

	return query, args
}

// groupEntries groups entries by the specified field
func groupEntries(entries []utils.Entry, groupBy string, loc *time.Location) []utils.Entry {
	// Simple implementation - could be enhanced for better grouping
	switch strings.ToLower(groupBy) {
	case "date":
		// Sort by date (already sorted by timestamp, so no change needed)
	case "project":
		// Sort by project (would need custom sorting)
	case "category":
		// Sort by category (would need custom sorting)
	}
	return entries
}

func init() {
	// Basic filters
	listCmd.Flags().StringVar(&since, "since", "", "Date/time filter (supports: yesterday, 'last week', '2 hours ago', 2025-01-15, etc.)")
	listCmd.Flags().IntVar(&limit, "limit", 50, "Maximum entries to show per page (default 50)")
	listCmd.Flags().IntVar(&page, "page", 1, "Page number to show (for pagination)")
	listCmd.Flags().StringVar(&format, "format", "default", "Output format: default, table, json, csv, compact, quiet")
	listCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	listCmd.Flags().StringVar(&groupBy, "group", "", "Group entries by: date, project, category")

	// Advanced filters
	listCmd.Flags().StringVar(&projects, "projects", "", "Filter by projects (comma-separated)")
	listCmd.Flags().StringVar(&categories, "categories", "", "Filter by categories (comma-separated)")
	listCmd.Flags().StringVar(&filterTags, "tags", "", "Filter by tags (comma-separated)")

	// Presets
	listCmd.Flags().StringVar(&preset, "preset", "", "Date preset: today, yesterday, week, month, year, last7days, last30days, last90days")
}
