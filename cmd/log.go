package cmd

import (
	"fmt"
	"strings"

	"github.com/ramanasai/pulse/internal/db"
	"github.com/spf13/cobra"
)

var (
	category string
	project  string
	tags     string
)

var logCmd = &cobra.Command{
	Use:   "log [text]",
	Short: "Add a quick log entry",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbh, err := db.Open()
		if err != nil {
			return err
		}
		defer dbh.Close()
		text := strings.Join(args, " ")
		_, err = dbh.Exec(`INSERT INTO entries(category, text, project, tags) VALUES(?,?,?,?)`, category, text, project, tags)
		if err != nil {
			return err
		}
		fmt.Println("Saved.")
		return nil
	},
}

func init() {
	logCmd.Flags().StringVarP(&category, "category", "c", "note", "Category: note|task|meeting|timer")
	logCmd.Flags().StringVarP(&project, "project", "p", "", "Project name")
	logCmd.Flags().StringVarP(&tags, "tags", "t", "", "Comma separated tags")
}
