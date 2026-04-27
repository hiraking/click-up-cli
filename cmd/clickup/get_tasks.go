// cmd/clickup/get_tasks.go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hiraking/click-up-client/internal/client"
	"github.com/hiraking/click-up-client/internal/dateparse"
	"github.com/spf13/cobra"
)

func newGetTasksCmd() *cobra.Command {
	var lists []string
	var statuses []string
	var dueAfterStr string
	var dueBeforeStr string
	var noSubtasks bool
	var query string

	cmd := &cobra.Command{
		Use:   "get-tasks",
		Short: "Get tasks as a JSON tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// --list 名 → list ID に解決
			var listIDs []string
			if len(lists) > 0 {
				listIDs = make([]string, 0, len(lists))
				for _, name := range lists {
					id, ok := cfg.Lists[name]
					if !ok {
						return fmt.Errorf("Error: Unknown list name '%s'. Available: %s",
							name, availableListNames(cfg.Lists))
					}
					listIDs = append(listIDs, id)
				}
			}

			var dueDateGt, dueDateLt *time.Time
			if dueAfterStr != "" {
				t, err := dateparse.ParseISO(dueAfterStr, "due-after")
				if err != nil {
					return err
				}
				dueDateGt = &t
			}
			if dueBeforeStr != "" {
				t, err := dateparse.ParseISO(dueBeforeStr, "due-before")
				if err != nil {
					return err
				}
				dueDateLt = &t
			}

			c := client.New(cfg.APIKey)
			tasks, err := c.GetTasks(context.Background(), cfg.TeamID, client.GetTasksOptions{
				IncludeSubtasks: !noSubtasks,
				ListIDs:         listIDs,
				Statuses:        statuses,
				DueDateGt:       dueDateGt,
				DueDateLt:       dueDateLt,
				Query:           query,
			})
			if err != nil {
				return err
			}
			return printJSON(tasks)
		},
	}

	cmd.Flags().StringArrayVar(&lists, "list", nil,
		"List name(s) defined in config.json (repeatable). Omit for all lists.")
	cmd.Flags().StringArrayVar(&statuses, "status", nil,
		"Status name(s) to filter by (repeatable), e.g. \"in progress\".")
	cmd.Flags().StringVar(&dueAfterStr, "due-after", "",
		"ISO 8601 datetime. Return only tasks with due date after this value.")
	cmd.Flags().StringVar(&dueBeforeStr, "due-before", "",
		"ISO 8601 datetime. Return only tasks with due date before this value.")
	cmd.Flags().BoolVar(&noSubtasks, "no-subtasks", false,
		"Exclude subtasks from results.")
	cmd.Flags().StringVar(&query, "query", "",
		"Case-insensitive substring to match against task name and description. Filtering is performed client-side after fetching all pages.")

	return cmd
}
