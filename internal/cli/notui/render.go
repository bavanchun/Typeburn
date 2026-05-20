package notui

import (
	"fmt"
	"io"

	"github.com/bavanchun/Typeburn/internal/metrics"
)

func RenderPrompt(w io.Writer, target string) {
	fmt.Fprintf(w, "typeburn raw test\n\n%s\n\n", target)
}

func RenderStatus(w io.Writer, done, total int, liveWPM float64) {
	fmt.Fprintf(w, "\rprogress %d/%d  wpm %.0f\033[K", done, total, liveWPM)
}

func RenderSummary(w io.Writer, result metrics.Result) {
	fmt.Fprintf(w, "\n\nnet %.2f wpm  raw %.2f wpm  acc %.2f%%  cons %.2f%%\n",
		result.NetWPM, result.RawWPM, result.Accuracy, result.Consistency)
}
