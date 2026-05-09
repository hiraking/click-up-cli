// cmd/clickup/config/cmd.go
package configcmd

import "github.com/spf13/cobra"

func NewCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}
	cmd.AddCommand(newShowCmd(configPath))
	return cmd
}
