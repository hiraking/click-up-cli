// cmd/clickup/time/cmd.go
package timecmd

import "github.com/spf13/cobra"

func NewCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time",
		Short: "Manage time entries",
	}
	cmd.AddCommand(newReportCmd(configPath))
	return cmd
}
