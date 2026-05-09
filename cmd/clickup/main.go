// cmd/clickup/main.go
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/hiraking/click-up-cli/cmd/clickup/task"
	configcmd "github.com/hiraking/click-up-cli/cmd/clickup/config"
	timecmd "github.com/hiraking/click-up-cli/cmd/clickup/time"
	"github.com/spf13/cobra"
)

var configPath string
var version = "dev"

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func main() {
	rootCmd := &cobra.Command{
		Use:           "clickup",
		Short:         "ClickUp API CLI wrapper",
		Version:       resolveVersion(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path (default: ~/.clickup/config.json)")

	rootCmd.AddCommand(task.NewCmd(&configPath))
	rootCmd.AddCommand(timecmd.NewCmd(&configPath))
	rootCmd.AddCommand(configcmd.NewCmd(&configPath))

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
