// cmd/clickup/update_task.go
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hiraking/click-up-client/internal/client"
	"github.com/hiraking/click-up-client/internal/dateparse"
	"github.com/hiraking/click-up-client/internal/models"
	"github.com/spf13/cobra"
)

var validClearFields = map[string]bool{
	"description":   true,
	"status":        true,
	"priority":      true,
	"due-date":      true,
	"start-date":    true,
	"time-estimate": true,
}

func newUpdateTaskCmd() *cobra.Command {
	var name string
	var description string
	var status string
	var priority string
	var dueDateStr string
	var startDateStr string
	var timeEstimateMin int
	var parentID string
	var clearFields []string

	cmd := &cobra.Command{
		Use:   "update-task <taskId>",
		Short: "Update an existing task and output it as JSON",
		Long: `Update an existing ClickUp task by task ID.

Only the flags you specify will be updated. Flags not provided are left unchanged.

Clearing fields:
  Use --clear FIELD to remove a field's value entirely.
  Accepted values: description, status, priority, due-date, start-date, time-estimate

  Note: 'name' cannot be cleared (required field).
        'parent' cannot be cleared (ClickUp API does not support removing parent).

  Examples:
    update-task abc123 --clear due-date
    update-task abc123 --clear due-date --clear priority
    update-task abc123 --name "New Name" --clear description`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			for _, f := range clearFields {
				if !validClearFields[f] {
					return fmt.Errorf("Error: invalid field for --clear: %q. Accepted: description, status, priority, due-date, start-date, time-estimate", f)
				}
			}

			changed := cmd.Flags().Changed
			if !changed("name") && !changed("description") && !changed("status") &&
				!changed("priority") && !changed("due-date") && !changed("start-date") &&
				!changed("time-estimate") && !changed("parent") && len(clearFields) == 0 {
				return fmt.Errorf("Error: no fields specified to update.")
			}

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			req := models.UpdateTaskRequest{
				ClearFields: clearFields,
			}

			if changed("name") {
				req.Name = &name
			}
			if changed("description") {
				req.Description = &description
			}
			if changed("status") {
				req.Status = &status
			}
			if changed("priority") {
				p, err := parsePriority(priority)
				if err != nil {
					return err
				}
				req.Priority = &p
			}
			if changed("due-date") {
				t, err := dateparse.ParseISO(dueDateStr, "due-date")
				if err != nil {
					return err
				}
				req.DueDate = &t
			}
			if changed("start-date") {
				t, err := dateparse.ParseISO(startDateStr, "start-date")
				if err != nil {
					return err
				}
				req.StartDate = &t
			}
			if changed("time-estimate") {
				d := time.Duration(timeEstimateMin) * time.Minute
				req.TimeEstimate = &d
			}
			if changed("parent") {
				if strings.TrimSpace(parentID) == "" {
					return fmt.Errorf("Error: '--parent' must not be empty or whitespace.")
				}
				req.Parent = &parentID
			}

			c := client.New(cfg.APIKey)
			task, err := c.UpdateTask(context.Background(), taskID, req)
			if err != nil {
				return err
			}
			return printJSON(task)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New task name.")
	cmd.Flags().StringVar(&description, "description", "", "New task description.")
	cmd.Flags().StringVar(&status, "status", "", "New status name (e.g. \"to do\", \"in progress\").")
	cmd.Flags().StringVar(&priority, "priority", "", "New priority: urgent, high, normal, or low.")
	cmd.Flags().StringVar(&dueDateStr, "due-date", "", "New due date as ISO 8601. Timezone-less values are treated as JST (+09:00).")
	cmd.Flags().StringVar(&startDateStr, "start-date", "", "New start date as ISO 8601. Timezone-less values are treated as JST (+09:00).")
	cmd.Flags().IntVar(&timeEstimateMin, "time-estimate", 0, "New time estimate in minutes.")
	cmd.Flags().StringVar(&parentID, "parent", "", "New parent task ID.")
	cmd.Flags().StringArrayVar(&clearFields, "clear", nil,
		"Field to clear (repeatable). Accepted: description, status, priority, due-date, start-date, time-estimate.\n"+
			"Use --clear FIELD to remove a field's value from the task.")

	return cmd
}
