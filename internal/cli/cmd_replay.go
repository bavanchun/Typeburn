package cli

import (
	"encoding/json"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/bavanchun/Typeburn/v2/internal/cli/output"
	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

const replaySchemaV1 = 1

// Bounds on the replay file. `replay <path>` is the one place a keystroke log
// arrives from outside the program, so both the read and the timestamps it
// carries are bounded before anything sized from them is allocated.
const (
	// maxReplayBytes bounds the read itself, so a path that names a device or
	// a pipe that never closes cannot be read until the machine gives out.
	maxReplayBytes = 8 << 20

	// maxReplaySpanMs bounds the range the timestamps may cover. Metrics
	// allocate one bucket per second of span, so an unchecked span lets the
	// file's author choose an allocation: a log holding 0 and 2^32 asks for
	// ~200 MiB, and one mixing 0 with epoch milliseconds asks for tens of GiB.
	// A day is far past any real test and still trivially cheap.
	maxReplaySpanMs = 24 * 60 * 60 * 1000
)

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
	data, err := readReplayFile(path)
	if err != nil {
		return replayInput{}, err
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
	if err := validateReplayTimestamps(input.Log, input.EndMs); err != nil {
		return replayInput{}, err
	}
	return input, nil
}

// readReplayFile reads at most maxReplayBytes+1 bytes, so an oversize file is
// reported rather than parsed from a truncated prefix — and never held whole.
func readReplayFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, ioError("could not read replay log: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxReplayBytes+1))
	if err != nil {
		return nil, ioError("could not read replay log: %w", err)
	}
	if len(data) > maxReplayBytes {
		return nil, ioError("replay log is larger than %d bytes", maxReplayBytes)
	}
	return data, nil
}

// validateReplayTimestamps rejects a log whose timestamps could not have come
// from a real run, before any work is sized from them.
//
// A keystroke log is recorded in order against a monotonic clock, so
// non-negative and non-decreasing is not a convention — it is the only shape
// the program can produce. Enforcing it here means the metrics path never has
// to defend against a span it cannot allocate.
func validateReplayTimestamps(log []typing.Keystroke, endMs int64) error {
	first := log[0].TimeMs
	if first < 0 {
		return ioError("keystroke timestamps must be non-negative, got %d", first)
	}
	prev := first
	for i, k := range log {
		if k.TimeMs < prev {
			return ioError("keystroke timestamps must be non-decreasing: index %d is %d after %d", i, k.TimeMs, prev)
		}
		prev = k.TimeMs
	}
	if prev-first > maxReplaySpanMs {
		return ioError("keystroke timestamps span %d ms, more than the %d ms limit", prev-first, maxReplaySpanMs)
	}
	if endMs-first > maxReplaySpanMs {
		return ioError("end_ms is %d ms after the first keystroke, more than the %d ms limit", endMs-first, maxReplaySpanMs)
	}
	return nil
}
