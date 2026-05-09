// cmd/clickup/task/delete.go
package task

import (
	"context"

	"github.com/hiraking/click-up-cli/cmd/clickup/cmdutil"
	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/spf13/cobra"
)

type deleteTaskResult struct {
	Deleted bool   `json:"deleted"`
	TaskID  string `json:"taskId"`
}

func newDeleteCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <taskId>",
		Short: "Delete a task by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			cfg, err := cmdutil.LoadConfig(*configPath)
			if err != nil {
				return err
			}
			c := client.New(cfg.APIKey)
			if err := c.DeleteTask(context.Background(), taskID); err != nil {
				return err
			}
			return cmdutil.PrintJSON(deleteTaskResult{Deleted: true, TaskID: taskID})
		},
	}
}
