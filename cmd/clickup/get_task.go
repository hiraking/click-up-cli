// cmd/clickup/get_task.go
package main

import (
	"context"

	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/spf13/cobra"
)

func newGetTaskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-task <taskId>",
		Short: "Get a single task by ID as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			c := client.New(cfg.APIKey)
			task, err := c.GetTask(context.Background(), args[0])
			if err != nil {
				return err
			}
			return printJSON(task)
		},
	}
}
