package cmd

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ramanasai/pulse/internal/config"
	"github.com/ramanasai/pulse/internal/db"
	"github.com/ramanasai/pulse/internal/utils"
	"github.com/spf13/cobra"
)

var (
	searchSince   string
	searchUntil   string
	searchLimit   int
	searchPage    int
	searchFormat  string
	searchNoColor bool
	searchProj    string
	searchTags    string
	searchCat     string
	searchPreset  string
)

// searchCmd performs an FTS5 search with enhanced features.
var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text search with highlights and advanced filtering",
	Long: `Examples:
	pulse search "deploy failed"                           # free text
	pulse search 'category:task project:api'              # field-specific search
	pulse search 'incid*'                                  # prefix search
	pulse search "error AND urgent"                        # boolean search
	pulse search "retro" --project devops                  # combine filters
	pulse search "error" --preset last7days                # date presets
	pulse search "meeting" --format json --page 2          # output formats`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config to get timezone
		cfg, _ := config.Load()
		loc := cfg.Location()

		// Setup renderer
		renderConfig := utils.DefaultRenderConfig()
		if searchNoColor {
			renderConfig.Color = false
		}
		if searchFormat != "" {
			renderConfig.Format = utils.OutputFormat(searchFormat)
		}
		renderConfig.Location = loc

		// Parse search query
		query := strings.Join(args, " ")
		processedQuery, filters, err := processSearchQuery(query)
		if err != nil {
			return fmt.Errorf("invalid search query: %w", err)
		}

		// Parse date range
		var sinceTime, untilTime time.Time

		if searchPreset != "" {
			sinceTime, untilTime, err = utils.GetDateRange(searchPreset, loc)
			if err != nil {
				return fmt.Errorf("invalid preset %q: %w", searchPreset, err)
			}
		} else if searchSince != "" {
			sinceTime, err = utils.ParseFlexibleDate(searchSince, loc)
			if err != nil {
				return fmt.Errorf("invalid --since date %q: %w", searchSince, err)
			}
		} else {
			sinceTime = time.Now().In(loc).Add(-90 * 24 * time.Hour) // default: last 90 days
		}

		if searchUntil != "" {
			untilTime, err = utils.ParseFlexibleDate(searchUntil, loc)
			if err != nil {
				return fmt.Errorf("invalid --until date %q: %w", searchUntil, err)
			}
		} else {
			untilTime = time.Now()
		}

		// Validate pagination
		if searchLimit <= 0 || searchLimit > 1000 {
			searchLimit = 50 // Reduced default for better UX
		}

		// Open database
		dbh, err := db.Open()
		if err != nil {
			return err
		}
		defer dbh.Close()

		// Build search query
		searchSQL, searchArgs, err := buildSearchQuery(processedQuery, sinceTime, untilTime, filters)
		if err != nil {
			return err
		}

		// Get total count for pagination
		countSQL, countArgs := buildSearchCountQuery(processedQuery, sinceTime, untilTime, filters)
		var total int
		if err := dbh.QueryRow(countSQL, countArgs...).Scan(&total); err != nil {
			return err
		}

		// Handle pagination
		pagination := utils.NewPagination(total, searchLimit, searchPage)
		limitSQL, offsetSQL := pagination.GetSQLLimitOffset()
		searchSQL += fmt.Sprintf(" LIMIT %d OFFSET %d", limitSQL, offsetSQL)

		// Execute search query
		rows, err := dbh.Query(searchSQL, searchArgs...)
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
			var rank float64
			var snippet sql.NullString

			if err := rows.Scan(&id, &ts, &cat, &proj, &tags, &text, &durationMinutes, &rank, &snippet); err != nil {
				return err
			}

			// Parse timestamp
			timestamp, err := time.Parse(time.RFC3339Nano, ts)
			if err != nil {
				continue
			}

			entry := utils.Entry{
				ID:              int64(id),
				Timestamp:       timestamp,
				Category:        cat,
				Text:            text,
				Project:         proj,
				Tags:            tags,
				DurationMinutes: int(durationMinutes.Int64),
				SearchRank:      rank,
			}

			if snippet.Valid && snippet.String != "" {
				entry.SearchSnippet = snippet.String
			}

			entries = append(entries, entry)
		}

		// Prepare entry list
		entryList := &utils.EntryList{
			Entries:    entries,
			Total:      total,
			Page:       pagination.Current,
			PerPage:    pagination.PerPage,
			TotalPages: pagination.TotalPages,
			Query:      query,
			Filters: map[string]string{
				"since": sinceTime.In(loc).Format("2006-01-02 03:04 PM MST"),
				"until": untilTime.In(loc).Format("2006-01-02 03:04 PM MST"),
			},
		}

		// Add additional filters to display
		if searchProj != "" {
			entryList.Filters["project"] = searchProj
		}
		if searchTags != "" {
			entryList.Filters["tags"] = searchTags
		}
		if searchCat != "" {
			entryList.Filters["category"] = searchCat
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

// SearchFilters represents parsed search filters from query
type SearchFilters struct {
	Category string
	Project  string
	Tags     []string
	Text     string
}

// processSearchQuery parses the search query and extracts field-specific filters
func processSearchQuery(query string) (string, *SearchFilters, error) {
	filters := &SearchFilters{}
	processedQuery := query

	// Parse field-specific searches: category:task, project:api, tags:urgent
	re := regexp.MustCompile(`(\w+):([^\s]+)`)
	matches := re.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		field := strings.ToLower(match[1])
		value := match[2]

		switch field {
		case "category", "cat":
			filters.Category = value
			processedQuery = strings.ReplaceAll(processedQuery, match[0], "")
		case "project", "proj":
			filters.Project = value
			processedQuery = strings.ReplaceAll(processedQuery, match[0], "")
		case "tags", "tag":
			filters.Tags = append(filters.Tags, strings.Split(value, ",")...)
			processedQuery = strings.ReplaceAll(processedQuery, match[0], "")
		case "text":
			filters.Text = value
			processedQuery = strings.ReplaceAll(processedQuery, match[0], "")
		}
	}

	// Clean up extra whitespace
	processedQuery = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(processedQuery), " ")

	// If no text query remaining, use any specified text filter or search for all entries
	if processedQuery == "" {
		if filters.Text != "" {
			processedQuery = filters.Text
		} else {
			// When only field filters are specified, search for all entries
			processedQuery = "*"
		}
	}

	return processedQuery, filters, nil
}

