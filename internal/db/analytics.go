package db

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// TimeReportEntry represents time data for a specific date
type TimeReportEntry struct {
	Date       time.Time
	TotalTime  time.Duration
	ByCategory map[string]time.Duration
	ByProject  map[string]time.Duration
	EntryCount int
}

// ProjectSummary represents summary data for a project
type ProjectSummary struct {
	Project    string
	TotalTime  time.Duration
	EntryCount int
	Categories map[string]time.Duration
	LastActive time.Time
	Trend      string // "up", "down", "stable"
}

// TagAnalytics represents analytics data for a tag
type TagAnalytics struct {
	Tag        string
	UsageCount int
	TotalTime  time.Duration
	Projects   []string
	Categories []string
	Trend      string
	LastUsed   time.Time
}

// LoadTimeReports loads time tracking data for the specified scope
func LoadTimeReports(dbh *sql.DB, loc *time.Location, scope int) ([]TimeReportEntry, error) {
	now := time.Now().In(loc)
	var startDate time.Time

	switch scope {
	case 0: // scopeToday
		year, month, day := now.Date()
		startDate = time.Date(year, month, day, 0, 0, 0, 0, loc)
	case 1: // scopeSince (not used for time reports)
		startDate = time.Now().AddDate(0, 0, -7).In(loc)
	case 2: // scopeAll
		startDate = time.Unix(0, 0).In(loc)
	case 3: // scopeThisWeek
		// Start of week (Sunday)
		weekday := int(now.Weekday())
		startDate = now.AddDate(0, 0, -weekday)
		year, month, day := startDate.Date()
		startDate = time.Date(year, month, day, 0, 0, 0, 0, loc)
	case 4: // scopeThisMonth
		year, month, _ := now.Date()
		startDate = time.Date(year, month, 1, 0, 0, 0, 0, loc)
	case 5: // scopeYesterday
		year, month, day := now.AddDate(0, 0, -1).Date()
		startDate = time.Date(year, month, day, 0, 0, 0, 0, loc)
	case 6: // scopeLastWeek
		weekday := int(now.Weekday())
		thisWeekStart := now.AddDate(0, 0, -weekday)
		year, month, day := thisWeekStart.Date()
		thisWeekStart = time.Date(year, month, day, 0, 0, 0, 0, loc)
		startDate = thisWeekStart.AddDate(0, 0, -7)
	case 7: // scopeLastMonth
		year, month, _ := now.AddDate(0, -1, 0).Date()
		startDate = time.Date(year, month, 1, 0, 0, 0, 0, loc)
	case 8: // scopeCustom (not used for time reports)
		startDate = now.AddDate(0, 0, -7).In(loc)
	default:
		startDate = now.AddDate(0, 0, -7).In(loc)
	}

	// Query entries within the date range
	query := `
		SELECT
			DATE(ts) as date,
			SUM(COALESCE(duration_minutes, 0)) as total_minutes,
			COUNT(*) as entry_count,
			CATEGORY,
			COALESCE(project, '') as project,
			COALESCE(duration_minutes, 0) as duration
		FROM entries
		WHERE ts >= ?
		GROUP BY DATE(ts), CATEGORY, COALESCE(project, '')
		ORDER BY DATE(ts) DESC
	`

	rows, err := dbh.Query(query, startDate.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to query time reports: %w", err)
	}
	defer rows.Close()

	// Organize data by date
	dateData := make(map[string]TimeReportEntry)

	for rows.Next() {
		var dateStr, category, project string
		var totalMinutes, entryCount, duration int

		if err := rows.Scan(&dateStr, &totalMinutes, &entryCount, &category, &project, &duration); err != nil {
			continue
		}

		// Parse date
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		date = date.In(loc)

		// Get or create entry for this date
		entry, exists := dateData[dateStr]
		if !exists {
			entry = TimeReportEntry{
				Date:       date,
				TotalTime:  0,
				ByCategory: make(map[string]time.Duration),
				ByProject:  make(map[string]time.Duration),
				EntryCount: 0,
			}
		}

		// Update totals
		entry.TotalTime += time.Duration(totalMinutes) * time.Minute
		entry.EntryCount = entryCount // This will be set to the count for the last group

		// Update category breakdown
		if duration > 0 {
			entry.ByCategory[category] += time.Duration(duration) * time.Minute
		}

		// Update project breakdown
		if project != "" && duration > 0 {
			entry.ByProject[project] += time.Duration(duration) * time.Minute
		}

		dateData[dateStr] = entry
	}

	// Convert to slice and sort by date
	var result []TimeReportEntry
	for _, entry := range dateData {
		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.After(result[j].Date)
	})

	return result, nil
}

