# Pulse 🕒
A fast, colorful **CLI personal logging and time tracking tool** built in Go.
Track work, notes, and timers right from your terminal — with SQLite storage, full-text search, advanced filtering, and multiple export formats.

---

## ✨ Features
- **CLI commands**
  - `pulse log "text"` → quick notes with categories, projects, and tags
  - `pulse start/stop` → track timers with projects and tags
  - `pulse list` → enhanced timeline view with pagination, filtering, and multiple output formats
  - `pulse summary` → daily breakdowns by category
  - `pulse search` → advanced full-text search with field-specific queries, highlighting, and filtering
- **🆕 Enhanced Output Formats**
  - Multiple output formats: `default`, `table`, `json`, `csv`, `compact`, `quiet`
  - Pagination support for large datasets
  - Export functionality for data analysis and scripting
  - Color-coded output with monochrome option
- **🆕 Advanced Search & Filtering**
  - Field-specific search: `category:task`, `project:api`, `tags:urgent`
  - Boolean search with natural language support
  - Date range filtering with natural language: "yesterday", "last week", "2 hours ago"
  - Preset date ranges: `--preset today`, `--preset last7days`, `--preset month`
  - Combined filters: projects, categories, tags, and date ranges
  - Search result ranking and snippet highlighting
- **🆕 Enhanced Date Handling**
  - Flexible date parsing: "2025-01-15", "Jan 15", "yesterday", "last week"
  - Relative date ranges: "2 days ago", "3 hours ago"
  - Timezone-aware display with configurable timezone
  - Multiple date presets for common time ranges
- **Timer Management**
  - Start/stop timers with automatic duration calculation
  - Support for concurrent timers with `--allow-multiple`
  - Add stop notes when completing tasks
- **Organization**
  - Categories: note, task, meeting, timer
  - Project-based organization
  - Tag system for flexible filtering
- **Reminders**
  - Configurable "end of day" reminder (default 17:00, Mon–Fri, skip holidays)
- **SQLite storage**
  - Local, portable, zero-config database
  - Full-text search (FTS5) for fast content discovery
