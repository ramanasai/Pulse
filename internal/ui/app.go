package ui

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ramanasai/pulse/internal/config"
	"github.com/ramanasai/pulse/internal/db"
	"github.com/ramanasai/pulse/internal/notify"
	"github.com/ramanasai/pulse/internal/version"
)

type focusPane int
type mode int
type scope int
type picker int

const (
	focusTimeline focusPane = iota
	focusSidebar
	focusThread
)

const (
	modeNormal mode = iota
	modeSearch
	modeReply
	modeEdit
	modeHelp
	modeSince
	modePicker
	modeFocus
	modeStats
	modeCreate
	modeDashboard
	modeCalendar
	modeTemplates
	modeExport
	modeAdvancedSearch
	modeTimeReports
	modeProjectSummary
	modeTagAnalytics
	modeCommandPalette
	modeRichTextEditor
	modeTemplateEdit
)

const (
	scopeToday scope = iota
	scopeSince
	scopeAll
	scopeThisWeek
	scopeThisMonth
	scopeYesterday
	scopeLastWeek
	scopeLastMonth
	scopeCustom
)

const (
	pickProjects picker = iota
	pickCategories
	pickTags
)

type entry struct {
	id      int
	when    time.Time
	cat     string
	project string
	tags    []string
	text    string
}

type block struct {
	rootID     int
	rootCat    string
	latest     time.Time
	entries    []entry // chronological
	monthLabel string  // for first entry
}

type facetItem struct {
	name  string
	count int
}

type Template struct {
	ID          string
	Name        string
	Category    string
	Content     string
	Description string
	Variables   []string
	IsCustom    bool
	UsageCount  int
	LastUsed    time.Time
	IsFavorite  bool
}

type TemplateCategory struct {
	Name        string
	Icon        string
	Description string
	Color       string
	Templates   []string // template IDs
}

// Command palette structures
type Command struct {
	ID          string
	Name        string
	Description string
	Shortcut    string
	Category    string
	Action      func(Model) (Model, tea.Cmd)
}

type CommandCategory struct {
	Name        string
	Icon        string
	Description string
	Color       string
}

// Time tracking analytics data structures (using types from db package)
type TimeReportEntry = db.TimeReportEntry
type ProjectSummary = db.ProjectSummary
type TagAnalytics = db.TagAnalytics

type Model struct {
	// layout
	width, height int
	showSidebar   bool // collapsed by default
	showThread    bool // collapsed by default
	focus         focusPane
	mode          mode
	scope         scope

	// time & tz
	loc *time.Location
	now time.Time

	// filters
	filterText string              // live search across text/project/tags
	filterProj string              // selected project
	filterCat  string              // selected category
	filterTags map[string]struct{} // multiple tags
	anyTags    bool

	// timeline data
	blocks      []block
	cursorBlock int
	cursorEntry int

	// thread pane data
	threadBlock block

	// sidebar facets
	projects       []facetItem
	categories     []facetItem
	tags           []facetItem
	sidebarCursor  int
	sidebarSection int // 0=projects, 1=categories, 2=tags

	// pickers
	activePicker picker
	pickerCursor int

	// editors
	editor        textarea.Model
	editTargetID  int // id being edited (modeEdit)
	replyParentID int // parent id (modeReply)

	// modal buttons
	okButtonRect     [4]int // x, y, width, height for OK button
	cancelButtonRect [4]int // x, y, width, height for Cancel button
	selectedButton   int    // 0 for OK, 1 for Cancel (for tab navigation)

	// since input
	sinceInput textinput.Model
	sinceValue time.Time // for scopeSince

	// create entry form
	createText       textinput.Model
	createProject    AutocompleteModel
	createCategory   textinput.Model
	createTags       AutocompleteModel
	createField      int // which field is currently focused (0=text,1=project,2=category,3=tags)

	// editor fields for edit/reply modes
	editProject      AutocompleteModel
	editTags         AutocompleteModel
	editField        int // which field is currently focused (0=text,1=project,2=tags)

	// advanced search
	advancedSearchQuery    textinput.Model
	advancedSearchProject  textinput.Model
	advancedSearchCategory textinput.Model
	advancedSearchTags     textinput.Model
	advancedSearchField    int // which field is currently focused
	advancedSearchResults  []entry

	// template search
	templateSearchInput   textinput.Model
	templateSearchField   bool // whether template search input is focused

	// templates
	templates           []Template
	templateCategories   []TemplateCategory
	templateCursor       int
	templateActive       bool
	templateCategoryCursor int
	templateSearchQuery     string
	templateFilterMode      bool // filter by category or search

	// calendar view
	calendarDate        time.Time
	calendarView        int // 0=month, 1=week, 2=day
	calendarSelectedDate time.Time
	calendarEntryCounts  map[string]int // date string -> entry count
	calendarPreviewMode  bool // showing entry preview for selected date

	// export settings
	exportFormat string // markdown, json, csv
	exportPath   string

	// quick actions and productivity
	pomodoroActive   bool
	pomodoroTimeLeft time.Duration
	pomodoroSession  int // 0=work, 1=break
	workSessionTime  time.Duration
	breakSessionTime time.Duration

	// view preferences
	viewMode      int    // 0=timeline, 1=cards, 2=table, 3=kanban
	groupBy       string // category, project, date, tags
	sortBy        string // date, category, project, priority
	sortDirection bool   // true=asc, false=desc

	// styles
	st style

	// db handle
	db *sql.DB

	// configuration
	cfg config.Config

	// status messages
	status string

	// additional features
	bookmarks        map[int]struct{} // entry IDs bookmarked
	theme            int              // current theme index
	notifications    []string         // recent notifications
	focusMode        bool             // focus mode enabled
	showQuickActions bool             // quick actions menu visible
	showDashboard    bool             // dashboard view enabled
	pinnedEntries    map[int]struct{} // entry IDs pinned to top
	archiveMode      bool             // show archived entries

	// quick actions scrolling
	quickActionsPage int // current page for quick actions (0-based)

	// help scrolling
	helpScrollOffset int // scroll offset for help view

	// timeline scrolling
	timelineScrollOffset int // scroll offset for timeline view
	cardsScrollOffset   int // scroll offset for cards view
	tableScrollOffset   int // scroll offset for table view
	kanbanScrollOffset  int // scroll offset for kanban view

	// time tracking analytics
	timeReportScope    scope // scope for time reports (today, week, month, all)
	timeReportData     []TimeReportEntry
	projectSummaryData []ProjectSummary
	tagAnalyticsData   []TagAnalytics
	analyticsCursor    int // cursor for navigation in analytics views

	// enhanced analytics view modes
	analyticsViewMode   int // 0=table, 1=chart, 2=summary, 3=details
	timeReportView      int // 0=daily, 1=weekly, 2=monthly, 3=category
	projectSortBy       int // 0=total_time, 1=entry_count, 2=last_active, 3=name
	tagSortBy           int // 0=usage_count, 1=total_time, 2=last_used, 3=name
	analyticsFilter     string // filter text for analytics views

	// command palette
	commandPalette      textinput.Model
	commandPaletteInput string
	commands            []Command
	commandCategories   []CommandCategory
	commandCursor       int
	selectedCategory    int
	filteredCommands    []Command

	// accessibility features
	accessibilityMode   bool // screen reader mode
	highContrast        bool // high contrast theme
	reducedMotion       bool // reduce animations
	screenReaderBuffer  []string // buffer for screen reader announcements
	announcePriority    int     // announcement priority level

	// rich text editor
	richTextMode        bool   // rich text editing mode
	richTextFormat      string // current format type (markdown, html, plain)
	richTextToolbar     int    // selected toolbar item
	richTextPreview     bool   // show preview pane

	// template management
	dbTemplates         []Template     // templates loaded from database
	templateEditID      string         // ID of template being edited
	templateEditMode    bool           // whether in template edit mode
	templateCreateMode  bool           // whether in template create mode
	templateEditName    textinput.Model // template name input
	templateEditDesc    textinput.Model // template description input
	templateEditContent textarea.Model // template content input
	templateEditCategory textinput.Model // template category input

	// pomodoro enhancements
	pomodoroWorkSessions     int     // total completed work sessions
	pomodoroTotalTime        time.Duration // total pomodoro time tracked
	pomodoroAutoLog          bool    // auto-create log entries for completed sessions
	pomodoroLongBreakEnabled bool    // enable long breaks after 4 sessions
	pomodoroSessionsCount    int     // count for long break tracking
}

type style struct {
	topBar      lipgloss.Style
	statusBar   lipgloss.Style
	panelTitle  lipgloss.Style
	borderFocus lipgloss.Style
	borderDim   lipgloss.Style

	textDim  lipgloss.Style
	textBold lipgloss.Style
	project  lipgloss.Style
	tags     lipgloss.Style
	age      lipgloss.Style
	month    lipgloss.Style

	quickBar lipgloss.Style
	summary  lipgloss.Style
	sepFaint lipgloss.Style

	modalBox   lipgloss.Style
	modalTitle lipgloss.Style
}

func Run() error {
	cfg, _ := config.Load()
	loc := cfg.Location()

	dbh, err := db.Open()
	if err != nil {
		return err
	}
	_ = db.EnsureThreadColumns(dbh)

	ed := textarea.New()
	ed.Placeholder = "Type here…  (Ctrl+Enter to save, Esc to cancel)"
	ed.SetHeight(8)
	ed.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#313244"))

	si := textinput.New()
	si.Placeholder = "today | yesterday | 7d | 30d | YYYY-MM-DD"
	si.CharLimit = 64
	si.Width = 40

	// Create entry form inputs
	createText := textinput.New()
	createText.Placeholder = "Enter your note text..."
	createText.Width = 60
	createText.CharLimit = 1000

	// Create autocomplete inputs
	createProject := NewAutocomplete(dbh, SourceProjects, 5)
	createProject.SetPlaceholder("Project name (optional)")
	createProject.SetWidth(30)

	createCategory := textinput.New()
	createCategory.Placeholder = "note|task|meeting|timer"
	createCategory.Width = 25
	createCategory.CharLimit = 20
	createCategory.SetValue("note")

	createTags := NewAutocomplete(dbh, SourceTags, 8)
	createTags.SetPlaceholder("tag1, tag2, tag3")
	createTags.SetWidth(30)

	// Advanced search inputs
	advancedSearchQuery := textinput.New()
	advancedSearchQuery.Placeholder = "Search in text, project, tags..."
	advancedSearchQuery.Width = 50
	advancedSearchQuery.CharLimit = 500

	advancedSearchProject := textinput.New()
	advancedSearchProject.Placeholder = "Project filter"
	advancedSearchProject.Width = 25

	advancedSearchCategory := textinput.New()
	advancedSearchCategory.Placeholder = "Category filter"
	advancedSearchCategory.Width = 20

	advancedSearchTags := textinput.New()
	advancedSearchTags.Placeholder = "Tags filter"
	advancedSearchTags.Width = 25

	// Initialize comprehensive template collection
	templateCategories := []TemplateCategory{
		{
			Name:        "Work",
			Icon:        "💼",
			Description: "Work-related templates",
			Color:       "#89b4fa",
			Templates:   []string{"meeting_notes", "daily_standup", "project_update", "1on1_meeting", "performance_review"},
		},
		{
			Name:        "Development",
			Icon:        "💻",
			Description: "Software development templates",
			Color:       "#cba6f7",
			Templates:   []string{"bug_report", "code_review", "technical_design", "api_docs", "deployment_checklist"},
		},
		{
			Name:        "Personal",
			Icon:        "🏠",
			Description: "Personal productivity templates",
			Color:       "#a6e3a1",
			Templates:   []string{"daily_journal", "goal_setting", "habit_tracker", "reflection", "decision_matrix"},
		},
		{
			Name:        "Creative",
			Icon:        "🎨",
			Description: "Creative thinking templates",
			Color:       "#fab387",
			Templates:   []string{"brainstorming", "mind_mapping", "story_outline", "content_ideas"},
		},
		{
			Name:        "Quick",
			Icon:        "⚡",
			Description: "Quick action templates",
			Color:       "#f9e2af",
			Templates:   []string{"quick_note", "task_list", "decision_making"},
		},
	}

	templates := []Template{
		// Work templates
		{
			ID:          "meeting_notes",
			Name:        "Meeting Notes",
			Category:    "Work",
			Content:     "📅 **Meeting:** {{date}}\n👥 **Attendees:** \n⏰ **Duration:** \n\n📋 **Agenda:**\n- \n\n💡 **Key Points:**\n- \n\n✅ **Action Items:**\n- [ ] \n\n🎯 **Decisions:**\n- ",
			Description: "Structured meeting notes with agenda and action items",
			Variables:   []string{"date", "attendees", "duration"},
			IsCustom:    false,
		},
		{
			ID:          "daily_standup",
			Name:        "Daily Standup",
			Category:    "Work",
			Content:     "🌅 **Daily Standup - {{date}}**\n\n**Yesterday:**\n- \n\n**Today:**\n- \n\n**Blockers:**\n- \n\n**Goals for today:**\n- [ ] ",
			Description: "Daily standup notes for agile teams",
			Variables:   []string{"date"},
			IsCustom:    false,
		},
		{
			ID:          "project_update",
			Name:        "Project Update",
			Category:    "Work",
			Content:     "🚀 **Project Update - {{date}}**\n\n**Progress:**\n- ✅ \n\n**Challenges:**\n- ⚠️ \n\n**Next Milestone:**\n- 🎯 \n\n**Timeline:**\n- Current: \n- Target: ",
			Description: "Project status update template",
			Variables:   []string{"date"},
			IsCustom:    false,
		},
		{
			ID:          "1on1_meeting",
			Name:        "1-on-1 Meeting",
			Category:    "Work",
			Content:     "🤝 **1-on-1 Meeting - {{date}}**\n\n**Previous Action Items:**\n- [ ] \n\n**Discussion Points:**\n- \n\n**Feedback:**\n- Positive: \n- Areas for improvement: \n\n**New Action Items:**\n- [ ] \n\n**Follow-up:** {{next_week_date}}",
			Description: "Structured 1-on-1 meeting template",
			Variables:   []string{"date", "next_week_date"},
			IsCustom:    false,
		},
		{
			ID:          "performance_review",
			Name:        "Performance Review",
			Category:    "Work",
			Content:     "📊 **Performance Review - {{period}}**\n\n**Achievements:**\n- \n\n**Areas of Excellence:**\n- \n\n**Development Areas:**\n- \n\n**Goals for Next Period:**\n- [ ] \n\n**Feedback & Comments:**\n- ",
			Description: "Comprehensive performance review template",
			Variables:   []string{"period"},
			IsCustom:    false,
		},

		// Development templates
		{
			ID:          "bug_report",
			Name:        "Bug Report",
			Category:    "Development",
			Content:     "🐛 **Bug Report**\n\n**Title:** \n\n**Environment:**\n- OS: \n- Version: \n- Browser/Device: \n\n**Steps to Reproduce:**\n1. \n2. \n3. \n\n**Expected Behavior:**\n- \n\n**Actual Behavior:**\n- \n\n**Error Messages/Screenshots:**\n- ",
			Description: "Detailed bug report template",
			Variables:   []string{"title", "environment"},
			IsCustom:    false,
		},
		{
			ID:          "code_review",
			Name:        "Code Review",
			Category:    "Development",
			Content:     "🔍 **Code Review - PR #{{pr_number}}**\n\n**Summary:**\n- \n\n**General Comments:**\n- ✅ Strengths: \n- 💡 Suggestions: \n\n**Specific Issues:**\n- \n\n**Approval:**\n- [ ] Approve\n- [ ] Request changes\n- [ ] Needs work",
			Description: "Structured code review template",
			Variables:   []string{"pr_number"},
			IsCustom:    false,
		},
		{
			ID:          "technical_design",
			Name:        "Technical Design",
			Category:    "Development",
			Content:     "🏗️ **Technical Design**\n\n**Problem:**\n- \n\n**Proposed Solution:**\n- \n\n**Architecture:**\n- Components: \n- Data Flow: \n- APIs: \n\n**Trade-offs:**\n- Pros: \n- Cons: \n\n**Implementation Plan:**\n1. \n2. \n3. ",
			Description: "Technical design document template",
			Variables:   []string{},
			IsCustom:    false,
		},
		{
			ID:          "api_docs",
			Name:        "API Documentation",
			Category:    "Development",
			Content:     "📚 **API Documentation**\n\n**Endpoint:** `{{method}} {{endpoint}}`\n\n**Description:**\n- \n\n**Parameters:**\n- `{{param1}}` ({{type}}): {{description}}\n\n**Request Body:**\n```json\n{\n  \"key\": \"value\"\n}\n```\n\n**Response:**\n```json\n{\n  \"status\": \"success\",\n  \"data\": {}\n}\n```\n\n**Example:**\n```bash\ncurl -X {{method}} {{endpoint}} \\\n  -H \"Content-Type: application/json\" \\\n  -d '{{body}}'\n```",
			Description: "REST API documentation template",
			Variables:   []string{"method", "endpoint", "param1", "type", "description", "body"},
			IsCustom:    false,
		},
		{
			ID:          "deployment_checklist",
			Name:        "Deployment Checklist",
			Category:    "Development",
			Content:     "🚀 **Deployment Checklist - {{date}}**\n\n**Pre-deployment:**\n- [ ] Code reviewed and approved\n- [ ] Tests passing ({{test_coverage}}% coverage)\n- [ ] Documentation updated\n- [ ] Migration scripts tested\n- [ ] Backup performed\n\n**Deployment:**\n- [ ] {{deployment_time}} - Deploy to staging\n- [ ] {{deployment_time}} - Smoke tests\n- [ ] {{deployment_time}} - Deploy to production\n\n**Post-deployment:**\n- [ ] Health checks passed\n- [ ] Monitor for {{monitoring_hours}} hours\n- [ ] Team notified\n- [ ] Rollback plan ready",
			Description: "Complete deployment checklist",
			Variables:   []string{"date", "test_coverage", "deployment_time", "monitoring_hours"},
			IsCustom:    false,
		},

		// Personal templates
		{
			ID:          "daily_journal",
			Name:        "Daily Journal",
			Category:    "Personal",
			Content:     "📖 **Daily Journal - {{date}}**\n\n**Today's Highlights:**\n- 😊 \n\n**Achievements:**\n- ✅ \n\n**Challenges:**\n- \n\n**What I Learned:**\n- \n\n**Gratitude:**\n- 🙏 \n\n**Tomorrow's Focus:**\n- ",
			Description: "Daily journaling template",
			Variables:   []string{"date"},
			IsCustom:    false,
		},
		{
			ID:          "goal_setting",
			Name:        "Goal Setting",
			Category:    "Personal",
			Content:     "🎯 **Goal Setting - {{timeframe}}**\n\n**SMART Goals:**\n\n**Specific:**\n- \n\n**Measurable:**\n- \n\n**Achievable:**\n- \n\n**Relevant:**\n- \n\n**Time-bound:**\n- Target: {{deadline}}\n\n**Action Steps:**\n1. \n2. \n3. \n\n**Progress Tracking:**\n- Current: \n- Target: ",
			Description: "SMART goal setting template",
			Variables:   []string{"timeframe", "deadline"},
			IsCustom:    false,
		},
		{
			ID:          "habit_tracker",
			Name:        "Habit Tracker",
			Category:    "Personal",
			Content:     "📊 **Habit Tracker - Week of {{week_date}}**\n\n**Daily Habits:**\n| Habit | Mon | Tue | Wed | Thu | Fri | Sat | Sun |\n|-------|-----|-----|-----|-----|-----|-----|-----|\n| 🏃 Exercise | ☐ | ☐ | ☐ | ☐ | ☐ | ☐ | ☐ |\n| 📚 Reading | ☐ | ☐ | ☐ | ☐ | ☐ | ☐ | ☐ |\n| 💧 Water 8x | ☐ | ☐ | ☐ | ☐ | ☐ | ☐ | ☐ |\n| 🧘 Meditation | ☐ | ☐ | ☐ | ☐ | ☐ | ☐ | ☐ |\n\n**Weekly Goal:**\n- \n\n**Reflection:**\n- ",
			Description: "Weekly habit tracking template",
			Variables:   []string{"week_date"},
			IsCustom:    false,
		},
		{
			ID:          "reflection",
			Name:        "Weekly Reflection",
			Category:    "Personal",
			Content:     "💭 **Weekly Reflection - {{week_date}}**\n\n**Wins:**\n- \n\n**Challenges:**\n- \n\n**Lessons Learned:**\n- \n\n**What to Improve:**\n- \n\n**Focus for Next Week:**\n- \n\n**Personal Growth:**\n- ",
			Description: "Weekly self-reflection template",
			Variables:   []string{"week_date"},
			IsCustom:    false,
		},
		{
			ID:          "decision_matrix",
			Name:        "Decision Matrix",
			Category:    "Personal",
			Content:     "⚖️ **Decision Matrix**\n\n**Decision:** {{decision_topic}}\n\n**Options:**\n1. {{option_1}}\n2. {{option_2}}\n3. {{option_3}}\n\n**Criteria:**\n- Cost (1-5)\n- Time (1-5)\n- Impact (1-5)\n- Effort (1-5)\n\n**Analysis:**\n| Option | Cost | Time | Impact | Effort | Total |\n|--------|------|------|--------|--------|-------|\n| 1 | | | | | |\n| 2 | | | | | |\n| 3 | | | | | |\n\n**Conclusion:**\n- Selected: ",
			Description: "Structured decision-making template",
			Variables:   []string{"decision_topic", "option_1", "option_2", "option_3"},
			IsCustom:    false,
		},

		// Creative templates
		{
			ID:          "brainstorming",
			Name:        "Brainstorming",
			Category:    "Creative",
			Content:     "💡 **Brainstorming Session**\n\n**Topic:** {{topic}}\n\n**Initial Ideas (No Filter):**\n- \n- \n- \n- \n- \n\n**Category Grouping:**\n**Theme 1:**\n- \n**Theme 2:**\n- \n\n**Top 3 Ideas:**\n1. \n2. \n3. \n\n**Next Steps:**\n- Research: \n- Prototype: \n- Test: ",
			Description: "Creative brainstorming template",
			Variables:   []string{"topic"},
			IsCustom:    false,
		},
		{
			ID:          "mind_mapping",
			Name:        "Mind Mapping",
			Category:    "Creative",
			Content:     "🧠 **Mind Map - {{central_idea}}**\n\n**Central Idea:** {{central_idea}}\n\n**Main Branches:**\n- **Branch 1:** {{branch_1}}\n  - Sub-branch: \n  - Sub-branch: \n\n- **Branch 2:** {{branch_2}}\n  - Sub-branch: \n  - Sub-branch: \n\n- **Branch 3:** {{branch_3}}\n  - Sub-branch: \n  - Sub-branch: \n\n**Connections:**\n- \n\n**Insights:**\n- ",
			Description: "Mind mapping template for visual thinking",
			Variables:   []string{"central_idea", "branch_1", "branch_2", "branch_3"},
			IsCustom:    false,
		},
		{
			ID:          "story_outline",
			Name:        "Story Outline",
			Category:    "Creative",
			Content:     "📚 **Story Outline**\n\n**Title:** {{title}}\n\n**Logline:** {{logline}}\n\n**Characters:**\n- **Protagonist:** {{protagonist}} - {{protagonist_goal}}\n- **Antagonist:** {{antagonist}} - {{antagonist_goal}}\n- **Supporting:** {{supporting_character}}\n\n**Plot Structure:**\n**Act 1: Setup**\n- Opening: \n- Inciting Incident: \n\n**Act 2: Confrontation**\n- Rising Action: \n- Midpoint: \n- Crisis: \n\n**Act 3: Resolution**\n- Climax: \n- Resolution: \n\n**Themes:**\n- \n\n**Setting:**\n- ",
			Description: "Story structure template",
			Variables:   []string{"title", "logline", "protagonist", "protagonist_goal", "antagonist", "antagonist_goal", "supporting_character"},
			IsCustom:    false,
		},
		{
			ID:          "content_ideas",
			Name:        "Content Ideas",
			Category:    "Creative",
			Content:     "🎨 **Content Ideas Brainstorm**\n\n**Target Audience:** {{audience}}\n\n**Content Pillars:**\n1. {{pillar_1}}\n2. {{pillar_2}}\n3. {{pillar_3}}\n\n**Content Formats:**\n- Blog posts\n- Videos\n- Social media\n- Podcasts\n- Infographics\n\n**Idea Generation:**\n\n**For {{pillar_1}}:**\n- \n- \n\n**For {{pillar_2}}:**\n- \n- \n\n**For {{pillar_3}}:**\n- \n- \n\n**Content Calendar:**\n- Week 1: \n- Week 2: \n- Week 3: \n- Week 4: \n\n**Keywords/Topics:**\n- ",
			Description: "Content planning template",
			Variables:   []string{"audience", "pillar_1", "pillar_2", "pillar_3"},
			IsCustom:    false,
		},

		// Quick templates
		{
			ID:          "quick_note",
			Name:        "Quick Note",
			Category:    "Quick",
			Content:     "📝 **Quick Note - {{date}} {{time}}**\n\n",
			Description: "Simple quick note template",
			Variables:   []string{"date", "time"},
			IsCustom:    false,
		},
		{
			ID:          "task_list",
			Name:        "Task List",
			Category:    "Quick",
			Content:     "✅ **Task List - {{date}}**\n\n**High Priority:**\n- [ ] \n- [ ] \n\n**Medium Priority:**\n- [ ] \n- [ ] \n\n**Low Priority:**\n- [ ] \n- [ ] \n\n**Completed:**\n- [x] ",
			Description: "Quick task list template",
			Variables:   []string{"date"},
			IsCustom:    false,
		},
		{
			ID:          "decision_making",
			Name:        "Quick Decision",
			Category:    "Quick",
			Content:     "⚖️ **Quick Decision**\n\n**Decision:** {{decision}}\n\n**Options:**\n✅ **Pros:**\n- \n- \n\n❌ **Cons:**\n- \n- \n\n**Decision:** \n**Rationale:** \n\n**Follow-up:** ",
			Description: "Quick decision-making template",
			Variables:   []string{"decision"},
			IsCustom:    false,
		},
	}

	m := Model{
		showSidebar:    false,
		showThread:     false,
		focus:          focusTimeline,
		mode:           modeNormal,
		scope:          scopeToday,
		loc:            loc,
		now:            time.Now().In(loc),
		db:             dbh,
		cfg:            cfg,
		filterTags:     map[string]struct{}{},
		editor:         ed,
		sinceInput:     si,
		createText:     createText,
		createProject:  createProject,
		createCategory: createCategory,
		createTags:     createTags,
		createField:    0,

		// Editor fields for edit/reply modes
		editProject:    NewAutocomplete(dbh, SourceProjects, 5),
		editTags:       NewAutocomplete(dbh, SourceTags, 8),
		editField:      0,

		// Advanced search
		advancedSearchQuery:    advancedSearchQuery,
		advancedSearchProject:  advancedSearchProject,
		advancedSearchCategory: advancedSearchCategory,
		advancedSearchTags:     advancedSearchTags,
		advancedSearchField:    0,
		advancedSearchResults:  []entry{},

		// Templates and calendar
		templates:           templates,
		templateCategories:   templateCategories,
		templateCursor:       0,
		templateActive:       false,
		templateCategoryCursor: 0,
		templateSearchQuery:   "",
		calendarDate:         time.Now().In(loc),
		calendarView:         0,
		calendarSelectedDate: time.Now().In(loc),
		calendarEntryCounts:  make(map[string]int),
		calendarPreviewMode:  false,

		// Export settings
		exportFormat: "markdown",
		exportPath:   "",

		// Productivity features
		pomodoroActive:   false,
		pomodoroTimeLeft: 0,
		pomodoroSession:  0,
		workSessionTime:  25 * time.Minute,
		breakSessionTime: 5 * time.Minute,

		// pomodoro enhancements
		pomodoroWorkSessions:     0,
		pomodoroTotalTime:        0,
		pomodoroAutoLog:          true,
		pomodoroLongBreakEnabled: true,
		pomodoroSessionsCount:    0,

		// View preferences
		viewMode:      0, // timeline
		groupBy:       "date",
		sortBy:        "date",
		sortDirection: false, // desc (newest first)

		bookmarks:            make(map[int]struct{}),
		pinnedEntries:        make(map[int]struct{}),
		theme:                0,
		notifications:        []string{},
		focusMode:            false,
		showQuickActions:     false,
		showDashboard:        false,
		archiveMode:          false,
		selectedButton:       0,
		quickActionsPage:     0,
		helpScrollOffset:     0,
		timelineScrollOffset: 0,
		cardsScrollOffset:     0,
		tableScrollOffset:     0,
		kanbanScrollOffset:    0,

		// time tracking analytics
		timeReportScope:      scopeThisWeek,
		timeReportData:       []TimeReportEntry{},
		projectSummaryData:   []ProjectSummary{},
		tagAnalyticsData:     []TagAnalytics{},
		analyticsCursor:      0,

		// enhanced analytics view modes
		analyticsViewMode:    0, // table view
		timeReportView:       0, // daily view
		projectSortBy:        0, // by total time
		tagSortBy:            0, // by usage count
		analyticsFilter:      "",

		// command palette
		commandPalette:       textinput.New(),
		commandPaletteInput:  "",
		commandCursor:        0,
		selectedCategory:     0,
		filteredCommands:     []Command{},

		st: style{
			topBar:      lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")).Bold(true).Padding(0, 1),
			statusBar:   lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")).Background(lipgloss.Color("#313244")).Padding(0, 1),
			panelTitle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#bac2de")).Bold(true),
			borderFocus: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#89B4FA")).Padding(0, 1),
			borderDim:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#585b70")).Padding(0, 1),

			textDim:  lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			textBold: lipgloss.NewStyle().Bold(true),
			project:  lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA")),
			tags:     lipgloss.NewStyle().Foreground(lipgloss.Color("#CBA6F7")).Faint(true),
			age:      lipgloss.NewStyle().Faint(true),
			month:    lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")).Bold(true),

			quickBar: lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")).Background(lipgloss.Color("#1e1e2e")).Padding(0, 1),
			summary:  lipgloss.NewStyle().Foreground(lipgloss.Color("#bac2de")).Padding(0, 1),
			sepFaint: lipgloss.NewStyle().Faint(true),

			modalBox:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#89B4FA")).Padding(1, 2).Width(70),
			modalTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cdd6f4")),
		},
	}

	// Initialize command palette
	m.commandPalette.Placeholder = "Type a command or search..."
	m.commandPalette.CharLimit = 156
	m.commandPalette.Width = 50

	// Initialize commands
	m.commands = []Command{
		// Navigation commands
		{ID: "goto_today", Name: "Go to Today", Description: "Jump to today's entries", Shortcut: "Ctrl+G", Category: "Navigation", Action: func(model Model) (Model, tea.Cmd) { model.scope = scopeToday; return model, model.loadTimelineCmd() }},
		{ID: "goto_this_week", Name: "This Week", Description: "Show this week's entries", Shortcut: "Ctrl+W", Category: "Navigation", Action: func(model Model) (Model, tea.Cmd) { model.scope = scopeThisWeek; return model, model.loadTimelineCmd() }},
		{ID: "goto_this_month", Name: "This Month", Description: "Show this month's entries", Shortcut: "Ctrl+M", Category: "Navigation", Action: func(model Model) (Model, tea.Cmd) { model.scope = scopeThisMonth; return model, model.loadTimelineCmd() }},
		{ID: "goto_all", Name: "All Time", Description: "Show all entries", Shortcut: "Ctrl+A", Category: "Navigation", Action: func(model Model) (Model, tea.Cmd) { model.scope = scopeAll; return model, model.loadTimelineCmd() }},

		// View commands
		{ID: "toggle_sidebar", Name: "Toggle Sidebar", Description: "Show/hide sidebar", Shortcut: "Ctrl+B", Category: "View", Action: func(model Model) (Model, tea.Cmd) { model.showSidebar = !model.showSidebar; return model, nil }},
		{ID: "toggle_theme", Name: "Toggle Theme", Description: "Cycle through themes", Shortcut: "Ctrl+T", Category: "View", Action: func(model Model) (Model, tea.Cmd) { model.theme = (model.theme + 1) % 4; model.addNotification("Theme changed"); return model, nil }},
		{ID: "toggle_focus", Name: "Focus Mode", Description: "Toggle distraction-free mode", Shortcut: "Ctrl+F", Category: "View", Action: func(model Model) (Model, tea.Cmd) { model.focusMode = !model.focusMode; return model, nil }},
		{ID: "dashboard", Name: "Dashboard", Description: "Show project dashboard", Shortcut: "Ctrl+D", Category: "View", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeDashboard; return model, model.loadProjectSummaryCmd() }},

		// Creation commands
		{ID: "create_note", Name: "New Note", Description: "Create a new note entry", Shortcut: "N", Category: "Create", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeCreate; model.createCategory.SetValue("note"); return model, nil }},
		{ID: "create_task", Name: "New Task", Description: "Create a new task entry", Shortcut: "T", Category: "Create", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeCreate; model.createCategory.SetValue("task"); return model, nil }},
		{ID: "create_meeting", Name: "New Meeting", Description: "Create a new meeting entry", Shortcut: "M", Category: "Create", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeCreate; model.createCategory.SetValue("meeting"); return model, nil }},
		{ID: "rich_text_editor", Name: "Rich Text Editor", Description: "Advanced rich text editor with markdown", Shortcut: "Ctrl+Shift+E", Category: "Create", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeRichTextEditor; return model, nil }},
		{ID: "templates", Name: "Templates", Description: "Browse template library", Shortcut: "Ctrl+Shift+T", Category: "Create", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeTemplates; return model, nil }},

		// Analytics commands
		{ID: "time_reports", Name: "Time Reports", Description: "View time tracking analytics", Shortcut: "Ctrl+R", Category: "Analytics", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeTimeReports; return model, model.loadTimeReportsCmd() }},
		{ID: "project_summary", Name: "Project Summary", Description: "View project analytics", Shortcut: "Ctrl+P", Category: "Analytics", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeProjectSummary; return model, model.loadProjectSummaryCmd() }},
		{ID: "tag_analytics", Name: "Tag Analytics", Description: "View tag usage statistics", Shortcut: "Ctrl+Shift+P", Category: "Analytics", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeTagAnalytics; return model, model.loadTagAnalyticsCmd() }},
		{ID: "calendar", Name: "Calendar View", Description: "Browse entries by date", Shortcut: "Ctrl+C", Category: "Analytics", Action: func(model Model) (Model, tea.Cmd) {
		model.mode = modeCalendar
		model.calendarDate = model.now
		model.calendarSelectedDate = model.now
		model.calendarPreviewMode = false
		model.loadCalendarEntryCounts()
		return model, nil
	}},

		// Search commands
		{ID: "search", Name: "Search", Description: "Search through entries", Shortcut: "/", Category: "Search", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeSearch; return model, nil }},
		{ID: "advanced_search", Name: "Advanced Search", Description: "Advanced search with filters", Shortcut: "Ctrl+/", Category: "Search", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeAdvancedSearch; return model, nil }},

		// Export commands
		{ID: "export", Name: "Export", Description: "Export entries to file", Shortcut: "Ctrl+E", Category: "Export", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeExport; return model, nil }},

		// Productivity commands
		{ID: "pomodoro_start", Name: "Start Pomodoro", Description: "Start a 25-minute work session", Shortcut: "Ctrl+Shift+S", Category: "Productivity", Action: func(model Model) (Model, tea.Cmd) {
			if !model.pomodoroActive {
				model.pomodoroActive = true
				model.pomodoroTimeLeft = model.workSessionTime
				model.pomodoroSession = 1
				model.addNotification("Pomodoro session started")
				return model, pomodoroTick()
			}
			return model, nil
		}},
		{ID: "pomodoro_break", Name: "Start Break", Description: "Start a 5-minute break", Shortcut: "Ctrl+Shift+B", Category: "Productivity", Action: func(model Model) (Model, tea.Cmd) {
			if !model.pomodoroActive {
				model.pomodoroActive = true
				model.pomodoroTimeLeft = model.breakSessionTime
				model.pomodoroSession = 0
				model.addNotification("Break started")
				return model, pomodoroTick()
			}
			return model, nil
		}},
		{ID: "pomodoro_stop", Name: "Stop Pomodoro", Description: "Stop the current Pomodoro session", Shortcut: "Ctrl+Shift+X", Category: "Productivity", Action: func(model Model) (Model, tea.Cmd) {
			if model.pomodoroActive {
				model.pomodoroActive = false
				model.addNotification("Pomodoro session stopped")
			}
			return model, nil
		}},
		{ID: "pomodoro_stats", Name: "Pomodoro Stats", Description: "View Pomodoro session statistics", Shortcut: "Ctrl+Shift+P", Category: "Productivity", Action: func(model Model) (Model, tea.Cmd) {
			statsMsg := fmt.Sprintf("🍅 Pomodoro Statistics\nTotal Sessions: %d\nTotal Focus Time: %s\nAvg Session: %s",
				model.pomodoroWorkSessions,
				model.pomodoroTotalTime.Round(time.Minute),
				func() time.Duration {
					if model.pomodoroWorkSessions > 0 {
						return model.pomodoroTotalTime / time.Duration(model.pomodoroWorkSessions)
					}
					return 0
				}().Round(time.Minute))
			model.addNotification(statsMsg)
			model.announceToScreenReader(statsMsg)
			return model, nil
		}},
		{ID: "pomodoro_toggle_autolog", Name: "Toggle Auto-Log", Description: "Toggle automatic logging of Pomodoro sessions", Shortcut: "Ctrl+Shift+L", Category: "Productivity", Action: func(model Model) (Model, tea.Cmd) {
			model.pomodoroAutoLog = !model.pomodoroAutoLog
			status := "disabled"
			if model.pomodoroAutoLog {
				status = "enabled"
			}
			model.addNotification(fmt.Sprintf("Pomodoro auto-logging %s", status))
			return model, nil
		}},

		// Utility commands
		{ID: "help", Name: "Help", Description: "Show keyboard shortcuts", Shortcut: "F1", Category: "Utility", Action: func(model Model) (Model, tea.Cmd) { model.mode = modeHelp; return model, nil }},
		{ID: "quit", Name: "Quit", Description: "Exit Pulse", Shortcut: "Ctrl+Q", Category: "Utility", Action: func(model Model) (Model, tea.Cmd) { return model, func() tea.Msg { return tea.Quit() } }},

		// Accessibility commands
		{ID: "toggle_screen_reader", Name: "Toggle Screen Reader", Description: "Enable/disable screen reader mode", Shortcut: "Ctrl+F12", Category: "Accessibility", Action: func(model Model) (Model, tea.Cmd) {
			model.accessibilityMode = !model.accessibilityMode
			if model.accessibilityMode {
				model.addNotification("Screen reader mode enabled")
				model.announceToScreenReader("Screen reader mode enabled")
			} else {
				model.addNotification("Screen reader mode disabled")
			}
			return model, nil
		}},
		{ID: "toggle_high_contrast", Name: "Toggle High Contrast", Description: "Enable/disable high contrast theme", Shortcut: "Ctrl+F11", Category: "Accessibility", Action: func(model Model) (Model, tea.Cmd) {
			model.highContrast = !model.highContrast
			model.applyAccessibilityTheme()
			if model.highContrast {
				model.addNotification("High contrast mode enabled")
				model.announceToScreenReader("High contrast mode enabled")
			} else {
				model.addNotification("High contrast mode disabled")
			}
			return model, nil
		}},
		{ID: "announce_context", Name: "Announce Context", Description: "Announce current context for screen reader", Shortcut: "Ctrl+F10", Category: "Accessibility", Action: func(model Model) (Model, tea.Cmd) {
			context := model.getCurrentContextForScreenReader()
			model.announceToScreenReader(context)
			return model, nil
		}},
	}

	// Initialize command categories
	m.commandCategories = []CommandCategory{
		{Name: "Navigation", Icon: "🧭", Description: "Navigate through different time ranges", Color: "#89B4FA"},
		{Name: "View", Icon: "👁️", Description: "Change interface layout and appearance", Color: "#CBA6F7"},
		{Name: "Create", Icon: "✏️", Description: "Create new entries and content", Color: "#A6E3A1"},
		{Name: "Analytics", Icon: "📊", Description: "View statistics and reports", Color: "#F9E2AF"},
		{Name: "Search", Icon: "🔍", Description: "Search and filter entries", Color: "#FAB387"},
		{Name: "Export", Icon: "📤", Description: "Export data to different formats", Color: "#94E2D5"},
		{Name: "Productivity", Icon: "⚡", Description: "Productivity tools and timers", Color: "#F2CDCD"},
		{Name: "Utility", Icon: "🛠️", Description: "Help and utility functions", Color: "#B4BEFE"},
		{Name: "Accessibility", Icon: "♿", Description: "Accessibility and screen reader features", Color: "#F2CDCD"},
	}

	// Initialize filtered commands with all commands
	m.filteredCommands = make([]Command, len(m.commands))
	copy(m.filteredCommands, m.commands)

	// Initialize accessibility features
	m.accessibilityMode = os.Getenv("SCREEN_READER") != "" || os.Getenv("ACCESSIBILITY") != ""
	m.highContrast = os.Getenv("HIGH_CONTRAST") != ""
	m.reducedMotion = os.Getenv("REDUCED_MOTION") != ""
	m.screenReaderBuffer = []string{}
	m.announcePriority = 0

	// Initialize rich text editor
	m.richTextMode = true // default to rich text
	m.richTextFormat = "markdown"
	m.richTextToolbar = 0
	m.richTextPreview = false

	// Initialize template management
	templateEditName := textinput.New()
	templateEditName.Placeholder = "Template name"
	templateEditName.Width = 40

	templateEditDesc := textinput.New()
	templateEditDesc.Placeholder = "Template description"
	templateEditDesc.Width = 40

	templateEditCategory := textinput.New()
	templateEditCategory.Placeholder = "Category"
	templateEditCategory.Width = 30

	templateEditContent := textarea.New()
	templateEditContent.Placeholder = "Template content..."
	templateEditContent.SetWidth(60)
	templateEditContent.SetHeight(15)

	m.dbTemplates = []Template{}
	m.templateEditID = ""
	m.templateEditMode = false
	m.templateCreateMode = false
	m.templateEditName = templateEditName
	m.templateEditDesc = templateEditDesc
	m.templateEditContent = templateEditContent
	m.templateEditCategory = templateEditCategory

	// Apply accessibility theme if needed
	m.applyAccessibilityTheme()

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
	_, runErr := p.Run()
	_ = dbh.Close()
	return runErr
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickNow(), m.loadFacetsCmd(), m.loadTimelineCmd(), m.loadTemplatesCmd())
}

