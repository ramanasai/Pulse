# Pulse 🕒
A fast, colorful **CLI personal logging and time tracking tool** built in Go.
Track work, notes, and timers right from your terminal — with SQLite storage, full-text search, and daily reminders.

---

## ✨ Features
- **CLI commands**
  - `pulse log "text"` → quick notes with categories, projects, and tags
  - `pulse start/stop` → track timers with projects and tags
  - `pulse list` → colorful timeline view with filtering options
  - `pulse summary` → daily breakdowns by category
  - `pulse search` → full-text search with highlights and filters
- **Timer Management**
  - Start/stop timers with automatic duration calculation
  - Support for concurrent timers with `--allow-multiple`
  - Add stop notes when completing tasks
- **Organization**
  - Categories: note, task, meeting, timer
  - Project-based organization
  - Tag system for flexible filtering
- **Reminders**
  Configurable "end of day" reminder (default 17:00, Mon–Fri, skip holidays)
- **SQLite storage**
  Local, portable, zero-config database
- **Colorful output** with [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Cross-platform** binaries for macOS, Linux, Windows

---

## 🚀 Installation

### From source
```bash
git clone https://github.com/ramanasai/Pulse.git
cd Pulse
make build
./bin/pulse --help
````

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

# View entries
pulse list                                                               # Show last 24h with colors
pulse list --limit 10                                                    # Limit to 10 entries
pulse list --since 2025-10-01T09:00:00                                   # Show entries since specific time

# Search and analyze
pulse search "deploy failed"                                             # Full-text search
pulse search "feature" --project myproject                               # Search within project
pulse search "urgent" --tags review                                      # Search by tags
pulse search "text:incident"                                             # Search specific field
pulse search "deploy*"                                                   # Wildcard search

# Daily summary
pulse summary                                                            # Show today's breakdown by category
```

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

### Viewing Data
```bash
pulse list                                 # List last 24h entries
pulse list --limit 50                     # Limit number of entries
pulse list --since 2025-10-01T09:00:00    # Entries since specific time
pulse summary                              # Daily summary by category
```

### Search
```bash
pulse search "query"                       # Basic full-text search
pulse search "text:specific"               # Search text field only
pulse search "pattern*"                    # Wildcard search
pulse search "query" --project proj        # Filter by project
pulse search "query" --tags tag1,tag2      # Filter by tags
pulse search "query" --since 2025-10-01    # Date range search
pulse search "query" --until 2025-10-07    # End date range
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

## 📦 Roadmap

* [ ] Export to CSV/Markdown
* [ ] Daily/weekly reports (Markdown)
* [ ] Packaging via GoReleaser

---

## 🤝 Contributing

PRs and issues welcome! Please file bugs and feature requests in [Issues](https://github.com/ramanasai/Pulse/issues).
