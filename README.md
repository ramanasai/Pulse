Here’s an updated **README.md** reflecting all the new features we wired up (filters, search, reminders, completion, packaging with GoReleaser, etc.) and showing how to install via prebuilt binaries and DEB/RPM packages.

---

````markdown
# Pulse 🕒  
A fast, colorful **CLI + TUI personal logging and time tracking tool** built in Go.  
Track work, notes, and timers right from your terminal — with SQLite storage, full-text search, and daily reminders.

---

## ✨ Features
- **CLI commands**
  - `pulse log "text"` → quick notes (auto-extracts `#tags`)
  - `pulse start/stop` → track timers with optional notes
  - `pulse list` → timeline view with colors + filters (`--project`, `--tags`, `--category`, `--any-tags`)
  - `pulse summary` → daily breakdowns
  - `pulse search` → full-text search with highlights
  - `pulse edit <id>` → edit entries inline or in `$EDITOR`
  - `pulse reply <id> "text"` → create threaded conversations (replies)
  - `pulse completion` → generate shell completions (bash/zsh/fish/powershell)
  - `pulse version` → shows version, commit, build date
- **TUI** (`pulse tui`)
  Rich terminal interface with:
  - Browse and search logs interactively
  - Time tracking reports and dashboards
  - Project-based time summaries
  - Tag analytics and insights
  - Threaded conversation view
  - Quick select picker for fast entry selection
  - Clean, resizable interface
- **Threading & Replies**
  Create threaded conversations with `pulse reply` - maintains parent-child relationships
- **Inline Editing**
  Edit existing entries with flags or open in `$EDITOR` with `pulse edit`
- **Smart Reply Selection**
  Reply to entries without knowing IDs using filters like `--last`, `--nth`, `--project-filter`, etc.
- **Reminders**
  Configurable "end of day" reminder (default 17:00, Mon–Fri, skip holidays)
- **SQLite storage**
  Local, portable, zero-config database with FTS5 full-text search