// ---------- messages & commands ----------

type tickMsg struct{ now time.Time }
type pomodoroTickMsg struct{}

func tickNow() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{now: time.Now()} })
}

func pomodoroTick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return pomodoroTickMsg{} })
}

type blocksLoadedMsg struct {
	blocks []block
	err    error
}
type facetsLoadedMsg struct {
	projects []facetItem
	cats     []facetItem
	tags     []facetItem
	err      error
}

// Analytics data loading messages
type timeReportsLoadedMsg struct {
	data []TimeReportEntry
	err  error
}

type projectSummaryLoadedMsg struct {
	data []ProjectSummary
	err  error
}

type tagAnalyticsLoadedMsg struct {
	data []TagAnalytics
	err  error
}

type templatesLoadedMsg struct {
	templates []Template
	err       error
}

func (m Model) loadTimelineCmd() tea.Cmd {
	return func() tea.Msg {
		blocks, err := loadBlocks(m.db, m.loc, m.scope, m.filterText, m.filterProj, m.filterCat, m.filterTags, m.anyTags, m.sinceValue)
		return blocksLoadedMsg{blocks: blocks, err: err}
	}
}
func (m Model) loadFacetsCmd() tea.Cmd {
	return func() tea.Msg {
		projects, cats, tags, err := loadFacets(m.db)
		return facetsLoadedMsg{projects: projects, cats: cats, tags: tags, err: err}
	}
}

// Analytics data loading commands
func (m Model) loadTimeReportsCmd() tea.Cmd {
	return func() tea.Msg {
		data, err := db.LoadTimeReports(m.db, m.loc, int(m.timeReportScope))
		return timeReportsLoadedMsg{data: data, err: err}
	}
}

func (m Model) loadProjectSummaryCmd() tea.Cmd {
	return func() tea.Msg {
		data, err := db.LoadProjectSummary(m.db, m.loc)
		return projectSummaryLoadedMsg{data: data, err: err}
	}
}

func (m Model) loadTagAnalyticsCmd() tea.Cmd {
	return func() tea.Msg {
		data, err := db.LoadTagAnalytics(m.db, m.loc)
		return tagAnalyticsLoadedMsg{data: data, err: err}
	}
}

func (m Model) loadTemplatesCmd() tea.Cmd {
	return func() tea.Msg {
		// Initialize default templates if needed
		if err := db.InitializeDefaultTemplates(m.db); err != nil {
			return templatesLoadedMsg{templates: []Template{}, err: err}
		}

		// Load templates from database
		dbTemplates, err := db.GetAllTemplates(m.db)
		if err != nil {
			return templatesLoadedMsg{templates: []Template{}, err: err}
		}

		// Convert DB templates to UI templates
		templates := make([]Template, len(dbTemplates))
		for i, dbTemplate := range dbTemplates {
			variables, _ := db.ParseTemplateVariables(dbTemplate.Variables)
			lastUsed := time.Time{}
			if dbTemplate.LastUsed.Valid {
				lastUsed = dbTemplate.LastUsed.Time
			}
			templates[i] = Template{
				ID:          dbTemplate.ID,
				Name:        dbTemplate.Name,
				Category:    dbTemplate.Category,
				Content:     dbTemplate.Content,
				Description: dbTemplate.Description,
				Variables:   variables,
				IsCustom:    dbTemplate.IsCustom,
				UsageCount:  dbTemplate.UsageCount,
				LastUsed:    lastUsed,
				IsFavorite:  dbTemplate.IsFavorite,
			}
		}

		return templatesLoadedMsg{templates: templates, err: nil}
	}
}

