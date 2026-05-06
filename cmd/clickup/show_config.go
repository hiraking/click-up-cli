// cmd/clickup/show_config.go
package main

import (
	"github.com/spf13/cobra"
)

func newShowConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-config",
		Short: "Show current configuration as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			lists := cfg.Lists
			if lists == nil {
				lists = map[string]string{}
			}
			out := struct {
				APIKey string            `json:"apiKey"`
				TeamID string            `json:"teamId"`
				Lists  map[string]string `json:"lists"`
			}{
				APIKey: maskAPIKey(cfg.APIKey),
				TeamID: cfg.TeamID,
				Lists:  lists,
			}
			return printJSON(out)
		},
	}
	return cmd
}