- **Beautiful UI**
  - Colorful output with [Lipgloss](https://github.com/charmbracelet/lipgloss)
  - Responsive terminal width detection
  - Search result highlighting and ranked results
- **Cross-platform** binaries for macOS, Linux, Windows

---

## 🚀 Installation

### From source
```bash
git clone https://github.com/ramanasai/Pulse.git
cd Pulse
make build
./bin/pulse --help
```

### Prebuilt binaries

Coming soon (via GitHub Actions releases).

---

## ⚡ Quickstart

```bash
# Log different types of entries
pulse log "Investigated incident 123"                                    # Basic note
pulse log "Team standup meeting" -c meeting -p backend -t daily           # Meeting with project & tags
pulse log "Code review PR #456" -c task -p frontend -t review,urgent     # Task with multiple tags

# Start and stop timers
pulse start "Working on feature X" -p myproject -t development           # Start timer with project & tags
pulse stop --note "Completed implementation"                             # Stop timer with note
pulse start "Quick research" --allow-multiple                            # Start concurrent timer
pulse stop -i 42 --note "Research done"                                  # Stop specific timer by ID

# View entries (NEW: Enhanced with pagination and formats!)
pulse list                                                               # Show last 24h with colors
pulse list --format table --limit 10                                     # Table format with pagination
pulse list --preset last7days --format json                              # Export last 7 days as JSON
pulse list --categories task,meeting --format compact                    # Filter by categories
pulse list --projects api,frontend --page 2                              # Filter by projects with pagination

# Advanced search (NEW: Field-specific queries and natural language!)
pulse search "deploy failed"                                             # Full-text search
pulse search "category:meeting"                                          # Field-specific search
pulse search "project:frontend urgent"                                   # Combined search
pulse search "test*" --since yesterday --format table                    # Wildcard search with date filter
pulse search "review" --tags urgent --preset last30days --format csv     # Complex filter with export

# Daily summary
pulse summary                                                            # Show today's breakdown by category
```

---

## 📋 Command Reference

### Logging Entries
```bash
pulse log "Your message"                    # Basic note entry
pulse log "Message" -c task                # Specify category (note|task|meeting|timer)
pulse log "Message" -p project-name         # Assign to project
pulse log "Message" -t tag1,tag2           # Add multiple tags
pulse log "Meeting notes" -c meeting -p client-work -t important,review
```

### Timer Management
```bash
pulse start "Working on feature"           # Start a timer
pulse start "Task" -p project -t dev       # With project and tags
pulse start "Task" --allow-multiple        # Allow concurrent timers
pulse stop                                 # Stop active timer
pulse stop --note "Task completed"         # Add stop note
pulse stop -i 42 --note "Done"             # Stop specific timer by ID
```

### Viewing Data (NEW: Enhanced!)
```bash
# Basic viewing
pulse list                                 # List last 24h entries
pulse list --limit 50                     # Limit number of entries
pulse list --since yesterday              # Natural language dates
pulse list --preset last7days             # Date presets

# Filtering (NEW!)
pulse list --categories task,meeting      # Filter by categories
pulse list --projects api,frontend        # Filter by projects
pulse list --tags urgent,review           # Filter by tags
pulse list --category meeting --project backend  # Combined filters

# Output formats (NEW!)
pulse list --format table                 # Table format
pulse list --format json                  # JSON export
pulse list --format csv                   # CSV export
pulse list --format compact               # Compact view
pulse list --format quiet                 # Text only (for scripting)
pulse list --no-color                     # Disable colors

# Pagination (NEW!)
pulse list --limit 20 --page 1            # Page 1, 20 entries
pulse list --page 2                       # Next page
pulse summary                              # Daily summary by category
```

### Advanced Search (NEW: Enhanced!)
```bash
# Basic search
pulse search "query"                       # Basic full-text search
pulse search "pattern*"                    # Wildcard search

# Field-specific search (NEW!)
pulse search "category:meeting"            # Search by category
pulse search "project:frontend"            # Search by project
pulse search "tags:urgent"                 # Search by tags
pulse search "text:deploy failed"          # Search text field only

# Combined search (NEW!)
pulse search "project:api category:task"   # Multiple field filters
pulse search "urgent review" --project frontend  # Query + filters

# Date filtering (NEW!)
pulse search "meeting" --since yesterday   # Natural language dates
pulse search "incident" --preset last7days # Date presets
pulse search "review" --since "last week" --until today  # Date ranges

# Output formats for search (NEW!)
pulse search "deploy" --format table       # Table format
pulse search "error" --format json         # JSON with search metadata
pulse search "urgent" --format csv         # CSV export
pulse search "meeting" --format compact    # Compact view

# Search features (NEW!)
pulse search "test" --limit 10 --page 2    # Paginated search results
pulse search "feature" --no-color          # Monochrome output
```

---

## 🎯 Usage Examples

### Data Analysis & Export
```bash
# Export last month's data for analysis
pulse list --preset last30days --format csv > monthly_report.csv

# Get all urgent tasks in JSON format for processing
pulse search "category:task tags:urgent" --format json > urgent_tasks.json

# Create a daily summary in table format
pulse list --preset today --format table --no-color

# Pipe search results to other tools
pulse search "deploy" --format quiet | grep "production"

# Generate weekly report
pulse list --preset last7days --projects client-work --format json
```

### Monitoring & Tracking
```bash
# Track all meetings from yesterday
pulse search "category:meeting" --since yesterday --format table

# Find all urgent items from the last week
pulse search "urgent" --preset last7days --format compact

# Monitor work on specific project
pulse list --project api --preset today --format table

# Review timer activities
pulse list --category timer --preset lastweek --format table
```

### Natural Language Power
```bash
# Use natural language dates
pulse list --since "2 hours ago"
pulse search "bug" --since yesterday --until "1 hour ago"
pulse list --preset "last business week"

# Complex date queries
pulse search "meeting" --since "last Monday" --until "last Friday"
pulse list --since "3 days ago" --format compact
```

---

## ⚙️ Configuration

Pulse loads config from `~/.config/pulse/config.yaml`. Example:

```yaml
theme: "default"

reminder:
  enabled: true
  time: "17:00"            # HH:MM
  timezone: "Asia/Kolkata" # optional; defaults to system local time
  workdays: ["Mon","Tue","Wed","Thu","Fri"]
  holidays:
    - "2025-01-26"
    - "2025-08-15"
```

---

## 🛠️ Development

### Useful commands

```bash
make build       # build binary into bin/pulse
make build-local # build local development binary
make run         # run in dev mode
make test        # run tests
make release     # cross-compile (dist/*)
make tidy        # clean up dependencies
```

### Dependencies

* Go 1.25+
* [Cobra](https://github.com/spf13/cobra) for CLI
* [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling
* [Beeep](https://github.com/gen2brain/beeep) for notifications
* [SQLite (modernc.org/sqlite)](https://pkg.go.dev/modernc.org/sqlite)

---

## 📦 Recent Updates (v2.0.0)

### ✅ Quality of Life Improvements
- **Enhanced Pagination**: Added `--page` and `--limit` flags with "Showing X-Y of Z results" display
- **Multiple Output Formats**: Support for `default`, `table`, `json`, `csv`, `compact`, `quiet` formats
- **Smart Date Parsing**: Natural language dates like "yesterday", "last week", "2 hours ago"
- **Date Presets**: Quick access to common date ranges (`--preset today`, `--preset last7days`, etc.)
- **Better UI**: Responsive terminal width detection and improved visual hierarchy

### ✅ Enhanced Search & Filtering
- **Field-Specific Search**: Search by `category:task`, `project:api`, `tags:urgent`
- **Advanced Filters**: Combine project, category, and tag filters simultaneously
- **Search Ranking**: Results ranked by relevance with BM25 scoring
- **Search Snippets**: Highlighted text snippets showing matched terms
- **Export Capabilities**: All results exportable to JSON, CSV, and other formats
- **Pagination**: Large search results now paginated for better performance

### ✅ Developer Experience
- **Script-Friendly**: `--format quiet` and `--format json` for automation and scripting
- **Pipe Support**: All output formats work well with Unix pipes
- **Monochrome Option**: `--no-color` flag for plain text output
- **Better Error Handling**: Improved error messages and validation

---

## 🗺️ Roadmap

* [ ] Fuzzy search and typo tolerance
* [ ] Saved searches and frequently used filter presets
* [ ] Daily/weekly reports (Markdown templates)
* [ ] Data visualization and charts
* [ ] Web dashboard interface
* [ ] Integration with calendar systems
* [ ] Packaging via GoReleaser
* [ ] Plugin system for custom commands

---

## 🤝 Contributing

PRs and issues welcome! Please file bugs and feature requests in [Issues](https://github.com/ramanasai/Pulse/issues).

### Development Focus Areas
We're especially interested in contributions for:
- Additional output formats (Markdown, HTML)
- Performance optimizations for large datasets
- Plugin system architecture
- Internationalization and localization
- Advanced search algorithms and indexing