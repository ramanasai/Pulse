package cmd

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanasai/pulse/internal/db"
	"github.com/spf13/cobra"
)

var (
	editText     string
	editCategory string
	editProject  string
	editTags     string
)

var editCmd = &cobra.Command{
	Use:   "edit [entry-id]",
	Short: "Edit an existing log entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse entry ID
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid entry ID: %v", err)
		}

		// Check if any field is being updated
		if editText == "" && editCategory == "" && editProject == "" && editTags == "" {
			return fmt.Errorf("nothing to update - specify at least one field to edit")
		}

		// Open database
		dbh, err := db.Open()
		if err != nil {
			return err
		}
		defer dbh.Close()

		// Verify entry exists
		var existingText string
		err = dbh.QueryRow("SELECT text FROM entries WHERE id = ?", id).Scan(&existingText)
		if err == sql.ErrNoRows {
			return fmt.Errorf("entry with ID %d not found", id)
		}
		if err != nil {
			return fmt.Errorf("error checking entry: %v", err)
		}

		// Build dynamic UPDATE query
		var updates []string
		var updateArgs []interface{}

		if editText != "" {
			updates = append(updates, "text = ?")
			updateArgs = append(updateArgs, editText)
		}
		if editCategory != "" {
			// Validate category
			validCategories := []string{"note", "task", "meeting", "timer"}
			isValid := false
			for _, cat := range validCategories {
				if editCategory == cat {
					isValid = true
					break
				}
			}
			if !isValid {
				return fmt.Errorf("invalid category '%s'. Valid categories: %s", editCategory, strings.Join(validCategories, ", "))
			}
			updates = append(updates, "category = ?")
			updateArgs = append(updateArgs, editCategory)
		}
		if editProject != "" {
			updates = append(updates, "project = ?")
			updateArgs = append(updateArgs, editProject)
		}
		if editTags != "" {
			updates = append(updates, "tags = ?")
			updateArgs = append(updateArgs, editTags)
		}

		if len(updates) == 0 {
			return fmt.Errorf("nothing to update")
		}

		// Add ID to args
		updateArgs = append(updateArgs, id)

		// Execute update
		query := fmt.Sprintf("UPDATE entries SET %s WHERE id = ?", strings.Join(updates, ", "))
		result, err := dbh.Exec(query, updateArgs...)
		if err != nil {
			return fmt.Errorf("error updating entry: %v", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("error checking update result: %v", err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("no entry was updated")
		}

		fmt.Printf("Entry %d updated successfully.\n", id)
		return nil
	},
}

func init() {
	editCmd.Flags().StringVarP(&editText, "text", "m", "", "New text/content for the entry")
	editCmd.Flags().StringVarP(&editCategory, "category", "c", "", "New category: note|task|meeting|timer")
	editCmd.Flags().StringVarP(&editProject, "project", "p", "", "New project name")
	editCmd.Flags().StringVarP(&editTags, "tags", "t", "", "New comma-separated tags")
}