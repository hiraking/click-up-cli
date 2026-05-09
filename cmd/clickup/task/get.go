// cmd/clickup/task/get.go
package task

import (
	"context"

	"github.com/hiraking/click-up-cli/cmd/clickup/cmdutil"
	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/spf13/cobra"
)

func newGetCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <taskId>",
		Short: "Get a single task by ID as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cmdutil.LoadConfig(*configPath)
			if err != nil {
				return err
			}
			c := client.New(cfg.APIKey)
			task, err := c.GetTask(context.Background(), args[0])
			if err != nil {
				return err
			}
			return cmdutil.PrintJSON(task)
		},
	}
}