- **Colorful output** with [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Cross-platform** binaries for macOS, Linux, Windows
- **Packages**
  Prebuilt `.deb` and `.rpm` installers for Linux (Ubuntu/Debian/Fedora/RHEL)

---

## 🚀 Installation

### Prebuilt binaries

Download the latest release from [Releases](https://github.com/ramanasai/Pulse/releases).

#### Linux (Ubuntu/Debian)
```bash
# DEB package
sudo dpkg -i pulse_<version>_amd64.deb

# or tarball
tar -xzf pulse_<version>_linux_amd64.tar.gz
./pulse --help
````

#### Linux (Fedora/RHEL)

```bash
sudo rpm -Uvh pulse_<version>_amd64.rpm
```

#### macOS

```bash
tar -xzf pulse_<version>_darwin_arm64.tar.gz
./pulse --help
```

#### Windows

Unzip `pulse_<version>_windows_amd64.zip` and run:

```powershell
.\pulse.exe --help
```

### From source

```bash
git clone https://github.com/ramanasai/Pulse.git
cd Pulse
make build
./bin/pulse --help
```

---

## ⚡ Quickstart

```bash
# Log a note (with inline tags)
pulse log "Investigated outage #devops #urgent"

# Start and stop timers
pulse start "Working on feature X" -p sesuite -t urgent
pulse stop --note "Finished draft"

# List entries (last 24h by default)
pulse list
pulse list --project sesuite --tags urgent,backend
pulse list --tags devops,infra --any-tags

# Search across history (with highlights)
pulse search "deploy failed" --project sesuite --tags bug

# Edit existing entries
pulse edit 123 --text "Updated text" --project newproject
pulse edit 456 --editor  # opens in $EDITOR

# Create threaded conversations
pulse reply 123 "Follow-up: Fixed the issue"
pulse reply "Quick update" --last  # replies to most recent entry
pulse reply "Status update" --project-filter sesuite --tags-filter urgent

# Quick select via TUI (fastest way to edit/reply)
pulse tui  # press 'q' for quick select, pick entry, choose action

# TUI interface
pulse tui
```

### TUI Features

The Terminal User Interface provides powerful analytics and browsing capabilities:

```bash
# Launch the TUI
pulse tui
```

**TUI Capabilities:**
- **Interactive Log Browser** - Scroll through entries with search and filtering
- **Time Tracking Dashboard** - Visual breakdown of time spent across projects and categories
- **Project Summaries** - Detailed time analytics per project with charts
- **Tag Analytics** - Insights into most used tags and time distribution
- **Thread View** - Navigate threaded conversations in a tree structure
- **Quick Select Picker** - Fast entry selection with keyboard navigation
- **Real-time Search** - Filter entries as you type
- **Keyboard Shortcuts** - Efficient navigation (j/k, gg, G, /, n, N, q)
- **Responsive Layout** - Adapts to terminal size

**Quick Select Actions:**
- `q` - Open quick select picker
- `Enter` - Select highlighted entry
- `Esc` - Close picker
- Arrow Keys/j/k - Navigate options
- `/` - Filter picker options
- `Enter` on selection - Perform action (edit, reply, view details)
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

## 📝 Advanced Usage

### Editing Entries

Edit existing entries using flags or your preferred editor:

```bash
# Edit specific fields
pulse edit 123 --text "Updated description" --project newproject --tags urgent,backend --category task

# Open in $EDITOR
pulse edit 456 --editor

# Quick text replacement
pulse edit 789 -m "Corrected information"
```

### Threading & Replies

Create threaded conversations to track follow-ups and related work:

```bash
# Reply to a specific entry by ID
pulse reply 123 "Fixed the root cause - updated firewall rules"

# Smart reply without knowing IDs
pulse reply "Quick status update" --last                    # most recent entry
pulse reply "Found the issue" --nth 3                      # 3rd most recent
pulse reply "Still investigating" --project-filter sesuite # most recent in project
pulse reply "Working on fix" --tags-filter bug,urgent --any-tags

# Advanced filtering for replies
pulse reply "Deployed hotfix" \
  --project-filter sesuite \
  --category-filter task \
  --tags-filter production \
  --match "database" \
  --since 2025-01-01T09:00:00+05:30
```

### Advanced Filtering

Powerful filtering options for list and search commands:

```bash
# Multiple filter combinations
pulse list --project sesuite --tags urgent,backend
pulse list --tags devops,infra --any-tags  # OR logic
pulse list --category task --since 2025-01-01

# Search with context
pulse search "API failure" --project backend --tags bug,urgent
```

---

## 🔄 Shell Completions

Generate completions:

```bash
pulse completion bash > /etc/bash_completion.d/pulse
pulse completion zsh  > "${fpath[1]}/_pulse"
pulse completion fish > ~/.config/fish/completions/pulse.fish
```

Restart your shell and enjoy autocompletion.

---

## 🛠️ Development

### Useful commands

```bash
make build     # build binary into bin/pulse
make run       # run in dev mode
make test      # run tests
make release   # cross-compile (dist/*) via GoReleaser
make tag VERSION=0.2.1   # create + push a release tag
```

### Dependencies

* Go 1.25+
* [Cobra](https://github.com/spf13/cobra) for CLI
* [Bubble Tea](https://github.com/charmbracelet/bubbletea) for TUI
* [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling
* [Beeep](https://github.com/gen2brain/beeep) for notifications
* [SQLite (modernc.org/sqlite)](https://pkg.go.dev/modernc.org/sqlite)
* [Viper](https://github.com/spf13/viper) for configuration
* [GoReleaser](https://goreleaser.com) for packaging

---

## 📦 Roadmap

* [ ] Export to CSV/Markdown
* [ ] Daily/weekly reports (Markdown)
* [ ] Web interface for browsing logs
* [ ] Integration with calendar apps
* [ ] Data backup/sync functionality

---

## 🤝 Contributing

PRs and issues welcome! Please file bugs and feature requests in [Issues](https://github.com/ramanasai/Pulse/issues).