// LoadProjectSummary loads project summary data
func LoadProjectSummary(dbh *sql.DB, loc *time.Location) ([]ProjectSummary, error) {
	query := `
		SELECT
			COALESCE(project, 'No Project') as project,
			COUNT(*) as entry_count,
			SUM(COALESCE(duration_minutes, 0)) as total_minutes,
			CATEGORY,
			COALESCE(duration_minutes, 0) as duration,
			MAX(ts) as last_active
		FROM entries
		GROUP BY COALESCE(project, ''), CATEGORY
		ORDER BY total_minutes DESC
	`

	rows, err := dbh.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query project summary: %w", err)
	}
	defer rows.Close()

	// Organize data by project
	projectData := make(map[string]ProjectSummary)

	for rows.Next() {
		var project, category string
		var entryCount, totalMinutes, duration int
		var lastActiveStr string

		if err := rows.Scan(&project, &entryCount, &totalMinutes, &category, &duration, &lastActiveStr); err != nil {
			continue
		}

		// Parse last active timestamp
		lastActive, err := time.Parse(time.RFC3339, lastActiveStr)
		if err != nil {
			continue
		}
		lastActive = lastActive.In(loc)

		// Get or create summary for this project
		summary, exists := projectData[project]
		if !exists {
			summary = ProjectSummary{
				Project:    project,
				TotalTime:  0,
				EntryCount: 0,
				Categories: make(map[string]time.Duration),
				LastActive: lastActive,
				Trend:      "stable",
			}
		}

		// Update totals
		summary.TotalTime += time.Duration(totalMinutes) * time.Minute
		summary.EntryCount += entryCount

		// Update last active if more recent
		if lastActive.After(summary.LastActive) {
			summary.LastActive = lastActive
		}

		// Update category breakdown
		if duration > 0 {
			summary.Categories[category] += time.Duration(duration) * time.Minute
		}

		projectData[project] = summary
	}

	// Convert to slice and sort by total time
	var result []ProjectSummary
	for _, summary := range projectData {
		// Determine trend (simplified - would need historical data for accurate trends)
		if summary.TotalTime > 4*time.Hour {
			summary.Trend = "up"
		} else if summary.TotalTime > time.Hour {
			summary.Trend = "stable"
		} else {
			summary.Trend = "down"
		}
		result = append(result, summary)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalTime > result[j].TotalTime
	})

	return result, nil
}

// LoadTagAnalytics loads tag analytics data
func LoadTagAnalytics(dbh *sql.DB, loc *time.Location) ([]TagAnalytics, error) {
	query := `
		SELECT
			TRIM(SUBSTR(tags, 1, INSTR(tags || ',', ',') - 1)) as first_tag,
			COUNT(*) as usage_count,
			SUM(COALESCE(duration_minutes, 0)) as total_minutes,
			GROUP_CONCAT(DISTINCT COALESCE(project, '')) as projects,
			GROUP_CONCAT(DISTINCT category) as categories,
			MAX(ts) as last_used
		FROM entries
		WHERE tags IS NOT NULL AND tags != ''
		GROUP BY first_tag

		UNION ALL

		SELECT
			TRIM(SUBSTR(SUBSTR(tags, INSTR(tags || ',', ',') + 1), 1, INSTR(SUBSTR(tags, INSTR(tags || ',', ',') + 1) || ',', ',') - 1)) as second_tag,
			COUNT(*) as usage_count,
			SUM(COALESCE(duration_minutes, 0)) as total_minutes,
			GROUP_CONCAT(DISTINCT COALESCE(project, '')) as projects,
			GROUP_CONCAT(DISTINCT category) as categories,
			MAX(ts) as last_used
		FROM entries
		WHERE tags IS NOT NULL AND tags != ''
			AND INSTR(SUBSTR(tags, INSTR(tags || ',', ',') + 1), ',') > 0
		GROUP BY second_tag

		ORDER BY usage_count DESC
	`

	rows, err := dbh.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tag analytics: %w", err)
	}
	defer rows.Close()

	var result []TagAnalytics
	for rows.Next() {
		var tag, projectsCSV, categoriesCSV, lastUsedStr string
		var usageCount, totalMinutes int

		if err := rows.Scan(&tag, &usageCount, &totalMinutes, &projectsCSV, &categoriesCSV, &lastUsedStr); err != nil {
			continue
		}

		// Parse last used timestamp
		lastUsed, err := time.Parse(time.RFC3339, lastUsedStr)
		if err != nil {
			continue
		}
		lastUsed = lastUsed.In(loc)

		// Parse projects and categories
		var projects []string
		if projectsCSV != "" {
			projects = strings.Split(projectsCSV, ",")
			for i, p := range projects {
				projects[i] = strings.TrimSpace(p)
			}
			// Limit to 10 projects
			if len(projects) > 10 {
				projects = projects[:10]
			}
		}

		var categories []string
		if categoriesCSV != "" {
			categories = strings.Split(categoriesCSV, ",")
			for i, c := range categories {
				categories[i] = strings.TrimSpace(c)
			}
			// Limit to 5 categories
			if len(categories) > 5 {
				categories = categories[:5]
			}
		}

		// Determine trend (simplified)
		var trend string
		if usageCount > 10 {
			trend = "up"
		} else if usageCount > 5 {
			trend = "stable"
		} else {
			trend = "down"
		}

		result = append(result, TagAnalytics{
			Tag:        tag,
			UsageCount: usageCount,
			TotalTime:  time.Duration(totalMinutes) * time.Minute,
			Projects:   projects,
			Categories: categories,
			Trend:      trend,
			LastUsed:   lastUsed,
		})
	}

	return result, nil
}