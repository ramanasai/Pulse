package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// DBTemplate represents a template in the database
type DBTemplate struct {
	ID          string        `db:"id"`
	Name        string        `db:"name"`
	Category    string        `db:"category"`
	Content     string        `db:"content"`
	Description string        `db:"description"`
	Variables   string        `db:"variables"` // JSON string
	IsCustom    bool          `db:"is_custom"`
	UsageCount  int           `db:"usage_count"`
	LastUsed    sql.NullTime  `db:"last_used"`
	IsFavorite  bool          `db:"is_favorite"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
}

// InitializeDefaultTemplates populates the database with default templates
func InitializeDefaultTemplates(dbh *sql.DB) error {
	// Check if templates already exist
	var count int
	err := dbh.QueryRow("SELECT COUNT(*) FROM templates WHERE is_custom = FALSE").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // Templates already initialized
	}

	defaultTemplates := []DBTemplate{
		{
			ID:          "meeting_notes",
			Name:        "Meeting Notes",
			Category:    "Work",
			Content:     "📅 **Meeting:** {{date}}\n👥 **Attendees:** \n⏰ **Duration:** \n\n📋 **Agenda:**\n- \n\n💡 **Key Points:**\n- \n\n✅ **Action Items:**\n- [ ] \n\n🎯 **Decisions:**\n- ",
			Description: "Structured meeting notes with agenda and action items",
			Variables:   `["date", "attendees", "duration"]`,
			IsCustom:    false,
		},
		{
			ID:          "daily_standup",
			Name:        "Daily Standup",
			Category:    "Work",
			Content:     "## Daily Standup - {{date}}\n\n**Yesterday:**\n- \n\n**Today:**\n- \n\n**Blockers:**\n- \n\n**Notes:**\n- ",
			Description: "Daily standup notes template",
			Variables:   `["date"]`,
			IsCustom:    false,
		},
		{
			ID:          "project_update",
			Name:        "Project Update",
			Category:    "Work",
			Content:     "## Project Update - {{date}}\n\n**Project:** {{project}}\n\n**Progress:**\n- \n\n**Completed:**\n- \n\n**Next Steps:**\n- \n\n**Challenges:**\n- ",
			Description: "Project status update template",
			Variables:   `["date", "project"]`,
			IsCustom:    false,
		},
		{
			ID:          "1on1_meeting",
			Name:        "1-on-1 Meeting",
			Category:    "Work",
			Content:     "## 1-on-1 Meeting - {{date}}\n\n**With:** \n\n**Discussion Topics:**\n- \n\n**Feedback:**\n- \n\n**Goals:**\n- \n\n**Follow-up Actions:**\n- ",
			Description: "1-on-1 meeting notes template",
			Variables:   `["date"]`,
			IsCustom:    false,
		},
		{
			ID:          "performance_review",
			Name:        "Performance Review",
			Category:    "Work",
			Content:     "## Performance Review - {{period}}\n\n**Strengths:**\n- \n\n**Areas for Improvement:**\n- \n\n**Achievements:**\n- \n\n**Goals for Next Period:**\n- \n\n**Feedback:**\n- ",
			Description: "Performance review template",
			Variables:   `["period"]`,
			IsCustom:    false,
		},
		{
			ID:          "bug_report",
			Name:        "Bug Report",
			Category:    "Development",
			Content:     "## Bug Report\n\n**Title:** \n\n**Description:**\n\n**Steps to Reproduce:**\n1. \n2. \n3. \n\n**Expected Behavior:**\n\n**Actual Behavior:**\n\n**Environment:**\n- OS: \n- Version: \n\n**Attachments:**\n- ",
			Description: "Bug report template for developers",
			Variables:   `[]`,
			IsCustom:    false,
		},
		{
			ID:          "code_review",
			Name:        "Code Review",
			Category:    "Development",
			Content:     "## Code Review - {{date}}\n\n**Repository:** {{repo}}\n**Branch:** {{branch}}\n**Pull Request:** #{{pr_number}}\n\n**Overall Assessment:**\n\n**Positive Points:**\n- \n\n**Suggestions:**\n- \n\n**Issues Found:**\n- \n\n**Approval:** ☐ Approve ☐ Request Changes",
			Description: "Code review checklist template",
			Variables:   `["date", "repo", "branch", "pr_number"]`,
			IsCustom:    false,
		},
		{
			ID:          "technical_design",
			Name:        "Technical Design",
			Category:    "Development",
			Content:     "## Technical Design - {{feature}}\n\n**Overview:**\n\n**Requirements:**\n- \n\n**Proposed Solution:**\n\n**Architecture:**\n\n**API Design:**\n\n**Database Changes:**\n\n**Testing Strategy:**\n\n**Risks and Mitigations:**\n\n**Timeline:**",
			Description: "Technical design document template",
			Variables:   `["feature"]`,
			IsCustom:    false,
		},
		{
			ID:          "api_docs",
			Name:        "API Documentation",
			Category:    "Development",
			Content:     "## API Documentation - {{endpoint}}\n\n**Method:** {{method}}\n**URL:** {{url}}\n\n**Description:**\n\n**Parameters:**\n\n**Request Body:**\n```json\n{\n  \n}\n```\n\n**Response:**\n```json\n{\n  \n}\n```\n\n**Error Codes:**\n\n**Examples:**",
			Description: "API documentation template",
			Variables:   `["endpoint", "method", "url"]`,
			IsCustom:    false,
		},
		{
			ID:          "deployment_checklist",
			Name:        "Deployment Checklist",
			Category:    "Development",
			Content:     "## Deployment Checklist - {{version}}\n\n**Pre-deployment:**\n- [ ] Tests pass\n- [ ] Code reviewed\n- [ ] Documentation updated\n- [ ] Backup created\n\n**Deployment Steps:**\n1. \n2. \n3. \n\n**Post-deployment:**\n- [ ] Verify functionality\n- [ ] Monitor performance\n- [ ] Update team\n\n**Rollback Plan:**",
			Description: "Deployment checklist template",
			Variables:   `["version"]`,
			IsCustom:    false,
		},
		{
			ID:          "daily_journal",
			Name:        "Daily Journal",
			Category:    "Personal",
			Content:     "## Daily Journal - {{date}}\n\n**Gratitude:**\n- \n\n**Today's Highlights:**\n- \n\n**Challenges:**\n- \n\n**What I Learned:**\n- \n\n**Tomorrow's Intentions:**\n- ",
			Description: "Daily journaling template",
			Variables:   `["date"]`,
			IsCustom:    false,
		},
		{
			ID:          "goal_setting",
			Name:        "Goal Setting",
			Category:    "Personal",
			Content:     "## Goal Setting - {{period}}\n\n**Overall Vision:**\n\n**Category Goals:**\n\n**Career:**\n- \n\n**Health:**\n- \n\n**Learning:**\n- \n\n**Relationships:**\n- \n\n**Financial:**\n- \n\n**Action Steps:**\n- \n\n**Success Metrics:**\n- ",
			Description: "Goal setting and planning template",
			Variables:   `["period"]`,
			IsCustom:    false,
		},
		{
			ID:          "habit_tracker",
			Name:        "Habit Tracker",
			Category:    "Personal",
			Content:     "## Habit Tracker - {{date}}\n\n**Daily Habits:**\n- [ ] Exercise\n- [ ] Read\n- [ ] Meditate\n- [ ] \n\n**Weekly Goals:**\n- \n\n**Reflections:**\n- What went well?\n- What could be improved?\n\n**Next Week Focus:**\n- ",
			Description: "Habit tracking template",
			Variables:   `["date"]`,
			IsCustom:    false,
		},
		{
			ID:          "reflection",
			Name:        "Weekly Reflection",
			Category:    "Personal",
			Content:     "## Weekly Reflection - Week of {{date}}\n\n**Wins:**\n- \n\n**Challenges:**\n- \n\n**Learnings:**\n- \n\n**Gratitude:**\n- \n\n**Areas for Improvement:**\n- \n\n**Next Week Focus:**\n- ",
			Description: "Weekly reflection template",
			Variables:   `["date"]`,
			IsCustom:    false,
		},
		{
			ID:          "decision_matrix",
			Name:        "Decision Matrix",
			Category:    "Personal",
			Content:     "## Decision Matrix - {{date}}\n\n**Decision:** \n\n**Options:**\n\n**Option 1:** \n- Pros: \n- Cons: \n- Score: \n\n**Option 2:** \n- Pros: \n- Cons: \n- Score: \n\n**Evaluation Criteria:**\n- Cost: \n- Time: \n- Impact: \n- Risk: \n\n**Final Decision:**",
			Description: "Decision-making matrix template",
			Variables:   `["date"]`,
			IsCustom:    false,
		},
		{
			ID:          "brainstorming",
			Name:        "Brainstorming Session",
			Category:    "Creative",
			Content:     "## Brainstorming - {{topic}}\n\n**Objective:**\n\n**Ideas (No Judging):**\n- \n- \n- \n\n**Themes:**\n- \n\n**Promising Concepts:**\n- \n\n**Next Steps:**\n- ",
			Description: "Brainstorming session template",
			Variables:   `["topic"]`,
			IsCustom:    false,
		},
		{
			ID:          "mind_mapping",
			Name:        "Mind Map",
			Category:    "Creative",
			Content:     "## Mind Map - {{central_idea}}\n\n**Main Branches:**\n\n**Branch 1:** \n- Sub-idea\n- Sub-idea\n\n**Branch 2:** \n- Sub-idea\n- Sub-idea\n\n**Branch 3:** \n- Sub-idea\n- Sub-idea\n\n**Connections:**\n- ",
			Description: "Mind mapping template",
			Variables:   `["central_idea"]`,
			IsCustom:    false,
		},
		{
			ID:          "story_outline",
			Name:        "Story Outline",
			Category:    "Creative",
			Content:     "## Story Outline - {{title}}\n\n**Logline:**\n\n**Characters:**\n- \n\n**Setting:**\n\n**Plot Points:**\n1. **Setup:** \n2. **Inciting Incident:** \n3. **Rising Action:** \n4. **Climax:** \n5. **Falling Action:** \n6. **Resolution:** \n\n**Themes:**\n- ",
			Description: "Story writing outline template",
			Variables:   `["title"]`,
			IsCustom:    false,
		},
		{
			ID:          "content_ideas",
			Name:        "Content Ideas",
			Category:    "Creative",
			Content:     "## Content Ideas - {{date}}\n\n**Target Audience:**\n\n**Content Pillars:**\n- \n\n**Ideas:**\n1. **Title:** \n   **Format:** \n   **Key Points:** \n   \n2. **Title:** \n   **Format:** \n   **Key Points:** \n   \n**Content Calendar:**\n- ",
			Description: "Content planning template",
			Variables:   `["date"]`,
			IsCustom:    false,
		},
		{
			ID:          "quick_note",
			Name:        "Quick Note",
			Category:    "Quick",
			Content:     "**Date:** {{date}}\n\n**Note:** \n\n**Tags:** \n\n**Follow-up:** \n- ",
			Description: "Quick note-taking template",
			Variables:   `["date"]`,
			IsCustom:    false,
		},
		{
			ID:          "task_list",
			Name:        "Task List",
			Category:    "Quick",
			Content:     "## Task List - {{date}}\n\n**Priority 1 (Urgent):**\n- [ ] \n\n**Priority 2 (Important):**\n- [ ] \n\n**Priority 3 (Can Wait):**\n- [ ] \n\n**Completed Today:**\n- [ ] ",
			Description: "Daily task list template",
			Variables:   `["date"]`,
			IsCustom:    false,
		},
		{
			ID:          "decision_making",
			Name:        "Quick Decision",
			Category:    "Quick",
			Content:     "## Decision - {{date}}\n\n**Decision:** \n\n**Options:**\n1. \n2. \n\n**Pros/Cons:**\n- Option 1: \n- Option 2: \n\n**Decision:** \n\n**Rationale:**",
			Description: "Quick decision-making template",
			Variables:   `["date"]`,
			IsCustom:    false,
		},
	}

	for _, template := range defaultTemplates {
		_, err := dbh.Exec(`
			INSERT OR IGNORE INTO templates
			(id, name, category, content, description, variables, is_custom)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, template.ID, template.Name, template.Category, template.Content,
		   template.Description, template.Variables, template.IsCustom)
		if err != nil {
			return fmt.Errorf("failed to insert template %s: %w", template.ID, err)
		}
	}

	return nil
}

