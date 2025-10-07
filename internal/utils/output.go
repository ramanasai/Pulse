package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// OutputFormat represents different output formats
type OutputFormat string

const (
	FormatDefault OutputFormat = "default"
	FormatTable   OutputFormat = "table"
	FormatJSON    OutputFormat = "json"
	FormatCSV     OutputFormat = "csv"
	FormatCompact OutputFormat = "compact"
	FormatQuiet   OutputFormat = "quiet"
)

// RenderConfig contains configuration for output rendering
type RenderConfig struct {
	Format       OutputFormat
	Width        int
	ShowID       bool
	ShowTime     bool
	ShowDate     bool
	ShowProject  bool
	ShowTags     bool
	ShowCategory bool
	ShowMeta     bool
	Color        bool
	Location     *time.Location
}

// DefaultRenderConfig returns a default render configuration
func DefaultRenderConfig() *RenderConfig {
	width := 100
	if colEnv := os.Getenv("COLUMNS"); colEnv != "" {
		if v, err := strconv.Atoi(colEnv); err == nil && v > 40 {
			width = v
		}
	}

	return &RenderConfig{
		Format:       FormatDefault,
		Width:        width,
		ShowID:       true,
		ShowTime:     true,
		ShowDate:     true,
		ShowProject:  true,
		ShowTags:     true,
		ShowCategory: true,
		ShowMeta:     true,
		Color:        true,
		Location:     time.UTC,
	}
}

