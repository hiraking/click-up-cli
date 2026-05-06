// cmd/clickup/time_report.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/hiraking/click-up-cli/internal/dateparse"
	"github.com/hiraking/click-up-cli/internal/timereport"
)

func newTimeReportCmd() *cobra.Command {
	var flagStart, flagEnd, flagOutput string
	var flagRows bool

	cmd := &cobra.Command{
		Use:   "time-report",
		Short: "Aggregate time entries and output a JSON report",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			start, err := dateparse.ParseISO(flagStart, "start", cfg.TimezoneLocation())
			if err != nil {
				return err
			}
			end, err := dateparse.ParseISO(flagEnd, "end", cfg.TimezoneLocation())
			if err != nil {
				return err
			}
			if !end.After(start) {
				return fmt.Errorf("--end must be after --start")
			}

			ctx := context.Background()
			c := client.New(cfg.APIKey)

			entries, err := c.GetTimeEntries(ctx, cfg.TeamID, client.GetTimeEntriesOptions{
				Start: start,
				End:   end,
			})
			if err != nil {
				return err
			}

			report, err := timereport.Build(ctx, entries, start, end, c.GetTask)
			if err != nil {
				return err
			}

			// --rows のデフォルト: --output あり → 含める、なし → 含めない
			includeRows := flagOutput != ""
			if cmd.Flags().Changed("rows") {
				includeRows = flagRows
			}
			if !includeRows {
				report.Rows = nil
			}

			var w io.Writer = os.Stdout
			if flagOutput != "" {
				f, err := os.Create(flagOutput)
				if err != nil {
					return fmt.Errorf("failed to create output file: %w", err)
				}
				defer f.Close()
				w = f
			}

			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			enc.SetEscapeHTML(false)
			return enc.Encode(report)
		},
	}

	cmd.Flags().StringVar(&flagStart, "start", "", "Report start datetime (ISO 8601, inclusive)")
	cmd.Flags().StringVar(&flagEnd, "end", "", "Report end datetime (ISO 8601, exclusive)")
	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().BoolVar(&flagRows, "rows", false, "Include normalized rows in output")

	_ = cmd.MarkFlagRequired("start")
	_ = cmd.MarkFlagRequired("end")

	return cmd
}