// GetTemplate retrieves a template by ID
func GetTemplate(dbh *sql.DB, id string) (DBTemplate, error) {
	var template DBTemplate
	err := dbh.QueryRow(`
		SELECT id, name, category, content, description, variables,
		       is_custom, usage_count, last_used, is_favorite, created_at, updated_at
		FROM templates WHERE id = ?
	`, id).Scan(&template.ID, &template.Name, &template.Category, &template.Content,
		&template.Description, &template.Variables, &template.IsCustom, &template.UsageCount,
		&template.LastUsed, &template.IsFavorite, &template.CreatedAt, &template.UpdatedAt)
	return template, err
}

// GetAllTemplates retrieves all templates
func GetAllTemplates(dbh *sql.DB) ([]DBTemplate, error) {
	rows, err := dbh.Query(`
		SELECT id, name, category, content, description, variables,
		       is_custom, usage_count, last_used, is_favorite, created_at, updated_at
		FROM templates ORDER BY category, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []DBTemplate
	for rows.Next() {
		var template DBTemplate
		err := rows.Scan(&template.ID, &template.Name, &template.Category, &template.Content,
			&template.Description, &template.Variables, &template.IsCustom, &template.UsageCount,
			&template.LastUsed, &template.IsFavorite, &template.CreatedAt, &template.UpdatedAt)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}
	return templates, nil
}

// GetTemplatesByCategory retrieves templates by category
func GetTemplatesByCategory(dbh *sql.DB, category string) ([]DBTemplate, error) {
	rows, err := dbh.Query(`
		SELECT id, name, category, content, description, variables,
		       is_custom, usage_count, last_used, is_favorite, created_at, updated_at
		FROM templates WHERE category = ? ORDER BY name
	`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []DBTemplate
	for rows.Next() {
		var template DBTemplate
		err := rows.Scan(&template.ID, &template.Name, &template.Category, &template.Content,
			&template.Description, &template.Variables, &template.IsCustom, &template.UsageCount,
			&template.LastUsed, &template.IsFavorite, &template.CreatedAt, &template.UpdatedAt)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}
	return templates, nil
}

// CreateTemplate creates a new custom template
func CreateTemplate(dbh *sql.DB, template DBTemplate) error {
	_, err := dbh.Exec(`
		INSERT INTO templates
		(id, name, category, content, description, variables, is_custom, is_favorite)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, template.ID, template.Name, template.Category, template.Content,
	   template.Description, template.Variables, true, template.IsFavorite)
	return err
}