// buildSearchQuery builds the FTS search SQL query
func buildSearchQuery(query string, since, until time.Time, filters *SearchFilters) (string, []interface{}, error) {
	conditions := []string{"e.ts BETWEEN ? AND ?"}
	args := []interface{}{since.UTC().Format(time.RFC3339), until.UTC().Format(time.RFC3339)}

	var useFTS bool
	if query != "*" && query != "" {
		useFTS = true
	}

	// Add command-line filters
	if strings.TrimSpace(searchProj) != "" {
		conditions = append(conditions, "e.project = ?")
		args = append(args, searchProj)
	}

	if strings.TrimSpace(searchCat) != "" {
		conditions = append(conditions, "e.category = ?")
		args = append(args, searchCat)
	}

	if strings.TrimSpace(searchTags) != "" {
		for _, tag := range strings.Split(searchTags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				conditions = append(conditions, "instr(e.tags, ?) > 0")
				args = append(args, tag)
			}
		}
	}

	// Add query filters
	if filters != nil {
		if filters.Project != "" {
			conditions = append(conditions, "e.project = ?")
			args = append(args, filters.Project)
		}

		if filters.Category != "" {
			conditions = append(conditions, "e.category = ?")
			args = append(args, filters.Category)
		}

		for _, tag := range filters.Tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				conditions = append(conditions, "instr(e.tags, ?) > 0")
				args = append(args, tag)
			}
		}
	}

	var searchSQL string
	whereClause := strings.Join(conditions, " AND ")

	if useFTS {
		// FTS search query
		searchSQL = `
			SELECT e.id, e.ts, e.category, COALESCE(e.project,''), COALESCE(e.tags,''),
			       e.text, e.duration_minutes,
			       bm25(entries_fts) AS rank,
			       snippet(entries_fts, 0, '[', ']', '…', 8) AS snippet
			FROM entries_fts
			JOIN entries e ON e.id = entries_fts.rowid
			WHERE entries_fts MATCH ? AND ` + whereClause + `
			ORDER BY rank ASC, e.ts DESC`
		args = append([]interface{}{query}, args...)
	} else {
		// Regular query without FTS (for field-only searches)
		searchSQL = `
			SELECT e.id, e.ts, e.category, COALESCE(e.project,''), COALESCE(e.tags,''),
			       e.text, e.duration_minutes,
			       0.0 AS rank,
			       '' AS snippet
			FROM entries e
			WHERE ` + whereClause + `
			ORDER BY e.ts DESC`
	}

	return searchSQL, args, nil
}