// ---------- Update ----------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.now = msg.now.In(m.loc)
		return m, tickNow()
	case pomodoroTickMsg:
		if m.pomodoroActive {
			m.pomodoroTimeLeft -= time.Second
			if m.pomodoroTimeLeft <= 0 {
				// Session completed
				if m.pomodoroSession == 0 {
					// Work session completed
					m.pomodoroWorkSessions++
					m.pomodoroTotalTime += m.workSessionTime
					m.pomodoroSessionsCount++

					// Create log entry if auto-log is enabled
					if m.pomodoroAutoLog {
						go m.createPomodoroLogEntry("work")
					}

					// Determine break length
					breakLength := m.breakSessionTime
					if m.pomodoroLongBreakEnabled && m.pomodoroSessionsCount >= 4 {
						breakLength = 15 * time.Minute // Long break
						m.pomodoroSessionsCount = 0

						// Send long break notification
						title, msg := notify.FormatPomodoroLongBreak()
						if m.cfg.Notifications.Enabled && m.cfg.Notifications.PomodoroSessions {
							_ = notify.Info(title, msg)
						}
						m.addNotification("Work session completed! Time for a long break 🎉")
					} else {
						// Send regular work session completion notification
						title, msg := notify.FormatPomodoroWorkComplete(m.pomodoroWorkSessions, m.pomodoroWorkSessions)
						if m.cfg.Notifications.Enabled && m.cfg.Notifications.PomodoroSessions {
							_ = notify.Info(title, msg)
						}
						m.addNotification("Work session completed! Time for a break 🎉")
					}

					m.pomodoroSession = 1
					m.pomodoroTimeLeft = breakLength
				} else {
					// Break completed, start work session
					if m.pomodoroAutoLog {
						go m.createPomodoroLogEntry("break")
					}

					// Send break completion notification
					title, msg := notify.FormatPomodoroBreakComplete()
					if m.cfg.Notifications.Enabled && m.cfg.Notifications.PomodoroSessions {
						_ = notify.Info(title, msg)
					}

					m.pomodoroSession = 0
					m.pomodoroTimeLeft = m.workSessionTime
					m.addNotification("Break completed! Back to work 💪")
				}
			}
			return m, pomodoroTick()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case blocksLoadedMsg:
		if msg.err != nil {
			m.status = "load error: " + msg.err.Error()
			return m, nil
		}
		m.blocks = msg.blocks
		if len(m.blocks) == 0 {
			m.cursorBlock, m.cursorEntry = 0, 0
			m.threadBlock = block{}
			return m, nil
		}
		if m.cursorBlock >= len(m.blocks) {
			m.cursorBlock = len(m.blocks) - 1
		}
		if m.cursorEntry >= len(m.blocks[m.cursorBlock].entries) {
			m.cursorEntry = len(m.blocks[m.cursorBlock].entries) - 1
		}
		m.threadBlock = m.blocks[m.cursorBlock]
		return m, nil

	case facetsLoadedMsg:
		if msg.err == nil {
			m.projects = msg.projects
			m.categories = msg.cats
			m.tags = msg.tags
		}
		return m, nil
	case timeReportsLoadedMsg:
		if msg.err == nil {
			m.timeReportData = msg.data
		} else {
			m.status = "Failed to load time reports: " + msg.err.Error()
		}
		return m, nil
	case projectSummaryLoadedMsg:
		if msg.err == nil {
			m.projectSummaryData = msg.data
		} else {
			m.status = "Failed to load project summary: " + msg.err.Error()
		}
		return m, nil
	case tagAnalyticsLoadedMsg:
		if msg.err == nil {
			m.tagAnalyticsData = msg.data
		} else {
			m.status = "Failed to load tag analytics: " + msg.err.Error()
		}
		return m, nil
	case templatesLoadedMsg:
		if msg.err == nil {
			m.dbTemplates = msg.templates
			m.status = "Templates loaded from database"
		} else {
			m.status = "Failed to load templates: " + msg.err.Error()
		}
		return m, nil

	case AutocompleteMsg:
		// Handle autocomplete messages in create mode
		if m.mode == modeCreate {
			// Update the autocomplete suggestions
			// The specific autocomplete model will handle this message
			var cmd tea.Cmd
			switch m.createField {
			case 1: // Project field
				m.createProject, cmd = m.createProject.Update(msg)
			case 3: // Tags field
				m.createTags, cmd = m.createTags.Update(msg)
			}
			return m, cmd
		}
		return m, nil

	case tea.MouseMsg:
		return m.updateMouse(msg)
	case tea.KeyMsg:
		k := msg.String()

		// global quit from normal/help modes
		if (k == "q" || k == "ctrl+c") && (m.mode == modeNormal || m.mode == modeHelp) {
			return m, tea.Quit
		}

		switch m.mode {
		case modeNormal:
			return m.updateNormal(k)
		case modeSearch:
			var cmd tea.Cmd
			m, cmd = m.updateSearch(msg)
			return m, cmd
		case modeReply, modeEdit:
			var cmd tea.Cmd
			m, cmd = m.updateEditor(msg)
			return m, cmd
		case modeHelp:
			switch k {
			case "esc", "?":
				m.mode = modeNormal
				m.helpScrollOffset = 0 // Reset scroll when closing help
			case "up", "k":
				m.helpScrollOffset = max(0, m.helpScrollOffset-1)
			case "down", "j":
				m.helpScrollOffset++
			case "pgup":
				m.helpScrollOffset = max(0, m.helpScrollOffset-15) // Page up
			case "pgdown":
				m.helpScrollOffset += 15 // Page down
			case "home", "g":
				m.helpScrollOffset = 0
			case "end", "G":
				// Will be calculated based on content length in helpView
				m.helpScrollOffset = -1 // Signal to go to end
			}
			return m, nil
		case modeSince:
			var cmd tea.Cmd
			m, cmd = m.updateSince(msg)
			return m, cmd
		case modePicker:
			return m.updatePicker(k)
		case modeFocus:
			return m.updateFocus(k)
		case modeStats:
			if k == "esc" || k == "ctrl+i" {
				m.mode = modeNormal
			}
			return m, nil
		case modeCreate:
			var cmd tea.Cmd
			m, cmd = m.updateCreate(msg)
			return m, cmd
		case modeDashboard:
			if k == "esc" || k == "ctrl+w" {
				m.showDashboard = false
				m.mode = modeNormal
			}
			return m, nil
		case modeCalendar:
			return m.updateCalendar(k)
		case modeTemplates:
			return m.updateTemplates(k)
		case modeExport:
			return m.updateExport(k)
		case modeAdvancedSearch:
			var cmd tea.Cmd
			m, cmd = m.updateAdvancedSearch(msg)
			return m, cmd
		case modeTimeReports:
			return m.updateTimeReports(k)
		case modeProjectSummary:
			return m.updateProjectSummary(k)
		case modeTagAnalytics:
			return m.updateTagAnalytics(k)
		case modeCommandPalette:
			var cmd tea.Cmd
			m, cmd = m.updateCommandPalette(msg)
			return m, cmd
		case modeRichTextEditor:
			var cmd tea.Cmd
			m, cmd = m.updateRichTextEditor(msg)
			return m, cmd
		case modeTemplateEdit:
			var cmd tea.Cmd
			m, cmd = m.updateTemplateEdit(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseLeft:
		// Handle mouse clicks based on current mode
		switch m.mode {
		case modeReply, modeEdit:
			// Calculate modal position (centered)
			modalWidth := 70
			modalHeight := 20 // approximate
			modalX := (m.width - modalWidth) / 2
			modalY := (m.height - modalHeight) / 3

			// Button area is near bottom of modal
			buttonY := modalY + modalHeight - 6

			// Check if click is in button area
			if msg.Y == buttonY {
				// OK button area
				if msg.X >= modalX+m.okButtonRect[0] && msg.X < modalX+m.okButtonRect[0]+m.okButtonRect[2] {
					m.selectedButton = 0
					return m.updateEditor(tea.KeyMsg{Type: tea.KeyEnter})
				}
				// Cancel button area
				if msg.X >= modalX+m.cancelButtonRect[0] && msg.X < modalX+m.cancelButtonRect[0]+m.cancelButtonRect[2] {
					m.selectedButton = 1
					return m.updateEditor(tea.KeyMsg{Type: tea.KeyEnter})
				}
			}
		case modeNormal:
			// Handle click on quick actions area for scrolling
			if msg.Y >= m.height-2 { // Approximate quick actions area
				if msg.X < m.width/2 {
					// Click on left half - previous page
					m.quickActionsPage--
					if m.quickActionsPage < 0 {
						m.quickActionsPage = 0
					}
				} else {
					// Click on right half - next page
					m.quickActionsPage++
					maxPages := m.getMaxQuickActionsPages()
					if m.quickActionsPage >= maxPages {
						m.quickActionsPage = maxPages - 1
					}
				}
				return m, nil
			}
			// Handle other timeline clicks, sidebar clicks, etc.
			return m, nil
		}
	case tea.MouseWheelUp:
		if m.mode == modeNormal {
			if m.focus == focusTimeline && len(m.blocks) > 0 {
				// Check current view mode
				switch m.viewMode {
				case 0: // Timeline view
					m.timelineScrollOffset = max(0, m.timelineScrollOffset-1)
				case 1: // Cards view
					m.cardsScrollOffset = max(0, m.cardsScrollOffset-1)
				case 2: // Table view
					m.tableScrollOffset = max(0, m.tableScrollOffset-1)
				case 3: // Kanban view
					m.kanbanScrollOffset = max(0, m.kanbanScrollOffset-1)
				}
				return m, nil
			} else {
				return m.updateNormal("up")
			}
		} else if m.mode == modeHelp {
			m.helpScrollOffset = max(0, m.helpScrollOffset-1)
			return m, nil
		}
	case tea.MouseWheelDown:
		if m.mode == modeNormal {
			if m.focus == focusTimeline && len(m.blocks) > 0 {
				// Check current view mode and scroll accordingly
				switch m.viewMode {
				case 0: // Timeline view
					// Use dynamic height calculation that accounts for layout
					topHeight := lipgloss.Height(m.renderTopBar())
					miniHeight := lipgloss.Height(m.renderMiniSummary())
					quickHeight := lipgloss.Height(m.renderQuickActions())
					statusHeight := lipgloss.Height(m.statusBar())
					availableHeight := m.height - topHeight - miniHeight - quickHeight - statusHeight - 4 // 4 for title and borders
					availableHeight = max(6, availableHeight) // minimum 6 lines for timeline
					maxVisibleBlocks := max(1, availableHeight/4)
					maxScroll := max(0, len(m.blocks)-maxVisibleBlocks)
					m.timelineScrollOffset = min(maxScroll, m.timelineScrollOffset+1)
				case 1: // Cards view
					// Calculate max scroll for cards view with dynamic height
					topHeight := lipgloss.Height(m.renderTopBar())
					miniHeight := lipgloss.Height(m.renderMiniSummary())
					quickHeight := lipgloss.Height(m.renderQuickActions())
					statusHeight := lipgloss.Height(m.statusBar())
					availableHeight := m.height - topHeight - miniHeight - quickHeight - statusHeight - 4 // 4 for title and borders
					availableHeight = max(8, availableHeight) // minimum 8 lines for cards
					cardHeight := 8
					maxVisibleCards := max(1, availableHeight/cardHeight)
					var allEntries []entry
					for _, block := range m.blocks {
						allEntries = append(allEntries, block.entries...)
					}
					maxScroll := max(0, len(allEntries)-maxVisibleCards)
					m.cardsScrollOffset = min(maxScroll, m.cardsScrollOffset+1)
				case 2: // Table view
					// Calculate max scroll for table view with dynamic height
					topHeight := lipgloss.Height(m.renderTopBar())
					miniHeight := lipgloss.Height(m.renderMiniSummary())
					quickHeight := lipgloss.Height(m.renderQuickActions())
					statusHeight := lipgloss.Height(m.statusBar())
					availableHeight := m.height - topHeight - miniHeight - quickHeight - statusHeight - 7 // 7 for title, header, and borders
					availableHeight = max(10, availableHeight) // minimum 10 lines for table
					maxVisibleRows := max(1, availableHeight)
					var allEntries []entry
					for _, block := range m.blocks {
						allEntries = append(allEntries, block.entries...)
					}
					maxScroll := max(0, len(allEntries)-maxVisibleRows)
					m.tableScrollOffset = min(maxScroll, m.tableScrollOffset+1)
				case 3: // Kanban view
					// Calculate max scroll for kanban view (horizontal scrolling)
					// Group entries by category for kanban view
					categories := make(map[string][]entry)
					for _, block := range m.blocks {
						for _, entry := range block.entries {
							cat := entry.cat
							if cat == "" {
								cat = "Uncategorized"
							}
							categories[cat] = append(categories[cat], entry)
						}
					}
					var sortedCats []string
					for cat := range categories {
						sortedCats = append(sortedCats, cat)
					}
					sort.Strings(sortedCats)

					// Use dynamic height calculation for vertical space
					topHeight := lipgloss.Height(m.renderTopBar())
					miniHeight := lipgloss.Height(m.renderMiniSummary())
					quickHeight := lipgloss.Height(m.renderQuickActions())
					statusHeight := lipgloss.Height(m.statusBar())
					availableHeight := m.height - topHeight - miniHeight - quickHeight - statusHeight - 4 // 4 for title and borders
					availableHeight = max(8, availableHeight) // minimum 8 lines for kanban

					numColumns := min(len(sortedCats), 4) // Max 4 columns visible at once
					if numColumns == 0 {
						numColumns = 1
					}
					maxScroll := max(0, len(sortedCats)-numColumns)
					m.kanbanScrollOffset = min(maxScroll, m.kanbanScrollOffset+1)
				}
				return m, nil
			} else {
				return m.updateNormal("down")
			}
		} else if m.mode == modeHelp {
			m.helpScrollOffset++
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateNormal(k string) (tea.Model, tea.Cmd) {
	switch k {
	// focus switch
	case "tab":
		m.cycleFocus()
		return m, nil
	case "right", "l":
		// Handle kanban horizontal scrolling if in kanban view
		if m.viewMode == 3 && m.focus == focusTimeline {
			// Calculate max scroll for kanban view
			categories := make(map[string][]entry)
			for _, block := range m.blocks {
				for _, entry := range block.entries {
					cat := entry.cat
					if cat == "" {
						cat = "Uncategorized"
					}
					categories[cat] = append(categories[cat], entry)
				}
			}
			var sortedCats []string
			for cat := range categories {
				sortedCats = append(sortedCats, cat)
			}
			sort.Strings(sortedCats)
			maxVisibleColumns := min(len(sortedCats), 4)
			if maxVisibleColumns == 0 {
				maxVisibleColumns = 1
			}
			maxScroll := max(0, len(sortedCats)-maxVisibleColumns)
			m.kanbanScrollOffset = min(maxScroll, m.kanbanScrollOffset+1)

			if len(sortedCats) > maxVisibleColumns {
				startCat := max(0, m.kanbanScrollOffset)
				endCat := min(len(sortedCats), startCat+maxVisibleColumns)
				m.addNotification(fmt.Sprintf("Kanban: %s - %s", sortedCats[startCat], sortedCats[endCat-1]))
			}
			return m, nil
		}
		// Original right/l behavior
		if m.focus == focusSidebar && m.showSidebar {
			m.showSidebar = false
			m.focus = focusTimeline
		} else {
			m.showSidebar = true
			m.focus = focusSidebar
		}
		return m, nil
	case "left", "h":
		// Handle kanban horizontal scrolling if in kanban view
		if m.viewMode == 3 && m.focus == focusTimeline {
			// Calculate max scroll for kanban view
			categories := make(map[string][]entry)
			for _, block := range m.blocks {
				for _, entry := range block.entries {
					cat := entry.cat
					if cat == "" {
						cat = "Uncategorized"
					}
					categories[cat] = append(categories[cat], entry)
				}
			}
			var sortedCats []string
			for cat := range categories {
				sortedCats = append(sortedCats, cat)
			}
			sort.Strings(sortedCats)
			maxVisibleColumns := min(len(sortedCats), 4)
			if maxVisibleColumns == 0 {
				maxVisibleColumns = 1
			}
			m.kanbanScrollOffset = max(0, m.kanbanScrollOffset-1)

			if len(sortedCats) > maxVisibleColumns {
				startCat := max(0, m.kanbanScrollOffset)
				endCat := min(len(sortedCats), startCat+maxVisibleColumns)
				m.addNotification(fmt.Sprintf("Kanban: %s - %s", sortedCats[startCat], sortedCats[endCat-1]))
			}
			return m, nil
		}
		// Original left/h behavior
		if m.focus == focusThread && m.showThread {
			m.showThread = false
			m.focus = focusTimeline
		} else {
			m.showThread = true
			m.focus = focusThread
		}
		return m, nil

	// live filter
	case "/":
		m.mode = modeSearch
		return m, nil

	// since picker
	case "s":
		m.mode = modeSince
		m.sinceInput.SetValue("")
		m.sinceInput.Focus()
		return m, nil

	// scope toggle (comprehensive)
	case "t":
		// Cycle through scopes: today -> this week -> this month -> all
		switch m.scope {
		case scopeToday:
			m.scope = scopeThisWeek
			m.addNotification("Scope: This Week")
		case scopeThisWeek:
			m.scope = scopeThisMonth
			m.addNotification("Scope: This Month")
		case scopeThisMonth:
			m.scope = scopeAll
			m.addNotification("Scope: All Time")
		default:
			m.scope = scopeToday
			m.addNotification("Scope: Today")
		}
		return m, m.loadTimelineCmd()

	// date navigation shortcuts
	case "1":
		m.scope = scopeToday
		m.addNotification("Scope: Today")
		return m, m.loadTimelineCmd()
	case "2":
		m.scope = scopeYesterday
		m.addNotification("Scope: Yesterday")
		return m, m.loadTimelineCmd()
	case "3":
		m.scope = scopeThisWeek
		m.addNotification("Scope: This Week")
		return m, m.loadTimelineCmd()
	case "4":
		m.scope = scopeLastWeek
		m.addNotification("Scope: Last Week")
		return m, m.loadTimelineCmd()
	case "5":
		m.scope = scopeThisMonth
		m.addNotification("Scope: This Month")
		return m, m.loadTimelineCmd()
	case "6":
		m.scope = scopeLastMonth
		m.addNotification("Scope: Last Month")
		return m, m.loadTimelineCmd()
	case "0":
		m.scope = scopeAll
		m.addNotification("Scope: All Time")
		return m, m.loadTimelineCmd()

	// pickers
	case "p":
		m.mode = modePicker
		m.activePicker = pickProjects
		m.pickerCursor = 0
		return m, nil
	case "c":
		m.mode = modePicker
		m.activePicker = pickCategories
		m.pickerCursor = 0
		return m, nil
	case "#":
		m.mode = modePicker
		m.activePicker = pickTags
		m.pickerCursor = 0
		return m, nil

	// advanced features
	case "F":
		m.mode = modeAdvancedSearch
		m.advancedSearchField = 0
		m.advancedSearchQuery.SetValue("")
		m.advancedSearchQuery.Focus()
		m.addNotification("Advanced Search Mode")
		return m, nil
	case "T":
		m.mode = modeTemplates
		m.templateCursor = 0
		m.templateActive = true
		m.addNotification("Template Selection")
		return m, nil
	case "C":
		m.mode = modeCalendar
		m.calendarDate = m.now
		m.calendarSelectedDate = m.now
		m.calendarPreviewMode = false
		m.loadCalendarEntryCounts()
		m.addNotification("Calendar View")
		return m, nil
	case "E":
		m.mode = modeExport
		m.addNotification("Export Options")
		return m, nil

	// analytics views
	case "R":
		m.mode = modeTimeReports
		m.timeReportScope = scopeThisWeek
		return m, m.loadTimeReportsCmd()
	case "J":
		m.mode = modeProjectSummary
		return m, m.loadProjectSummaryCmd()
	case "A":
		m.mode = modeTagAnalytics
		return m, m.loadTagAnalyticsCmd()

	// view mode switching
	case "v":
		m.viewMode = (m.viewMode + 1) % 4
		viewNames := []string{"Timeline", "Cards", "Table", "Kanban"}
		m.addNotification(fmt.Sprintf("View: %s", viewNames[m.viewMode]))
		return m, nil

	// sorting options
	case "o":
		// Cycle sort by: date -> category -> project -> priority
		switch m.sortBy {
		case "date":
			m.sortBy = "category"
			m.addNotification("Sort: Category")
		case "category":
			m.sortBy = "project"
			m.addNotification("Sort: Project")
		case "project":
			m.sortBy = "priority"
			m.addNotification("Sort: Priority")
		default:
			m.sortBy = "date"
			m.addNotification("Sort: Date")
		}
		return m, m.loadTimelineCmd()
	case "O":
		m.sortDirection = !m.sortDirection
		if m.sortDirection {
			m.addNotification("Sort: Ascending")
		} else {
			m.addNotification("Sort: Descending")
		}
		return m, m.loadTimelineCmd()

	// productivity features
	case "P":
		m.pomodoroActive = !m.pomodoroActive
		if m.pomodoroActive {
			m.pomodoroSession = 0
			m.pomodoroTimeLeft = m.workSessionTime
			m.addNotification("Pomodoro Timer Started (25 min work)")
			return m, pomodoroTick()
		} else {
			m.addNotification("Pomodoro Timer Stopped")
		}
		return m, nil

	// entry management
	case "d":
		if len(m.blocks) > 0 {
			// Delete current entry (with confirmation would be better)
			entryID := m.blocks[m.cursorBlock].entries[m.cursorEntry].id
			_, err := m.db.Exec("DELETE FROM entries WHERE id = ?", entryID)
			if err != nil {
				m.status = "Failed to delete entry: " + err.Error()
			} else {
				m.status = fmt.Sprintf("Deleted entry #%d", entryID)
				return m, m.loadTimelineCmd()
			}
		}
		return m, nil
	case "D":
		if len(m.blocks) > 0 {
			// Duplicate current entry
			entry := m.blocks[m.cursorBlock].entries[m.cursorEntry]
			_, err := m.db.Exec(`
				INSERT INTO entries(category, text, project, tags)
				VALUES(?,?,?,?)
			`, entry.cat, entry.text+" (copy)", entry.project, strings.Join(entry.tags, ","))
			if err != nil {
				m.status = "Failed to duplicate entry: " + err.Error()
			} else {
				m.status = "Entry duplicated"
				return m, m.loadTimelineCmd()
			}
		}
		return m, nil

	// archive management
	case "a":
		m.archiveMode = !m.archiveMode
		if m.archiveMode {
			m.addNotification("Archive Mode: Showing archived entries")
		} else {
			m.addNotification("Archive Mode: Showing active entries")
		}
		return m, m.loadTimelineCmd()

	// help
	case "?":
		m.mode = modeHelp
		return m, nil

	// quick category creation
	case "alt+n":
		m.mode = modeCreate
		m.createField = 0
		m.createText.SetValue("")
		m.createProject.SetValue("")
		m.createCategory.SetValue("note")
		m.createTags.SetValue("")
		m.createText.Focus()
		return m, nil
	case "alt+t":
		m.mode = modeCreate
		m.createField = 0
		m.createText.SetValue("")
		m.createProject.SetValue("")
		m.createCategory.SetValue("task")
		m.createTags.SetValue("")
		m.createText.Focus()
		return m, nil
	case "alt+m":
		m.mode = modeCreate
		m.createField = 0
		m.createText.SetValue("")
		m.createProject.SetValue("")
		m.createCategory.SetValue("meeting")
		m.createTags.SetValue("")
		m.createText.Focus()
		return m, nil

	// nav
	case "up", "k":
		switch m.focus {
		case focusTimeline:
			if len(m.blocks) == 0 {
				return m, nil
			}
			if m.cursorEntry > 0 {
				m.cursorEntry--
			} else if m.cursorBlock > 0 {
				m.cursorBlock--
				m.cursorEntry = len(m.blocks[m.cursorBlock].entries) - 1
				m.threadBlock = m.blocks[m.cursorBlock]
			}

			// Auto-scroll to keep cursor visible in timeline view
			if m.viewMode == 0 { // Timeline view
				// Scroll up if cursor is above visible area
				if m.cursorBlock < m.timelineScrollOffset {
					m.timelineScrollOffset = max(0, m.cursorBlock)
				}
			} else if m.viewMode == 1 { // Cards view
				// Calculate flat index for current cursor position
				flatIndex := 0
				for bi := 0; bi < m.cursorBlock; bi++ {
					flatIndex += len(m.blocks[bi].entries)
				}
				flatIndex += m.cursorEntry

				// Scroll up if cursor is above visible area
				if flatIndex < m.cardsScrollOffset {
					m.cardsScrollOffset = max(0, flatIndex)
				}
			} else if m.viewMode == 2 { // Table view
				// Calculate flat index for current cursor position
				flatIndex := 0
				for bi := 0; bi < m.cursorBlock; bi++ {
					flatIndex += len(m.blocks[bi].entries)
				}
				flatIndex += m.cursorEntry

				// Scroll up if cursor is above visible area
				if flatIndex < m.tableScrollOffset {
					m.tableScrollOffset = max(0, flatIndex)
				}
			}
			return m, nil
		case focusSidebar:
			if m.sidebarCursor > 0 {
				m.sidebarCursor--
			} else if m.sidebarSection > 0 {
				// Jump to previous section
				m.sidebarSection--
				if m.sidebarSection == 2 {
					m.sidebarCursor = len(m.tags) - 1
				} else if m.sidebarSection == 1 {
					m.sidebarCursor = len(m.categories) - 1
				} else {
					m.sidebarCursor = len(m.projects) - 1
				}
			}
			return m, nil
		case focusThread:
			return m, nil
		}
	case "down", "j":
		switch m.focus {
		case focusTimeline:
			if len(m.blocks) == 0 {
				return m, nil
			}
			cur := &m.blocks[m.cursorBlock]
			if m.cursorEntry < len(cur.entries)-1 {
				m.cursorEntry++
			} else if m.cursorBlock < len(m.blocks)-1 {
				m.cursorBlock++
				m.cursorEntry = 0
				m.threadBlock = m.blocks[m.cursorBlock]
			}

			// Auto-scroll to keep cursor visible in timeline view
			if m.viewMode == 0 { // Timeline view
				// Use dynamic height calculation that accounts for layout
				topHeight := lipgloss.Height(m.renderTopBar())
				miniHeight := lipgloss.Height(m.renderMiniSummary())
				quickHeight := lipgloss.Height(m.renderQuickActions())
				statusHeight := lipgloss.Height(m.statusBar())
				availableHeight := m.height - topHeight - miniHeight - quickHeight - statusHeight - 4 // 4 for title and borders
				availableHeight = max(6, availableHeight) // minimum 6 lines for timeline
				maxVisibleBlocks := max(1, availableHeight/4)
				maxScroll := max(0, len(m.blocks)-maxVisibleBlocks)

				// Scroll down if cursor is below visible area
				if m.cursorBlock >= m.timelineScrollOffset+maxVisibleBlocks {
					m.timelineScrollOffset = min(maxScroll, m.cursorBlock-maxVisibleBlocks+1)
				}
			} else if m.viewMode == 1 { // Cards view
				// Use dynamic height calculation for cards view
				topHeight := lipgloss.Height(m.renderTopBar())
				miniHeight := lipgloss.Height(m.renderMiniSummary())
				quickHeight := lipgloss.Height(m.renderQuickActions())
				statusHeight := lipgloss.Height(m.statusBar())
				availableHeight := m.height - topHeight - miniHeight - quickHeight - statusHeight - 4 // 4 for title and borders
				availableHeight = max(8, availableHeight) // minimum 8 lines for cards
				cardHeight := 8
				maxVisibleCards := max(1, availableHeight/cardHeight)

				// Calculate flat index for current cursor position
				flatIndex := 0
				for bi := 0; bi < m.cursorBlock; bi++ {
					flatIndex += len(m.blocks[bi].entries)
				}
				flatIndex += m.cursorEntry

				// Calculate total entries and max scroll
				var totalEntries int
				for _, block := range m.blocks {
					totalEntries += len(block.entries)
				}
				maxScroll := max(0, totalEntries-maxVisibleCards)

				// Scroll down if cursor is below visible area
				if flatIndex >= m.cardsScrollOffset+maxVisibleCards {
					m.cardsScrollOffset = min(maxScroll, flatIndex-maxVisibleCards+1)
				}
			} else if m.viewMode == 2 { // Table view
				// Use dynamic height calculation for table view
				topHeight := lipgloss.Height(m.renderTopBar())
				miniHeight := lipgloss.Height(m.renderMiniSummary())
				quickHeight := lipgloss.Height(m.renderQuickActions())
				statusHeight := lipgloss.Height(m.statusBar())
				availableHeight := m.height - topHeight - miniHeight - quickHeight - statusHeight - 7 // 7 for title, header, and borders
				availableHeight = max(10, availableHeight) // minimum 10 lines for table
				maxVisibleRows := max(1, availableHeight)

				// Calculate flat index for current cursor position
				flatIndex := 0
				for bi := 0; bi < m.cursorBlock; bi++ {
					flatIndex += len(m.blocks[bi].entries)
				}
				flatIndex += m.cursorEntry

				// Calculate total entries and max scroll
				var totalEntries int
				for _, block := range m.blocks {
					totalEntries += len(block.entries)
				}
				maxScroll := max(0, totalEntries-maxVisibleRows)

				// Scroll down if cursor is below visible area
				if flatIndex >= m.tableScrollOffset+maxVisibleRows {
					m.tableScrollOffset = min(maxScroll, flatIndex-maxVisibleRows+1)
				}
			}
			return m, nil
		case focusSidebar:
			currentSectionLength := 0
			if m.sidebarSection == 0 {
				currentSectionLength = len(m.projects)
			} else if m.sidebarSection == 1 {
				currentSectionLength = len(m.categories)
			} else if m.sidebarSection == 2 {
				currentSectionLength = len(m.tags)
			}

			if m.sidebarCursor < currentSectionLength-1 {
				m.sidebarCursor++
			} else if m.sidebarSection < 2 {
				// Jump to next section
				m.sidebarSection++
				m.sidebarCursor = 0
			}
			return m, nil
		case focusThread:
			return m, nil
		}
	case " ":
		if m.focus == focusSidebar {
			// Select all functionality
			switch m.sidebarSection {
			case 0: // Projects
				if len(m.projects) > 0 {
					// Clear all projects or select all based on current state
					if m.filterProj != "" {
						m.filterProj = ""
						m.addNotification("Cleared all project filters")
					} else {
						// Select all projects by clearing filter (shows all)
						m.addNotification("Showing all projects")
					}
				}
			case 1: // Categories
				if len(m.categories) > 0 {
					if m.filterCat != "" {
						m.filterCat = ""
						m.addNotification("Cleared category filter")
					} else {
						m.addNotification("Showing all categories")
					}
				}
			case 2: // Tags
				if len(m.tags) > 0 {
					if len(m.filterTags) > 0 {
						m.filterTags = make(map[string]struct{})
						m.addNotification("Cleared all tag filters")
					} else {
						// Select all tags
						for _, tag := range m.tags {
							m.filterTags[tag.name] = struct{}{}
						}
						m.addNotification(fmt.Sprintf("Selected all %d tags", len(m.tags)))
					}
				}
			}
			return m, m.loadTimelineCmd()
		}

	// open thread
	case "enter":
		if len(m.blocks) > 0 {
			m.threadBlock = m.blocks[m.cursorBlock]
			m.showThread = true
			m.focus = focusThread
		}
		return m, nil

	// reply/edit
	case "r":
		if len(m.blocks) == 0 {
			return m, nil
		}
		parent := m.blocks[m.cursorBlock].entries[m.cursorEntry]
		m.replyParentID = parent.id
		m.editor.SetValue("")
		m.editProject.SetValue(parent.project)
		m.editTags.SetValue(strings.Join(parent.tags, ", "))
		m.editField = 0 // Start with text field
		m.editor.Focus()
		m.mode = modeReply
		return m, nil
	case "e":
		if len(m.blocks) == 0 {
			return m, nil
		}
		target := m.blocks[m.cursorBlock].entries[m.cursorEntry]
		m.editTargetID = target.id
		m.editor.SetValue(target.text)
		m.editProject.SetValue(target.project)
		m.editTags.SetValue(strings.Join(target.tags, ", "))
		m.editField = 0 // Start with text field
		m.editor.Focus()
		m.mode = modeEdit
		return m, nil

	// export
	case "x":
		if len(m.blocks) == 0 {
			return m, nil
		}
		cur := m.blocks[m.cursorBlock]
		path, err := exportThreadMarkdown(cur, m.loc)
		if err != nil {
			m.status = "export failed: " + err.Error()
		} else {
			m.status = "exported: " + path
		}
		return m, nil

	// enhanced shortcuts
	case "ctrl+b":
		m.showSidebar = !m.showSidebar
		if m.showSidebar {
			m.addNotification("Sidebar opened")
		} else {
			m.addNotification("Sidebar closed")
		}
		return m, nil
	case "ctrl+k":
		m.mode = modeCommandPalette
		m.commandPalette.SetValue("")
		m.commandPaletteInput = ""
		m.commandCursor = 0
		m.commandPalette.Focus()
		m.filteredCommands = make([]Command, len(m.commands))
		copy(m.filteredCommands, m.commands)
		return m, nil
	case "ctrl+f12":
		// Toggle accessibility mode
		m.accessibilityMode = !m.accessibilityMode
		if m.accessibilityMode {
			m.addNotification("Screen reader mode enabled")
			m.announceToScreenReader("Screen reader mode enabled")
		} else {
			m.addNotification("Screen reader mode disabled")
		}
		return m, nil
	case "ctrl+f11":
		// Toggle high contrast mode
		m.highContrast = !m.highContrast
		if m.highContrast {
			m.addNotification("High contrast mode enabled")
			m.announceToScreenReader("High contrast mode enabled")
		} else {
			m.addNotification("High contrast mode disabled")
		}
		return m, nil
	case "ctrl+f10":
		// Announce current context
		context := m.getCurrentContextForScreenReader()
		m.announceToScreenReader(context)
		return m, nil
	case "n":
		m.mode = modeCreate
		m.createField = 0
		m.createText.SetValue("")
		m.createProject.SetValue("")
		m.createCategory.SetValue("note")
		m.createTags.SetValue("")
		m.createText.Focus()
		return m, nil
	case "ctrl+f":
		m.focusMode = !m.focusMode
		if m.focusMode {
			m.mode = modeFocus
			m.showSidebar = false
			m.showThread = false
			m.addNotification("Focus mode enabled")
		} else {
			m.mode = modeNormal
			m.addNotification("Focus mode disabled")
		}
		return m, nil
	case "ctrl+t":
		m.theme = (m.theme + 1) % 3
		m.applyTheme(m.theme)
		m.addNotification(fmt.Sprintf("Theme changed to %d", m.theme+1))
		return m, nil
	case "ctrl+g":
		m.scope = scopeToday
		m.cursorBlock, m.cursorEntry = 0, 0
		m.addNotification("Jumped to today")
		return m, m.loadTimelineCmd()
	case "ctrl+d":
		if len(m.blocks) > 0 {
			entryID := m.blocks[m.cursorBlock].entries[m.cursorEntry].id
			if _, bookmarked := m.bookmarks[entryID]; bookmarked {
				delete(m.bookmarks, entryID)
				m.addNotification("Bookmark removed")
			} else {
				m.bookmarks[entryID] = struct{}{}
				m.addNotification("Entry bookmarked")
			}
		}
		return m, nil
	case "ctrl+w":
		m.showDashboard = !m.showDashboard
		if m.showDashboard {
			m.mode = modeDashboard
			m.addNotification("Dashboard opened")
		} else {
			m.mode = modeNormal
			m.addNotification("Dashboard closed")
		}
		return m, nil
	case "ctrl+i":
		if m.mode == modeStats {
			m.mode = modeNormal
		} else {
			m.mode = modeStats
		}
		return m, nil
	case "ctrl+r":
		m.mode = modeTimeReports
		m.timeReportScope = scopeThisWeek
		return m, m.loadTimeReportsCmd()
	case "ctrl+p":
		m.mode = modeProjectSummary
		return m, m.loadProjectSummaryCmd()
	case "ctrl+a":
		m.mode = modeTagAnalytics
		return m, m.loadTagAnalyticsCmd()

	// timeline scrolling (only when focused on timeline)
	case "pgup":
		if m.focus == focusTimeline && len(m.blocks) > 0 {
			// Scroll up by roughly one page (about 5 blocks)
			m.timelineScrollOffset = max(0, m.timelineScrollOffset-5)
			// Move cursor with scroll to keep it visible
			if m.cursorBlock >= m.timelineScrollOffset && m.cursorBlock < m.timelineScrollOffset+5 {
				// Cursor is already in visible range, don't move it
			} else {
				m.cursorBlock = min(m.cursorBlock, max(0, m.timelineScrollOffset+4))
				if m.cursorBlock < len(m.blocks) {
					m.threadBlock = m.blocks[m.cursorBlock]
				}
			}
		}
		return m, nil
	case "pgdown":
		if m.focus == focusTimeline && len(m.blocks) > 0 {
			// Scroll down by roughly one page using dynamic height calculation
			topHeight := lipgloss.Height(m.renderTopBar())
			miniHeight := lipgloss.Height(m.renderMiniSummary())
			quickHeight := lipgloss.Height(m.renderQuickActions())
			statusHeight := lipgloss.Height(m.statusBar())
			availableHeight := m.height - topHeight - miniHeight - quickHeight - statusHeight - 4 // 4 for title and borders
			availableHeight = max(6, availableHeight) // minimum 6 lines for timeline
			maxVisibleBlocks := max(1, availableHeight/4)
			pageSize := max(1, maxVisibleBlocks-1) // Scroll by almost a full page, leaving one item visible
			m.timelineScrollOffset = min(max(0, len(m.blocks)-maxVisibleBlocks), m.timelineScrollOffset+pageSize)
			// Move cursor with scroll to keep it visible
			if m.cursorBlock >= m.timelineScrollOffset && m.cursorBlock < m.timelineScrollOffset+maxVisibleBlocks {
				// Cursor is already in visible range, don't move it
			} else {
				m.cursorBlock = max(m.timelineScrollOffset, min(m.cursorBlock, m.timelineScrollOffset+maxVisibleBlocks-1))
				if m.cursorBlock < len(m.blocks) {
					m.threadBlock = m.blocks[m.cursorBlock]
				}
			}
		}
		return m, nil
	case "home":
		if m.focus == focusTimeline {
			m.timelineScrollOffset = 0
			m.cursorBlock = 0
			m.cursorEntry = 0
			if len(m.blocks) > 0 {
				m.threadBlock = m.blocks[0]
			}
		}
		return m, nil
	case "end":
		if m.focus == focusTimeline && len(m.blocks) > 0 {
			// Use dynamic height calculation for end navigation
			topHeight := lipgloss.Height(m.renderTopBar())
			miniHeight := lipgloss.Height(m.renderMiniSummary())
			quickHeight := lipgloss.Height(m.renderQuickActions())
			statusHeight := lipgloss.Height(m.statusBar())
			availableHeight := m.height - topHeight - miniHeight - quickHeight - statusHeight - 4 // 4 for title and borders
			availableHeight = max(6, availableHeight) // minimum 6 lines for timeline
			maxVisibleBlocks := max(1, availableHeight/4)
			m.timelineScrollOffset = max(0, len(m.blocks)-maxVisibleBlocks)
			m.cursorBlock = len(m.blocks) - 1
			m.cursorEntry = 0
			m.threadBlock = m.blocks[m.cursorBlock]
		}
		return m, nil

	// quick actions scrolling
	case "[":
		m.quickActionsPage--
		if m.quickActionsPage < 0 {
			m.quickActionsPage = 0
		}
		return m, nil
	case "]":
		m.quickActionsPage++
		maxPages := m.getMaxQuickActionsPages()
		if m.quickActionsPage >= maxPages {
			m.quickActionsPage = maxPages - 1
		}
		return m, nil
	case "ctrl+left":
		m.quickActionsPage--
		if m.quickActionsPage < 0 {
			m.quickActionsPage = 0
		}
		return m, nil
	case "ctrl+right":
		m.quickActionsPage++
		maxPages := m.getMaxQuickActionsPages()
		if m.quickActionsPage >= maxPages {
			m.quickActionsPage = maxPages - 1
		}
		return m, nil
	case "ctrl+[":
		m.quickActionsPage = 0 // Go to first page
		return m, nil
	case "ctrl+]":
		m.quickActionsPage = m.getMaxQuickActionsPages() - 1 // Go to last page
		return m, nil
	}
	return m, nil
}

// ----- quick actions helpers -----

func (m Model) getMaxQuickActionsPages() int {
	actions := m.getAllQuickActions()
	// Each page can show about 80 characters worth of actions
	maxCharsPerPage := 80
	totalChars := len(actions)
	if totalChars <= maxCharsPerPage {
		return 1
	}
	return (totalChars + maxCharsPerPage - 1) / maxCharsPerPage
}

func (m Model) getAllQuickActions() string {
	return "Quick: [n] new  [F] search  [T] templates  [C] calendar  [E] export  [r] reply  [e] edit  [d] delete  [D] duplicate  [/] filter  [t] scope  [v] view  [o] sort  [P] pomodoro  [Ctrl+W] dashboard  [Ctrl+I] stats  [Ctrl+R] time reports  [Ctrl+P] projects  [Ctrl+A] tags  [?] help"
}

func (m Model) getQuickActionsPage(page int) string {
	allActions := m.getAllQuickActions()
	maxCharsPerPage := 80

	if len(allActions) <= maxCharsPerPage {
		return allActions
	}

	// Split actions into chunks that fit within the character limit
	actions := strings.Fields(allActions)
	var pages []string
	var currentPage strings.Builder

	for _, action := range actions {
		testPage := currentPage.String()
		if testPage != "" {
			testPage += " "
		}
		testPage += action

		if len(testPage) <= maxCharsPerPage {
			currentPage.WriteString(action)
			currentPage.WriteString(" ")
		} else {
			if currentPage.Len() > 0 {
				pages = append(pages, strings.TrimSpace(currentPage.String()))
			}
			currentPage.Reset()
			currentPage.WriteString(action)
			currentPage.WriteString(" ")
		}
	}

	if currentPage.Len() > 0 {
		pages = append(pages, strings.TrimSpace(currentPage.String()))
	}

	if page >= 0 && page < len(pages) {
		pageContent := pages[page]
		if len(pages) > 1 {
			pageContent = fmt.Sprintf("%s [%d/%d]", pageContent, page+1, len(pages))
		}
		return pageContent
	}

	return pages[0] // fallback
}

// ----- search (live) -----

func (m Model) updateSearch(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
		m.filterText = ""
		return m, m.loadTimelineCmd()
	case tea.KeyEnter:
		m.mode = modeNormal
		return m, nil
	case tea.KeyBackspace:
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			return m, m.loadTimelineCmd()
		}
	default:
		// printable
		if ch := msg.String(); len(ch) == 1 {
			m.filterText += ch
			return m, m.loadTimelineCmd()
		}
	}
	return m, nil
}

// ----- since picker -----

func (m Model) updateSince(msg tea.Msg) (Model, tea.Cmd) {
	switch t := msg.(type) {
	case tea.KeyMsg:
		switch t.Type {
		case tea.KeyEsc:
			m.mode = modeNormal
			return m, nil
		case tea.KeyEnter:
			val := strings.TrimSpace(m.sinceInput.Value())
			ts, err := parseSince(val, m.loc)
			if err != nil {
				m.status = "invalid date: " + err.Error()
				return m, nil
			}
			m.scope = scopeSince
			m.sinceValue = ts
			m.mode = modeNormal
			return m, m.loadTimelineCmd()
		}
	}
	var cmd tea.Cmd
	m.sinceInput, cmd = m.sinceInput.Update(msg)
	return m, cmd
}

// ----- picker -----

func (m Model) updatePicker(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "up", "k":
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
		return m, nil
	case "down", "j":
		m.pickerCursor++
		return m, nil
	case "enter":
		switch m.activePicker {
		case pickProjects:
			if len(m.projects) == 0 {
				return m, nil
			}
			i := clamp(m.pickerCursor, 0, len(m.projects)-1)
			if m.filterProj == m.projects[i].name {
				m.filterProj = "" // toggle off
			} else {
				m.filterProj = m.projects[i].name
			}
		case pickCategories:
			if len(m.categories) == 0 {
				return m, nil
			}
			i := clamp(m.pickerCursor, 0, len(m.categories)-1)
			if m.filterCat == strings.ToLower(m.categories[i].name) {
				m.filterCat = ""
			} else {
				m.filterCat = strings.ToLower(m.categories[i].name)
			}
		case pickTags:
			if len(m.tags) == 0 {
				return m, nil
			}
			i := clamp(m.pickerCursor, 0, len(m.tags)-1)
			name := m.tags[i].name
			if _, ok := m.filterTags[name]; ok {
				delete(m.filterTags, name)
			} else {
				m.filterTags[name] = struct{}{}
			}
		}
		m.mode = modeNormal
		return m, tea.Batch(m.loadTimelineCmd(), m.loadFacetsCmd())
	}
	return m, nil
}

// ----- focus mode -----

func (m Model) updateFocus(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "esc":
		m.focusMode = false
		m.mode = modeNormal
		return m, nil
	case "ctrl+f":
		m.focusMode = false
		m.mode = modeNormal
		return m, nil
	}
	return m, nil
}

// ----- create entry form -----

func (m Model) updateCreate(msg tea.Msg) (Model, tea.Cmd) {
	// Handle autocomplete messages
	if acMsg, ok := msg.(AutocompleteMsg); ok {
		// Update the appropriate autocomplete model based on which field is focused
		switch m.createField {
		case 1: // Project field
			m.createProject, _ = m.createProject.Update(acMsg)
		case 3: // Tags field
			m.createTags, _ = m.createTags.Update(acMsg)
		}
		return m, nil
	}

	if km, ok := msg.(tea.KeyMsg); ok {
		k := km.String()
		switch k {
		case "esc":
			m.mode = modeNormal
			m.selectedButton = 0
			return m, nil
		case "tab":
			// Cycle through fields
			m.createField = (m.createField + 1) % 4
			switch m.createField {
			case 0:
				m.createText.Focus()
				m.createProject.Blur()
				m.createCategory.Blur()
				m.createTags.Blur()
			case 1:
				m.createText.Blur()
				m.createProject.Focus()
				m.createCategory.Blur()
				m.createTags.Blur()
			case 2:
				m.createText.Blur()
				m.createProject.Blur()
				m.createCategory.Focus()
				m.createTags.Blur()
			case 3:
				m.createText.Blur()
				m.createProject.Blur()
				m.createCategory.Blur()
				m.createTags.Focus()
			}
			return m, nil
		case "enter":
			// Save the entry
			text := strings.TrimSpace(m.createText.Value())
			if text == "" {
				m.status = "Text cannot be empty"
				return m, nil
			}

			project := strings.TrimSpace(m.createProject.Value())
			category := strings.ToLower(strings.TrimSpace(m.createCategory.Value()))
			if category == "" {
				category = "note"
			}
			tags := strings.TrimSpace(m.createTags.Value())

			// Insert into database
			res, err := m.db.Exec(`
				INSERT INTO entries(category, text, project, tags)
				VALUES(?,?,?,?)
			`, category, text, nullIfEmpty(project), nullIfEmpty(tags))
			if err != nil {
				m.status = "Failed to create entry: " + err.Error()
				return m, nil
			}

			id, _ := res.LastInsertId()
			m.status = fmt.Sprintf("Created entry #%d", id)
			m.mode = modeNormal
			m.selectedButton = 0
			return m, m.loadTimelineCmd()
		}
	}

	// Update the currently focused field
	var cmd tea.Cmd
	switch m.createField {
	case 0:
		m.createText, cmd = m.createText.Update(msg)
	case 1:
		m.createProject, cmd = m.createProject.Update(msg)
	case 2:
		m.createCategory, cmd = m.createCategory.Update(msg)
	case 3:
		m.createTags, cmd = m.createTags.Update(msg)
	}
	return m, cmd
}

// ----- advanced search -----

func (m Model) updateAdvancedSearch(msg tea.Msg) (Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		k := km.String()
		switch k {
		case "esc":
			m.mode = modeNormal
			return m, nil
		case "tab":
			// Cycle through search fields
			m.advancedSearchField = (m.advancedSearchField + 1) % 4
			switch m.advancedSearchField {
			case 0:
				m.advancedSearchQuery.Focus()
				m.advancedSearchProject.Blur()
				m.advancedSearchCategory.Blur()
				m.advancedSearchTags.Blur()
			case 1:
				m.advancedSearchQuery.Blur()
				m.advancedSearchProject.Focus()
				m.advancedSearchCategory.Blur()
				m.advancedSearchTags.Blur()
			case 2:
				m.advancedSearchQuery.Blur()
				m.advancedSearchProject.Blur()
				m.advancedSearchCategory.Focus()
				m.advancedSearchTags.Blur()
			case 3:
				m.advancedSearchQuery.Blur()
				m.advancedSearchProject.Blur()
				m.advancedSearchCategory.Blur()
				m.advancedSearchTags.Focus()
			}
			return m, nil
		case "enter":
			// Perform search
			return m.performAdvancedSearch()
		}
	}

	// Update the currently focused field
	var cmd tea.Cmd
	switch m.advancedSearchField {
	case 0:
		m.advancedSearchQuery, cmd = m.advancedSearchQuery.Update(msg)
	case 1:
		m.advancedSearchProject, cmd = m.advancedSearchProject.Update(msg)
	case 2:
		m.advancedSearchCategory, cmd = m.advancedSearchCategory.Update(msg)
	case 3:
		m.advancedSearchTags, cmd = m.advancedSearchTags.Update(msg)
	}
	return m, cmd
}