// UpdateTemplate updates an existing template
func UpdateTemplate(dbh *sql.DB, template DBTemplate) error {
	_, err := dbh.Exec(`
		UPDATE templates
		SET name = ?, category = ?, content = ?, description = ?,
		    variables = ?, is_favorite = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?
	`, template.Name, template.Category, template.Content, template.Description,
	   template.Variables, template.IsFavorite, template.ID)
	return err
}

// DeleteTemplate deletes a template
func DeleteTemplate(dbh *sql.DB, id string) error {
	_, err := dbh.Exec("DELETE FROM templates WHERE id = ? AND is_custom = TRUE", id)
	return err
}

// UpdateTemplateUsage updates the usage count and last used timestamp
func UpdateTemplateUsage(dbh *sql.DB, id string) error {
	_, err := dbh.Exec(`
		UPDATE templates
		SET usage_count = usage_count + 1,
		    last_used = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?
	`, id)
	return err
}

// GetTemplateCategories retrieves all unique template categories
func GetTemplateCategories(dbh *sql.DB) ([]string, error) {
	rows, err := dbh.Query("SELECT DISTINCT category FROM templates ORDER BY category")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}

// SearchTemplates searches templates by name, description, or content
func SearchTemplates(dbh *sql.DB, query string) ([]DBTemplate, error) {
	rows, err := dbh.Query(`
		SELECT id, name, category, content, description, variables,
		       is_custom, usage_count, last_used, is_favorite, created_at, updated_at
		FROM templates
		WHERE name LIKE ? OR description LIKE ? OR content LIKE ?
		ORDER BY usage_count DESC, name
	`, "%"+query+"%", "%"+query+"%", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []DBTemplate
	for rows.Next() {
		var template DBTemplate
		err := rows.Scan(&template.ID, &template.Name, &template.Category, &template.Content,
			&template.Description, &template.Variables, &template.IsCustom, &template.UsageCount,
			&template.LastUsed, &template.IsFavorite, &template.CreatedAt, &template.UpdatedAt)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}
	return templates, nil
}

// GetFavoriteTemplates retrieves all favorite templates
func GetFavoriteTemplates(dbh *sql.DB) ([]DBTemplate, error) {
	rows, err := dbh.Query(`
		SELECT id, name, category, content, description, variables,
		       is_custom, usage_count, last_used, is_favorite, created_at, updated_at
		FROM templates WHERE is_favorite = TRUE ORDER BY usage_count DESC, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []DBTemplate
	for rows.Next() {
		var template DBTemplate
		err := rows.Scan(&template.ID, &template.Name, &template.Category, &template.Content,
			&template.Description, &template.Variables, &template.IsCustom, &template.UsageCount,
			&template.LastUsed, &template.IsFavorite, &template.CreatedAt, &template.UpdatedAt)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}
	return templates, nil
}

// ToggleTemplateFavorite toggles the favorite status of a template
func ToggleTemplateFavorite(dbh *sql.DB, id string) error {
	_, err := dbh.Exec("UPDATE templates SET is_favorite = NOT is_favorite WHERE id = ?", id)
	return err
}

// ParseTemplateVariables parses the variables JSON field into a string slice
func ParseTemplateVariables(variablesJSON string) ([]string, error) {
	if variablesJSON == "" {
		return []string{}, nil
	}

	var variables []string
	err := json.Unmarshal([]byte(variablesJSON), &variables)
	return variables, err
}

// SerializeTemplateVariables serializes a string slice into JSON
func SerializeTemplateVariables(variables []string) (string, error) {
	if len(variables) == 0 {
		return "[]", nil
	}

	data, err := json.Marshal(variables)
	return string(data), err
}