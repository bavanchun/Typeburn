package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/bavanchun/Typeburn/v2/internal/app"
	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/ui"
	"github.com/bavanchun/Typeburn/v2/internal/words"
)

type runFlags struct {
	mode     string
	duration int
	words    int
	quoteLen string
	theme    string
	text     string
	noTUI    bool
	json     bool
}

type runRequest struct {
	mode     config.Mode
	length   int
	quoteLen words.QuoteLen
	theme    string
	textPath string
	noTUI    bool
	json     bool
}

func newRunCmd(e env) *cobra.Command {
	var flags runFlags
	cmd := &cobra.Command{
		Use:           "run",
		Short:         "Start a typing test",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := buildRunRequest(cmd, flags, e.loadSettings())
			if err != nil {
				return err
			}
			return runTUICommand(cmd.Context(), e, req)
		},
	}
	cmd.Flags().StringVar(&flags.mode, "mode", "", "test mode: time, words, quote, code")
	cmd.Flags().IntVar(&flags.duration, "duration", 0, "duration in seconds for time mode")
	cmd.Flags().IntVar(&flags.words, "words", 0, "word count for words mode")
	cmd.Flags().StringVar(&flags.quoteLen, "quote-len", "", "quote length: short, medium, long")
	cmd.Flags().StringVar(&flags.theme, "theme", "", "temporary theme override")
	cmd.Flags().StringVar(&flags.text, "text", "", "file to load for code mode (\"-\" = stdin)")
	cmd.Flags().BoolVar(&flags.noTUI, "no-tui", false, "run in raw terminal mode")
	cmd.Flags().BoolVar(&flags.json, "json", false, "emit JSON result with --no-tui")
	return cmd
}

func runTUICommand(ctx context.Context, e env, req runRequest) error {
	if req.noTUI {
		return runNoTUI(ctx, e, req)
	}
	settings := e.loadSettings()
	settings.DefaultMode = req.mode
	if req.length > 0 {
		settings.DefaultLength = req.length
	}
	themeName := settings.Theme
	if req.theme != "" {
		themeName = req.theme
	}
	codeText := ""
	if req.mode == config.ModeCode {
		text, err := e.loadCode(req.textPath)
		if err != nil {
			return ioError("%s", codeHintFor(err))
		}
		codeText = text
	}
	hint := resolveUpdateHint(ctx, settings)
	model := app.New(theme.Load(themeName, theme.EnvNoColor()), settings, codeText, "", hint)
	start := ui.StartTestMsg{
		Mode: req.mode, Length: req.length, QuoteLen: req.quoteLen, CodeText: codeText,
	}
	next, cmd := model.Update(start)
	if cmd != nil {
		if msg := cmd(); msg != nil {
			next, _ = next.Update(msg)
		}
	}
	return e.runModel(ctx, next)
}
