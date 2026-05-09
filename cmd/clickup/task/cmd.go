// cmd/clickup/task/cmd.go
package task

import "github.com/spf13/cobra"

func NewCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
	}
	cmd.AddCommand(newListCmd(configPath))
	cmd.AddCommand(newGetCmd(configPath))
	cmd.AddCommand(newCreateCmd(configPath))
	cmd.AddCommand(newUpdateCmd(configPath))
	cmd.AddCommand(newDeleteCmd(configPath))
	return cmd
}
