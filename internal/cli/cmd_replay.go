package cli

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/bavanchun/Typeburn/v2/internal/cli/output"
	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

const replaySchemaV1 = 1

type replayInput struct {
	SchemaVersion int                `json:"schema_version"`
	Mode          config.Mode        `json:"mode"`
	EndMs         int64              `json:"end_ms"`
	Log           []typing.Keystroke `json:"log"`
}

type replayOutput struct {
	SchemaVersion int          `json:"schema_version"`
	Mode          config.Mode  `json:"mode"`
	EndMs         int64        `json:"end_ms"`
	Result        metricOutput `json:"result"`
}

func newReplayCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:           "replay <log.json>",
		Short:         "Replay a keystroke log and compute metrics",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := loadReplayInput(args[0])
			if err != nil {
				return err
			}
			result := metrics.Compute(input.Log, input.Mode, input.EndMs)
			if asJSON {
				return output.RenderJSON(cmd.OutOrStdout(), replayOutput{
					SchemaVersion: replaySchemaV1,
					Mode:          input.Mode,
					EndMs:         input.EndMs,
					Result:        newMetricOutput(result),
				})
			}
			return output.RenderTable(cmd.OutOrStdout(), []string{"metric", "value"}, metricTableRows(result))
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print replay result as JSON")
	return cmd
}

func loadReplayInput(path string) (replayInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return replayInput{}, ioError("could not read replay log: %w", err)
	}
	var input replayInput
	if err := json.Unmarshal(data, &input); err != nil {
		return replayInput{}, ioError("malformed replay log: %w", err)
	}
	if input.SchemaVersion != replaySchemaV1 {
		return replayInput{}, ioError("unsupported schema version %d", input.SchemaVersion)
	}
	if !validMode(input.Mode) {
		return replayInput{}, ioError("invalid replay mode %q", input.Mode)
	}
	if input.EndMs < 0 {
		return replayInput{}, ioError("end_ms must be non-negative")
	}
	if len(input.Log) == 0 {
		return replayInput{}, ioError("replay log is empty")
	}
	return input, nil
}
