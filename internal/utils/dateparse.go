package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseFlexibleDate attempts to parse various date formats and natural language
func ParseFlexibleDate(input string, loc *time.Location) (time.Time, error) {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return time.Time{}, fmt.Errorf("empty date input")
	}

	now := time.Now().In(loc)

	// Handle natural language patterns
	switch input {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), nil
	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, loc), nil
	case "tomorrow":
		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, loc), nil
	case "now":
		return now, nil
	}

	// Handle relative patterns
	if strings.HasSuffix(input, " ago") {
		durationStr := strings.TrimSuffix(input, " ago")
		if duration, err := parseDuration(durationStr); err == nil {
			return now.Add(-duration), nil
		}
	}

	if strings.HasPrefix(input, "last ") {
		period := strings.TrimPrefix(input, "last ")
		switch period {
		case "week":
			return now.AddDate(0, 0, -7), nil
		case "month":
			return now.AddDate(0, -1, 0), nil
		case "year":
			return now.AddDate(-1, 0, 0), nil
		case "day":
			return now.AddDate(0, 0, -1), nil
		}
	}

	if strings.HasPrefix(input, "this ") {
		period := strings.TrimPrefix(input, "this ")
		switch period {
		case "week":
			weekday := int(now.Weekday())
			if weekday == 0 { // Sunday
				weekday = 7
			}
			return now.AddDate(0, 0, -(weekday - 1)), nil
		case "month":
			return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc), nil
		case "year":
			return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc), nil
		}
	}

	// Handle "N days/weeks/months/years" patterns
	re := regexp.MustCompile(`^(\d+)\s+(day|days|week|weeks|month|months|year|years)$`)
	if matches := re.FindStringSubmatch(input); matches != nil {
		num, _ := strconv.Atoi(matches[1])
		unit := matches[2]
		var duration time.Duration
		switch unit {
		case "day", "days":
			duration = time.Duration(num) * 24 * time.Hour
		case "week", "weeks":
			duration = time.Duration(num) * 7 * 24 * time.Hour
		case "month", "months":
			return now.AddDate(0, -num, 0), nil
		case "year", "years":
			return now.AddDate(-num, 0, 0), nil
		}
		return now.Add(-duration), nil
	}

	// Try various date formats
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"02/01/2006", // European format
		"Jan 2, 2006",
		"2 Jan 2006",
		"January 2, 2006",
		"2 January 2006",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, input, loc); err == nil {
			return t, nil
		}
	}

	// Try without location (fallback to UTC)
	for _, format := range formats {
		if t, err := time.Parse(format, input); err == nil {
			return t.In(loc), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", input)
}

// parseDuration parses simple duration strings like "2h", "30m", "1d"
func parseDuration(input string) (time.Duration, error) {
	re := regexp.MustCompile(`^(\d+)([smhdwy])$`)
	matches := re.FindStringSubmatch(input)
	if matches == nil {
		return 0, fmt.Errorf("invalid duration format: %s", input)
	}

	num, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	switch unit {
	case "s":
		return time.Duration(num) * time.Second, nil
	case "m":
		return time.Duration(num) * time.Minute, nil
	case "h":
		return time.Duration(num) * time.Hour, nil
	case "d":
		return time.Duration(num) * 24 * time.Hour, nil
	case "w":
		return time.Duration(num) * 7 * 24 * time.Hour, nil
	case "y":
		return time.Duration(num) * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}
}

// GetDateRange returns start and end time for common presets
func GetDateRange(preset string, loc *time.Location) (time.Time, time.Time, error) {
	now := time.Now().In(loc)

	switch strings.ToLower(preset) {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		end := start.Add(24 * time.Hour)
		return start, end, nil
	case "yesterday":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Add(-24 * time.Hour)
		end := start.Add(24 * time.Hour)
		return start, end, nil
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 { // Sunday
			weekday = 7
		}
		start := now.AddDate(0, 0, -(weekday - 1))
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
		end := start.Add(7 * 24 * time.Hour)
		return start, end, nil
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		end := start.AddDate(0, 1, 0)
		return start, end, nil
	case "year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)
		end := start.AddDate(1, 0, 0)
		return start, end, nil
	case "last7days", "last-7-days":
		start := now.AddDate(0, 0, -7)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
		end := now
		return start, end, nil
	case "last30days", "last-30-days":
		start := now.AddDate(0, 0, -30)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
		end := now
		return start, end, nil
	case "last90days", "last-90-days":
		start := now.AddDate(0, 0, -90)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
		end := now
		return start, end, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unknown date preset: %s", preset)
	}
}