// buildSearchCountQuery builds the count query for search pagination
func buildSearchCountQuery(query string, since, until time.Time, filters *SearchFilters) (string, []interface{}) {
	conditions := []string{"e.ts BETWEEN ? AND ?"}
	args := []interface{}{since.UTC().Format(time.RFC3339), until.UTC().Format(time.RFC3339)}

	var useFTS bool
	if query != "*" && query != "" {
		useFTS = true
	}

	// Add command-line filters
	if strings.TrimSpace(searchProj) != "" {
		conditions = append(conditions, "e.project = ?")
		args = append(args, searchProj)
	}

	if strings.TrimSpace(searchCat) != "" {
		conditions = append(conditions, "e.category = ?")
		args = append(args, searchCat)
	}

	if strings.TrimSpace(searchTags) != "" {
		for _, tag := range strings.Split(searchTags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				conditions = append(conditions, "instr(e.tags, ?) > 0")
				args = append(args, tag)
			}
		}
	}

	// Add query filters
	if filters != nil {
		if filters.Project != "" {
			conditions = append(conditions, "e.project = ?")
			args = append(args, filters.Project)
		}

		if filters.Category != "" {
			conditions = append(conditions, "e.category = ?")
			args = append(args, filters.Category)
		}

		for _, tag := range filters.Tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				conditions = append(conditions, "instr(e.tags, ?) > 0")
				args = append(args, tag)
			}
		}
	}

	whereClause := strings.Join(conditions, " AND ")

	var countSQL string
	if useFTS {
		// FTS count query
		countSQL = `
			SELECT COUNT(*)
			FROM entries_fts
			JOIN entries e ON e.id = entries_fts.rowid
			WHERE entries_fts MATCH ? AND ` + whereClause
		args = append([]interface{}{query}, args...)
	} else {
		// Regular count query (for field-only searches)
		countSQL = `
			SELECT COUNT(*)
			FROM entries e
			WHERE ` + whereClause
	}

	return countSQL, args
}

func init() {
	// Basic filters
	searchCmd.Flags().StringVar(&searchSince, "since", "", "Date/time filter (supports: yesterday, 'last week', '2 hours ago', 2025-01-15, etc.)")
	searchCmd.Flags().StringVar(&searchUntil, "until", "", "End date filter")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "Maximum results to show per page (default 50)")
	searchCmd.Flags().IntVar(&searchPage, "page", 1, "Page number to show (for pagination)")
	searchCmd.Flags().StringVar(&searchFormat, "format", "default", "Output format: default, table, json, csv, compact, quiet")
	searchCmd.Flags().BoolVar(&searchNoColor, "no-color", false, "Disable colored output")

	// Advanced filters
	searchCmd.Flags().StringVar(&searchProj, "project", "", "Filter by project")
	searchCmd.Flags().StringVar(&searchTags, "tags", "", "Filter by tags (comma-separated)")
	searchCmd.Flags().StringVar(&searchCat, "category", "", "Filter by category")

	// Presets
	searchCmd.Flags().StringVar(&searchPreset, "preset", "", "Date preset: today, yesterday, week, month, year, last7days, last30days, last90days")
}
