package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/bavanchun/Typeburn/internal/cli/output"
	"github.com/bavanchun/Typeburn/internal/storage"
)

func newHistoryCmd(e env) *cobra.Command {
	var limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use:           "history",
		Short:         "Print saved typing history",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return usageError("--limit must be positive")
			}
			records := newestRecords(e.loadHistory(), limit)
			if asJSON {
				return output.RenderJSON(cmd.OutOrStdout(), records)
			}
			if len(records) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no history yet")
				return nil
			}
			return output.RenderTable(cmd.OutOrStdout(), historyHeaders(), historyRows(records))
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "maximum records to print")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print records as JSON")
	return cmd
}

func newestRecords(records []storage.Record, limit int) []storage.Record {
	out := make([]storage.Record, len(records))
	copy(out, records)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Time.After(out[j].Time)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func historyHeaders() []string {
	return []string{"when", "mode", "len", "wpm", "acc", "cons"}
}

func historyRows(records []storage.Record) [][]string {
	rows := make([][]string, 0, len(records))
	for _, r := range records {
		rows = append(rows, []string{
			r.Time.Format("2006-01-02T15:04:05Z07:00"),
			r.Mode,
			fmt.Sprint(r.Length),
			fmt.Sprint(r.WPM),
			fmt.Sprintf("%.0f%%", r.Accuracy),
			fmt.Sprintf("%.0f%%", r.Consistency),
		})
	}
	return rows
}