func (m Model) performAdvancedSearch() (Model, tea.Cmd) {
	query := strings.TrimSpace(m.advancedSearchQuery.Value())
	project := strings.TrimSpace(m.advancedSearchProject.Value())
	category := strings.TrimSpace(m.advancedSearchCategory.Value())
	tags := strings.TrimSpace(m.advancedSearchTags.Value())

	if query == "" && project == "" && category == "" && tags == "" {
		m.status = "Please enter at least one search criterion"
		return m, nil
	}

	// Build the search query
	conditions := []string{}
	args := []any{}

	if query != "" {
		conditions = append(conditions, "(instr(text, ?) > 0 OR instr(project, ?) > 0 OR instr(tags, ?) > 0)")
		args = append(args, query, query, query)
	}
	if project != "" {
		conditions = append(conditions, "project = ?")
		args = append(args, project)
	}
	if category != "" {
		conditions = append(conditions, "lower(category) = lower(?)")
		args = append(args, category)
	}
	if tags != "" {
		tagList := strings.Split(tags, ",")
		for _, tag := range tagList {
			if strings.TrimSpace(tag) != "" {
				conditions = append(conditions, "instr(tags, ?) > 0")
				args = append(args, strings.TrimSpace(tag))
			}
		}
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Execute search
	rows, err := m.db.Query(`
		SELECT id, ts, category, COALESCE(project,''), COALESCE(tags,''), COALESCE(text,'')
		FROM entries `+whereClause+`
		ORDER BY ts DESC, id DESC
		LIMIT 50
	`, args...)

	if err != nil {
		m.status = "Search failed: " + err.Error()
		return m, nil
	}
	defer rows.Close()

	var results []entry
	for rows.Next() {
		var e entry
		var tsStr, projS, tagsS, text string
		if err := rows.Scan(&e.id, &tsStr, &e.cat, &projS, &tagsS, &text); err != nil {
			continue
		}
		e.when = parseAny(tsStr).In(m.loc)
		e.project = projS
		e.tags = splitTags(tagsS)
		e.text = strings.TrimSpace(text)
		results = append(results, e)
	}

	m.advancedSearchResults = results
	m.status = fmt.Sprintf("Found %d results", len(results))
	return m, nil
}

// ----- calendar view -----

func (m Model) updateCalendar(k string) (Model, tea.Cmd) {
	switch k {
	case "esc":
		if m.calendarPreviewMode {
			m.calendarPreviewMode = false
			m.addNotification("Calendar View")
		} else {
			m.mode = modeNormal
		}
		return m, nil
	case "left", "h":
		switch m.calendarView {
		case 0: // month
			m.calendarDate = m.calendarDate.AddDate(0, -1, 0)
		case 1: // week
			m.calendarDate = m.calendarDate.AddDate(0, 0, -7)
		case 2: // day
			m.calendarDate = m.calendarDate.AddDate(0, 0, -1)
		}
		m.loadCalendarEntryCounts()
		return m, nil
	case "right", "l":
		switch m.calendarView {
		case 0: // month
			m.calendarDate = m.calendarDate.AddDate(0, 1, 0)
		case 1: // week
			m.calendarDate = m.calendarDate.AddDate(0, 0, 7)
		case 2: // day
			m.calendarDate = m.calendarDate.AddDate(0, 0, 1)
		}
		m.loadCalendarEntryCounts()
		return m, nil
	case "v":
		m.calendarView = (m.calendarView + 1) % 3
		viewNames := []string{"Month", "Week", "Day"}
		m.addNotification(fmt.Sprintf("Calendar View: %s", viewNames[m.calendarView]))
		m.loadCalendarEntryCounts()
		return m, nil
	case "t":
		m.calendarDate = m.now
		m.loadCalendarEntryCounts()
		return m, nil
	case "enter":
		if !m.calendarPreviewMode {
			m.calendarPreviewMode = true
			m.addNotification(fmt.Sprintf("Entries for %s", m.calendarSelectedDate.Format("2006-01-02")))
		}
		return m, nil
	case "up", "k", "down", "j":
		if !m.calendarPreviewMode {
			// Navigate dates within current view
			switch k {
			case "up", "k":
				if m.calendarView == 0 { // month
					m.calendarSelectedDate = m.calendarSelectedDate.AddDate(0, 0, -7)
				} else if m.calendarView == 1 { // week
					m.calendarSelectedDate = m.calendarSelectedDate.AddDate(0, 0, -1)
				}
			case "down", "j":
				if m.calendarView == 0 { // month
					m.calendarSelectedDate = m.calendarSelectedDate.AddDate(0, 0, 7)
				} else if m.calendarView == 1 { // week
					m.calendarSelectedDate = m.calendarSelectedDate.AddDate(0, 0, 1)
				}
			}
		}
		return m, nil
	case "n":
		// Create new entry for selected date
		m.mode = modeCreate
		m.createField = 0
		m.createText.SetValue("")
		m.createProject.SetValue("")
		m.createCategory.SetValue("note")
		m.createTags.SetValue("")
		m.createText.Focus()
		m.addNotification(fmt.Sprintf("Creating entry for %s", m.calendarSelectedDate.Format("2006-01-02")))
		return m, nil
	}
	return m, nil
}

func (m Model) loadCalendarEntryCounts() {
	var startDate, endDate time.Time

	switch m.calendarView {
	case 0: // month view - load entire month
		year, month, _ := m.calendarDate.Date()
		startDate = time.Date(year, month, 1, 0, 0, 0, 0, m.loc).UTC()
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
	case 1: // week view - load current week
		weekday := int(m.calendarDate.Weekday())
		startDate = m.calendarDate.AddDate(0, 0, -weekday).UTC()
		endDate = startDate.AddDate(0, 0, 7).Add(-time.Second)
	case 2: // day view - load current day
		startDate = time.Date(m.calendarDate.Year(), m.calendarDate.Month(), m.calendarDate.Day(), 0, 0, 0, 0, m.loc).UTC()
		endDate = startDate.AddDate(0, 0, 1).Add(-time.Second)
	}

	counts, err := db.GetEntryCountsByDate(m.db, startDate, endDate)
	if err == nil {
		m.calendarEntryCounts = counts
	}
}

// ----- templates -----

func (m Model) updateTemplates(k string) (Model, tea.Cmd) {
	switch k {
	case "esc":
		m.mode = modeNormal
		m.templateFilterMode = false
		m.templateSearchQuery = ""
		return m, nil
	case "tab":
		// Toggle between category and template selection
		m.templateFilterMode = !m.templateFilterMode
		if m.templateFilterMode {
			m.templateCategoryCursor = 0
		} else {
			m.templateCursor = 0
		}
		return m, nil
	case "/":
		// Toggle search mode
		m.templateFilterMode = !m.templateFilterMode
		if m.templateFilterMode {
			m.templateSearchQuery = ""
		}
		return m, nil
	case "1", "2", "3", "4", "5", "6", "7", "8", "9", "0":
		// Quick category selection
		catIndex := 0
		if k == "0" {
			catIndex = 9
		} else {
			catIndex = int(k[0] - '1')
		}
		if catIndex < len(m.templateCategories) {
			m.templateCategoryCursor = catIndex
			m.templateCursor = 0
			m.templateFilterMode = false
			m.addNotification(fmt.Sprintf("Selected: %s %s", m.templateCategories[catIndex].Icon, m.templateCategories[catIndex].Name))
		}
		return m, nil
	case "left", "h":
		if !m.templateFilterMode {
			if m.templateCategoryCursor > 0 {
				m.templateCategoryCursor--
				m.templateCursor = 0
			}
		}
		return m, nil
	case "right", "l":
		if !m.templateFilterMode {
			if m.templateCategoryCursor < len(m.templateCategories)-1 {
				m.templateCategoryCursor++
				m.templateCursor = 0
			}
		}
		return m, nil
	case "up", "k":
		if m.templateFilterMode {
			// Search mode - navigate categories
			if m.templateCategoryCursor > 0 {
				m.templateCategoryCursor--
			}
		} else {
			// Navigate templates in current category
			if m.templateCursor > 0 {
				m.templateCursor--
			} else if m.templateCategoryCursor > 0 {
				// Move to previous category and select last template
				m.templateCategoryCursor--
				newCategory := m.getCurrentCategoryTemplates()
				m.templateCursor = len(newCategory) - 1
			}
		}
		return m, nil
	case "down", "j":
		if m.templateFilterMode {
			// Search mode - navigate categories
			if m.templateCategoryCursor < len(m.templateCategories)-1 {
				m.templateCategoryCursor++
			}
		} else {
			// Navigate templates in current category
			currentCategory := m.getCurrentCategoryTemplates()
			if m.templateCursor < len(currentCategory)-1 {
				m.templateCursor++
			} else if m.templateCategoryCursor < len(m.templateCategories)-1 {
				// Move to next category and select first template
				m.templateCategoryCursor++
				m.templateCursor = 0
			}
		}
		return m, nil
	case "enter":
		// Use selected template
		if m.templateFilterMode {
			// In category selection mode, switch to template selection
			m.templateFilterMode = false
			m.templateCursor = 0
		} else {
			// Use the selected template
			currentTemplates := m.getCurrentCategoryTemplates()
			if m.templateCursor < len(currentTemplates) {
				selectedTemplate := currentTemplates[m.templateCursor]

				// Process template variables
				content := m.processTemplateVariables(selectedTemplate.Content)

				// Create new entry with template
				m.mode = modeCreate
				m.createField = 0
				m.createText.SetValue(content)
				m.createProject.SetValue("")
				m.createCategory.SetValue(strings.ToLower(selectedTemplate.Category))
				m.createTags.SetValue("")
				m.createText.Focus()

				// Update usage stats
				for i, template := range m.templates {
					if template.ID == selectedTemplate.ID {
						m.templates[i].UsageCount++
						m.templates[i].LastUsed = time.Now()
						break
					}
				}

				m.addNotification(fmt.Sprintf("Created entry from template: %s", selectedTemplate.Name))
			}
		}
		return m, nil
	}
	return m, nil
}

// Helper function to get templates in current category
func (m Model) getCurrentCategoryTemplates() []Template {
	if m.templateCategoryCursor >= len(m.templateCategories) {
		return []Template{}
	}

	category := m.templateCategories[m.templateCategoryCursor]
	var categoryTemplates []Template

	for _, template := range m.templates {
		if template.Category == category.Name {
			categoryTemplates = append(categoryTemplates, template)
		}
	}

	return categoryTemplates
}

// Helper function to process template variables
func (m Model) processTemplateVariables(content string) string {
	now := time.Now()

	// Replace common variables
	replacements := map[string]string{
		"{{date}}":          now.Format("2006-01-02"),
		"{{time}}":          now.Format("15:04"),
		"{{datetime}}":      now.Format("2006-01-02 15:04"),
		"{{week_date}}":     fmt.Sprintf("%s-%s",
			now.Format("2006-01-02"),
			now.AddDate(0, 0, 7).Format("2006-01-02")),
		"{{next_week_date}}": now.AddDate(0, 0, 7).Format("2006-01-02"),
		"{{deadline}}":      now.AddDate(0, 1, 0).Format("2006-01-02"), // Default: 1 month
		"{{period}}":        fmt.Sprintf("%s %d", now.Month().String(), now.Year()),
		"{{timeframe}}":     "Q" + fmt.Sprintf("%d", (now.Month()-1)/3+1),
	}

	result := content
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// ----- export -----

func (m Model) updateExport(k string) (Model, tea.Cmd) {
	switch k {
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "1":
		m.exportFormat = "markdown"
		m.addNotification("Export format: Markdown")
		return m, nil
	case "2":
		m.exportFormat = "json"
		m.addNotification("Export format: JSON")
		return m, nil
	case "3":
		m.exportFormat = "csv"
		m.addNotification("Export format: CSV")
		return m, nil
	case "e":
		return m.performExport()
	}
	return m, nil
}

func (m Model) performExport() (Model, tea.Cmd) {
	home, _ := os.UserHomeDir()
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("pulse-export-%s.%s", timestamp, m.exportFormat)
	path := filepath.Join(home, ".config", "pulse", "exports", filename)

	// Ensure export directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.status = "Failed to create export directory: " + err.Error()
		return m, nil
	}

	// Collect all entries for export
	var allEntries []entry
	for _, b := range m.blocks {
		allEntries = append(allEntries, b.entries...)
	}

	var err error
	switch m.exportFormat {
	case "markdown":
		err = m.exportMarkdown(allEntries, path)
	case "json":
		err = m.exportJSON(allEntries, path)
	case "csv":
		err = m.exportCSV(allEntries, path)
	}

	if err != nil {
		m.status = "Export failed: " + err.Error()
	} else {
		m.status = "Exported to: " + path
		m.mode = modeNormal
	}
	return m, nil
}

// ----- editor (reply/edit) -----

func (m Model) updateEditor(msg tea.Msg) (Model, tea.Cmd) {
	// Handle autocomplete messages
	if acMsg, ok := msg.(AutocompleteMsg); ok {
		// Update the appropriate autocomplete model based on which field is focused
		switch m.editField {
		case 1: // Project field
			m.editProject, _ = m.editProject.Update(acMsg)
		case 2: // Tags field
			m.editTags, _ = m.editTags.Update(acMsg)
		}
		return m, nil
	}

	// handle save/cancel
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			m.mode = modeNormal
			m.selectedButton = 0 // reset button selection
			return m, nil
		case "ctrl+enter":
			text := strings.TrimSpace(m.editor.Value())
			if text == "" {
				m.status = "nothing to save"
				m.mode = modeNormal
				m.selectedButton = 0 // reset button selection
				return m, nil
			}
			if m.mode == modeReply {
				project := strings.TrimSpace(m.editProject.Value())
				tags := strings.TrimSpace(m.editTags.Value())
				if err := insertReplyWithProjectTags(m.db, m.replyParentID, text, project, tags); err != nil {
					m.status = "reply failed: " + err.Error()
				} else {
					m.status = "replied"
				}
			} else if m.mode == modeEdit {
				project := strings.TrimSpace(m.editProject.Value())
				tags := strings.TrimSpace(m.editTags.Value())
				if err := updateEntryTextProjectTags(m.db, m.editTargetID, text, project, tags); err != nil {
					m.status = "edit failed: " + err.Error()
				} else {
					m.status = "updated"
				}
			}
			m.mode = modeNormal
			m.selectedButton = 0 // reset button selection
			return m, m.loadTimelineCmd()
		case "tab":
			// Cycle through fields: text -> project -> tags -> buttons
			m.editField = (m.editField + 1) % 4
			switch m.editField {
			case 0: // Text field
				m.editor.Focus()
				m.editProject.Blur()
				m.editTags.Blur()
			case 1: // Project field
				m.editor.Blur()
				m.editProject.Focus()
				m.editTags.Blur()
			case 2: // Tags field
				m.editor.Blur()
				m.editProject.Blur()
				m.editTags.Focus()
			case 3: // Buttons
				m.editor.Blur()
				m.editProject.Blur()
				m.editTags.Blur()
				m.selectedButton = 0
			}
			return m, nil
		case "enter":
			// Handle button selection with Enter key
			if m.selectedButton == 0 {
				// OK button - same as Ctrl+Enter
				text := strings.TrimSpace(m.editor.Value())
				if text == "" {
					m.status = "nothing to save"
					m.mode = modeNormal
					m.selectedButton = 0
					return m, nil
				}
				if m.mode == modeReply {
					project := strings.TrimSpace(m.editProject.Value())
					tags := strings.TrimSpace(m.editTags.Value())
					if err := insertReplyWithProjectTags(m.db, m.replyParentID, text, project, tags); err != nil {
						m.status = "reply failed: " + err.Error()
					} else {
						m.status = "replied"
					}
				} else if m.mode == modeEdit {
					project := strings.TrimSpace(m.editProject.Value())
					tags := strings.TrimSpace(m.editTags.Value())
					if err := updateEntryTextProjectTags(m.db, m.editTargetID, text, project, tags); err != nil {
						m.status = "edit failed: " + err.Error()
					} else {
						m.status = "updated"
					}
				}
				m.mode = modeNormal
				m.selectedButton = 0
				return m, m.loadTimelineCmd()
			} else {
				// Cancel button - same as Esc
				m.mode = modeNormal
				m.selectedButton = 0
				return m, nil
			}
		}
	}

	// Update the currently focused field
	var cmd tea.Cmd
	switch m.editField {
	case 0: // Text field
		m.editor, cmd = m.editor.Update(msg)
	case 1: // Project field
		m.editProject, cmd = m.editProject.Update(msg)
	case 2: // Tags field
		m.editTags, cmd = m.editTags.Update(msg)
	case 3: // Buttons - no update needed
		// Buttons are handled by the key switch above
	}
	return m, cmd
}

// ---------- View ----------

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	top := m.renderTopBar()
	mini := m.renderMiniSummary()
	quick := m.renderQuickActions()
	status := m.statusBar()

	innerH := m.height - lipgloss.Height(top) - lipgloss.Height(mini) - lipgloss.Height(quick) - lipgloss.Height(status)
	if innerH < 10 {
		innerH = 10
	}

	var ui string

	// Focus mode: simplified UI
	if m.focusMode {
		timelineW := m.width
		quick = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Render("🎯 FOCUS MODE • Ctrl+F to exit • Ctrl+T for themes • Ctrl+G for today • Ctrl+D to bookmark")
		timeline := m.renderTimeline(timelineW, innerH)
		ui = lipgloss.JoinVertical(lipgloss.Left, top, timeline, status)
	} else {
		sidebarW := 0
		threadW := 0
		if m.showSidebar {
			sidebarW = max(24, m.width/5)
		}
		if m.showThread {
			threadW = max(36, m.width/3)
		}
		timelineW := m.width - sidebarW - threadW
		if timelineW < 38 {
			def := 38 - timelineW
			if threadW > 0 {
				threadW = max(24, threadW-def/2)
			}
			if sidebarW > 0 {
				sidebarW = max(18, sidebarW-def/2)
			}
			timelineW = m.width - sidebarW - threadW
		}

		var panes []string
		if m.showSidebar {
			panes = append(panes, m.renderSidebar(sidebarW, innerH))
		}
		panes = append(panes, m.renderTimeline(timelineW, innerH))
		if m.showThread {
			panes = append(panes, m.renderThread(threadW, innerH))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, panes...)

		ui = lipgloss.JoinVertical(lipgloss.Left, top, row, mini, quick, status)
	}

	// overlays
	switch m.mode {
	case modeSearch:
		box := m.modal("Filter", lipgloss.NewStyle().Width(60).Render("Type to filter…  Enter to keep, Esc to clear\n\n> "+m.filterText))
		ui = overlayCenter(ui, box)
	case modeHelp:
		ui = overlayCenter(ui, m.helpView())
	case modeSince:
		content := "Set scope start (local time)\n\n" + m.sinceInput.View() + "\n\nExamples: today, yesterday, 7d, 30d, 2025-09-29"
		ui = overlayCenter(ui, m.modal("Since…", content))
	case modePicker:
		ui = overlayCenter(ui, m.renderPickerModal())
	case modeReply:
		ui = overlayCenter(ui, m.renderReplyModal())
	case modeEdit:
		ui = overlayCenter(ui, m.renderEditModal())
	case modeCreate:
		ui = overlayCenter(ui, m.renderCreateModal())
	case modeStats:
		ui = overlayCenter(ui, m.renderStatsView())
	case modeDashboard:
		ui = overlayCenter(ui, m.renderDashboardView())
	case modeCalendar:
		ui = overlayCenter(ui, m.renderCalendarView())
	case modeFocus:
		ui = overlayCenter(ui, m.renderFocusView())
	case modeTemplates:
		ui = overlayCenter(ui, m.renderTemplatesView())
	case modeExport:
		ui = overlayCenter(ui, m.renderExportView())
	case modeAdvancedSearch:
		ui = overlayCenter(ui, m.renderAdvancedSearchView())
	case modeTimeReports:
		ui = overlayCenter(ui, m.renderTimeReportsView())
	case modeProjectSummary:
		ui = overlayCenter(ui, m.renderProjectSummaryView())
	case modeTagAnalytics:
		ui = overlayCenter(ui, m.renderTagAnalyticsView())
	case modeCommandPalette:
		ui = overlayCenter(ui, m.renderCommandPaletteView())
	case modeRichTextEditor:
		ui = overlayCenter(ui, m.renderRichTextEditorView())
	case modeTemplateEdit:
		ui = overlayCenter(ui, m.renderTemplateEditView())
	}
	return ui
}

func (m Model) renderTopBar() string {
	scopeText := "Today"
	if m.scope == scopeAll {
		scopeText = "All time"
	} else if m.scope == scopeSince {
		scopeText = "Since " + m.sinceValue.In(m.loc).Format("Jan 02 03:04 PM")
	}
	var filters []string
	if strings.TrimSpace(m.filterText) != "" {
		filters = append(filters, "q="+m.filterText)
	}
	if m.filterProj != "" {
		filters = append(filters, "p="+m.filterProj)
	}
	if m.filterCat != "" {
		filters = append(filters, "c="+m.filterCat)
	}
	if len(m.filterTags) > 0 {
		var tags []string
		for t := range m.filterTags {
			tags = append(tags, t)
		}
		sort.Strings(tags)
		filters = append(filters, "tags="+strings.Join(tags, ","))
	}
	filterText := ""
	if len(filters) > 0 {
		filterText = "  |  " + strings.Join(filters, " · ")
	}

	right := m.now.In(m.loc).Format("Jan 02 03:04 PM")

	// Add Pomodoro timer if active
	pomodoroText := ""
	if m.pomodoroActive {
		minutes := int(m.pomodoroTimeLeft.Minutes())
		seconds := int(m.pomodoroTimeLeft.Seconds()) % 60
		sessionType := "WORK"
		if m.pomodoroSession == 1 {
			sessionType = "BREAK"
		}
		pomodoroText = fmt.Sprintf(" | 🍅 %s %02d:%02d", sessionType, minutes, seconds)
	}

	viewModeText := ""
	if m.viewMode != 0 {
		viewNames := []string{"Timeline", "Cards", "Table", "Kanban"}
		viewModeText = fmt.Sprintf(" | %s", viewNames[m.viewMode])
	}

	title := fmt.Sprintf("Pulse • %s%s%s%s  |  %s", scopeText, filterText, pomodoroText, viewModeText, right)
	return m.st.topBar.Render(title)
}

func (m Model) renderQuickActions() string {
	quickActionsContent := m.getQuickActionsPage(m.quickActionsPage)
	return m.st.quickBar.Render(quickActionsContent)
}

func (m Model) renderMiniSummary() string {
	var total, notes, tasks, meets, timers int
	for _, b := range m.blocks {
		for _, e := range b.entries {
			total++
			switch strings.ToLower(e.cat) {
			case "note":
				notes++
			case "task":
				tasks++
			case "meeting":
				meets++
			case "timer":
				timers++
			}
		}
	}
	line := fmt.Sprintf("Summary: total=%d • notes=%d tasks=%d meetings=%d timers=%d", total, notes, tasks, meets, timers)
	return m.st.summary.Render(line)
}

func (m Model) statusBar() string {
	focus := "Timeline"
	if m.focus == focusSidebar {
		focus = "Sidebar"
	} else if m.focus == focusThread {
		focus = "Thread"
	}
	mode := ""
	switch m.mode {
	case modeSearch:
		mode = " | SEARCH"
	case modeSince:
		mode = " | SINCE"
	case modePicker:
		mode = " | PICK"
	case modeReply:
		mode = " | REPLY"
	case modeEdit:
		mode = " | EDIT"
	case modeHelp:
		mode = " | HELP"
	case modeFocus:
		mode = " | FOCUS"
	case modeStats:
		mode = " | STATS"
	case modeCreate:
		mode = " | CREATE"
	case modeDashboard:
		mode = " | DASHBOARD"
	case modeCalendar:
		mode = " | CALENDAR"
	case modeTemplates:
		mode = " | TEMPLATES"
	case modeExport:
		mode = " | EXPORT"
	case modeAdvancedSearch:
		mode = " | SEARCH"
	case modeTimeReports:
		mode = " | TIME REPORTS"
	case modeProjectSummary:
		mode = " | PROJECTS"
	case modeTagAnalytics:
		mode = " | TAGS"
	}
	hints := "j/k/↑/↓ scroll • Tab/←/→ panes • q quit"
	if m.status != "" {
		hints = m.status
	}
	return m.st.statusBar.Render(fmt.Sprintf("Focus: %s%s   |   %s", focus, mode, hints))
}

// ----- panes -----

func (m Model) renderSidebar(w, h int) string {
	title := m.st.panelTitle.Render("Filters")
	lines := []string{
		"",
	}

	// Projects section
	projectsTitle := "Projects"
	if m.sidebarSection == 0 && m.focus == focusSidebar {
		projectsTitle = "➤ " + projectsTitle
	}
	lines = append(lines, m.st.textBold.Render(projectsTitle))

	for i, it := range m.projects {
		cur := (m.sidebarSection == 0 && m.focus == focusSidebar && m.sidebarCursor == i)
		prefix := "  "
		if cur {
			prefix = "→ "
		}
		active := ""
		if m.filterProj == it.name {
			active = " [x]"
		}
		lines = append(lines, fmt.Sprintf("%s%s (%d)%s", prefix, it.name, it.count, active))
	}

	lines = append(lines, "")

	// Categories section
	categoriesTitle := "Categories"
	if m.sidebarSection == 1 && m.focus == focusSidebar {
		categoriesTitle = "➤ " + categoriesTitle
	}
	lines = append(lines, m.st.textBold.Render(categoriesTitle))

	for i, it := range m.categories {
		cur := (m.sidebarSection == 1 && m.focus == focusSidebar && m.sidebarCursor == i)
		prefix := "  "
		if cur {
			prefix = "→ "
		}
		active := ""
		if m.filterCat == strings.ToLower(it.name) {
			active = " [x]"
		}
		lines = append(lines, fmt.Sprintf("%s%s (%d)%s", prefix, it.name, it.count, active))
	}

	lines = append(lines, "")

	// Tags section
	tagsTitle := "Tags"
	if m.sidebarSection == 2 && m.focus == focusSidebar {
		tagsTitle = "➤ " + tagsTitle
	}
	lines = append(lines, m.st.textBold.Render(tagsTitle))

	for i, it := range m.tags {
		cur := (m.sidebarSection == 2 && m.focus == focusSidebar && m.sidebarCursor == i)
		prefix := "  "
		if cur {
			prefix = "→ "
		}
		active := ""
		if _, ok := m.filterTags[it.name]; ok {
			active = " [x]"
		}
		lines = append(lines, fmt.Sprintf("%s#%s (%d)%s", prefix, it.name, it.count, active))
	}

	// Add help text for space key
	if m.focus == focusSidebar {
		lines = append(lines, "",
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a6adc8")).
				Faint(true).
				Render("Space: Select/clear all"))
	}

	content := strings.Join(lines, "\n")
	body := lipgloss.NewStyle().Width(w - 4).Height(h - 4).Render(content)
	box := m.st.border(m.focus == focusSidebar).Width(w).Height(h).Render(lipgloss.JoinVertical(lipgloss.Left, title, body))
	return box
}

func (m Model) renderThread(w, h int) string {
	title := m.st.panelTitle.Render("Thread")
	body := m.renderBlock(w-4, 0, m.threadBlock, -1, m.now)
	box := m.st.border(m.focus == focusThread).Width(w).Height(h).Render(lipgloss.JoinVertical(lipgloss.Left, title, body))
	return box
}

func (m Model) renderTimeline(w, h int) string {
	// Route to appropriate view renderer based on viewMode
	switch m.viewMode {
	case 0:
		return m.renderTimelineView(w, h)
	case 1:
		return m.renderCardsView(w, h)
	case 2:
		return m.renderTableView(w, h)
	case 3:
		return m.renderKanbanView(w, h)
	default:
		return m.renderTimelineView(w, h)
	}
}

func (m Model) renderTimelineView(w, h int) string {
	// Enhanced header with padding and status info - always show header
	headerInfo := ""
	if len(m.blocks) > 0 {
		if len(m.blocks) > 999 {
			headerInfo = fmt.Sprintf(" (%d+ blocks)", len(m.blocks))
		} else {
			headerInfo = fmt.Sprintf(" (%d blocks)", len(m.blocks))
		}
		if m.timelineScrollOffset > 0 {
			if len(headerInfo) < 30 { // Keep header reasonably short
				headerInfo += fmt.Sprintf(" • offset %d", m.timelineScrollOffset)
			}
		}
	} else {
		// Show scope info even when no blocks
		scopeName := ""
		switch m.scope {
		case scopeToday:
			scopeName = "Today"
		case scopeThisWeek:
			scopeName = "This Week"
		case scopeThisMonth:
			scopeName = "This Month"
		case scopeAll:
			scopeName = "All Time"
		default:
			scopeName = "Custom"
		}
		headerInfo = fmt.Sprintf(" (%s - no entries)", scopeName)
	}

	// Ensure title doesn't exceed available width
	titleText := "Timeline" + headerInfo
	maxTitleWidth := max(10, w-8) // Leave room for borders, minimum 10 chars
	if len(titleText) > maxTitleWidth {
		titleText = "Timeline" // Fallback to just title
	}
	title := m.st.panelTitle.Render(titleText)

	// Add a separator line for better visual separation
	separatorWidth := max(10, w-4) // Ensure minimum width
	separator := m.st.sepFaint.Render(strings.Repeat("─", separatorWidth))

	// Calculate visible blocks based on scroll offset
	availableHeight := h - 6 // Account for title, separator, padding and borders
	availableHeight = max(4, availableHeight) // Ensure minimum height for content

	// Estimate how many blocks we can fit (rough estimate: 3-5 lines per block)
	maxVisibleBlocks := max(1, availableHeight/4)

	// Ensure scroll offset is within bounds
	maxScroll := max(0, len(m.blocks)-maxVisibleBlocks)
	if m.timelineScrollOffset > maxScroll {
		m.timelineScrollOffset = maxScroll
	}

	// Determine which blocks to show
	startBlock := max(0, m.timelineScrollOffset)
	endBlock := min(len(m.blocks), startBlock+maxVisibleBlocks)

	// If cursor is outside visible range, adjust scroll
	if m.focus == focusTimeline && len(m.blocks) > 0 {
		if m.cursorBlock < startBlock {
			m.timelineScrollOffset = m.cursorBlock
		} else if m.cursorBlock >= endBlock {
			m.timelineScrollOffset = max(0, m.cursorBlock-maxVisibleBlocks+1)
		}
		// Recalculate bounds after adjustment
		startBlock = max(0, m.timelineScrollOffset)
		endBlock = min(len(m.blocks), startBlock+maxVisibleBlocks)
	}

	lines := []string{}
	for bi := startBlock; bi < endBlock; bi++ {
		b := m.blocks[bi]
		hl := (m.focus == focusTimeline && bi == m.cursorBlock)
		lines = append(lines, m.renderBlock(w-4, 0, b, m.cursorEntryIf(hl), m.now))
		lines = append(lines, m.st.sepFaint.Render(strings.Repeat("─", min(w-4, 120))))
	}

	// Constrain content width to prevent overflow
	contentStyle := lipgloss.NewStyle().Width(max(10, w-4)).Height(max(1, availableHeight))
	content := contentStyle.Render(strings.Join(lines, "\n"))

	// Add scroll indicator if there are more blocks than can be shown
	if len(m.blocks) > maxVisibleBlocks {
		scrollInfo := fmt.Sprintf("Blocks %d-%d of %d", startBlock+1, endBlock, len(m.blocks))
		if m.timelineScrollOffset > 0 || endBlock < len(m.blocks) {
			scrollStyle := lipgloss.NewStyle().
				Width(max(10, w-4)).
				Foreground(lipgloss.Color("#a6adc8")).
				Faint(true).
				AlignHorizontal(lipgloss.Center)
			content += "\n" + scrollStyle.Render(scrollInfo)
		}
	}

	// Add visual padding and separator - ensure proper layout
	contentWithPadding := lipgloss.JoinVertical(lipgloss.Left, "", content) // Add padding after title
	finalContent := lipgloss.JoinVertical(lipgloss.Left, title, separator, contentWithPadding)

	// Ensure the final box has proper dimensions
	box := m.st.border(m.focus == focusTimeline).Width(w).Height(h).Render(finalContent)
	return box
}

