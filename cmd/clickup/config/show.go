// cmd/clickup/config/show.go
package configcmd

import (
	"github.com/hiraking/click-up-cli/cmd/clickup/cmdutil"
	"github.com/spf13/cobra"
)

func newShowCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cmdutil.LoadConfig(*configPath)
			if err != nil {
				return err
			}
			lists := cfg.Lists
			if lists == nil {
				lists = map[string]string{}
			}
			out := struct {
				APIKey   string            `json:"apiKey"`
				TeamID   string            `json:"teamId"`
				Lists    map[string]string `json:"lists"`
				Timezone string            `json:"timezone,omitempty"`
			}{
				APIKey:   cmdutil.MaskAPIKey(cfg.APIKey),
				TeamID:   cfg.TeamID,
				Lists:    lists,
				Timezone: cfg.Timezone,
			}
			return cmdutil.PrintJSON(out)
		},
	}
}
