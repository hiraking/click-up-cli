// cmd/clickup/task/create.go
package task

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hiraking/click-up-cli/cmd/clickup/cmdutil"
	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/hiraking/click-up-cli/internal/dateparse"
	"github.com/hiraking/click-up-cli/internal/models"
	"github.com/spf13/cobra"
)

func newCreateCmd(configPath *string) *cobra.Command {
	var listName string
	var description string
	var parentID string
	var status string
	var priority string
	var dueDateStr string
	var startDateStr string
	var timeEstimateMin int
	var taskTypeStr string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new task and output it as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := cmdutil.LoadConfig(*configPath)
			if err != nil {
				return err
			}

			listID, ok := cfg.Lists[listName]
			if !ok {
				return fmt.Errorf("Error: Unknown list name '%s'. Available: %s",
					listName, cmdutil.AvailableListNames(cfg.Lists))
			}

			req := models.CreateTaskRequest{Name: name}

			if cmd.Flags().Changed("description") {
				req.Description = &description
			}
			if cmd.Flags().Changed("parent") {
				if strings.TrimSpace(parentID) == "" {
					return fmt.Errorf("Error: '--parent' must not be empty or whitespace.")
				}
				req.ParentID = &parentID
			}
			if cmd.Flags().Changed("status") {
				req.Status = &status
			}
			if cmd.Flags().Changed("priority") {
				p, err := parsePriority(priority)
				if err != nil {
					return err
				}
				req.Priority = &p
			}
			if cmd.Flags().Changed("due-date") {
				t, err := dateparse.ParseISO(dueDateStr, "due-date", cfg.TimezoneLocation())
				if err != nil {
					return err
				}
				req.DueDate = &t
			}
			if cmd.Flags().Changed("start-date") {
				t, err := dateparse.ParseISO(startDateStr, "start-date", cfg.TimezoneLocation())
				if err != nil {
					return err
				}
				req.StartDate = &t
			}
			if cmd.Flags().Changed("time-estimate") {
				d := time.Duration(timeEstimateMin) * time.Minute
				req.TimeEstimate = &d
			}
			if cmd.Flags().Changed("task-type") {
				id, err := lookupTaskType(cfg.TaskTypes, taskTypeStr)
				if err != nil {
					return err
				}
				req.CustomItemID = &id
			}

			c := client.New(cfg.APIKey)
			task, err := c.CreateTask(context.Background(), listID, req)
			if err != nil {
				return err
			}
			return cmdutil.PrintJSON(task)
		},
	}

	cmd.Flags().StringVar(&listName, "list", "", "List name defined in config.json.")
	_ = cmd.MarkFlagRequired("list")
	cmd.Flags().StringVar(&description, "description", "", "Task description.")
	cmd.Flags().StringVar(&parentID, "parent", "", "Parent task ID. Creates a subtask.")
	cmd.Flags().StringVar(&status, "status", "", "Status name (e.g. \"to do\", \"in progress\").")
	cmd.Flags().StringVar(&priority, "priority", "", "Priority: urgent, high, normal, or low.")
	cmd.Flags().StringVar(&dueDateStr, "due-date", "", "Due date as ISO 8601. Timezone-less values use the timezone from config (default UTC).")
	cmd.Flags().StringVar(&startDateStr, "start-date", "", "Start date as ISO 8601. Timezone-less values use the timezone from config (default UTC).")
	cmd.Flags().IntVar(&timeEstimateMin, "time-estimate", 0, "Time estimate in minutes.")
	cmd.Flags().StringVar(&taskTypeStr, "task-type", "", "Task type name as defined in the taskTypes config.")

	return cmd
}

func parsePriority(s string) (models.TaskPriority, error) {
	switch strings.ToLower(s) {
	case "urgent":
		return models.PriorityUrgent, nil
	case "high":
		return models.PriorityHigh, nil
	case "normal":
		return models.PriorityNormal, nil
	case "low":
		return models.PriorityLow, nil
	default:
		return 0, fmt.Errorf("Error: Invalid priority '%s'. Use urgent, high, normal, or low.", s)
	}
}

func lookupTaskType(taskTypes map[string]int, name string) (int, error) {
	if len(taskTypes) == 0 {
		return 0, fmt.Errorf("Error: No task types configured. Add a \"taskTypes\" mapping to config.json.")
	}
	id, ok := taskTypes[name]
	if !ok {
		keys := sortedStringKeys(taskTypes)
		return 0, fmt.Errorf("Error: Unknown task type '%s'. Available: %s", name, strings.Join(keys, ", "))
	}
	return id, nil
}

func sortedStringKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