func (m Model) renderCardsView(w, h int) string {
	// Enhanced header with padding and status info - always show header
	var allEntries []entry
	for _, block := range m.blocks {
		allEntries = append(allEntries, block.entries...)
	}

	headerInfo := ""
	if len(allEntries) > 0 {
		visibleCount := min(len(allEntries), max(1, (h-4)/8))
		headerInfo = fmt.Sprintf(" (%d entries, showing %d)", len(allEntries), visibleCount)
		if m.cardsScrollOffset > 0 {
			headerInfo += fmt.Sprintf(" • offset %d", m.cardsScrollOffset)
		}
	} else {
		// Show scope info even when no entries
		scopeName := ""
		switch m.scope {
		case scopeToday:
			scopeName = "Today"
		case scopeThisWeek:
			scopeName = "This Week"
		case scopeThisMonth:
			scopeName = "This Month"
		case scopeAll:
			scopeName = "All Time"
		default:
			scopeName = "Custom"
		}
		headerInfo = fmt.Sprintf(" (%s - no entries)", scopeName)
	}
	title := m.st.panelTitle.Render("Cards" + headerInfo)

	// Add a separator line for better visual separation
	separator := m.st.sepFaint.Render(strings.Repeat("─", w-4))

	// Calculate visible cards based on scroll offset
	availableHeight := h - 6 // Account for title, separator, padding and borders
	cardHeight := 8 // Estimated height per card
	maxVisibleCards := max(1, availableHeight/cardHeight)

	// Ensure scroll offset is within bounds
	maxScroll := max(0, len(allEntries)-maxVisibleCards)
	if m.cardsScrollOffset > maxScroll {
		m.cardsScrollOffset = maxScroll
	}

	// Determine which entries to show
	startEntry := max(0, m.cardsScrollOffset)
	endEntry := min(len(allEntries), startEntry+maxVisibleCards)

	// If cursor is outside visible range, adjust scroll
	if m.focus == focusTimeline && len(allEntries) > 0 {
		if m.cursorBlock < len(m.blocks) {
			// Convert cursorBlock/cursorEntry to flat index
			flatIndex := 0
			for bi, block := range m.blocks {
				for ei := range block.entries {
					if bi == m.cursorBlock && ei == m.cursorEntry {
						goto foundIndex
					}
					flatIndex++
				}
			}
			foundIndex:
			if flatIndex < startEntry {
				m.cardsScrollOffset = flatIndex
			} else if flatIndex >= endEntry {
				m.cardsScrollOffset = max(0, flatIndex-maxVisibleCards+1)
			}
		}
		// Recalculate bounds after adjustment
		startEntry = max(0, m.cardsScrollOffset)
		endEntry = min(len(allEntries), startEntry+maxVisibleCards)
	}

	lines := []string{}
	for i := startEntry; i < endEntry; i++ {
		if i >= len(allEntries) {
			break
		}
		entry := allEntries[i]

		// Check if this entry should be highlighted
		highlight := false
		if m.focus == focusTimeline && len(m.blocks) > 0 {
			// Convert flat index back to block/entry indices
			flatCount := 0
			for bi, block := range m.blocks {
				for ei := range block.entries {
					if flatCount == i {
						highlight = (bi == m.cursorBlock && ei == m.cursorEntry)
						goto foundHighlight
					}
					flatCount++
				}
			}
			foundHighlight:
		}

		lines = append(lines, m.renderCard(w-4, entry, highlight, m.now))
		if i < endEntry-1 {
			lines = append(lines, "")
		}
	}

	content := lipgloss.NewStyle().Width(w - 4).Render(strings.Join(lines, "\n"))

	// Add scroll indicator if there are more entries than can be shown
	if len(allEntries) > maxVisibleCards {
		scrollInfo := fmt.Sprintf("Cards %d-%d of %d", startEntry+1, endEntry, len(allEntries))
		if m.cardsScrollOffset > 0 || endEntry < len(allEntries) {
			content += "\n" + lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a6adc8")).
				Faint(true).
				AlignHorizontal(lipgloss.Center).
				Render(scrollInfo)
		}
	}

	// Add visual padding and separator
	contentWithPadding := lipgloss.JoinVertical(lipgloss.Left, "", content) // Add padding after title
	finalContent := lipgloss.JoinVertical(lipgloss.Left, title, separator, contentWithPadding)

	box := m.st.border(m.focus == focusTimeline).Width(w).Height(h).Render(finalContent)
	return box
}

func (m Model) renderCard(w int, entry entry, highlight bool, now time.Time) string {
	cardWidth := min(w, 80)
	if cardWidth < 40 {
		cardWidth = 40
	}

	// Card border style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6e6a86")).
		Padding(0, 1).
		Width(cardWidth)

	if highlight {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("#9ccfd8"))
	}

	// Header with date and category
	dateStr := entry.when.Format("2006-01-02 15:04")
	categoryColor := colorForCategory(entry.cat)
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(categoryColor).
		Render(fmt.Sprintf("%s | %s", dateStr, entry.cat))

	// Project and tags
	var meta []string
	if entry.project != "" {
		meta = append(meta, fmt.Sprintf("📁 %s", entry.project))
	}
	if len(entry.tags) > 0 {
		meta = append(meta, fmt.Sprintf("🏷️ %s", strings.Join(entry.tags, ", ")))
	}
	metaLine := lipgloss.NewStyle().Faint(true).Render(strings.Join(meta, " • "))

	// Content text (truncated if too long)
	content := entry.text
	maxContentWidth := cardWidth - 4 // Account for padding
	if len(content) > maxContentWidth {
		content = content[:maxContentWidth-3] + "..."
	}
	contentStyle := lipgloss.NewStyle().Width(maxContentWidth).Render(content)

	return borderStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, metaLine, "", contentStyle))
}

func (m Model) renderTableView(w, h int) string {
	// Flatten all entries for table view to calculate header info
	var allEntries []entry
	for _, block := range m.blocks {
		allEntries = append(allEntries, block.entries...)
	}

	// Enhanced header with padding and status info - always show header
	headerInfo := ""
	if len(allEntries) > 0 {
		visibleCount := min(len(allEntries), max(1, h-7))
		headerInfo = fmt.Sprintf(" (%d rows, showing %d)", len(allEntries), visibleCount)
		if m.tableScrollOffset > 0 {
			headerInfo += fmt.Sprintf(" • offset %d", m.tableScrollOffset)
		}
	} else {
		// Show scope info even when no entries
		scopeName := ""
		switch m.scope {
		case scopeToday:
			scopeName = "Today"
		case scopeThisWeek:
			scopeName = "This Week"
		case scopeThisMonth:
			scopeName = "This Month"
		case scopeAll:
			scopeName = "All Time"
		default:
			scopeName = "Custom"
		}
		headerInfo = fmt.Sprintf(" (%s - no entries)", scopeName)
	}
	title := m.st.panelTitle.Render("Table" + headerInfo)

	// Add a separator line for better visual separation
	separator := m.st.sepFaint.Render(strings.Repeat("─", w-4))

	// Calculate visible rows based on scroll offset
	availableHeight := h - 9 // Account for title, separator, padding, header and borders
	maxVisibleRows := max(1, availableHeight)

	// Ensure scroll offset is within bounds
	maxScroll := max(0, len(allEntries)-maxVisibleRows)
	if m.tableScrollOffset > maxScroll {
		m.tableScrollOffset = maxScroll
	}

	// Determine which entries to show
	startEntry := max(0, m.tableScrollOffset)
	endEntry := min(len(allEntries), startEntry+maxVisibleRows)

	// If cursor is outside visible range, adjust scroll
	if m.focus == focusTimeline && len(allEntries) > 0 {
		if m.cursorBlock < len(m.blocks) {
			// Convert cursorBlock/cursorEntry to flat index
			flatIndex := 0
			for bi, block := range m.blocks {
				for ei := range block.entries {
					if bi == m.cursorBlock && ei == m.cursorEntry {
						goto foundTableIndex
					}
					flatIndex++
				}
			}
			foundTableIndex:
			if flatIndex < startEntry {
				m.tableScrollOffset = flatIndex
			} else if flatIndex >= endEntry {
				m.tableScrollOffset = max(0, flatIndex-maxVisibleRows+1)
			}
		}
		// Recalculate bounds after adjustment
		startEntry = max(0, m.tableScrollOffset)
		endEntry = min(len(allEntries), startEntry+maxVisibleRows)
	}

	// Table dimensions
	dateWidth := 16
	catWidth := 12
	projectWidth := 15
	tagsWidth := 20
	contentWidth := w - dateWidth - catWidth - projectWidth - tagsWidth - 5 // account for separators
	if contentWidth < 20 {
		contentWidth = 20
		tagsWidth = max(10, tagsWidth - (20 - contentWidth))
	}

	// Header
	header := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("%-*s %-*s %-*s %-*s %s",
			dateWidth, "Date",
			catWidth, "Category",
			projectWidth, "Project",
			tagsWidth, "Tags",
			"Content"))

	lines := []string{header, m.st.sepFaint.Render(strings.Repeat("─", w-4))}

	// Table rows
	for i := startEntry; i < endEntry; i++ {
		if i >= len(allEntries) {
			break
		}
		entry := allEntries[i]

		// Check if this entry should be highlighted
		highlight := false
		if m.focus == focusTimeline && len(m.blocks) > 0 {
			// Convert flat index back to block/entry indices
			flatCount := 0
			for bi, block := range m.blocks {
				for ei := range block.entries {
					if flatCount == i {
						highlight = (bi == m.cursorBlock && ei == m.cursorEntry)
						goto foundTableHighlight
					}
					flatCount++
				}
			}
			foundTableHighlight:
		}

		// Format row data
		dateStr := entry.when.Format("2006-01-02 15:04")
		catStr := entry.cat
		projectStr := entry.project
		tagsStr := strings.Join(entry.tags, ",")
		contentStr := entry.text

		// Truncate if too long
		if len(tagsStr) > tagsWidth-2 {
			tagsStr = tagsStr[:tagsWidth-5] + "..."
		}
		if len(contentStr) > contentWidth {
			contentStr = contentStr[:contentWidth-3] + "..."
		}

		rowStyle := lipgloss.NewStyle()
		if highlight {
			rowStyle = rowStyle.Background(lipgloss.Color("#313244")).Foreground(lipgloss.Color("#cdd6f4"))
		}

		row := rowStyle.Render(
			fmt.Sprintf("%-*s %-*s %-*s %-*s %s",
				dateWidth, dateStr,
				catWidth, catStr,
				projectWidth, projectStr,
				tagsWidth, tagsStr,
				contentStr))

		lines = append(lines, row)
	}

	content := lipgloss.NewStyle().Width(w - 4).Render(strings.Join(lines, "\n"))

	// Add scroll indicator if there are more entries than can be shown
	if len(allEntries) > maxVisibleRows {
		scrollInfo := fmt.Sprintf("Rows %d-%d of %d", startEntry+1, endEntry, len(allEntries))
		if m.tableScrollOffset > 0 || endEntry < len(allEntries) {
			content += "\n" + lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a6adc8")).
				Faint(true).
				AlignHorizontal(lipgloss.Center).
				Render(scrollInfo)
		}
	}

	// Add visual padding and separator
	contentWithPadding := lipgloss.JoinVertical(lipgloss.Left, "", content) // Add padding after title
	finalContent := lipgloss.JoinVertical(lipgloss.Left, title, separator, contentWithPadding)

	box := m.st.border(m.focus == focusTimeline).Width(w).Height(h).Render(finalContent)
	return box
}

func (m Model) renderKanbanView(w, h int) string {
	// Group entries by category for kanban view
	categories := make(map[string][]entry)
	for _, block := range m.blocks {
		for _, entry := range block.entries {
			cat := entry.cat
			if cat == "" {
				cat = "Uncategorized"
			}
			categories[cat] = append(categories[cat], entry)
		}
	}

	// Sort categories alphabetically
	var sortedCats []string
	for cat := range categories {
		sortedCats = append(sortedCats, cat)
	}
	sort.Strings(sortedCats)

	// Enhanced header with padding and status info - always show header
	headerInfo := ""
	if len(sortedCats) > 0 {
		maxVisibleColumns := min(len(sortedCats), 4)
		headerInfo = fmt.Sprintf(" (%d categories, showing %d)", len(sortedCats), maxVisibleColumns)
		if m.kanbanScrollOffset > 0 {
			headerInfo += fmt.Sprintf(" • offset %d", m.kanbanScrollOffset)
		}
	} else {
		// Show scope info even when no categories
		scopeName := ""
		switch m.scope {
		case scopeToday:
			scopeName = "Today"
		case scopeThisWeek:
			scopeName = "This Week"
		case scopeThisMonth:
			scopeName = "This Month"
		case scopeAll:
			scopeName = "All Time"
		default:
			scopeName = "Custom"
		}
		headerInfo = fmt.Sprintf(" (%s - no entries)", scopeName)
	}
	title := m.st.panelTitle.Render("Kanban" + headerInfo)

	// Add a separator line for better visual separation
	separator := m.st.sepFaint.Render(strings.Repeat("─", w-4))

	// Calculate layout
	availableHeight := h - 6 // Account for title, separator, padding and borders
	availableWidth := w - 4
	maxVisibleColumns := min(len(sortedCats), 4) // Max 4 columns visible at once
	if maxVisibleColumns == 0 {
		maxVisibleColumns = 1
	}

	// Ensure scroll offset is within bounds
	maxScroll := max(0, len(sortedCats)-maxVisibleColumns)
	if m.kanbanScrollOffset > maxScroll {
		m.kanbanScrollOffset = maxScroll
	}

	// Determine which categories to show
	startCat := max(0, m.kanbanScrollOffset)
	endCat := min(len(sortedCats), startCat+maxVisibleColumns)

	columnWidth := (availableWidth - (maxVisibleColumns-1)*3) / maxVisibleColumns // 3 spaces between columns
	if columnWidth < 20 {
		columnWidth = 20
	}

	// Build kanban columns
	var columns []string
	for i := startCat; i < endCat; i++ {
		cat := sortedCats[i]
		entries := categories[cat]

		// Column header
		headerColor := colorForCategory(cat)
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(headerColor).
			AlignHorizontal(lipgloss.Center).
			Width(columnWidth)
		header := headerStyle.Render(fmt.Sprintf("%s (%d)", cat, len(entries)))

		// Column entries
		var columnLines []string
		columnLines = append(columnLines, header)
		columnLines = append(columnLines, m.st.sepFaint.Render(strings.Repeat("─", columnWidth)))

		maxEntries := max(1, (availableHeight-3)/4) // 4 lines per entry min
		for i, entry := range entries {
			if i >= maxEntries {
				// Show indicator if there are more entries
				moreStyle := lipgloss.NewStyle().
					Faint(true).
					AlignHorizontal(lipgloss.Center).
					Width(columnWidth)
				moreIndicator := moreStyle.Render(fmt.Sprintf("... %d more", len(entries)-maxEntries))
				columnLines = append(columnLines, moreIndicator)
				break
			}

			// Check if this entry should be highlighted
			highlight := false
			if m.focus == focusTimeline && len(m.blocks) > 0 {
				// Find this entry in the original blocks
				for bi, block := range m.blocks {
					for ei, e := range block.entries {
						if e.id == entry.id {
							highlight = (bi == m.cursorBlock && ei == m.cursorEntry)
							goto foundKanbanHighlight
						}
					}
				}
				foundKanbanHighlight:
			}

			cardText := m.renderKanbanCard(columnWidth-2, entry, highlight)
			columnLines = append(columnLines, cardText)
			if i < len(entries)-1 && i < maxEntries-1 {
				columnLines = append(columnLines, "")
			}
		}

		columns = append(columns, strings.Join(columnLines, "\n"))
	}

	// Join columns horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, columns...)

	// Add scroll info if needed
	totalEntries := 0
	for _, entries := range categories {
		totalEntries += len(entries)
	}
	if totalEntries > 0 {
		infoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a6adc8")).
			Faint(true).
			AlignHorizontal(lipgloss.Center).
			Width(availableWidth)

		var info string
		if len(sortedCats) > maxVisibleColumns {
			// Show horizontal scroll info when there are more categories than can be displayed
			info = fmt.Sprintf("Categories %d-%d of %d | %d entries", startCat+1, endCat, len(sortedCats), totalEntries)
		} else {
			info = fmt.Sprintf("%d entries across %d categories", totalEntries, len(sortedCats))
		}

		content = lipgloss.JoinVertical(lipgloss.Left, content, infoStyle.Render(info))
	}

	// Add visual padding and separator
	contentWithPadding := lipgloss.JoinVertical(lipgloss.Left, "", content) // Add padding after title
	finalContent := lipgloss.JoinVertical(lipgloss.Left, title, separator, contentWithPadding)

	box := m.st.border(m.focus == focusTimeline).Width(w).Height(h).Render(finalContent)
	return box
}

func (m Model) renderKanbanCard(w int, entry entry, highlight bool) string {
	// Simplified card for kanban column
	maxLines := 3
	maxWidth := w

	// Date and first line of text
	dateStr := entry.when.Format("01-02 15:04")
	textLines := strings.Split(entry.text, "\n")

	var cardLines []string

	// First line: date + text (truncated)
	firstLine := dateStr + " " + textLines[0]
	if len(firstLine) > maxWidth {
		firstLine = firstLine[:maxWidth-3] + "..."
	}
	cardLines = append(cardLines, firstLine)

	// Add additional lines if space permits
	for i := 1; i < len(textLines) && i < maxLines-1; i++ {
		line := textLines[i]
		if len(line) > maxWidth {
			line = line[:maxWidth-3] + "..."
		}
		cardLines = append(cardLines, line)
	}

	// Add project info if space allows
	if entry.project != "" && len(cardLines) < maxLines {
		projectLine := "📁 " + entry.project
		if len(projectLine) > maxWidth {
			projectLine = "📁 " + entry.project[:maxWidth-5] + "..."
		}
		cardLines = append(cardLines, projectLine)
	}

	cardStyle := lipgloss.NewStyle().
		Width(maxWidth).
		Height(maxLines)

	if highlight {
		cardStyle = cardStyle.Background(lipgloss.Color("#313244"))
	}

	return cardStyle.Render(strings.Join(cardLines, "\n"))
}

func (m Model) cursorEntryIf(hl bool) int {
	if !hl {
		return -1
	}
	return m.cursorEntry
}

func (m Model) renderBlock(w int, _ int, b block, cursorEntry int, now time.Time) string {
	if len(b.entries) == 0 {
		return ""
	}
	leftW := 14
	threadW := 2
	rightW := 26
	bodyW := w - leftW - threadW - 1 - rightW - 1
	if bodyW < 20 {
		bodyW = 20
	}
	rootCol := colorForCategory(b.rootCat)
	dot := lipgloss.NewStyle().Foreground(rootCol).Render("●")
	pipe := lipgloss.NewStyle().Foreground(rootCol).Render("│")
	tee := lipgloss.NewStyle().Foreground(rootCol).Render("├")
	elb := lipgloss.NewStyle().Foreground(rootCol).Render("└")

	var out []string
	prevMonth := ""

	for i, e := range b.entries {
		mLabel := monthOrToday(e.when, now)
		if mLabel != prevMonth {
			prevMonth = mLabel
			out = append(out, m.st.month.Render(padRight(mLabel, leftW)))
		}

		isFirst := i == 0
		isLast := i == len(b.entries)-1
		glyph := tee
		if isFirst && isLast {
			glyph = dot
		} else if isFirst {
			glyph = dot
		} else if isLast {
			glyph = elb
		}

		abs := absTimeFor(e.when, now)
		rel := humanizeAge(e.when, now)
		right := m.st.age.Width(rightW).AlignHorizontal(lipgloss.Right).Render(fmt.Sprintf("%s • %s", abs, rel))

		bodyLines := wrapText(e.text, bodyW)
		if len(bodyLines) == 0 {
			bodyLines = []string{""}
		}

		leftGutter := padRight("", leftW)
		threadPad := padRight("", threadW-1)

		bold := (cursorEntry == i)
		bodyStyle := lipgloss.NewStyle().Width(bodyW)
		if bold {
			bodyStyle = bodyStyle.Bold(true)
		}

		out = append(out, fmt.Sprintf("%s%s%s %s %s", leftGutter, threadPad, glyph, bodyStyle.Render(bodyLines[0]), right))
		for _, ln := range bodyLines[1:] {
			out = append(out, fmt.Sprintf("%s%s%s %s", leftGutter, threadPad, pipe, lipgloss.NewStyle().Width(bodyW).Render(ln)))
		}

		// meta line: CAT  [project]  #tags  [#id]
		metaParts := make([]string, 0, 4)
		if e.cat != "" {
			metaParts = append(metaParts, lipgloss.NewStyle().Bold(true).Foreground(colorForCategory(e.cat)).Render(strings.ToUpper(e.cat)))
		}
		if strings.TrimSpace(e.project) != "" {
			metaParts = append(metaParts, m.st.project.Render("["+strings.TrimSpace(e.project)+"]"))
		}
		if len(e.tags) > 0 {
			metaParts = append(metaParts, m.st.tags.Render("#"+strings.Join(e.tags, " #")))
		}
		// Add bookmark indicator
		bookmarkIndicator := ""
		if _, bookmarked := m.bookmarks[e.id]; bookmarked {
			bookmarkIndicator = " 🔖"
		}
		metaParts = append(metaParts, lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("[#%d]%s", e.id, bookmarkIndicator)))

		out = append(out, fmt.Sprintf("%s%s%s %s", leftGutter, threadPad, pipe, strings.Join(metaParts, "  ")))
		if !isLast {
			out = append(out, fmt.Sprintf("%s%s%s", leftGutter, threadPad, pipe))
		}
	}
	return strings.Join(out, "\n")
}

// ---------- data loading ----------

func loadBlocks(dbh *sql.DB, loc *time.Location, sc scope, textFilter, proj, cat string, tags map[string]struct{}, anyTags bool, sinceValue time.Time) ([]block, error) {
	fromLocal := time.Now().In(loc)
	switch sc {
	case scopeAll:
		fromLocal = time.Unix(0, 0).In(loc)
	case scopeToday:
		y, m, d := fromLocal.Date()
		fromLocal = time.Date(y, m, d, 0, 0, 0, 0, loc)
	case scopeThisWeek:
		// Start of week (Sunday)
		weekday := int(fromLocal.Weekday())
		fromLocal = fromLocal.AddDate(0, 0, -weekday)
		y, m, d := fromLocal.Date()
		fromLocal = time.Date(y, m, d, 0, 0, 0, 0, loc)
	case scopeThisMonth:
		y, m, _ := fromLocal.Date()
		fromLocal = time.Date(y, m, 1, 0, 0, 0, 0, loc)
	case scopeYesterday:
		y, m, d := fromLocal.AddDate(0, 0, -1).Date()
		fromLocal = time.Date(y, m, d, 0, 0, 0, 0, loc)
	case scopeLastWeek:
		weekday := int(fromLocal.Weekday())
		thisWeekStart := fromLocal.AddDate(0, 0, -weekday)
		y, m, d := thisWeekStart.Date()
		thisWeekStart = time.Date(y, m, d, 0, 0, 0, 0, loc)
		fromLocal = thisWeekStart.AddDate(0, 0, -7)
	case scopeLastMonth:
		y, m, _ := fromLocal.AddDate(0, -1, 0).Date()
		fromLocal = time.Date(y, m, 1, 0, 0, 0, 0, loc)
	case scopeSince:
		if !sinceValue.IsZero() {
			fromLocal = sinceValue.In(loc)
		}
	}
	fromUTC := fromLocal.UTC().Format(time.RFC3339)

	conds := []string{"ts >= ?"}
	argsQ := []any{fromUTC}

	if strings.TrimSpace(textFilter) != "" {
		conds = append(conds, "(instr(text, ?) > 0 OR instr(project, ?) > 0 OR instr(tags, ?) > 0)")
		argsQ = append(argsQ, textFilter, textFilter, textFilter)
	}
	if strings.TrimSpace(proj) != "" {
		conds = append(conds, "project = ?")
		argsQ = append(argsQ, proj)
	}
	if strings.TrimSpace(cat) != "" {
		conds = append(conds, "lower(category) = lower(?)")
		argsQ = append(argsQ, cat)
	}
	if len(tags) > 0 {
		var tagConds []string
		for t := range tags {
			tagConds = append(tagConds, "instr(tags, ?) > 0")
			argsQ = append(argsQ, t)
		}
		if anyTags {
			conds = append(conds, "("+strings.Join(tagConds, " OR ")+")")
		} else {
			conds = append(conds, strings.Join(tagConds, " AND "))
		}
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	// discover roots & max ts
	rows, err := dbh.Query(`
		SELECT COALESCE(thread_id, id) AS root, MAX(ts) AS latest
		FROM entries
		`+where+`
		GROUP BY root
		ORDER BY latest DESC
	`, argsQ...)
	if err != nil {
		return nil, err
	}
	type rootRec struct {
		root   int
		latest time.Time
	}
	var roots []rootRec
	for rows.Next() {
		var root int
		var latestStr string
		if err := rows.Scan(&root, &latestStr); err != nil {
			_ = rows.Close()
			return nil, err
		}
		roots = append(roots, rootRec{root: root, latest: parseAny(latestStr).In(loc)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	_ = rows.Close()

	if len(roots) == 0 {
		return []block{}, nil
	}

	// load each block
	var blocks []block
	for _, r := range roots {
		tr, err := dbh.Query(`
			SELECT id, ts, category, COALESCE(project,''), COALESCE(tags,''), COALESCE(text,'')
			FROM entries
			WHERE id = ? OR thread_id = ?
			ORDER BY ts ASC, id ASC
		`, r.root, r.root)
		if err != nil {
			return nil, err
		}
		var items []entry
		var rootCat string
		var monthLabel string
		for tr.Next() {
			var id int
			var tsStr, catS, projS, tagsS, text string
			if err := tr.Scan(&id, &tsStr, &catS, &projS, &tagsS, &text); err != nil {
				_ = tr.Close()
				return nil, err
			}
			t := parseAny(tsStr).In(loc)
			if rootCat == "" {
				rootCat = strings.ToLower(catS)
				monthLabel = monthOrToday(t, time.Now().In(loc))
			}
			items = append(items, entry{
				id:      id,
				when:    t,
				cat:     strings.ToLower(catS),
				project: projS,
				tags:    splitTags(tagsS),
				text:    strings.TrimSpace(text),
			})
		}
		_ = tr.Close()
		if len(items) == 0 {
			continue
		}
		blocks = append(blocks, block{
			rootID:     r.root,
			rootCat:    rootCat,
			latest:     r.latest,
			entries:    items,
			monthLabel: monthLabel,
		})
	}
	sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].latest.After(blocks[j].latest) })
	return blocks, nil
}

func loadFacets(dbh *sql.DB) (projects, cats, tags []facetItem, err error) {
	// projects
	{
		rows, e := dbh.Query(`SELECT COALESCE(project,''), COUNT(*) FROM entries GROUP BY 1 ORDER BY 2 DESC, 1`)
		if e != nil {
			return nil, nil, nil, e
		}
		for rows.Next() {
			var name string
			var c int
			if er := rows.Scan(&name, &c); er != nil {
				_ = rows.Close()
				return nil, nil, nil, er
			}
			if strings.TrimSpace(name) != "" {
				projects = append(projects, facetItem{name: name, count: c})
			}
		}
		_ = rows.Close()
	}
	// categories
	{
		rows, e := dbh.Query(`SELECT lower(category), COUNT(*) FROM entries GROUP BY 1 ORDER BY 2 DESC`)
		if e != nil {
			return nil, nil, nil, e
		}
		for rows.Next() {
			var name string
			var c int
			if er := rows.Scan(&name, &c); er != nil {
				_ = rows.Close()
				return nil, nil, nil, er
			}
			cats = append(cats, facetItem{name: name, count: c})
		}
		_ = rows.Close()
	}
	// tags (split CSV)
	{
		rows, e := dbh.Query(`SELECT COALESCE(tags,'') FROM entries WHERE tags IS NOT NULL AND tags <> ''`)
		if e != nil {
			return nil, nil, nil, e
		}
		counter := map[string]int{}
		for rows.Next() {
			var csv string
			if er := rows.Scan(&csv); er != nil {
				_ = rows.Close()
				return nil, nil, nil, er
			}
			for _, t := range splitTags(csv) {
				counter[t]++
			}
		}
		_ = rows.Close()
		for k, v := range counter {
			tags = append(tags, facetItem{name: k, count: v})
		}
		sort.Slice(tags, func(i, j int) bool {
			if tags[i].count == tags[j].count {
				return tags[i].name < tags[j].name
			}
			return tags[i].count > tags[j].count
		})
	}
	return
}

// ---------- actions (db) ----------

func insertReply(dbh *sql.DB, parentID int, text string) error {
	// fetch parent to inherit project/category/thread
	parent, err := db.GetEntry(dbh, parentID)
	if err != nil {
		return err
	}
	root := db.RootThreadID(parent)
	cat := "note"
	if strings.TrimSpace(parent.Category) != "" {
		cat = strings.ToLower(parent.Category)
	}
	project := ""
	if parent.Project.Valid {
		project = strings.TrimSpace(parent.Project.String)
	}
	tags := ""
	if parent.Tags.Valid {
		tags = strings.TrimSpace(parent.Tags.String)
	}
	_, err = dbh.Exec(`
		INSERT INTO entries(category, text, project, tags, thread_id, parent_id)
		VALUES(?,?,?,?,?,?)
	`, cat, text, nullIfEmpty(project), nullIfEmpty(tags), root, parentID)
	return err
}

func updateEntryText(dbh *sql.DB, id int, text string) error {
	_, err := dbh.Exec(`UPDATE entries SET text = ? WHERE id = ?`, text, id)
	return err
}

func insertReplyWithProjectTags(dbh *sql.DB, parentID int, text, project, tags string) error {
	// fetch parent to inherit category/thread
	parent, err := db.GetEntry(dbh, parentID)
	if err != nil {
		return err
	}
	root := db.RootThreadID(parent)
	cat := "note"
	if strings.TrimSpace(parent.Category) != "" {
		cat = strings.ToLower(parent.Category)
	}

	// Use provided project and tags, fall back to parent's values if empty
	if project == "" {
		if parent.Project.Valid {
			project = parent.Project.String
		} else {
			project = ""
		}
	}
	if tags == "" {
		if parent.Tags.Valid {
			tags = parent.Tags.String
		} else {
			tags = ""
		}
	}

	_, err = dbh.Exec(`
		INSERT INTO entries(category, text, project, tags, thread_id, parent_id)
		VALUES(?,?,?,?,?,?)
	`, cat, text, nullIfEmpty(project), nullIfEmpty(tags), root, parentID)
	return err
}

func updateEntryTextProjectTags(dbh *sql.DB, id int, text, project, tags string) error {
	// Get the current entry to preserve other fields if needed
	_, err := dbh.Exec(`
		UPDATE entries SET
			text = ?,
			project = COALESCE(NULLIF(?, ''), project),
			tags = COALESCE(NULLIF(?, ''), tags)
		WHERE id = ?
	`, text, project, tags, id)
	return err
}

func exportThreadMarkdown(b block, loc *time.Location) (string, error) {
	home, _ := os.UserHomeDir()
	outDir := filepath.Join(home, ".config", "pulse", "exports")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("thread-%d-%s.md", b.rootID, time.Now().Format("20060102-150405"))
	path := filepath.Join(outDir, filename)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Thread %d\n\n", b.rootID))
	for _, e := range b.entries {
		ts := e.when.In(loc).Format("2006-01-02 03:04 PM")
		meta := []string{strings.ToUpper(e.cat)}
		if strings.TrimSpace(e.project) != "" {
			meta = append(meta, "["+e.project+"]")
		}
		if len(e.tags) > 0 {
			meta = append(meta, "#"+strings.Join(e.tags, " #"))
		}
		sb.WriteString(fmt.Sprintf("## #%d  %s  — %s\n\n", e.id, ts, strings.Join(meta, "  ")))
		if strings.TrimSpace(e.text) != "" {
			sb.WriteString(e.text + "\n\n")
		}
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ---------- helpers: UI/time/text/colors ----------

func (m *Model) cycleFocus() {
	switch m.focus {
	case focusTimeline:
		if m.showSidebar {
			m.focus = focusSidebar
		} else if m.showThread {
			m.focus = focusThread
		} else {
			m.showSidebar = true
			m.focus = focusSidebar
		}
	case focusSidebar:
		if m.showThread {
			m.focus = focusThread
		} else {
			m.focus = focusTimeline
		}
	case focusThread:
		m.focus = focusTimeline
	}
}

func (s style) border(focused bool) lipgloss.Style {
	if focused {
		return s.borderFocus
	}
	return s.borderDim
}

func parseAny(s string) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}

func splitTags(csv string) []string {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func monthOrToday(t, now time.Time) string {
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return "Today"
	}
	return t.Format("Jan 2006")
}

func absTimeFor(t time.Time, now time.Time) string {
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("03:04 PM")
	}
	return t.Format("Jan 02 03:04 PM")
}

func humanizeAge(t time.Time, now time.Time) string {
	d := now.Sub(t)
	if d < time.Minute {
		sec := int(d.Seconds())
		if sec <= 1 {
			return "just now"
		}
		return fmt.Sprintf("%ds ago", sec)
	}
	if d < time.Hour {
		min := int(d.Minutes())
		if min == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", min)
	}
	if d < 24*time.Hour {
		hrs := int(d.Hours())
		if hrs == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hrs)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1d ago"
	}
	if days < 7 {
		return fmt.Sprintf("%dd ago", days)
	}
	weeks := days / 7
	if weeks == 1 {
		return "1w ago"
	}
	if weeks < 5 {
		return fmt.Sprintf("%dw ago", weeks)
	}
	months := int(float64(days) / 30.4)
	if months == 1 {
		return "1mo ago"
	}
	if months < 12 {
		return fmt.Sprintf("%dmo ago", months)
	}
	years := months / 12
	if years == 1 {
		return "1y ago"
	}
	return fmt.Sprintf("%dy ago", years)
}

