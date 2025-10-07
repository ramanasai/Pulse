package utils

import (
	"fmt"
	"math"
	"strings"
)

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	Total      int
	PerPage    int
	Current    int
	Offset     int
	TotalPages int
}

// NewPagination creates pagination info
func NewPagination(total, perPage, current int) *PaginationInfo {
	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	if totalPages == 0 {
		totalPages = 1
	}

	if current < 1 {
		current = 1
	}
	if current > totalPages {
		current = totalPages
	}

	offset := (current - 1) * perPage

	return &PaginationInfo{
		Total:      total,
		PerPage:    perPage,
		Current:    current,
		Offset:     offset,
		TotalPages: totalPages,
	}
}

// GetSQLLimitOffset returns LIMIT and OFFSET for SQL queries
func (p *PaginationInfo) GetSQLLimitOffset() (limit int, offset int) {
	return p.PerPage, p.Offset
}

// GetRange returns the range of items on the current page (1-indexed)
func (p *PaginationInfo) GetRange() (start, end int) {
	start = p.Offset + 1
	end = p.Offset + p.PerPage
	if end > p.Total {
		end = p.Total
	}
	return start, end
}

// HasNext returns true if there's a next page
func (p *PaginationInfo) HasNext() bool {
	return p.Current < p.TotalPages
}

// HasPrev returns true if there's a previous page
func (p *PaginationInfo) HasPrev() bool {
	return p.Current > 1
}

// GetNextPage returns the next page number
func (p *PaginationInfo) GetNextPage() int {
	if p.HasNext() {
		return p.Current + 1
	}
	return p.Current
}

// GetPrevPage returns the previous page number
func (p *PaginationInfo) GetPrevPage() int {
	if p.HasPrev() {
		return p.Current - 1
	}
	return p.Current
}

// FormatSummary returns a human-readable summary
func (p *PaginationInfo) FormatSummary() string {
	if p.Total == 0 {
		return "No results"
	}

	start, end := p.GetRange()
	if p.TotalPages == 1 {
		return fmt.Sprintf("Showing %d-%d of %d result%s", start, end, p.Total, plural(p.Total))
	}
	return fmt.Sprintf("Showing %d-%d of %d result%s (page %d of %d)",
		start, end, p.Total, plural(p.Total), p.Current, p.TotalPages)
}

// FormatNavigation returns navigation hints for CLI
func (p *PaginationInfo) FormatNavigation() string {
	if p.TotalPages <= 1 {
		return ""
	}

	var hints []string
	if p.HasPrev() {
		hints = append(hints, fmt.Sprintf("use --page %d for previous", p.GetPrevPage()))
	}
	if p.HasNext() {
		hints = append(hints, fmt.Sprintf("use --page %d for next", p.GetNextPage()))
	}

	return strings.Join(hints, ", ")
}

// plural returns "s" if count is not 1
func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// ParsePage parses page number from string with validation
func ParsePage(pageStr string, totalPages int) (int, error) {
	if pageStr == "" {
		return 1, nil
	}

	page := 1
	if pageStr != "" {
		var err error
		page, err = parsePageNumber(pageStr)
		if err != nil {
			return 1, fmt.Errorf("invalid page number: %w", err)
		}
	}

	if page < 1 {
		page = 1
	}
	if totalPages > 0 && page > totalPages {
		page = totalPages
	}

	return page, nil
}

// parsePageNumber parses various page number formats
func parsePageNumber(input string) (int, error) {
	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "first", "start", "beginning":
		return 1, nil
	case "last", "end":
		return 0, nil // Special case to be handled by caller
	case "next", "n":
		return -1, nil // Special case to be handled by caller
	case "prev", "previous", "p":
		return -2, nil // Special case to be handled by caller
	}

	// Try to parse as number
	var page int
	_, err := fmt.Sscanf(input, "%d", &page)
	if err != nil {
		return 1, fmt.Errorf("invalid page format: %s", input)
	}

	if page < 1 {
		return 1, fmt.Errorf("page number must be positive")
	}

	return page, nil
}