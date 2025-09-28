# Pulse 🕒  
A fast, colorful **CLI + TUI personal logging and time tracking tool** built in Go.  
Track work, notes, and timers right from your terminal — with SQLite storage, full-text search, and daily reminders.

---

## ✨ Features
- **CLI commands**
  - `pulse log "text"` → quick notes
  - `pulse start/stop` → track timers
  - `pulse list` → timeline view with colors
  - `pulse summary` → daily breakdowns
  - `pulse search` → full-text search with highlights
- **TUI** (`pulse tui`)  
  Scroll through logs with a clean, resizable interface
- **Reminders**  
  Configurable “end of day” reminder (default 17:00, Mon–Fri, skip holidays)
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
# Log a note
pulse log "Investigated incident 123"

# Start and stop timers
pulse start "Working on feature X" -p sesuite -t urgent
pulse stop --note "Finished draft"

# List entries (last 24h by default)
pulse list

# Search across history (with highlights)
pulse search "deploy failed" --project sesuite --tags bug

# TUI interface
pulse tui
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
make build     # build binary into bin/pulse
make run       # run in dev mode
make test      # run tests
make release   # cross-compile (dist/*)
```

### Dependencies

* Go 1.25+
* [Cobra](https://github.com/spf13/cobra) for CLI
* [Bubble Tea](https://github.com/charmbracelet/bubbletea) for TUI
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