func colorForCategory(cat string) lipgloss.Color {
	switch strings.ToLower(cat) {
	case "task":
		return lipgloss.Color("#F9E2AF")
	case "meeting":
		return lipgloss.Color("#F5C2E7")
	case "timer":
		return lipgloss.Color("#A6E3A1")
	case "note":
		return lipgloss.Color("#89B4FA")
	default:
		return lipgloss.Color("#94E2D5")
	}
}

func padRight(s string, w int) string {
	if len([]rune(s)) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len([]rune(s)))
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)

	if len(words) == 0 {
		return []string{""}
	}

	currentLine := ""
	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				// Word is longer than width, split it
				for len(word) > width {
					lines = append(lines, word[:width])
					word = word[width:]
				}
				currentLine = word
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// overlays

func (m Model) modal(title, content string) string {
	box := lipgloss.JoinVertical(lipgloss.Left,
		m.st.modalTitle.Render(title),
		content,
	)
	return m.st.modalBox.Render(box)
}

func (m Model) renderReplyModal() string {
	content := ""

	// Text field
	textLabel := "Reply Text"
	if m.editField == 0 {
		textLabel = "➤ Reply Text"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(textLabel), m.editor.View())

	// Project field
	projectLabel := "Project (optional)"
	if m.editField == 1 {
		projectLabel = "➤ Project (optional)"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(projectLabel), m.editProject.View())

	// Tags field
	tagsLabel := "Tags (optional)"
	if m.editField == 2 {
		tagsLabel = "➤ Tags (optional)"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(tagsLabel), m.editTags.View())

	// Create button styles with visual feedback for selection
	okButtonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1e1e2e")).
		Background(lipgloss.Color("#a6e3a1")).
		Padding(0, 2).
		Bold(true)

	cancelButtonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdd6f4")).
		Background(lipgloss.Color("#585b70")).
		Padding(0, 2)

	// Highlight selected button
	if m.selectedButton == 0 {
		okButtonStyle = okButtonStyle.
			Foreground(lipgloss.Color("#1e1e2e")).
			Background(lipgloss.Color("#94e2d5")).
			Underline(true)
	} else {
		cancelButtonStyle = cancelButtonStyle.
			Foreground(lipgloss.Color("#1e1e2e")).
			Background(lipgloss.Color("#f38ba8")).
			Underline(true)
	}

	okText := okButtonStyle.Render("OK (Enter)")
	cancelText := cancelButtonStyle.Render("Cancel (Esc)")

	buttons := lipgloss.JoinHorizontal(lipgloss.Left, okText, "  ", cancelText)
	content += buttons

	// Add hint for tab navigation
	content += "\n\n" + lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Faint(true).
		Render("Tip: Use Tab to switch between buttons, Enter to select")

	// Store button positions for mouse detection (approximate)
	m.okButtonRect = [4]int{0, 0, 15, 1}      // x, y, width, height
	m.cancelButtonRect = [4]int{18, 0, 15, 1} // x, y, width, height

	return m.modal("Reply", content)
}

func (m Model) renderEditModal() string {
	content := ""

	// Text field
	textLabel := "Entry Text"
	if m.editField == 0 {
		textLabel = "➤ Entry Text"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(textLabel), m.editor.View())

	// Project field
	projectLabel := "Project (optional)"
	if m.editField == 1 {
		projectLabel = "➤ Project (optional)"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(projectLabel), m.editProject.View())

	// Tags field
	tagsLabel := "Tags (optional)"
	if m.editField == 2 {
		tagsLabel = "➤ Tags (optional)"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(tagsLabel), m.editTags.View())

	// Create button styles with visual feedback for selection
	okButtonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1e1e2e")).
		Background(lipgloss.Color("#a6e3a1")).
		Padding(0, 2).
		Bold(true)

	cancelButtonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdd6f4")).
		Background(lipgloss.Color("#585b70")).
		Padding(0, 2)

	// Highlight selected button
	if m.selectedButton == 0 {
		okButtonStyle = okButtonStyle.
			Foreground(lipgloss.Color("#1e1e2e")).
			Background(lipgloss.Color("#94e2d5")).
			Underline(true)
	} else {
		cancelButtonStyle = cancelButtonStyle.
			Foreground(lipgloss.Color("#1e1e2e")).
			Background(lipgloss.Color("#f38ba8")).
			Underline(true)
	}

	okText := okButtonStyle.Render("OK (Enter)")
	cancelText := cancelButtonStyle.Render("Cancel (Esc)")

	buttons := lipgloss.JoinHorizontal(lipgloss.Left, okText, "  ", cancelText)
	content += buttons

	// Add hint for tab navigation
	content += "\n\n" + lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Faint(true).
		Render("Tip: Use Tab to switch between buttons, Enter to select")

	// Store button positions for mouse detection (approximate)
	m.okButtonRect = [4]int{0, 0, 15, 1}      // x, y, width, height
	m.cancelButtonRect = [4]int{18, 0, 15, 1} // x, y, width, height

	return m.modal("Edit Entry", content)
}

func (m Model) renderCreateModal() string {
	content := ""

	// Text field (always highlighted as it's required)
	textLabel := "Text *"
	if m.createField == 0 {
		textLabel = "➤ Text *"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(textLabel), m.createText.View())

	// Project field
	projectLabel := "Project"
	if m.createField == 1 {
		projectLabel = "➤ Project"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(projectLabel), m.createProject.View())

	// Category field
	categoryLabel := "Category"
	if m.createField == 2 {
		categoryLabel = "➤ Category"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(categoryLabel), m.createCategory.View())

	// Tags field
	tagsLabel := "Tags"
	if m.createField == 3 {
		tagsLabel = "➤ Tags"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(tagsLabel), m.createTags.View())

	// Help text
	content += lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Faint(true).
		Render("Tab: Next field  •  Enter: Save entry  •  Esc: Cancel")

	return m.modal("Create New Entry", content)
}

func (m Model) renderAdvancedSearchView() string {
	content := ""

	// Query field
	queryLabel := "Search Query"
	if m.advancedSearchField == 0 {
		queryLabel = "➤ Search Query"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(queryLabel), m.advancedSearchQuery.View())

	// Project field
	projectLabel := "Project"
	if m.advancedSearchField == 1 {
		projectLabel = "➤ Project"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(projectLabel), m.advancedSearchProject.View())

	// Category field
	categoryLabel := "Category"
	if m.advancedSearchField == 2 {
		categoryLabel = "➤ Category"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(categoryLabel), m.advancedSearchCategory.View())

	// Tags field
	tagsLabel := "Tags"
	if m.advancedSearchField == 3 {
		tagsLabel = "➤ Tags"
	}
	content += fmt.Sprintf("%s\n%s\n\n", m.st.textBold.Render(tagsLabel), m.advancedSearchTags.View())

	// Results section
	if len(m.advancedSearchResults) > 0 {
		content += "\n" + m.st.textBold.Render("Results (Top 10):") + "\n\n"
		maxResults := 10
		if len(m.advancedSearchResults) < maxResults {
			maxResults = len(m.advancedSearchResults)
		}
		for i := 0; i < maxResults; i++ {
			result := m.advancedSearchResults[i]
			preview := result.text
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}
			content += fmt.Sprintf("#%d %s: %s\n", result.id, strings.ToUpper(result.cat), preview)
		}
		if len(m.advancedSearchResults) > 10 {
			content += fmt.Sprintf("... and %d more\n", len(m.advancedSearchResults)-10)
		}
	}

	// Help text
	content += "\n" + lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Faint(true).
		Render("Tab: Next field  •  Enter: Search  •  Esc: Cancel")

	return m.modal("Advanced Search", content)
}

func (m Model) renderCalendarView() string {
	var content string

	if m.calendarPreviewMode {
		content = m.renderDateEntryPreview()
	} else {
		switch m.calendarView {
		case 0:
			content = m.renderMonthView()
		case 1:
			content = m.renderWeekView()
		case 2:
			content = m.renderDayView()
		}
	}

	return m.modal("📅 Calendar", content)
}

func (m Model) renderMonthView() string {
	year, month, _ := m.calendarDate.Date()
	monthName := m.calendarDate.Format("January 2006")

	// Calculate first day of month and number of days
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, m.loc)
	lastDay := firstDay.AddDate(0, 1, -1)
	daysInMonth := lastDay.Day()

	// Calculate starting weekday (0 = Sunday)
	startWeekday := int(firstDay.Weekday())

	// Build calendar grid
	var grid strings.Builder

	// Header with month and navigation
	grid.WriteString(fmt.Sprintf("        %s\n\n", monthName))

	// Weekday headers
	grid.WriteString(" Su  Mo  Tu  We  Th  Fr  Sa\n")
	grid.WriteString(" ──────────────────────────\n")

	// Calendar grid
	for week := 0; week < 6; week++ {
		for day := 0; day < 7; day++ {
			cellDate := week*7 + day - startWeekday + 1

			if cellDate < 1 || cellDate > daysInMonth {
				grid.WriteString("    ") // Empty cell
			} else {
				dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, cellDate)
				entryCount := m.calendarEntryCounts[dateStr]

				// Check if this is today
				isToday := m.now.Year() == year && m.now.Month() == month && m.now.Day() == cellDate
				// Check if this is selected date
				isSelected := m.calendarSelectedDate.Year() == year && m.calendarSelectedDate.Month() == month && m.calendarSelectedDate.Day() == cellDate

				// Format the cell
				cell := fmt.Sprintf("%2d", cellDate)
				if entryCount > 0 {
					cell += "*"
				} else {
					cell += " "
				}

				// Apply styling based on state
				if isSelected {
					cell = fmt.Sprintf("[%s]", cell)
				} else if isToday {
					cell = fmt.Sprintf("(%s)", cell)
				} else {
					cell = fmt.Sprintf(" %s ", cell)
				}

				grid.WriteString(cell)
			}
		}
		grid.WriteString("\n")
	}

	// Legend
	grid.WriteString("\n Legend:\n")
	grid.WriteString(" *  = Has entries\n")
	grid.WriteString(" () = Today's date\n")
	grid.WriteString(" [] = Selected date\n")

	// Navigation help
	grid.WriteString("\n Navigation:\n")
	grid.WriteString(" ←/h  : Previous month\n")
	grid.WriteString(" →/l  : Next month\n")
	grid.WriteString(" ↑/k  : Week up\n")
	grid.WriteString(" ↓/j  : Week down\n")
	grid.WriteString(" v    : Switch to week view\n")
	grid.WriteString(" Enter: View entries for selected date\n")
	grid.WriteString(" t    : Go to today\n")
	grid.WriteString(" Esc  : Exit calendar")

	return grid.String()
}

func (m Model) renderWeekView() string {
	// Find start of week (Sunday)
	weekday := int(m.calendarDate.Weekday())
	startOfWeek := m.calendarDate.AddDate(0, 0, -weekday)

	var grid strings.Builder
	grid.WriteString(fmt.Sprintf("    Week of %s\n\n", startOfWeek.Format("Jan 2, 2006")))

	// Weekday headers
	weekdays := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for _, day := range weekdays {
		grid.WriteString(fmt.Sprintf(" %-10s", day))
	}
	grid.WriteString("\n")
	grid.WriteString(" ───────────────────────────────────────\n")

	// Week days with entry counts
	for i := 0; i < 7; i++ {
		currentDate := startOfWeek.AddDate(0, 0, i)
		dateStr := currentDate.Format("2006-01-02")
		entryCount := m.calendarEntryCounts[dateStr]

		isToday := m.now.Year() == currentDate.Year() && m.now.Month() == currentDate.Month() && m.now.Day() == currentDate.Day()
		isSelected := m.calendarSelectedDate.Year() == currentDate.Year() && m.calendarSelectedDate.Month() == currentDate.Month() && m.calendarSelectedDate.Day() == currentDate.Day()

		dayCell := fmt.Sprintf("%s %2d", currentDate.Format("Mon"), currentDate.Day())
		if entryCount > 0 {
			dayCell += fmt.Sprintf(" (%d)", entryCount)
		}

		if isSelected {
			dayCell = fmt.Sprintf("[%s]", dayCell)
		} else if isToday {
			dayCell = fmt.Sprintf("(%s)", dayCell)
		}

		grid.WriteString(fmt.Sprintf(" %-10s", dayCell))
	}
	grid.WriteString("\n")

	// Navigation help
	grid.WriteString("\n Navigation:\n")
	grid.WriteString(" ←/h  : Previous week\n")
	grid.WriteString(" →/l  : Next week\n")
	grid.WriteString(" ↑/k  : Previous day\n")
	grid.WriteString(" ↓/j  : Next day\n")
	grid.WriteString(" v    : Switch to day view\n")
	grid.WriteString(" Enter: View entries for selected date\n")
	grid.WriteString(" t    : Go to today\n")
	grid.WriteString(" Esc  : Exit calendar")

	return grid.String()
}

func (m Model) renderDayView() string {
	dateStr := m.calendarDate.Format("2006-01-02")
	entryCount := m.calendarEntryCounts[dateStr]

	var grid strings.Builder
	grid.WriteString(fmt.Sprintf("   %s\n\n", m.calendarDate.Format("Monday, January 2, 2006")))

	// Entry count
	grid.WriteString(fmt.Sprintf(" Entries: %d\n", entryCount))

	if entryCount > 0 {
		grid.WriteString("\n ✨ This day has journal entries!\n")
		grid.WriteString(" Press Enter to view them")
	} else {
		grid.WriteString("\n 📝 No entries for this day\n")
		grid.WriteString(" Press 'n' to create a new entry")
	}

	// Navigation help
	grid.WriteString("\n\n Navigation:\n")
	grid.WriteString(" ←/h  : Previous day\n")
	grid.WriteString(" →/l  : Next day\n")
	grid.WriteString(" v    : Switch to month view\n")
	grid.WriteString(" Enter: View/Create entries\n")
	grid.WriteString(" t    : Go to today\n")
	grid.WriteString(" Esc  : Exit calendar")

	return grid.String()
}

func (m Model) renderDateEntryPreview() string {
	entries, err := db.GetEntriesByDate(m.db, m.calendarSelectedDate, m.loc)

	var grid strings.Builder
	grid.WriteString(fmt.Sprintf("   Entries for %s\n\n", m.calendarSelectedDate.Format("Monday, January 2, 2006")))

	if err != nil || len(entries) == 0 {
		grid.WriteString(" No entries found for this date.\n\n")
		grid.WriteString(" Press 'n' to create a new entry")
	} else {
		grid.WriteString(fmt.Sprintf(" Found %d entries:\n\n", len(entries)))

		for i, entry := range entries {
			if i >= 10 { // Limit to 10 entries
				grid.WriteString(fmt.Sprintf(" ... and %d more entries\n", len(entries)-10))
				break
			}

			// Parse timestamp
			ts, _ := time.Parse(time.RFC3339, entry.TS)
			timeStr := ts.Format("3:04 PM")

			// Entry preview
			text := entry.Text.String
			if len(text) > 50 {
				text = text[:47] + "..."
			}

			grid.WriteString(fmt.Sprintf(" %s  %s", timeStr, text))
			if entry.Project.Valid && entry.Project.String != "" {
				grid.WriteString(fmt.Sprintf(" [%s]", entry.Project.String))
			}
			grid.WriteString("\n")
		}
	}

	// Navigation help
	grid.WriteString("\n Navigation:\n")
	grid.WriteString(" Esc  : Back to calendar\n")
	grid.WriteString(" n    : Create new entry for this date")

	return grid.String()
}

func (m Model) renderTemplatesView() string {
	// Create two-column layout: categories on left, templates on right
	categoryWidth := 25
	templateWidth := 45

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#f9e2af")).
		AlignHorizontal(lipgloss.Center).
		Render("📋 Template Library")

	// Category panel
	categoryPanel := m.renderTemplateCategories(categoryWidth)

	// Template panel
	templatePanel := m.renderTemplateList(templateWidth)

	// Combine panels
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, categoryPanel, templatePanel)

	// Help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Faint(true).
		AlignHorizontal(lipgloss.Center).
		Render("1-5: Quick cat  •  ←/→: Categories  •  ↑/↓: Navigate  •  Tab: Toggle  •  Enter: Select  •  Esc: Cancel")

	// Full content
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		mainContent,
		"",
		helpText,
	)

	return m.modal("📋 Templates", content)
}

func (m Model) renderTemplateCategories(width int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#89b4fa")).
		Render("Categories")

	lines := []string{title, ""}

	for i, category := range m.templateCategories {
		// Determine if this category is selected
		isSelected := (i == m.templateCategoryCursor && !m.templateFilterMode) ||
		             (i == m.templateCategoryCursor && m.templateFilterMode)

		// Count templates in this category
		templateCount := 0
		for _, template := range m.templates {
			if template.Category == category.Name {
				templateCount++
			}
		}

		// Style based on selection
		if isSelected {
			lines = append(lines, lipgloss.NewStyle().
				Background(lipgloss.Color(category.Color)).
				Foreground(lipgloss.Color("#1e1e2e")).
				Bold(true).
				Padding(0, 1).
				Width(width-2).
				Render(fmt.Sprintf("%s %s (%d)", category.Icon, category.Name, templateCount)))
		} else {
			lines = append(lines, lipgloss.NewStyle().
				Foreground(lipgloss.Color(category.Color)).
				Padding(0, 1).
				Width(width-2).
				Render(fmt.Sprintf("%s %s (%d)", category.Icon, category.Name, templateCount)))
		}
	}

	// Add quick selection hints
	quickHints := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6e6a86")).
		Faint(true).
		Render("Quick: 1-5")
	lines = append(lines, "", quickHints)

	return lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6e6a86")).
		Padding(1).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderTemplateList(width int) string {
	if m.templateCategoryCursor >= len(m.templateCategories) {
		return ""
	}

	category := m.templateCategories[m.templateCategoryCursor]
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(category.Color)).
		Render(fmt.Sprintf("%s %s Templates", category.Icon, category.Name))

	lines := []string{title, ""}

	// Get templates for current category
	currentTemplates := m.getCurrentCategoryTemplates()

	if len(currentTemplates) == 0 {
		noTemplates := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6e6a86")).
			Faint(true).
			AlignHorizontal(lipgloss.Center).
			Render("No templates in this category")
		lines = append(lines, noTemplates)
	} else {
		for i, template := range currentTemplates {
			isSelected := i == m.templateCursor && !m.templateFilterMode

			// Template name
			templateName := template.Name
			if len(templateName) > width-8 {
				templateName = templateName[:width-11] + "..."
			}

			// Usage indicator
			usageIndicator := ""
			if template.UsageCount > 0 {
				usageIndicator = fmt.Sprintf(" (%d×)", template.UsageCount)
			}

			if isSelected {
				// Selected template - show full details
				lines = append(lines, lipgloss.NewStyle().
					Background(lipgloss.Color("#313244")).
					Foreground(lipgloss.Color("#cdd6f4")).
					Bold(true).
					Padding(0, 1).
					Width(width-4).
					Render(fmt.Sprintf("📄 %s%s", templateName, usageIndicator)))

				// Show description
				desc := template.Description
				if len(desc) > width-8 {
					desc = desc[:width-11] + "..."
				}
				lines = append(lines, lipgloss.NewStyle().
					Foreground(lipgloss.Color("#a6adc8")).
					Faint(true).
					Padding(0, 2).
					Width(width-6).
					Render(fmt.Sprintf("   %s", desc)))

				// Show first few lines of content preview
				contentLines := strings.Split(template.Content, "\n")
				for j, line := range contentLines {
					if j >= 2 { // Show max 2 lines
						break
					}
					if strings.TrimSpace(line) != "" {
						preview := line
						if len(preview) > width-10 {
							preview = preview[:width-13] + "..."
						}
						lines = append(lines, lipgloss.NewStyle().
							Foreground(lipgloss.Color("#6e6a86")).
							Faint(true).
							Padding(0, 3).
							Width(width-7).
							Render(fmt.Sprintf("   %s", preview)))
					}
				}
			} else {
				// Regular template entry
				lines = append(lines, lipgloss.NewStyle().
					Foreground(lipgloss.Color("#cdd6f4")).
					Padding(0, 1).
					Width(width-4).
					Render(fmt.Sprintf("📄 %s%s", templateName, usageIndicator)))
			}
		}
	}

	// Add mode indicator
	var modeIndicator string
	if m.templateFilterMode {
		modeIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f38ba8")).
			Italic(true).
			Render("🔍 Category Selection")
	} else {
		modeIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a6e3a1")).
			Italic(true).
			Render("📝 Template Selection")
	}
	lines = append(lines, "", modeIndicator)

	return lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6e6a86")).
		Padding(1).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderExportView() string {
	content := fmt.Sprintf(`📤 Export Options

Current Scope: %s
Entries to Export: %d

Export Formats:
  1. Markdown [M] - Structured markdown format
  2. JSON [J]     - Machine-readable JSON format
  3. CSV [C]      - Spreadsheet-compatible CSV format

Selected: %s

Controls:
  1/2/3: Select format
  e: Export to ~/.config/pulse/exports/
  Esc: Cancel

All entries in current scope will be exported.`,
		func() string {
			switch m.scope {
			case scopeToday:
				return "Today"
			case scopeThisWeek:
				return "This Week"
			case scopeThisMonth:
				return "This Month"
			case scopeAll:
				return "All Time"
			default:
				return "Custom Range"
			}
		}(),
		func() int {
			count := 0
			for _, b := range m.blocks {
				count += len(b.entries)
			}
			return count
		}(),
		strings.ToUpper(m.exportFormat),
	)

	return m.modal("📤 Export", content)
}

func overlayCenter(base, modal string) string {
	// naive center overlay using vertical join with blank lines
	baseH := lipgloss.Height(base)
	mh := lipgloss.Height(modal)
	topPad := max(0, (baseH-mh)/3)
	return lipgloss.JoinVertical(lipgloss.Left, strings.Repeat("\n", topPad), lipgloss.PlaceHorizontal(lipgloss.Width(base), lipgloss.Center, modal), "")
}

// parseSince supports human shorthands or YYYY-MM-DD
func parseSince(s string, loc *time.Location) (time.Time, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	now := time.Now().In(loc)
	switch s {
	case "", "today":
		y, m, d := now.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, loc), nil
	case "yesterday":
		y, m, d := now.Add(-24 * time.Hour).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, loc), nil
	case "7d":
		t := now.Add(-7 * 24 * time.Hour)
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, loc), nil
	case "30d":
		t := now.Add(-30 * 24 * time.Hour)
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, loc), nil
	default:
		// YYYY-MM-DD
		if len(s) == 10 && s[4] == '-' && s[7] == '-' {
			t, err := time.ParseInLocation("2006-01-02", s, loc)
			if err != nil {
				return time.Time{}, err
			}
			return t, nil
		}
		// fallback RFC3339
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t.In(loc), nil
		}
		return time.Time{}, errors.New("use: today | yesterday | 7d | 30d | YYYY-MM-DD | RFC3339")
	}
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// ----- export functions -----

func (m Model) exportMarkdown(entries []entry, path string) error {
	var sb strings.Builder
	sb.WriteString("# Pulse Export\n\n")
	sb.WriteString(fmt.Sprintf("Exported on: %s\n\n", m.now.Format("2006-01-02 15:04:05")))

	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("## Entry #%d - %s\n\n", e.id, e.when.Format("2006-01-02 15:04")))
		sb.WriteString(fmt.Sprintf("**Category:** %s\n", strings.ToUpper(e.cat)))
		if e.project != "" {
			sb.WriteString(fmt.Sprintf("**Project:** %s\n", e.project))
		}
		if len(e.tags) > 0 {
			sb.WriteString(fmt.Sprintf("**Tags:** #%s\n", strings.Join(e.tags, " #")))
		}
		sb.WriteString(fmt.Sprintf("\n%s\n\n", e.text))
		sb.WriteString("---\n\n")
	}

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

func (m Model) exportJSON(entries []entry, path string) error {
	type ExportEntry struct {
		ID        int       `json:"id"`
		Timestamp time.Time `json:"timestamp"`
		Category  string    `json:"category"`
		Project   string    `json:"project,omitempty"`
		Tags      []string  `json:"tags,omitempty"`
		Text      string    `json:"text"`
	}

	var exportEntries []ExportEntry
	for _, e := range entries {
		exportEntries = append(exportEntries, ExportEntry{
			ID:        e.id,
			Timestamp: e.when,
			Category:  e.cat,
			Project:   e.project,
			Tags:      e.tags,
			Text:      e.text,
		})
	}

	data := struct {
		ExportDate time.Time     `json:"export_date"`
		Entries    []ExportEntry `json:"entries"`
	}{
		ExportDate: m.now,
		Entries:    exportEntries,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, jsonData, 0o644)
}

func (m Model) exportCSV(entries []entry, path string) error {
	var sb strings.Builder
	sb.WriteString("ID,Timestamp,Category,Project,Tags,Text\n")

	for _, e := range entries {
		tagsStr := strings.Join(e.tags, ";")
		text := strings.ReplaceAll(e.text, "\"", "\"\"")
		sb.WriteString(fmt.Sprintf("%d,\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n",
			e.id, e.when.Format(time.RFC3339), e.cat, e.project, tagsStr, text))
	}

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// ---------- new feature helpers ----------

func (m *Model) addNotification(msg string) {
	m.notifications = append(m.notifications, msg)
	if len(m.notifications) > 5 {
		m.notifications = m.notifications[1:]
	}
	m.status = msg

	// Send desktop notification if enabled
	if m.cfg.Notifications.Enabled {
		_ = notify.Info("Pulse", msg)
	}

	// Announce to screen reader if accessibility mode is enabled
	if m.accessibilityMode {
		m.announceToScreenReader(msg)
	}
}

func (m *Model) addNotificationWithType(msg string, notificationType notify.NotificationType) {
	m.notifications = append(m.notifications, msg)
	if len(m.notifications) > 5 {
		m.notifications = m.notifications[1:]
	}
	m.status = msg

	// Send desktop notification with specific type if enabled
	_ = notify.SendNotification(m.cfg.Notifications, notificationType, "Pulse", msg)

	// Announce to screen reader if accessibility mode is enabled
	if m.accessibilityMode {
		m.announceToScreenReader(msg)
	}
}

func (m *Model) applyTheme(themeIndex int) {
	themes := []struct {
		topBar      lipgloss.Style
		statusBar   lipgloss.Style
		panelTitle  lipgloss.Style
		borderFocus lipgloss.Style
		borderDim   lipgloss.Style
	}{
		// Theme 0: Default (dark blue)
		{
			topBar:      lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")).Bold(true).Padding(0, 1),
			statusBar:   lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")).Background(lipgloss.Color("#313244")).Padding(0, 1),
			panelTitle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#bac2de")).Bold(true),
			borderFocus: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#89B4FA")).Padding(0, 1),
			borderDim:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#585b70")).Padding(0, 1),
		},
		// Theme 1: Dark green
		{
			topBar:      lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Bold(true).Padding(0, 1),
			statusBar:   lipgloss.NewStyle().Foreground(lipgloss.Color("#94e2d5")).Background(lipgloss.Color("#1e1e2e")).Padding(0, 1),
			panelTitle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Bold(true),
			borderFocus: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#a6e3a1")).Padding(0, 1),
			borderDim:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#585b70")).Padding(0, 1),
		},
		// Theme 2: Dark purple
		{
			topBar:      lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7")).Bold(true).Padding(0, 1),
			statusBar:   lipgloss.NewStyle().Foreground(lipgloss.Color("#f5c2e7")).Background(lipgloss.Color("#313244")).Padding(0, 1),
			panelTitle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7")).Bold(true),
			borderFocus: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#cba6f7")).Padding(0, 1),
			borderDim:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#585b70")).Padding(0, 1),
		},
	}

	if themeIndex >= 0 && themeIndex < len(themes) {
		theme := themes[themeIndex]
		m.st.topBar = theme.topBar
		m.st.statusBar = theme.statusBar
		m.st.panelTitle = theme.panelTitle
		m.st.borderFocus = theme.borderFocus
		m.st.borderDim = theme.borderDim
	}
}

func (m Model) renderStatsView() string {
	var total, notes, tasks, meets, timers, bookmarks int
	var todayEntries, weekEntries, monthEntries int
	var projectCounts = make(map[string]int)
	var tagCounts = make(map[string]int)

	// Calculate time boundaries
	now := m.now
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, m.loc)
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday())) // Start of week
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, m.loc)

	for _, b := range m.blocks {
		for _, e := range b.entries {
			total++
			switch strings.ToLower(e.cat) {
			case "note":
				notes++
			case "task":
				tasks++
			case "meeting":
				meets++
			case "timer":
				timers++
			}
			if _, ok := m.bookmarks[e.id]; ok {
				bookmarks++
			}

			// Time-based stats
			if e.when.After(todayStart) {
				todayEntries++
			}
			if e.when.After(weekStart) {
				weekEntries++
			}
			if e.when.After(monthStart) {
				monthEntries++
			}

			// Project and tag stats
			if e.project != "" {
				projectCounts[e.project]++
			}
			for _, tag := range e.tags {
				tagCounts[tag]++
			}
		}
	}

	// Get top projects
	var topProjects []string
	for project, count := range projectCounts {
		topProjects = append(topProjects, fmt.Sprintf("%s (%d)", project, count))
	}
	sort.Slice(topProjects, func(i, j int) bool {
		// Simple sort by count (extract number from string)
		iCount := 0
		jCount := 0
		fmt.Sscanf(topProjects[i], "%*s (%d)", &iCount)
		fmt.Sscanf(topProjects[j], "%*s (%d)", &jCount)
		return iCount > jCount
	})
	if len(topProjects) > 5 {
		topProjects = topProjects[:5]
	}

	// Activity indicators
	todayActivity := "🔴 Low"
	if todayEntries >= 5 {
		todayActivity = "🟢 High"
	} else if todayEntries >= 2 {
		todayActivity = "🟡 Medium"
	}

	content := fmt.Sprintf(`📊 Detailed Statistics - %s

📈 Overview
   Total: %d  |  Today: %d %s  |  This Week: %d  |  This Month: %d

📝 Category Breakdown
   📄 Notes:    %d (%.1f%%)
   ✅ Tasks:    %d (%.1f%%)
   🤝 Meetings: %d (%.1f%%)
   ⏱️  Timers:   %d (%.1f%%)

🔖 Engagement
   Bookmarked: %d entries (%.1f%% of total)

🏗️  Top Projects
   %s

🎯 Current Context
   Scope: %s  |  View: %s  |  Sort: %s (%s)
   Project Filter: %s  |  Category Filter: %s
   Active Filters: %d

📅 Productivity Insights
   Daily Average: %.1f entries/day
   Most Active Day: %s
   Current Streak: %d days

Press Ctrl+I or Esc to close`,
		now.Format("Jan 02, 2006"),
		total, todayEntries, todayActivity, weekEntries, monthEntries,
		notes, float64(notes)/float64(total)*100,
		tasks, float64(tasks)/float64(total)*100,
		meets, float64(meets)/float64(total)*100,
		timers, float64(timers)/float64(total)*100,
		bookmarks, float64(bookmarks)/float64(total)*100,
		func() string {
			if len(topProjects) == 0 {
				return "No projects yet"
			}
			return strings.Join(topProjects, "  •  ")
		}(),
		func() string {
			switch m.scope {
			case scopeToday:
				return "Today"
			case scopeThisWeek:
				return "This Week"
			case scopeThisMonth:
				return "This Month"
			case scopeAll:
				return "All Time"
			default:
				return "Custom"
			}
		}(),
		func() string {
			viewNames := []string{"Timeline", "Cards", "Table", "Kanban"}
			return viewNames[m.viewMode]
		}(),
		m.sortBy,
		func() string {
			if m.sortDirection {
				return "↑"
			}
			return "↓"
		}(),
		func() string {
			if m.filterProj == "" {
				return "None"
			}
			return m.filterProj
		}(),
		func() string {
			if m.filterCat == "" {
				return "None"
			}
			return m.filterCat
		}(),
		len(m.filterTags),
		func() float64 {
			days := 1
			if !monthStart.IsZero() {
				days = int(now.Sub(monthStart).Hours()/24) + 1
			}
			return float64(monthEntries) / float64(days)
		}(),
		func() string {
			// This would require more complex date analysis
			// For now, return today
			return now.Format("Monday")
		}(),
		func() int {
			// Placeholder for streak calculation
			if todayEntries > 0 {
				return 1
			}
			return 0
		}(),
	)

	return m.modal("📊 Statistics", content)
}