// Entry represents a single entry for output formatting
type Entry struct {
	ID              int64     `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	Category        string    `json:"category"`
	Text            string    `json:"text"`
	Project         string    `json:"project"`
	Tags            string    `json:"tags"`
	DurationMinutes int       `json:"duration_minutes,omitempty"`
	SearchRank      float64   `json:"search_rank,omitempty"`
	SearchSnippet   string    `json:"search_snippet,omitempty"`
}

// EntryList represents a list of entries with pagination info
type EntryList struct {
	Entries     []Entry         `json:"entries"`
	Total       int             `json:"total"`
	Page        int             `json:"page,omitempty"`
	PerPage     int             `json:"per_page,omitempty"`
	TotalPages  int             `json:"total_pages,omitempty"`
	Query       string          `json:"query,omitempty"`
	Filters     map[string]string `json:"filters,omitempty"`
}

// Renderer handles output formatting
type Renderer struct {
	config *RenderConfig
	styles *Styles
}

// Styles contains lipgloss styles for different elements
type Styles struct {
	Title      lipgloss.Style
	Separator  lipgloss.Style
	Meta       lipgloss.Style
	ID         lipgloss.Style
	Time       lipgloss.Style
	Date       lipgloss.Style
	Category   lipgloss.Style
	Project    lipgloss.Style
	Tags       lipgloss.Style
	Text       lipgloss.Style
	Highlight  lipgloss.Style
	Success    lipgloss.Style
	Error      lipgloss.Style
	Warning    lipgloss.Style
}

// NewRenderer creates a new renderer with the given config
func NewRenderer(config *RenderConfig) *Renderer {
	if config == nil {
		config = DefaultRenderConfig()
	}

	r := &Renderer{
		config: config,
		styles: initStyles(config.Color),
	}

	return r
}

// initStyles initializes the style set
func initStyles(color bool) *Styles {
	styles := &Styles{}

	if color {
		styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1"))
		styles.Separator = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
		styles.Meta = lipgloss.NewStyle().Faint(true)
		styles.ID = lipgloss.NewStyle().Faint(true)
		styles.Time = lipgloss.NewStyle().Faint(true)
		styles.Date = lipgloss.NewStyle().Faint(true)
		styles.Category = lipgloss.NewStyle().Bold(true)
		styles.Project = lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA"))
		styles.Tags = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#CBA6F7"))
		styles.Text = lipgloss.NewStyle()
		styles.Highlight = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F9E2AF"))
		styles.Success = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))
		styles.Error = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
		styles.Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAB387"))
	} else {
		// Monochrome styles
		styles.Title = lipgloss.NewStyle().Bold(true)
		styles.Separator = lipgloss.NewStyle()
		styles.Meta = lipgloss.NewStyle()
		styles.ID = lipgloss.NewStyle()
		styles.Time = lipgloss.NewStyle()
		styles.Date = lipgloss.NewStyle()
		styles.Category = lipgloss.NewStyle().Bold(true)
		styles.Project = lipgloss.NewStyle()
		styles.Tags = lipgloss.NewStyle()
		styles.Text = lipgloss.NewStyle()
		styles.Highlight = lipgloss.NewStyle().Bold(true)
		styles.Success = lipgloss.NewStyle()
		styles.Error = lipgloss.NewStyle()
		styles.Warning = lipgloss.NewStyle()
	}

	return styles
}

// RenderEntryList renders a list of entries according to the configured format
func (r *Renderer) RenderEntryList(list *EntryList) (string, error) {
	switch r.config.Format {
	case FormatJSON:
		return r.renderJSON(list)
	case FormatCSV:
		return r.renderCSV(list)
	case FormatTable:
		return r.renderTable(list)
	case FormatCompact:
		return r.renderCompact(list)
	case FormatQuiet:
		return r.renderQuiet(list)
	default:
		return r.renderDefault(list)
	}
}

// renderDefault renders entries in the default format (similar to current)
func (r *Renderer) renderDefault(list *EntryList) (string, error) {
	var builder strings.Builder

	// Header
	if list.Query != "" {
		builder.WriteString(r.styles.Title.Render("Search Results"))
		builder.WriteString("  ")
		builder.WriteString(r.styles.Separator.Render("query: "))
		builder.WriteString(list.Query)
	} else {
		builder.WriteString(r.styles.Title.Render("Recent Entries"))
		if list.Filters != nil && list.Filters["since"] != "" {
			builder.WriteString("  ")
			builder.WriteString(r.styles.Separator.Render("since "))
			builder.WriteString(r.styles.Meta.Render(list.Filters["since"]))
		}
	}
	builder.WriteString("\n")
	builder.WriteString(r.styles.Separator.Render(strings.Repeat("─", min(r.config.Width, 120))))
	builder.WriteString("\n")

	// Pagination info
	if list.TotalPages > 1 {
		start, end := list.Page*list.PerPage - list.PerPage + 1, list.Page*list.PerPage
		if end > list.Total {
			end = list.Total
		}
		builder.WriteString(r.styles.Meta.Render(fmt.Sprintf("Page %d of %d | Showing %d-%d of %d entries",
			list.Page, list.TotalPages, start, end, list.Total)))
		builder.WriteString("\n")
		builder.WriteString(r.styles.Separator.Render(strings.Repeat("─", min(r.config.Width, 120))))
		builder.WriteString("\n")
	}

	// Entries
	for _, entry := range list.Entries {
		builder.WriteString(r.renderSingleEntry(entry))
		builder.WriteString(r.styles.Separator.Render(strings.Repeat("─", min(r.config.Width, 120))))
		builder.WriteString("\n")
	}

	// Navigation hints
	if list.TotalPages > 1 {
		pagination := NewPagination(list.Total, list.PerPage, list.Page)
		if nav := pagination.FormatNavigation(); nav != "" {
			builder.WriteString(r.styles.Meta.Render(nav))
			builder.WriteString("\n")
		}
	}

	return builder.String(), nil
}

// renderSingleEntry renders a single entry
func (r *Renderer) renderSingleEntry(entry Entry) string {
	var builder strings.Builder

	// Meta line: [id] time date category project tags
	var metaParts []string

	if r.config.ShowID {
		metaParts = append(metaParts, r.styles.ID.Render(fmt.Sprintf("[%d]", entry.ID)))
	}

	if r.config.ShowTime {
		timeStr := entry.Timestamp.In(r.config.Location).Format("03:04 PM")
		metaParts = append(metaParts, r.styles.Time.Render(timeStr))
	}

	if r.config.ShowDate {
		dateStr := entry.Timestamp.In(r.config.Location).Format("2006-01-02")
		metaParts = append(metaParts, r.styles.Date.Render(dateStr))
	}

	if r.config.ShowCategory {
		catColor := colorForCategory(entry.Category)
		categoryStyle := r.styles.Category.Copy().Foreground(catColor)
		metaParts = append(metaParts, categoryStyle.Render(entry.Category))
	}

	if r.config.ShowProject && entry.Project != "" {
		metaParts = append(metaParts, r.styles.Project.Render("["+entry.Project+"]"))
	}

	if r.config.ShowTags && entry.Tags != "" {
		tags := strings.ReplaceAll(entry.Tags, ",", " #")
		metaParts = append(metaParts, r.styles.Tags.Render("#"+tags))
	}

	if len(metaParts) > 0 {
		builder.WriteString(strings.Join(metaParts, "  "))
		builder.WriteString("\n")
	}

	// Text content or search snippet
	text := entry.Text
	if entry.SearchSnippet != "" {
		// Highlight search matches
		highlighted := strings.ReplaceAll(entry.SearchSnippet, "[", r.styles.Highlight.Render("["))
		highlighted = strings.ReplaceAll(highlighted, "]", r.styles.Highlight.Render("]"))
		text = highlighted
	}

	if text != "" {
		builder.WriteString(r.styles.Text.Render("  " + text))
		builder.WriteString("\n")
	}

	// Additional metadata
	if r.config.ShowMeta {
		var metaInfo []string
		if entry.DurationMinutes > 0 {
			metaInfo = append(metaInfo, fmt.Sprintf("duration: %dm", entry.DurationMinutes))
		}
		if entry.SearchRank > 0 {
			metaInfo = append(metaInfo, fmt.Sprintf("rank: %.2f", entry.SearchRank))
		}
		if len(metaInfo) > 0 {
			builder.WriteString(r.styles.Meta.Render("  " + strings.Join(metaInfo, " | ")))
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// renderJSON renders entries as JSON
func (r *Renderer) renderJSON(list *EntryList) (string, error) {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// renderCSV renders entries as CSV
func (r *Renderer) renderCSV(list *EntryList) (string, error) {
	var builder strings.Builder

	// Header
	headers := []string{"id", "timestamp", "category", "text", "project", "tags", "duration_minutes"}
	if list.Query != "" {
		headers = append(headers, "search_rank", "search_snippet")
	}
	builder.WriteString(strings.Join(headers, ","))
	builder.WriteString("\n")

	// Data rows
	for _, entry := range list.Entries {
		row := []string{
			fmt.Sprintf("%d", entry.ID),
			entry.Timestamp.Format(time.RFC3339),
			entry.Category,
			escapeCSV(entry.Text),
			escapeCSV(entry.Project),
			escapeCSV(entry.Tags),
			fmt.Sprintf("%d", entry.DurationMinutes),
		}
		if list.Query != "" {
			row = append(row, fmt.Sprintf("%.2f", entry.SearchRank))
			row = append(row, escapeCSV(entry.SearchSnippet))
		}
		builder.WriteString(strings.Join(row, ","))
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// renderTable renders entries in a table format
func (r *Renderer) renderTable(list *EntryList) (string, error) {
	// This is a simplified table - could be enhanced with proper column alignment
	var builder strings.Builder

	// Header
	builder.WriteString("ID\tTime\tCategory\tProject\tTags\tText\n")
	builder.WriteString(strings.Repeat("-", r.config.Width))
	builder.WriteString("\n")

	// Data rows
	for _, entry := range list.Entries {
		timeStr := entry.Timestamp.In(r.config.Location).Format("15:04")
		tags := strings.ReplaceAll(entry.Tags, ",", " #")
		text := strings.ReplaceAll(entry.Text, "\n", " ")
		if len(text) > 50 {
			text = text[:47] + "..."
		}

		row := []string{
			fmt.Sprintf("%d", entry.ID),
			timeStr,
			entry.Category,
			entry.Project,
			tags,
			text,
		}
		builder.WriteString(strings.Join(row, "\t"))
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// renderCompact renders entries in a compact format
func (r *Renderer) renderCompact(list *EntryList) (string, error) {
	var builder strings.Builder

	for _, entry := range list.Entries {
		timeStr := entry.Timestamp.In(r.config.Location).Format("15:04")
		text := strings.ReplaceAll(entry.Text, "\n", " ")
		if len(text) > 80 {
			text = text[:77] + "..."
		}

		line := fmt.Sprintf("%s %s %s",
			r.styles.Time.Render(timeStr),
			r.styles.Category.Render(entry.Category),
			text)

		if entry.Project != "" {
			line += " " + r.styles.Project.Render("["+entry.Project+"]")
		}

		builder.WriteString(line)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// renderQuiet renders only the entry text (for scripting)
func (r *Renderer) renderQuiet(list *EntryList) (string, error) {
	var builder strings.Builder

	for _, entry := range list.Entries {
		builder.WriteString(entry.Text)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

func escapeCSV(s string) string {
	if strings.Contains(s, ",") || strings.Contains(s, "\"") || strings.Contains(s, "\n") {
		s = strings.ReplaceAll(s, "\"", "\"\"")
		return "\"" + s + "\""
	}
	return s
}