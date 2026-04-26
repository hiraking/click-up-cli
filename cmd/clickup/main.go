// cmd/clickup/main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "clickup",
		Short:         "ClickUp API CLI wrapper",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(newGetTaskCmd())
	rootCmd.AddCommand(newGetTasksCmd())
	// create-task は後続タスクで追加する

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