func (m Model) renderDashboardView() string {
	var total, notes, tasks, meets, timers, bookmarks int
	var recentEntries []string
	var topProjects []string
	var topTags []string

	// Calculate statistics and collect data
	projectCounts := make(map[string]int)
	tagCounts := make(map[string]int)

	for _, b := range m.blocks {
		for _, e := range b.entries {
			total++
			switch strings.ToLower(e.cat) {
			case "note":
				notes++
			case "task":
				tasks++
			case "meeting":
				meets++
			case "timer":
				timers++
			}
			if _, ok := m.bookmarks[e.id]; ok {
				bookmarks++
			}

			// Collect recent entries (show last 5)
			if len(recentEntries) < 5 {
				preview := e.text
				if len(preview) > 50 {
					preview = preview[:47] + "..."
				}
				recentEntries = append(recentEntries, fmt.Sprintf("• #%d %s: %s", e.id, strings.ToUpper(e.cat), preview))
			}

			// Count projects and tags
			if e.project != "" {
				projectCounts[e.project]++
			}
			for _, tag := range e.tags {
				tagCounts[tag]++
			}
		}
	}

	// Get top projects (max 5)
	for project, count := range projectCounts {
		topProjects = append(topProjects, fmt.Sprintf("%s (%d)", project, count))
		if len(topProjects) >= 5 {
			break
		}
	}

	// Get top tags (max 5)
	for tag, count := range tagCounts {
		topTags = append(topTags, fmt.Sprintf("#%s (%d)", tag, count))
		if len(topTags) >= 5 {
			break
		}
	}

	// Build dashboard content
	content := fmt.Sprintf(`📊 Pulse Dashboard - %s

📈 Overview
   Total Entries: %d
   Notes: %d  •  Tasks: %d  •  Meetings: %d  •  Timers: %d
   Bookmarked: %d

🔥 Recent Activity
   %s

🏗️  Top Projects
   %s

🏷️  Top Tags
   %s

⚡ Quick Actions
   • Press 'n' to create new entry
   • Press '/' to search entries
   • Press 'p' to filter by project
   • Press 'c' to filter by category
   • Press '#' to filter by tags

Press Ctrl+W or Esc to close dashboard`,
		m.now.In(m.loc).Format("2006-01-02 03:04 PM"),
		total, notes, tasks, meets, timers, bookmarks,
		func() string {
			if len(recentEntries) == 0 {
				return "No recent entries"
			}
			return strings.Join(recentEntries, "\n   ")
		}(),
		func() string {
			if len(topProjects) == 0 {
				return "No projects yet"
			}
			return strings.Join(topProjects, "\n   ")
		}(),
		func() string {
			if len(topTags) == 0 {
				return "No tags yet"
			}
			return strings.Join(topTags, "\n   ")
		}(),
	)

	return m.modal("📊 Dashboard", content)
}

// DefaultTheme provides simple styling for CLI commands
var DefaultTheme = struct {
	Title   lipgloss.Style
	Value   lipgloss.Style
	Success lipgloss.Style
}{
	Title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89B4FA")),
	Value:   lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")),
	Success: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1")),
}

func (m Model) helpView() string {
	content := `🚀 ` + version.GetVersionInfo() + ` - Comprehensive Help

📍 BASIC NAVIGATION
  j/k, ↑/↓        Move cursor up/down
  Enter           Open thread view / Select items
  Tab, ←/→        Switch focus between panes
  q, Ctrl+C        Quit application

📝 ENTRY MANAGEMENT
  n               Create new entry (full form)
  r               Reply to selected entry
  e               Edit selected entry
  d               Delete selected entry
  D               Duplicate selected entry
  x               Export thread to markdown
  Alt+N           Quick create note
  Alt+T           Quick create task
  Alt+M           Quick create meeting

🔍 SEARCH & FILTERS
  /               Live search mode
  F               Advanced search (multi-field)
  p               Project picker
  c               Category picker
  #               Tag picker
  s               Since date picker
  Space           (Sidebar: select/clear all)

📊 VIEWS & MODES
  v               Cycle view modes (Timeline/Cards/Table/Kanban)
  t               Cycle scopes (Today/Week/Month/All)
  1-6, 0          Quick scope selection (1=Today, 0=All)
  C               Calendar view with date navigation and entry browsing
  T               Templates selection
  E               Export options
  R               Time tracking reports
  J               Project summaries
  A               Tag analytics
  Ctrl+W          Dashboard view
  Ctrl+I          Statistics view
  Ctrl+R          Time tracking reports (alternative)
  Ctrl+P          Project summaries (alternative)
  Ctrl+A          Tag analytics (alternative)
  ?               Toggle this help

🎯 PRODUCTIVITY FEATURES
  P               Toggle Pomodoro timer (25min work / 5min break)
  Ctrl+B          Toggle sidebar
  Ctrl+F          Toggle focus mode
  Ctrl+G          Go to today
  Ctrl+D          Toggle bookmark on entry
  a               Toggle archive mode

⚙️ SORTING & ORGANIZATION
  o               Cycle sort by (Date/Category/Project/Priority)
  O               Toggle sort direction (Asc/Desc)
  g               Group by options (when implemented)

🎨 CUSTOMIZATION
  Ctrl+T          Cycle themes (3 color themes)
  Ctrl+L          Clear all filters
  Ctrl+R          Refresh data

📜 TIMELINE SCROLLING
  PgUp/PgDn       Scroll timeline up/down by page
  Home/End        Jump to top/bottom of timeline
  Mouse wheel     Scroll timeline (when timeline focused)

📜 QUICK ACTIONS SCROLLING
  [ and ]         Navigate quick actions pages
  Ctrl+←/→        Previous/Next quick actions page
  Ctrl+[ and ]    Jump to first/last quick actions page
  Mouse click     Click left/right half of quick actions bar

📋 QUICK REFERENCE
  Scopes: 1=Today, 2=Yesterday, 3=This Week, 4=Last Week, 5=This Month, 6=Last Month, 0=All
  Views: Timeline → Cards → Table → Kanban
  Sort: Date → Category → Project → Priority
  Analytics: R (Time Reports), J (Projects), A (Tags) or Ctrl+R, Ctrl+P, Ctrl+A
  Templates: Meeting Notes, Daily Standup, Brainstorm, Bug Report, Project Update

💡 PRO TIPS
  • Use Tab to navigate between form fields
  • Press Space in sidebar to select all items in current section
  • Use number keys 1-6 and 0 for quick scope switching
  • Timeline supports page scrolling with PgUp/PgDn and Home/End keys
  • Quick actions bar supports multiple scrolling methods (see help section above)
  • Mouse wheel works on both timeline (when focused) and help views
  • Pomodoro timer shows work/break status in top bar
  • Export supports Markdown, JSON, and CSV formats
  • Advanced search supports text, project, category, and tag filters
  • Analytics: Use R/J/A for quick access or Ctrl+R/P/A for alternative shortcuts
  • In analytics views, press Enter to filter by selected item, Esc to close

📊 ANALYTICS CONTROLS
  • v/V: Cycle view modes (Table/Chart/Summary/Details)
  • v (Time Reports): Daily/Weekly/Monthly/Category views
  • o: Sort projects/tags by different criteria
  • t: Change time scope (Today/Week/Month/All)
  • ↑/↓: Navigate through data items
  • Charts: ASCII bar charts for time distribution

🔧 FILTER EXAMPLES
  Date: today, yesterday, 7d, 30d, YYYY-MM-DD
  Tags: #urgent, #bug, #feature
  Projects: project-name, client-name
  Categories: note, task, meeting, timer

📈 ANALYTICS VIEW MODES
  • Table View: Structured data in tabular format
  • Chart View: ASCII bar charts and visual indicators
  • Summary View: Key metrics and top items with progress bars
  • Details View: In-depth breakdown with navigation

⏱️ TIME REPORT VIEWS
  • Daily: Individual days with time and percentage breakdown
  • Weekly: Week-over-week summaries with daily averages
  • Monthly: Monthly aggregates with daily averages
  • Category: Time distribution across categories and projects

📖 HELP NAVIGATION
  ↑/↓, j/k       Scroll help content
  PgUp/PgDn       Scroll faster
  Home/g, End/G   Jump to top/bottom
  Mouse wheel     Scroll (if supported)

Press Esc, ?, or any other key to close help • Happy logging! 🎉`

	// Split content into lines for scrolling
	lines := strings.Split(content, "\n")

	// Handle "go to end" signal
	if m.helpScrollOffset == -1 {
		m.helpScrollOffset = max(0, len(lines)-20) // Show last ~20 lines
	}

	// Calculate how many lines can fit in the modal
	maxVisibleLines := 20 // Approximate modal height

	// Ensure scroll offset is within bounds
	maxScroll := max(0, len(lines)-maxVisibleLines)
	if m.helpScrollOffset > maxScroll {
		m.helpScrollOffset = maxScroll
	}

	// Extract visible portion of content
	start := max(0, m.helpScrollOffset)
	end := min(len(lines), start+maxVisibleLines)
	visibleContent := strings.Join(lines[start:end], "\n")

	// Add scroll indicator if content is longer than visible area
	if len(lines) > maxVisibleLines {
		scrollIndicator := fmt.Sprintf("Line %d-%d of %d", start+1, end, len(lines))
		visibleContent += "\n\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a6adc8")).
			Faint(true).
			AlignHorizontal(lipgloss.Right).
			Render(scrollIndicator)
	}

	return m.modal("📖 Comprehensive Help (↑/↓ to scroll)", visibleContent)
}

func (m Model) renderFocusView() string {
	content := `🎯 Focus Mode

Focus Mode provides a distraction-free environment to help you concentrate on your current tasks.

📋 FEATURES:
• Clean, minimal interface
• Reduced visual distractions
• Quick access to essential functions
• Enhanced productivity

⚡ QUICK ACTIONS:
• Ctrl+F: Exit Focus Mode
• Ctrl+T: Change theme
• Ctrl+G: Jump to today
• Ctrl+D: Bookmark current entry
• n: Create new entry
• r: Reply to selected entry
• e: Edit selected entry

🎨 CUSTOMIZATION:
• Themes: 9 beautiful color schemes
• Font styles: Multiple text styles
• Layout: Optimized for focus work

📊 STATUS:
• Pomodoro Timer: ` + func() string {
		if m.pomodoroActive {
			minutes := int(m.pomodoroTimeLeft.Minutes())
			seconds := int(m.pomodoroTimeLeft.Seconds()) % 60
			sessionType := "Work"
			if m.pomodoroSession == 1 {
				sessionType = "Break"
			}
			return fmt.Sprintf("🍅 %s %02d:%02d", sessionType, minutes, seconds)
		}
		return "🍅 Inactive (Press P to start)"
	}() + `
• Current Scope: ` + func() string {
		switch m.scope {
		case scopeToday:
			return "Today"
		case scopeThisWeek:
			return "This Week"
		case scopeThisMonth:
			return "This Month"
		case scopeAll:
			return "All Time"
		default:
			return "Custom"
		}
	}() + `
• Entries: ` + fmt.Sprintf("%d", func() int {
		count := 0
		for _, block := range m.blocks {
			count += len(block.entries)
		}
		return count
	}()) + `

💡 TIPS:
• Use Focus Mode when you need to concentrate
• Combine with Pomodoro Timer for productivity
• Press ? to see full help without leaving Focus Mode
• All keyboard shortcuts work normally in Focus Mode

Press Esc or Ctrl+F to exit Focus Mode and return to normal view.`

	return m.modal("🎯 Focus Mode", content)
}

func (m Model) renderPickerModal() string {
	var title, content string
	switch m.activePicker {
	case pickProjects:
		title = "Select Project"
		if len(m.projects) == 0 {
			content = "No projects found"
		} else {
			lines := make([]string, len(m.projects))
			for i, item := range m.projects {
				prefix := "  "
				if i == m.pickerCursor {
					prefix = "➤ "
				}
				selected := ""
				if m.filterProj == item.name {
					selected = " [x]"
				}
				lines[i] = fmt.Sprintf("%s%s (%d)%s", prefix, item.name, item.count, selected)
			}
			content = strings.Join(lines, "\n")
		}
	case pickCategories:
		title = "Select Category"
		if len(m.categories) == 0 {
			content = "No categories found"
		} else {
			lines := make([]string, len(m.categories))
			for i, item := range m.categories {
				prefix := "  "
				if i == m.pickerCursor {
					prefix = "➤ "
				}
				selected := ""
				if m.filterCat == strings.ToLower(item.name) {
					selected = " [x]"
				}
				lines[i] = fmt.Sprintf("%s%s (%d)%s", prefix, item.name, item.count, selected)
			}
			content = strings.Join(lines, "\n")
		}
	case pickTags:
		title = "Select Tags"
		if len(m.tags) == 0 {
			content = "No tags found"
		} else {
			lines := make([]string, len(m.tags))
			for i, item := range m.tags {
				prefix := "  "
				if i == m.pickerCursor {
					prefix = "➤ "
				}
				selected := ""
				if _, ok := m.filterTags[item.name]; ok {
					selected = " [x]"
				}
				lines[i] = fmt.Sprintf("%s#%s (%d)%s", prefix, item.name, item.count, selected)
			}
			content = strings.Join(lines, "\n")
		}
	}

	return m.modal(title, content)
}

// ----- Analytics update functions -----

func (m Model) updateTimeReports(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "t":
		// Cycle through scopes: today -> this week -> this month -> all
		switch m.timeReportScope {
		case scopeToday:
			m.timeReportScope = scopeThisWeek
			m.addNotification("Time Reports: This Week")
		case scopeThisWeek:
			m.timeReportScope = scopeThisMonth
			m.addNotification("Time Reports: This Month")
		case scopeThisMonth:
			m.timeReportScope = scopeAll
			m.addNotification("Time Reports: All Time")
		default:
			m.timeReportScope = scopeToday
			m.addNotification("Time Reports: Today")
		}
		return m, m.loadTimeReportsCmd()
	case "v":
		// Cycle through view modes: daily -> weekly -> monthly -> category
		m.timeReportView = (m.timeReportView + 1) % 4
		viewNames := []string{"Daily View", "Weekly View", "Monthly View", "Category View"}
		m.addNotification(fmt.Sprintf("Time Report View: %s", viewNames[m.timeReportView]))
		return m, nil
	case "V":
		// Cycle through analytics display modes
		m.analyticsViewMode = (m.analyticsViewMode + 1) % 4
		viewNames := []string{"Table View", "Chart View", "Summary View", "Details View"}
		m.addNotification(fmt.Sprintf("Analytics Mode: %s", viewNames[m.analyticsViewMode]))
		return m, nil
	case "up", "k":
		if m.analyticsCursor > 0 {
			m.analyticsCursor--
		}
		return m, nil
	case "down", "j":
		if m.analyticsCursor < len(m.timeReportData)-1 {
			m.analyticsCursor++
		}
		return m, nil
	case "f":
		// Filter functionality (placeholder for now)
		m.addNotification("Filter: Enter filter text (feature coming soon)")
		return m, nil
	}
	return m, nil
}

func (m Model) updateProjectSummary(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "r":
		return m, m.loadProjectSummaryCmd()
	case "v":
		// Cycle through analytics display modes
		m.analyticsViewMode = (m.analyticsViewMode + 1) % 4
		viewNames := []string{"Table View", "Chart View", "Summary View", "Details View"}
		m.addNotification(fmt.Sprintf("Analytics Mode: %s", viewNames[m.analyticsViewMode]))
		return m, nil
	case "o":
		// Cycle through sort options: total_time -> entry_count -> last_active -> name
		m.projectSortBy = (m.projectSortBy + 1) % 4
		sortNames := []string{"Sort: Total Time", "Sort: Entry Count", "Sort: Last Active", "Sort: Name"}
		m.addNotification(sortNames[m.projectSortBy])
		// Re-sort the data
		m.sortProjectSummaryData()
		return m, nil
	case "up", "k":
		if m.analyticsCursor > 0 {
			m.analyticsCursor--
		}
		return m, nil
	case "down", "j":
		if m.analyticsCursor < len(m.projectSummaryData)-1 {
			m.analyticsCursor++
		}
		return m, nil
	case "enter":
		if len(m.projectSummaryData) > 0 && m.analyticsCursor < len(m.projectSummaryData) {
			project := m.projectSummaryData[m.analyticsCursor].Project
			m.filterProj = project
			m.mode = modeNormal
			m.addNotification(fmt.Sprintf("Filtering by project: %s", project))
			return m, m.loadTimelineCmd()
		}
		return m, nil
	case "f":
		// Filter functionality
		m.addNotification("Filter: Enter filter text (feature coming soon)")
		return m, nil
	}
	return m, nil
}

func (m Model) updateTagAnalytics(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "r":
		return m, m.loadTagAnalyticsCmd()
	case "v":
		// Cycle through analytics display modes
		m.analyticsViewMode = (m.analyticsViewMode + 1) % 4
		viewNames := []string{"Table View", "Chart View", "Summary View", "Details View"}
		m.addNotification(fmt.Sprintf("Analytics Mode: %s", viewNames[m.analyticsViewMode]))
		return m, nil
	case "o":
		// Cycle through sort options: usage_count -> total_time -> last_used -> name
		m.tagSortBy = (m.tagSortBy + 1) % 4
		sortNames := []string{"Sort: Usage Count", "Sort: Total Time", "Sort: Last Used", "Sort: Name"}
		m.addNotification(sortNames[m.tagSortBy])
		// Re-sort the data
		m.sortTagAnalyticsData()
		return m, nil
	case "up", "k":
		if m.analyticsCursor > 0 {
			m.analyticsCursor--
		}
		return m, nil
	case "down", "j":
		if m.analyticsCursor < len(m.tagAnalyticsData)-1 {
			m.analyticsCursor++
		}
		return m, nil
	case "enter":
		if len(m.tagAnalyticsData) > 0 && m.analyticsCursor < len(m.tagAnalyticsData) {
			tag := m.tagAnalyticsData[m.analyticsCursor].Tag
			m.filterTags = map[string]struct{}{tag: {}}
			m.mode = modeNormal
			m.addNotification(fmt.Sprintf("Filtering by tag: #%s", tag))
			return m, tea.Batch(m.loadTimelineCmd(), m.loadFacetsCmd())
		}
		return m, nil
	case "f":
		// Filter functionality
		m.addNotification("Filter: Enter filter text (feature coming soon)")
		return m, nil
	}
	return m, nil
}

// ----- Analytics helper functions -----

func (m *Model) sortProjectSummaryData() {
	switch m.projectSortBy {
	case 0: // total_time
		sort.Slice(m.projectSummaryData, func(i, j int) bool {
			return m.projectSummaryData[i].TotalTime > m.projectSummaryData[j].TotalTime
		})
	case 1: // entry_count
		sort.Slice(m.projectSummaryData, func(i, j int) bool {
			return m.projectSummaryData[i].EntryCount > m.projectSummaryData[j].EntryCount
		})
	case 2: // last_active
		sort.Slice(m.projectSummaryData, func(i, j int) bool {
			return m.projectSummaryData[i].LastActive.After(m.projectSummaryData[j].LastActive)
		})
	case 3: // name
		sort.Slice(m.projectSummaryData, func(i, j int) bool {
			return m.projectSummaryData[i].Project < m.projectSummaryData[j].Project
		})
	}
}

func (m *Model) sortTagAnalyticsData() {
	switch m.tagSortBy {
	case 0: // usage_count
		sort.Slice(m.tagAnalyticsData, func(i, j int) bool {
			return m.tagAnalyticsData[i].UsageCount > m.tagAnalyticsData[j].UsageCount
		})
	case 1: // total_time
		sort.Slice(m.tagAnalyticsData, func(i, j int) bool {
			return m.tagAnalyticsData[i].TotalTime > m.tagAnalyticsData[j].TotalTime
		})
	case 2: // last_used
		sort.Slice(m.tagAnalyticsData, func(i, j int) bool {
			return m.tagAnalyticsData[i].LastUsed.After(m.tagAnalyticsData[j].LastUsed)
		})
	case 3: // name
		sort.Slice(m.tagAnalyticsData, func(i, j int) bool {
			return m.tagAnalyticsData[i].Tag < m.tagAnalyticsData[j].Tag
		})
	}
}

// ----- Analytics view rendering functions -----

func (m Model) renderTimeReportsView() string {
	switch m.analyticsViewMode {
	case 0:
		return m.renderTimeReportsTableView()
	case 1:
		return m.renderTimeReportsChartView()
	case 2:
		return m.renderTimeReportsSummaryView()
	case 3:
		return m.renderTimeReportsDetailsView()
	default:
		return m.renderTimeReportsTableView()
	}
}

func (m Model) renderTimeReportsTableView() string {
	var totalTime time.Duration
	var entryCount int
	categoryTime := make(map[string]time.Duration)
	projectTime := make(map[string]time.Duration)
	dailyTime := make(map[string]time.Duration)

	// Calculate statistics from time report data
	for _, report := range m.timeReportData {
		totalTime += report.TotalTime
		entryCount += report.EntryCount

		for cat, duration := range report.ByCategory {
			categoryTime[cat] += duration
		}

		for proj, duration := range report.ByProject {
			projectTime[proj] += duration
		}

		dateKey := report.Date.Format("2006-01-02")
		dailyTime[dateKey] = report.TotalTime
	}

	// Calculate daily average
	var daysCount int
	switch m.timeReportScope {
	case scopeToday:
		daysCount = 1
	case scopeThisWeek:
		daysCount = 7
	case scopeThisMonth:
		daysCount = 30
	case scopeAll:
		daysCount = max(1, len(dailyTime))
	}
	dailyAvg := totalTime / time.Duration(daysCount)

	var content strings.Builder
	content.WriteString(fmt.Sprintf("⏱️ Time Tracking Report - %s (Table View)\n\n", m.getTimeReportScopeLabel()))

	// Overview section
	content.WriteString("📊 Overview\n")
	content.WriteString(fmt.Sprintf("   Total Time: %s  •  Entries: %d  •  Daily Avg: %s  •  Active Days: %d\n\n",
		formatDuration(totalTime), entryCount, formatDuration(dailyAvg), len(dailyTime)))

	// View data based on current time report view
	switch m.timeReportView {
	case 0: // Daily view
		content.WriteString(m.renderDailyTimeTable(dailyTime))
	case 1: // Weekly view
		content.WriteString(m.renderWeeklyTimeTable(dailyTime))
	case 2: // Monthly view
		content.WriteString(m.renderMonthlyTimeTable(dailyTime))
	case 3: // Category view
		content.WriteString(m.renderCategoryTimeTable(categoryTime, projectTime))
	}

	// Controls
	content.WriteString("\n⌨️  Controls\n")
	content.WriteString("   t: Scope  •  v: View mode  •  V: Display mode  •  ↑/↓: Navigate  •  Esc: Close")

	return m.modal("⏱️ Time Reports", content.String())
}

func (m Model) renderTimeReportsChartView() string {
	var totalTime time.Duration
	categoryTime := make(map[string]time.Duration)
	dailyTime := make(map[string]time.Duration)

	// Calculate statistics
	for _, report := range m.timeReportData {
		totalTime += report.TotalTime
		for cat, duration := range report.ByCategory {
			categoryTime[cat] += duration
		}
		dateKey := report.Date.Format("2006-01-02")
		dailyTime[dateKey] = report.TotalTime
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("⏱️ Time Tracking Report - %s (Chart View)\n\n", m.getTimeReportScopeLabel()))

	if len(dailyTime) == 0 {
		content.WriteString("No data available for chart visualization.")
	} else {
		// ASCII bar chart for time distribution
		content.WriteString("📊 Time Distribution Chart\n\n")
		content.WriteString(m.renderTimeChart(dailyTime, categoryTime))
	}

	// Controls
	content.WriteString("\n⌨️  Controls\n")
	content.WriteString("   t: Scope  •  v: View mode  •  V: Display mode  •  Esc: Close")

	return m.modal("⏱️ Time Reports", content.String())
}

func (m Model) renderTimeReportsSummaryView() string {
	var totalTime time.Duration
	var entryCount int
	categoryTime := make(map[string]time.Duration)
	projectTime := make(map[string]time.Duration)
	dailyTime := make(map[string]time.Duration)

	// Calculate statistics
	for _, report := range m.timeReportData {
		totalTime += report.TotalTime
		entryCount += report.EntryCount
		for cat, duration := range report.ByCategory {
			categoryTime[cat] += duration
		}
		for proj, duration := range report.ByProject {
			projectTime[proj] += duration
		}
		dateKey := report.Date.Format("2006-01-02")
		dailyTime[dateKey] = report.TotalTime
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("⏱️ Time Tracking Report - %s (Summary View)\n\n", m.getTimeReportScopeLabel()))

	// Summary statistics with visual indicators
	content.WriteString("📈 Summary Statistics\n")
	content.WriteString(fmt.Sprintf("   Total Time: %s (%d entries)\n", formatDuration(totalTime), entryCount))

	// Productivity indicator
	avgDaily := totalTime / time.Duration(max(1, len(dailyTime)))
	prodIndicator := "🟢 High"
	if avgDaily < 2*time.Hour {
		prodIndicator = "🟡 Medium"
	}
	if avgDaily < 1*time.Hour {
		prodIndicator = "🔴 Low"
	}
	content.WriteString(fmt.Sprintf("   Daily Average: %s %s\n", formatDuration(avgDaily), prodIndicator))
	content.WriteString(fmt.Sprintf("   Active Days: %d\n", len(dailyTime)))
	content.WriteString("\n")

	// Top categories with visual bars
	content.WriteString("📝 Top Categories\n")
	for cat, duration := range categoryTime {
		if duration <= 0 {
			continue
		}
		percentage := float64(duration) / float64(totalTime) * 100
		bar := m.renderProgressBar(percentage, 20)
		content.WriteString(fmt.Sprintf("   %-12s %s %s (%.1f%%)\n",
			strings.ToUpper(cat), bar, formatDuration(duration), percentage))
	}
	content.WriteString("\n")

	// Top projects
	content.WriteString("🏗️  Top Projects\n")
	for proj, duration := range projectTime {
		if duration <= 0 || proj == "" {
			continue
		}
		percentage := float64(duration) / float64(totalTime) * 100
		bar := m.renderProgressBar(percentage, 20)
		content.WriteString(fmt.Sprintf("   %-12s %s %s (%.1f%%)\n",
			proj, bar, formatDuration(duration), percentage))
	}

	// Controls
	content.WriteString("\n⌨️  Controls\n")
	content.WriteString("   t: Scope  •  v: View mode  •  V: Display mode  •  Esc: Close")

	return m.modal("⏱️ Time Reports", content.String())
}

func (m Model) renderTimeReportsDetailsView() string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("⏱️ Time Tracking Report - %s (Details View)\n\n", m.getTimeReportScopeLabel()))

	if len(m.timeReportData) == 0 {
		content.WriteString("No detailed data available for this time period.")
	} else {
		content.WriteString("📅 Daily Breakdown\n\n")

		for i, report := range m.timeReportData {
			cursor := " "
			if i == m.analyticsCursor {
				cursor = "➤ "
			}

			content.WriteString(fmt.Sprintf("%s%s\n", cursor, report.Date.Format("2006-01-02")))
			content.WriteString(fmt.Sprintf("   Total: %s  •  Entries: %d\n",
				formatDuration(report.TotalTime), report.EntryCount))

			// Category breakdown for this day
			if len(report.ByCategory) > 0 {
				var cats []string
				for cat, duration := range report.ByCategory {
					if duration > 0 {
						cats = append(cats, fmt.Sprintf("%s:%s", cat, formatDuration(duration)))
					}
				}
				sort.Strings(cats)
				content.WriteString(fmt.Sprintf("   Categories: %s\n", strings.Join(cats, "  ")))
			}
			content.WriteString("\n")
		}
	}

	// Controls
	content.WriteString("⌨️  Controls\n")
	content.WriteString("   t: Scope  •  v: View mode  •  V: Display mode  •  ↑/↓: Navigate  •  Esc: Close")

	return m.modal("⏱️ Time Reports", content.String())
}

func (m Model) renderProjectSummaryView() string {
	if len(m.projectSummaryData) == 0 {
		return m.modal("🏗️ Project Summary", "No project data available. Start tracking time with projects!")
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("🏗️ Project Summary Overview - %s\n\n", m.now.Format("Jan 02, 2006")))

	// Summary statistics
	var totalProjectsTime time.Duration
	var totalEntries int
	activeProjects := 0

	for _, summary := range m.projectSummaryData {
		totalProjectsTime += summary.TotalTime
		totalEntries += summary.EntryCount
		if summary.TotalTime > 0 {
			activeProjects++
		}
	}

	content.WriteString(fmt.Sprintf("📊 Overall Statistics\n"))
	content.WriteString(fmt.Sprintf("   Total Projects: %d active (%d total)\n", activeProjects, len(m.projectSummaryData)))
	content.WriteString(fmt.Sprintf("   Total Time: %s\n", formatDuration(totalProjectsTime)))
	content.WriteString(fmt.Sprintf("   Total Entries: %d\n", totalEntries))
	content.WriteString(fmt.Sprintf("   Average per Project: %s\n\n", formatDuration(totalProjectsTime/time.Duration(max(1, activeProjects)))))

	// Project details
	content.WriteString("📋 Project Details\n\n")

	for i, summary := range m.projectSummaryData {
		cursor := " "
		if i == m.analyticsCursor {
			cursor = "➤ "
		}

		trendIcon := "•"
		switch summary.Trend {
		case "up":
			trendIcon = "📈"
		case "down":
			trendIcon = "📉"
		case "stable":
			trendIcon = "➡️"
		}

		content.WriteString(fmt.Sprintf("%s%s %s\n", cursor, trendIcon, summary.Project))
		content.WriteString(fmt.Sprintf("   Time: %s  •  Entries: %d  •  Last: %s\n",
			formatDuration(summary.TotalTime),
			summary.EntryCount,
			summary.LastActive.Format("Jan 02")))

		// Category breakdown
		if len(summary.Categories) > 0 {
			var catBreakdown []string
			for cat, duration := range summary.Categories {
				if duration > 0 {
					catBreakdown = append(catBreakdown, fmt.Sprintf("%s: %s", cat, formatDuration(duration)))
				}
			}
			sort.Slice(catBreakdown, func(i, j int) bool { return len(catBreakdown[i]) > len(catBreakdown[j]) })
			if len(catBreakdown) > 0 {
				content.WriteString(fmt.Sprintf("   Categories: %s\n", strings.Join(catBreakdown[:min(3, len(catBreakdown))], ", ")))
			}
		}
		content.WriteString("\n")
	}

	content.WriteString("⌨️  Controls\n")
	content.WriteString("   ↑/↓: Navigate  •  Enter: Filter by project  •  r: Refresh  •  Esc: Close")

	return m.modal("🏗️ Project Summary", content.String())
}

