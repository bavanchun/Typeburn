package notui

import (
	"bufio"
	"context"
	"io"
	"os"
	"time"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/runner"
)

type ResultWriter func(io.Writer, metrics.Result) error

func Run(ctx context.Context, in *os.File, out io.Writer, session runner.Session, write ResultWriter) error {
	return runRaw(ctx, in, out, func(r io.Reader, w io.Writer) error {
		return runLoop(r, w, session, write, time.Now)
	})
}

func runLoop(
	in io.Reader,
	out io.Writer,
	session runner.Session,
	write ResultWriter,
	now func() time.Time,
) error {
	RenderPrompt(out, session.Target)
	rd := bufio.NewReader(in)
	for {
		ev, err := ReadEvent(rd)
		if err != nil {
			return err
		}
		nowMs := now().UnixMilli()
		switch ev.Kind {
		case EventNone:
			continue
		case EventAbort:
			return ErrAbort
		case EventBackspace:
			session.Engine.Backspace(nowMs)
		case EventRune:
			session.Engine.Apply(ev.Rune, nowMs)
		}
		done, total := session.Engine.Progress()
		elapsed := nowMs - session.Engine.StartMs()
		live := metrics.Compute(session.Engine.Log(), session.Mode, nowMs).NetWPM
		if elapsed <= 0 {
			live = 0
		}
		RenderStatus(out, done, total, live)
		if completed(session, nowMs) {
			result := metrics.Compute(session.Engine.Log(), session.Mode, endMs(session, nowMs))
			if write != nil {
				return write(out, result)
			}
			RenderSummary(out, result)
			return nil
		}
	}
}

func completed(session runner.Session, nowMs int64) bool {
	if session.Mode == config.ModeTime {
		start := session.Engine.StartMs()
		return start > 0 && nowMs-start >= int64(session.Length*1000)
	}
	return session.Engine.Complete(nowMs)
}

func endMs(session runner.Session, nowMs int64) int64 {
	if session.Mode == config.ModeTime {
		return session.Engine.StartMs() + int64(session.Length*1000)
	}
	return nowMs
}
