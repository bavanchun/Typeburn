package cli

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/bavanchun/Typeburn/internal/cli/notui"
	"github.com/bavanchun/Typeburn/internal/cli/output"
	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/runner"
)

func runNoTUI(ctx context.Context, e env, req runRequest) error {
	in, ok := e.stdin.(*os.File)
	if !ok {
		return usageError("stdin is not a terminal")
	}
	session, err := runSession(e, req)
	if err != nil {
		return err
	}
	write := func(w io.Writer, result metrics.Result) error {
		if req.json {
			return output.RenderJSON(w, newMetricOutput(result))
		}
		notui.RenderSummary(w, result)
		return nil
	}
	err = notui.Run(ctx, in, e.stdout, session, write)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, notui.ErrAbort):
		return abortError("aborted")
	case errors.Is(err, notui.ErrNotTerminal):
		return usageError("stdin is not a terminal")
	default:
		return ioError("%w", err)
	}
}

func runSession(e env, req runRequest) (runner.Session, error) {
	settings := e.loadSettings()
	if req.mode == config.ModeCode {
		text, err := e.loadCode(req.textPath)
		if err != nil {
			return runner.Session{}, ioError("%s", codeHintFor(err))
		}
		return runner.NewCodeSession(text, settings.StrictMode), nil
	}
	return runner.NewSession(req.mode, req.length, req.quoteLen, 0, settings.StrictMode), nil
}
