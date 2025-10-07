package db

import (
	"database/sql"
	"strings"
	"time"
)

// Entry is a minimal projection of a row.
type Entry struct {
	ID        int
	TS        string
	Category  string
	Project   sql.NullString
	Tags      sql.NullString
	Text      sql.NullString
	Duration sql.NullInt64
	ThreadID  sql.NullInt64
	ParentID  sql.NullInt64
	Encrypted bool
}

// GetEntry returns one entry by id.
func GetEntry(dbh *sql.DB, id int) (Entry, error) {
	var e Entry
	err := dbh.QueryRow(`
		SELECT id, ts, category, project, tags, text, duration_minutes, thread_id, parent_id, encrypted
		FROM entries WHERE id = ?
	`, id).Scan(&e.ID, &e.TS, &e.Category, &e.Project, &e.Tags, &e.Text, &e.Duration, &e.ThreadID, &e.ParentID, &e.Encrypted)
	return e, err
}

// RootThreadID returns the root thread id for a given entry (its own id if root).
func RootThreadID(e Entry) int {
	if e.ThreadID.Valid && e.ThreadID.Int64 > 0 {
		return int(e.ThreadID.Int64)
	}
	return e.ID
}

// AddEntry creates a new entry in the database
func AddEntry(dbh *sql.DB, entry *Entry) error {
	query := `
		INSERT INTO entries (category, project, tags, text, ts)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := dbh.Exec(query, entry.Category, entry.Project, entry.Tags, entry.Text, entry.TS)
	return err
}

// StartTimer starts a new timer entry
func StartTimer(dbh *sql.DB, note, project string) error {
	query := `
		INSERT INTO entries (category, project, text, ts, duration_minutes)
		VALUES ('timer', ?, ?, ?, 0)
	`
	_, err := dbh.Exec(query, project, note, time.Now().Format(time.RFC3339))
	return err
}

// StopTimer stops the most recent active timer
func StopTimer(dbh *sql.DB, note string) error {
	// Find the most recent timer entry
	var id int
	var startTime time.Time

	query := `
		SELECT id, ts
		FROM entries
		WHERE category = 'timer' AND duration_minutes = 0
		ORDER BY ts DESC
		LIMIT 1
	`

	err := dbh.QueryRow(query).Scan(&id, &startTime)
	if err != nil {
		return err
	}

	// Calculate duration in minutes
	duration := int(time.Since(startTime).Minutes())

	// Update the entry with duration and note
	updateQuery := `
		UPDATE entries
		SET duration_minutes = ?, text = ?
		WHERE id = ?
	`

	_, err = dbh.Exec(updateQuery, duration, note, id)
	return err
}

// GetActiveTimer returns the currently active timer (if any)
func GetActiveTimer(dbh *sql.DB) (*Entry, error) {
	query := `
		SELECT id, ts, category, project, tags, text, duration_minutes, thread_id, parent_id, encrypted
		FROM entries
		WHERE category = 'timer' AND duration_minutes = 0
		ORDER BY ts DESC
		LIMIT 1
	`

	var entry Entry
	err := dbh.QueryRow(query).Scan(
		&entry.ID, &entry.TS, &entry.Category,
		&entry.Project, &entry.Tags, &entry.Text,
		&entry.Duration, &entry.ThreadID, &entry.ParentID, &entry.Encrypted,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &entry, nil
}

// GetRecentProjects returns the most recently used projects
func GetRecentProjects(dbh *sql.DB, limit int) ([]string, error) {
	query := `
		SELECT DISTINCT project
		FROM entries
		WHERE project IS NOT NULL AND project != ''
		ORDER BY ts DESC
		LIMIT ?
	`

	rows, err := dbh.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []string
	for rows.Next() {
		var project string
		if err := rows.Scan(&project); err != nil {
			continue
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// GetRecentTags returns the most recently used tags
func GetRecentTags(dbh *sql.DB, limit int) ([]string, error) {
	query := `
		SELECT DISTINCT tags
		FROM entries
		WHERE tags IS NOT NULL AND tags != ''
		ORDER BY ts DESC
		LIMIT ?
	`

	rows, err := dbh.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allTags []string
	for rows.Next() {
		var tags string
		if err := rows.Scan(&tags); err != nil {
			continue
		}

		// Split comma-separated tags
		for _, tag := range strings.Split(tags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				allTags = append(allTags, tag)
			}
		}
	}

	return allTags, nil
}

// SearchProjects searches projects by query
func SearchProjects(dbh *sql.DB, query string, limit int) ([]string, error) {
	searchQuery := "%" + strings.ToLower(query) + "%"
	sqlQuery := `
		SELECT DISTINCT project
		FROM entries
		WHERE project IS NOT NULL AND project != '' AND LOWER(project) LIKE ?
		ORDER BY
			CASE WHEN LOWER(project) = ? THEN 1 ELSE 2 END,
			CASE WHEN LOWER(project) LIKE ? THEN 3 ELSE 4 END,
			ts DESC
		LIMIT ?
	`

	rows, err := dbh.Query(sqlQuery, searchQuery, query, searchQuery+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []string
	for rows.Next() {
		var project string
		if err := rows.Scan(&project); err != nil {
			continue
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// SearchTags searches tags by query
func SearchTags(dbh *sql.DB, query string, limit int) ([]string, error) {
	searchQuery := "%" + strings.ToLower(query) + "%"
	sqlQuery := `
		SELECT DISTINCT tags
		FROM entries
		WHERE tags IS NOT NULL AND tags != '' AND LOWER(tags) LIKE ?
		ORDER BY ts DESC
		LIMIT ?
	`

	rows, err := dbh.Query(sqlQuery, searchQuery, limit*2) // Get more to account for comma-separated tags
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allTags []string
	for rows.Next() {
		var tags string
		if err := rows.Scan(&tags); err != nil {
			continue
		}

		// Split comma-separated tags and filter by query
		for _, tag := range strings.Split(tags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" && strings.Contains(strings.ToLower(tag), strings.ToLower(query)) {
				allTags = append(allTags, tag)
			}
		}
	}

	// Deduplicate and limit results
	seen := make(map[string]bool)
	var result []string
	for _, tag := range allTags {
		if !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// SearchCategories searches categories by query
func SearchCategories(dbh *sql.DB, query string, limit int) ([]string, error) {
	searchQuery := "%" + strings.ToLower(query) + "%"
	sqlQuery := `
		SELECT DISTINCT category
		FROM entries
		WHERE LOWER(category) LIKE ?
		ORDER BY
			CASE WHEN LOWER(category) = ? THEN 1 ELSE 2 END,
			CASE WHEN LOWER(category) LIKE ? THEN 3 ELSE 4 END,
			ts DESC
		LIMIT ?
	`

	rows, err := dbh.Query(sqlQuery, searchQuery, query, searchQuery+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			continue
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// GetEntryCountsByDate returns a map of date strings to entry counts for the given date range
func GetEntryCountsByDate(dbh *sql.DB, startDate, endDate time.Time) (map[string]int, error) {
	query := `
		SELECT DATE(ts) as date, COUNT(*) as count
		FROM entries
		WHERE ts >= ? AND ts <= ?
		GROUP BY DATE(ts)
		ORDER BY date
	`

	rows, err := dbh.Query(query, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			continue
		}
		counts[date] = count
	}

	return counts, nil
}

// GetEntriesByDate returns all entries for a specific date
func GetEntriesByDate(dbh *sql.DB, date time.Time, loc *time.Location) ([]Entry, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc).UTC()
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT id, ts, category, project, tags, text, duration_minutes, thread_id, parent_id, encrypted
		FROM entries
		WHERE ts >= ? AND ts < ?
		ORDER BY ts ASC
	`

	rows, err := dbh.Query(query, startOfDay.Format(time.RFC3339), endOfDay.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.TS, &e.Category, &e.Project, &e.Tags, &e.Text, &e.Duration, &e.ThreadID, &e.ParentID, &e.Encrypted); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	return entries, nil
}