func (m Model) renderTagAnalyticsView() string {
	if len(m.tagAnalyticsData) == 0 {
		return m.modal("🏷️ Tag Analytics", "No tag data available. Start using tags in your entries!")
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("🏷️ Tag Analytics & Insights - %s\n\n", m.now.Format("Jan 02, 2006")))

	// Summary statistics
	var totalUsages int
	var totalTaggedTime time.Duration
	activeTags := 0

	for _, analytics := range m.tagAnalyticsData {
		totalUsages += analytics.UsageCount
		totalTaggedTime += analytics.TotalTime
		if analytics.UsageCount > 0 {
			activeTags++
		}
	}

	content.WriteString("📊 Overall Tag Statistics\n")
	content.WriteString(fmt.Sprintf("   Active Tags: %d (%d total)\n", activeTags, len(m.tagAnalyticsData)))
	content.WriteString(fmt.Sprintf("   Total Tag Usage: %d\n", totalUsages))
	content.WriteString(fmt.Sprintf("   Total Tagged Time: %s\n", formatDuration(totalTaggedTime)))
	content.WriteString(fmt.Sprintf("   Average Usage per Tag: %.1f\n\n", float64(totalUsages)/float64(max(1, activeTags))))

	// Tag details
	content.WriteString("🏷️ Tag Details\n\n")

	for i, analytics := range m.tagAnalyticsData {
		if analytics.UsageCount == 0 {
			continue
		}

		cursor := " "
		if i == m.analyticsCursor {
			cursor = "➤ "
		}

		trendIcon := "•"
		switch analytics.Trend {
		case "up":
			trendIcon = "📈"
		case "down":
			trendIcon = "📉"
		case "stable":
			trendIcon = "➡️"
		}

		content.WriteString(fmt.Sprintf("%s%s #%s\n", cursor, trendIcon, analytics.Tag))
		content.WriteString(fmt.Sprintf("   Used: %d times  •  Time: %s  •  Last: %s\n",
			analytics.UsageCount,
			formatDuration(analytics.TotalTime),
			analytics.LastUsed.Format("Jan 02")))

		// Project associations
		if len(analytics.Projects) > 0 {
			content.WriteString(fmt.Sprintf("   Projects: %s\n", strings.Join(analytics.Projects[:min(3, len(analytics.Projects))], ", ")))
		}

		// Category associations
		if len(analytics.Categories) > 0 {
			content.WriteString(fmt.Sprintf("   Categories: %s\n", strings.Join(analytics.Categories[:min(3, len(analytics.Categories))], ", ")))
		}
		content.WriteString("\n")
	}

	content.WriteString("⌨️  Controls\n")
	content.WriteString("   ↑/↓: Navigate  •  Enter: Filter by tag  •  r: Refresh  •  Esc: Close")

	return m.modal("🏷️ Tag Analytics", content.String())
}

// Helper function to format duration
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "0m"
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// ----- Analytics view helper functions -----

func (m Model) getTimeReportScopeLabel() string {
	switch m.timeReportScope {
	case scopeToday:
		return "Today"
	case scopeThisWeek:
		return "This Week"
	case scopeThisMonth:
		return "This Month"
	case scopeAll:
		return "All Time"
	default:
		return "Custom"
	}
}

func (m Model) renderProgressBar(percentage float64, width int) string {
	if percentage <= 0 {
		return strings.Repeat("░", width)
	}

	filled := int(percentage / 100 * float64(width))
	if filled > width {
		filled = width
	}

	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func (m Model) renderDailyTimeTable(dailyTime map[string]time.Duration) string {
	if len(dailyTime) == 0 {
		return "No daily data available.\n"
	}

	var content strings.Builder
	content.WriteString("📅 Daily Breakdown\n\n")

	// Get sorted dates
	var dates []string
	for date := range dailyTime {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	// Show table header
	content.WriteString("   Date        • Time    • % of Total\n")
	content.WriteString("   ─────────────────────────────\n")

	// Calculate total for percentage calculation
	var total time.Duration
	for _, duration := range dailyTime {
		total += duration
	}

	// Show daily entries (limit to last 14 days for readability)
	start := max(0, len(dates)-14)
	for i := start; i < len(dates); i++ {
		date := dates[i]
		duration := dailyTime[date]
		percentage := float64(duration) / float64(total) * 100
		content.WriteString(fmt.Sprintf("   %-10s  • %-7s • %.1f%%\n",
			date, formatDuration(duration), percentage))
	}

	return content.String()
}

func (m Model) renderWeeklyTimeTable(dailyTime map[string]time.Duration) string {
	if len(dailyTime) == 0 {
		return "No weekly data available.\n"
	}

	var content strings.Builder
	content.WriteString("📊 Weekly Summary\n\n")

	// Group by week
	weeklyData := make(map[string]time.Duration)
	for dateStr, duration := range dailyTime {
		date, _ := time.Parse("2006-01-02", dateStr)
		year, week := date.ISOWeek()
		weekKey := fmt.Sprintf("%d-W%02d", year, week)
		weeklyData[weekKey] += duration
	}

	// Get sorted weeks
	var weeks []string
	for week := range weeklyData {
		weeks = append(weeks, week)
	}
	sort.Strings(weeks)

	// Calculate total
	var total time.Duration
	for _, duration := range weeklyData {
		total += duration
	}

	content.WriteString("   Week     • Time    • Daily Avg\n")
	content.WriteString("   ────────────────────────────\n")

	for _, week := range weeks {
		duration := weeklyData[week]
		dailyAvg := duration / 7 // Approximate
		content.WriteString(fmt.Sprintf("   %-7s  • %-7s • %s\n",
			week, formatDuration(duration), formatDuration(dailyAvg)))
	}

	return content.String()
}

func (m Model) renderMonthlyTimeTable(dailyTime map[string]time.Duration) string {
	if len(dailyTime) == 0 {
		return "No monthly data available.\n"
	}

	var content strings.Builder
	content.WriteString("📅 Monthly Summary\n\n")

	// Group by month
	monthlyData := make(map[string]time.Duration)
	for dateStr, duration := range dailyTime {
		date, _ := time.Parse("2006-01-02", dateStr)
		monthKey := date.Format("2006-01")
		monthlyData[monthKey] += duration
	}

	// Get sorted months
	var months []string
	for month := range monthlyData {
		months = append(months, month)
	}
	sort.Strings(months)

	// Calculate total
	var total time.Duration
	for _, duration := range monthlyData {
		total += duration
	}

	content.WriteString("   Month    • Time    • Daily Avg • Entries\n")
	content.WriteString("   ──────────────────────────────────\n")

	for _, month := range months {
		duration := monthlyData[month]
		// Count days in month for average
		date, _ := time.Parse("2006-01", month)
		daysInMonth := time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		dailyAvg := duration / time.Duration(daysInMonth)
		content.WriteString(fmt.Sprintf("   %-7s  • %-7s • %-8s • %d\n",
			month, formatDuration(duration), formatDuration(dailyAvg), daysInMonth))
	}

	return content.String()
}

func (m Model) renderCategoryTimeTable(categoryTime map[string]time.Duration, projectTime map[string]time.Duration) string {
	if len(categoryTime) == 0 {
		return "No category data available.\n"
	}

	var content strings.Builder
	content.WriteString("📝 Category & Project Breakdown\n\n")

	// Calculate total
	var total time.Duration
	for _, duration := range categoryTime {
		total += duration
	}

	content.WriteString("   Category    • Time    • % of Total\n")
	content.WriteString("   ─────────────────────────────\n")

	// Show categories sorted by time
	var categories []string
	for cat := range categoryTime {
		categories = append(categories, cat)
	}
	sort.Slice(categories, func(i, j int) bool {
		return categoryTime[categories[i]] > categoryTime[categories[j]]
	})

	for _, cat := range categories {
		duration := categoryTime[cat]
		percentage := float64(duration) / float64(total) * 100
		content.WriteString(fmt.Sprintf("   %-10s  • %-7s • %.1f%%\n",
			strings.ToUpper(cat), formatDuration(duration), percentage))
	}

	return content.String()
}

func (m Model) renderTimeChart(dailyTime map[string]time.Duration, categoryTime map[string]time.Duration) string {
	var content strings.Builder

	// Daily time chart
	if len(dailyTime) > 0 {
		content.WriteString("📊 Daily Time Distribution\n\n")

		// Get sorted dates and find max for scaling
		var dates []string
		var maxDuration time.Duration
		for date, duration := range dailyTime {
			dates = append(dates, date)
			if duration > maxDuration {
				maxDuration = duration
			}
		}
		sort.Strings(dates)

		// Show last 10 days
		start := max(0, len(dates)-10)
		for i := start; i < len(dates); i++ {
			date := dates[i]
			duration := dailyTime[date]
			barWidth := int(float64(duration) / float64(maxDuration) * 20) // Max 20 chars
			if barWidth < 1 && duration > 0 {
				barWidth = 1
			}
			bar := strings.Repeat("█", barWidth) + strings.Repeat("░", 20-barWidth)
			content.WriteString(fmt.Sprintf("%s %s %s\n", date, bar, formatDuration(duration)))
		}
		content.WriteString("\n")
	}

	// Category distribution chart
	if len(categoryTime) > 0 {
		content.WriteString("📈 Category Distribution\n\n")

		// Calculate total and sort categories
		var total time.Duration
		var categories []string
		for cat, duration := range categoryTime {
			total += duration
			categories = append(categories, cat)
		}
		sort.Slice(categories, func(i, j int) bool {
			return categoryTime[categories[i]] > categoryTime[categories[j]]
		})

		for _, cat := range categories {
			duration := categoryTime[cat]
			percentage := float64(duration) / float64(total) * 100
			barWidth := int(percentage / 100 * 20) // Max 20 chars
			if barWidth < 1 && duration > 0 {
				barWidth = 1
			}
			bar := strings.Repeat("█", barWidth) + strings.Repeat("░", 20-barWidth)
			content.WriteString(fmt.Sprintf("%-10s %s %s (%.1f%%)\n",
				strings.ToUpper(cat), bar, formatDuration(duration), percentage))
		}
	}

	return content.String()
}

// ---------- Command Palette Functions ----------

func (m Model) updateCommandPalette(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			m.mode = modeNormal
			m.commandPalette.Blur()
			return m, nil

		case tea.KeyEnter:
			// Execute selected command
			if m.commandCursor < len(m.filteredCommands) {
				selectedCmd := m.filteredCommands[m.commandCursor]
				m.mode = modeNormal
				m.commandPalette.Blur()
				return selectedCmd.Action(m)
			}
			return m, nil

		case tea.KeyUp, tea.KeyTab:
			// Move cursor up
			if m.commandCursor > 0 {
				m.commandCursor--
			}
			return m, nil

		case tea.KeyDown, tea.KeyShiftTab:
			// Move cursor down
			if m.commandCursor < len(m.filteredCommands)-1 {
				m.commandCursor++
			}
			return m, nil

		case tea.KeyCtrlP:
			// Previous category
			if m.selectedCategory > 0 {
				m.selectedCategory--
				m.filterCommandsByCategory()
				m.commandCursor = 0
			}
			return m, nil

		case tea.KeyCtrlN:
			// Next category
			if m.selectedCategory < len(m.commandCategories)-1 {
				m.selectedCategory++
				m.filterCommandsByCategory()
				m.commandCursor = 0
			}
			return m, nil

		case tea.KeyCtrlR:
			// Reset filters
			m.selectedCategory = 0
			m.commandPaletteInput = ""
			m.commandPalette.SetValue("")
			m.filteredCommands = make([]Command, len(m.commands))
			copy(m.filteredCommands, m.commands)
			m.commandCursor = 0
			return m, nil

		default:
			// Handle text input
			var cmd tea.Cmd
			m.commandPalette, cmd = m.commandPalette.Update(msg)
			newInput := m.commandPalette.Value()

			// Update filtered commands when input changes
			if newInput != m.commandPaletteInput {
				m.commandPaletteInput = newInput
				m.filterCommands()
				m.commandCursor = 0
			}

			return m, cmd
		}
	}

	return m, nil
}

func (m Model) renderCommandPaletteView() string {
	var content strings.Builder

	// Title
	title := m.st.modalTitle.Render("🔍 Command Palette")
	content.WriteString(title)
	content.WriteString("\n\n")

	// Search input
	content.WriteString(m.st.textBold.Render("Search: "))
	content.WriteString(m.commandPalette.View())
	content.WriteString("\n\n")

	// Category filter
	if m.selectedCategory == 0 {
		content.WriteString(m.st.textBold.Render("Category: All Categories") + "\n")
	} else {
		category := m.commandCategories[m.selectedCategory-1]
		content.WriteString(fmt.Sprintf("%sCategory: %s %s",
			m.st.textBold.Render("Category: "),
			category.Icon,
			category.Name))
		content.WriteString("\n")
	}
	content.WriteString("  (Ctrl+P/N to change, Ctrl+R to reset)\n\n")

	// Commands list
	if len(m.filteredCommands) == 0 {
		content.WriteString("No commands found.\n")
	} else {
		content.WriteString(m.st.textBold.Render("Commands:\n"))

		// Group commands by category
		currentCategory := ""
		for i, cmd := range m.filteredCommands {
			// Show category header
			if currentCategory != cmd.Category {
				currentCategory = cmd.Category
				// Find category color
				categoryColor := "#f9e2af" // default
				for _, cat := range m.commandCategories {
					if cat.Name == cmd.Category {
						categoryColor = cat.Color
						break
					}
				}

				content.WriteString(fmt.Sprintf("\n%s%s%s\n",
					lipgloss.NewStyle().Foreground(lipgloss.Color(categoryColor)).Bold(true).Render("▸ "),
					lipgloss.NewStyle().Foreground(lipgloss.Color(categoryColor)).Bold(true).Render(currentCategory),
					lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf(" (%d)", m.countCommandsInCategory(currentCategory)))))
			}

			// Highlight selected command
			cursor := " "
			if i == m.commandCursor {
				cursor = "➤"
				content.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("#45475a")).Render(
					fmt.Sprintf("%s %s - %s", cursor, cmd.Name, cmd.Description)))
			} else {
				content.WriteString(fmt.Sprintf("%s %s - %s", cursor, cmd.Name, cmd.Description))
			}

			// Show shortcut
			if cmd.Shortcut != "" {
				content.WriteString(fmt.Sprintf(" [%s]", lipgloss.NewStyle().Faint(true).Render(cmd.Shortcut)))
			}

			content.WriteString("\n")
		}
	}

	// Help text
	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Faint(true).Render(
		"↑/↓ or Tab/Shift+Tab: Navigate | Enter: Execute | Esc: Exit"))

	return content.String()
}

func (m Model) filterCommands() {
	input := strings.ToLower(m.commandPaletteInput)
	if input == "" {
		// Show all commands from selected category
		m.filterCommandsByCategory()
		return
	}

	m.filteredCommands = []Command{}
	for _, cmd := range m.commands {
		// Check category filter
		if m.selectedCategory > 0 {
			category := m.commandCategories[m.selectedCategory-1]
			if cmd.Category != category.Name {
				continue
			}
		}

		// Check text search
		if strings.Contains(strings.ToLower(cmd.Name), input) ||
		   strings.Contains(strings.ToLower(cmd.Description), input) ||
		   strings.Contains(strings.ToLower(cmd.Shortcut), input) {
			m.filteredCommands = append(m.filteredCommands, cmd)
		}
	}
}

func (m Model) filterCommandsByCategory() {
	if m.selectedCategory == 0 {
		// Show all commands
		m.filteredCommands = make([]Command, len(m.commands))
		copy(m.filteredCommands, m.commands)
	} else {
		// Show commands from selected category
		category := m.commandCategories[m.selectedCategory-1]
		m.filteredCommands = []Command{}
		for _, cmd := range m.commands {
			if cmd.Category == category.Name {
				m.filteredCommands = append(m.filteredCommands, cmd)
			}
		}
	}

	// Apply search filter if there's active input
	if m.commandPaletteInput != "" {
		input := strings.ToLower(m.commandPaletteInput)
		filtered := []Command{}
		for _, cmd := range m.filteredCommands {
			if strings.Contains(strings.ToLower(cmd.Name), input) ||
			   strings.Contains(strings.ToLower(cmd.Description), input) ||
			   strings.Contains(strings.ToLower(cmd.Shortcut), input) {
				filtered = append(filtered, cmd)
			}
		}
		m.filteredCommands = filtered
	}
}

func (m Model) countCommandsInCategory(category string) int {
	count := 0
	for _, cmd := range m.commands {
		if cmd.Category == category {
			count++
		}
	}
	return count
}

// ---------- Accessibility Functions ----------

func (m *Model) announceToScreenReader(message string) {
	if m.accessibilityMode {
		// Add timestamp for ordering
		timestampedMsg := fmt.Sprintf("[%s] %s", m.now.Format("15:04:05"), message)
		m.screenReaderBuffer = append(m.screenReaderBuffer, timestampedMsg)

		// Keep buffer size manageable
		if len(m.screenReaderBuffer) > 100 {
			m.screenReaderBuffer = m.screenReaderBuffer[len(m.screenReaderBuffer)-100:]
		}

		// Print to stderr for screen readers to capture
		fmt.Fprintln(os.Stderr, timestampedMsg)
	}
}

func (m Model) getCurrentContextForScreenReader() string {
	var context strings.Builder

	// Current mode
	modeName := "normal view"
	switch m.mode {
	case modeCommandPalette:
		modeName = "command palette"
	case modeCreate:
		modeName = "create entry"
	case modeSearch:
		modeName = "search"
	case modeHelp:
		modeName = "help"
	case modeDashboard:
		modeName = "dashboard"
	case modeTimeReports:
		modeName = "time reports"
	case modeProjectSummary:
		modeName = "project summary"
	case modeTagAnalytics:
		modeName = "tag analytics"
	case modeCalendar:
		modeName = "calendar"
	case modeTemplates:
		modeName = "templates"
	case modeExport:
		modeName = "export"
	case modeAdvancedSearch:
		modeName = "advanced search"
	}

	context.WriteString(fmt.Sprintf("Current mode: %s. ", modeName))

	// Current scope
	scopeName := "today"
	switch m.scope {
	case scopeToday:
		scopeName = "today"
	case scopeThisWeek:
		scopeName = "this week"
	case scopeThisMonth:
		scopeName = "this month"
	case scopeAll:
		scopeName = "all time"
	case scopeYesterday:
		scopeName = "yesterday"
	case scopeLastWeek:
		scopeName = "last week"
	case scopeLastMonth:
		scopeName = "last month"
	}

	context.WriteString(fmt.Sprintf("Time scope: %s. ", scopeName))

	// Entry count
	totalEntries := 0
	for _, block := range m.blocks {
		totalEntries += len(block.entries)
	}
	context.WriteString(fmt.Sprintf("Showing %d entries. ", totalEntries))

	// Current focus
	switch m.focus {
	case focusTimeline:
		context.WriteString("Timeline has focus. ")
		if len(m.blocks) > 0 && m.cursorBlock < len(m.blocks) && m.cursorEntry < len(m.blocks[m.cursorBlock].entries) {
			entry := m.blocks[m.cursorBlock].entries[m.cursorEntry]
			context.WriteString(fmt.Sprintf("Current entry: %s, category: %s, project: %s. ",
				entry.text[:min(50, len(entry.text))], entry.cat, entry.project))
		}
	case focusSidebar:
		context.WriteString("Sidebar has focus. ")
		sectionName := "projects"
		switch m.sidebarSection {
		case 0:
			sectionName = "projects"
		case 1:
			sectionName = "categories"
		case 2:
			sectionName = "tags"
		}
		context.WriteString(fmt.Sprintf("Current section: %s. ", sectionName))
	case focusThread:
		context.WriteString("Thread view has focus. ")
		if len(m.threadBlock.entries) > 0 {
			context.WriteString(fmt.Sprintf("Thread has %d entries. ", len(m.threadBlock.entries)))
		}
	}

	// Current filters
	if m.filterText != "" {
		context.WriteString(fmt.Sprintf("Search filter: %s. ", m.filterText))
	}
	if m.filterProj != "" {
		context.WriteString(fmt.Sprintf("Project filter: %s. ", m.filterProj))
	}
	if m.filterCat != "" {
		context.WriteString(fmt.Sprintf("Category filter: %s. ", m.filterCat))
	}
	if len(m.filterTags) > 0 {
		tags := make([]string, 0, len(m.filterTags))
		for tag := range m.filterTags {
			tags = append(tags, tag)
		}
		context.WriteString(fmt.Sprintf("Tag filters: %s. ", strings.Join(tags, ", ")))
	}

	return context.String()
}

// ---------- Rich Text Editor Functions ----------

func (m Model) updateRichTextEditor(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			m.mode = modeNormal
			return m, nil

		case tea.KeyCtrlS:
			// Save entry
			if err := m.saveRichTextEntry(); err != nil {
				m.addNotification(fmt.Sprintf("Error saving entry: %v", err))
			} else {
				m.addNotification("Entry saved successfully")
			}
			return m, nil

		case tea.KeyCtrlP:
			// Toggle preview
			m.richTextPreview = !m.richTextPreview
			if m.richTextPreview {
				m.addNotification("Preview enabled")
			} else {
				m.addNotification("Preview disabled")
			}
			return m, nil

		case tea.KeyCtrlM:
			// Cycle format
			formats := []string{"markdown", "html", "plain"}
			currentIndex := 0
			for index, f := range formats {
				if f == m.richTextFormat {
					currentIndex = index
					break
				}
			}
			nextIndex := (currentIndex + 1) % len(formats)
			m.richTextFormat = formats[nextIndex]
			m.addNotification(fmt.Sprintf("Format: %s", m.richTextFormat))
			return m, nil

		case tea.KeyTab:
			// Move between toolbar and editor
			if m.richTextToolbar == -1 {
				m.richTextToolbar = 0
			} else {
				m.richTextToolbar = -1
			}
			return m, nil

		case tea.KeyLeft:
			// Navigate toolbar
			if m.richTextToolbar >= 0 {
				m.richTextToolbar = (m.richTextToolbar - 1 + 6) % 6
				return m, nil
			}
			// Otherwise pass to textarea

		case tea.KeyRight:
			// Navigate toolbar
			if m.richTextToolbar >= 0 {
				m.richTextToolbar = (m.richTextToolbar + 1) % 6
				return m, nil
			}
			// Otherwise pass to textarea

		// case tea.KeyEnter:
		//	// Apply toolbar action if toolbar is focused
		//	if m.richTextToolbar >= 0 {
		//		return m.applyRichTextAction()
		//	}
		//	// Otherwise pass to textarea

		default:
			// Pass to textarea if toolbar is not focused
			if m.richTextToolbar == -1 {
				var cmd tea.Cmd
				m.createText, cmd = m.createText.Update(msg)
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m Model) renderRichTextEditorView() string {
	var content strings.Builder

	// Title
	title := m.st.modalTitle.Render("📝 Rich Text Editor")
	content.WriteString(title)
	content.WriteString("\n\n")

	// Format selector
	formats := []string{"markdown", "html", "plain"}
	content.WriteString("Format: ")
	for _, format := range formats {
		if format == m.richTextFormat {
			content.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("#89b4fa")).Render(fmt.Sprintf(" [%s] ", format)))
		} else {
			content.WriteString(fmt.Sprintf(" %s ", format))
		}
	}
	content.WriteString(" (Ctrl+M to change)")
	content.WriteString("\n\n")

	// Toolbar
	toolbar := []string{"Bold", "Italic", "Code", "Link", "List", "Quote"}
	content.WriteString("Toolbar: ")
	for i, tool := range toolbar {
		prefix := " "
		if i == m.richTextToolbar {
			prefix = "➤"
		}
		content.WriteString(fmt.Sprintf("%s[%s] ", prefix, tool))
	}
	content.WriteString(" (Tab to focus, Enter to apply)")
	content.WriteString("\n\n")

	// Editor area
	if m.richTextPreview {
		// Show preview
		content.WriteString(m.st.textBold.Render("Preview:"))
		content.WriteString("\n")
		preview := m.renderMarkdownPreview(m.createText.Value())
		content.WriteString(preview)
	} else {
		// Show editor
		content.WriteString(m.st.textBold.Render("Content:"))
		content.WriteString("\n")
		content.WriteString(m.createText.View())
	}

	// Help text
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Faint(true).Render(
		"Ctrl+S: Save | Ctrl+P: Toggle Preview | Ctrl+M: Change Format | Tab: Focus Toolbar | Esc: Exit"))

	return content.String()
}

func (m Model) applyRichTextAction() (Model, tea.Cmd) {
	actions := []string{
		"**bold**",
		"*italic*",
		"`code`",
		"[text](url)",
		"- item",
		"> quote",
	}

	if m.richTextToolbar >= 0 && m.richTextToolbar < len(actions) {
		action := actions[m.richTextToolbar]
		currentText := m.createText.Value()

		// Append action to current text (simplified)
		newText := currentText + action
		m.createText.SetValue(newText)

		m.addNotification(fmt.Sprintf("Applied %s formatting", actions[m.richTextToolbar]))
	}

	return m, nil
}

func (m Model) renderMarkdownPreview(text string) string {
	// Simple markdown renderer for preview
	lines := strings.Split(text, "\n")
	var preview strings.Builder

	inCodeBlock := false
	inList := false
	inQuote := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				preview.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("#45475a")).Render(" CODE BLOCK "))
				preview.WriteString("\n")
			}
			continue
		}

		if inCodeBlock {
			preview.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Render(line))
			preview.WriteString("\n")
			continue
		}

		// Headers
		if strings.HasPrefix(line, "# ") {
			preview.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f9e2af")).Render(strings.TrimPrefix(line, "# ")))
			preview.WriteString("\n")
			continue
		}

		// Lists
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			if !inList {
				inList = true
			}
			preview.WriteString("  • " + strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* "))
			preview.WriteString("\n")
			continue
		} else if inList && strings.TrimSpace(line) == "" {
			inList = false
		}

		// Quotes
		if strings.HasPrefix(line, "> ") {
			if !inQuote {
				inQuote = true
			}
			preview.WriteString(lipgloss.NewStyle().Faint(true).Render("│ " + strings.TrimPrefix(line, "> ")))
			preview.WriteString("\n")
			continue
		} else if inQuote && strings.TrimSpace(line) == "" {
			inQuote = false
		}

		// Bold text
		line = strings.ReplaceAll(line, "**", "")

		// Italic text
		line = strings.ReplaceAll(line, "*", "")

		// Code
		line = strings.ReplaceAll(line, "`", "")

		// Links - simple format
		line = strings.ReplaceAll(line, "[", "")
		line = strings.ReplaceAll(line, "](", " → ")
		line = strings.ReplaceAll(line, ")", "")

		preview.WriteString(line)
		preview.WriteString("\n")
	}

	return preview.String()
}

func (m Model) saveRichTextEntry() error {
	// Save the rich text entry to database
	text := m.createText.Value()
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("cannot save empty entry")
	}

	// For now, use existing create logic with rich text
	project := strings.TrimSpace(m.createProject.Value())
	tags := strings.TrimSpace(m.createTags.Value())
	category := strings.ToLower(m.createCategory.Value())
	if category == "" {
		category = "note"
	}

	_, err := m.db.Exec(`
		INSERT INTO entries(category, text, project, tags)
		VALUES(?,?,?,?)
	`, category, text, nullIfEmpty(project), nullIfEmpty(tags))

	if err != nil {
		return err
	}

	// Reload timeline to show new entry
	cmd := m.loadTimelineCmd()
	cmd()
	return nil
}

// ---------- Template Edit Functions ----------

func (m Model) updateTemplateEdit(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			m.mode = modeNormal
			m.templateEditMode = false
			m.templateCreateMode = false
			return m, nil

		case tea.KeyCtrlS:
			// Save template
			if m.templateCreateMode {
				if err := m.createTemplate(); err != nil {
					m.addNotification(fmt.Sprintf("Error creating template: %v", err))
				} else {
					m.addNotification("Template created successfully")
					m.mode = modeNormal
					return m, m.loadTemplatesCmd()
				}
			} else if m.templateEditMode {
				if err := m.updateTemplate(); err != nil {
					m.addNotification(fmt.Sprintf("Error updating template: %v", err))
				} else {
					m.addNotification("Template updated successfully")
					m.mode = modeNormal
					return m, m.loadTemplatesCmd()
				}
			}
			return m, nil

		case tea.KeyTab:
			// Navigate between fields
			// Simple cycling through fields
			return m, nil

		default:
			// Handle text input
			var cmd tea.Cmd
			m.templateEditName, cmd = m.templateEditName.Update(msg)
			m.templateEditDesc, cmd = m.templateEditDesc.Update(msg)
			m.templateEditCategory, cmd = m.templateEditCategory.Update(msg)
			m.templateEditContent, cmd = m.templateEditContent.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) renderTemplateEditView() string {
	var content strings.Builder

	// Title
	title := "Edit Template"
	if m.templateCreateMode {
		title = "Create Template"
	}
	content.WriteString(m.st.modalTitle.Render("📝 "+title))
	content.WriteString("\n\n")

	// Template name
	content.WriteString(m.st.textBold.Render("Name:"))
	content.WriteString("\n")
	content.WriteString(m.templateEditName.View())
	content.WriteString("\n\n")

	// Category
	content.WriteString(m.st.textBold.Render("Category:"))
	content.WriteString("\n")
	content.WriteString(m.templateEditCategory.View())
	content.WriteString("\n\n")

	// Description
	content.WriteString(m.st.textBold.Render("Description:"))
	content.WriteString("\n")
	content.WriteString(m.templateEditDesc.View())
	content.WriteString("\n\n")

	// Content
	content.WriteString(m.st.textBold.Render("Content:"))
	content.WriteString("\n")
	content.WriteString(m.templateEditContent.View())
	content.WriteString("\n\n")

	// Help text
	content.WriteString(lipgloss.NewStyle().Faint(true).Render(
		"Ctrl+S: Save | Esc: Cancel | Tab: Navigate fields"))

	return content.String()
}

func (m Model) createTemplate() error {
	name := strings.TrimSpace(m.templateEditName.Value())
	if name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	category := strings.TrimSpace(m.templateEditCategory.Value())
	if category == "" {
		return fmt.Errorf("template category cannot be empty")
	}

	content := strings.TrimSpace(m.templateEditContent.Value())
	if content == "" {
		return fmt.Errorf("template content cannot be empty")
	}

	description := strings.TrimSpace(m.templateEditDesc.Value())

	// Generate ID from name
	id := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	id = strings.ReplaceAll(id, "[^a-z0-9_]", "")

	// Create template in database
	dbTemplate := db.DBTemplate{
		ID:          id,
		Name:        name,
		Category:    category,
		Content:     content,
		Description: description,
		IsCustom:    true,
		IsFavorite:  false,
	}

	return db.CreateTemplate(m.db, dbTemplate)
}

func (m Model) updateTemplate() error {
	if m.templateEditID == "" {
		return fmt.Errorf("no template ID specified")
	}

	name := strings.TrimSpace(m.templateEditName.Value())
	if name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	category := strings.TrimSpace(m.templateEditCategory.Value())
	if category == "" {
		return fmt.Errorf("template category cannot be empty")
	}

	content := strings.TrimSpace(m.templateEditContent.Value())
	if content == "" {
		return fmt.Errorf("template content cannot be empty")
	}

	description := strings.TrimSpace(m.templateEditDesc.Value())

	// Update template in database
	dbTemplate := db.DBTemplate{
		ID:          m.templateEditID,
		Name:        name,
		Category:    category,
		Content:     content,
		Description: description,
		IsCustom:    true, // Only custom templates can be edited
		IsFavorite:  false,
	}

	return db.UpdateTemplate(m.db, dbTemplate)
}

// createPomodoroLogEntry creates a log entry for a completed pomodoro session
func (m Model) createPomodoroLogEntry(sessionType string) {
	var content string
	var category string
	sessionTime := time.Now().In(m.loc)

	if sessionType == "work" {
		content = fmt.Sprintf("🍅 Completed Pomodoro work session #%d\nTotal work sessions today: %d\nTotal focus time: %s",
			m.pomodoroWorkSessions, m.pomodoroWorkSessions, m.pomodoroTotalTime.Round(time.Minute))
		category = "timer"
	} else {
		content = fmt.Sprintf("☕ Completed Pomodoro break\nBack to work! 💪")
		category = "timer"
	}

	_, err := m.db.Exec(`
		INSERT INTO entries(category, text, ts, duration_minutes)
		VALUES(?, ?, ?, ?)
	`, category, content, sessionTime.UTC().Format(time.RFC3339), int(m.workSessionTime.Minutes()))

	if err != nil {
		// Log error but don't interrupt the flow
		fmt.Fprintf(os.Stderr, "Failed to create pomodoro log entry: %v\n", err)
	}
}

func (m *Model) applyAccessibilityTheme() {
	if m.highContrast {
		// Apply high contrast colors
		m.st.topBar = m.st.topBar.Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#000000"))
		m.st.statusBar = m.st.statusBar.Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#000000"))
		m.st.borderFocus = m.st.borderFocus.BorderForeground(lipgloss.Color("#FFFFFF"))
		m.st.borderDim = m.st.borderDim.BorderForeground(lipgloss.Color("#808080"))
		m.st.textBold = m.st.textBold.Foreground(lipgloss.Color("#FFFFFF"))
		m.st.textDim = m.st.textDim.Foreground(lipgloss.Color("#CCCCCC"))
		m.st.project = m.st.project.Foreground(lipgloss.Color("#FFFFFF"))
		m.st.tags = m.st.tags.Foreground(lipgloss.Color("#FFFFFF"))
		m.st.modalBox = m.st.modalBox.BorderForeground(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#000000"))
		m.st.modalTitle = m.st.modalTitle.Foreground(lipgloss.Color("#FFFFFF"))
	}
}
