package updateui

import (
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/update"
)

// pollInterval is how often the model samples the update's progress. The update
// runs on its own goroutine and publishes state, so the renderer reads at its
// own cadence instead of being driven by the download's write pattern.
const pollInterval = 40 * time.Millisecond

// Snapshot returns the update's current progress. Progress is state rather than
// an event stream, so the newest value is always the correct one — nothing can
// be missed by sampling.
type Snapshot func() update.Progress

// Result reports the terminal outcome of the run the model was watching.
type Result struct {
	Outcome update.Outcome
	Err     error
}

// Model renders the framed progress block for a single `typeburn update` run.
// It does not own the update itself: the caller runs update.Apply and supplies
// a Snapshot, which keeps this package free of download and swap concerns.
type Model struct {
	from, to string
	theme    theme.Theme
	noColor  bool

	spin spinner.Model
	bar  progress.Model

	snapshot Snapshot
	results  <-chan Result

	cur    update.Progress
	done   bool
	result Result

	// settled records that a terminal state actually reached the model.
	// Bubble Tea's event loop returns directly on QuitMsg (SIGTERM) and
	// InterruptMsg without consulting the model, so without this flag a killed
	// program would surface a zero Result that reads as a successful update.
	settled bool
}

// tickMsg drives the progress poll.
type tickMsg time.Time

// resultMsg carries the finished run's outcome into the event loop.
type resultMsg Result

// New builds the model. results must deliver exactly one value when the update
// finishes, successfully or not.
func New(from, to string, th theme.Theme, snapshot Snapshot, results <-chan Result) Model {
	return Model{
		from:     from,
		to:       to,
		theme:    th,
		noColor:  isNoColor(th),
		spin:     newSpinner(),
		bar:      newBar(th),
		snapshot: snapshot,
		results:  results,
	}
}

// Result reports what the watched run returned, and whether a terminal state
// reached the model at all. A false second return means the program was killed
// out from under the update — the caller must not report success.
func (m Model) Result() (Result, bool) { return m.result, m.settled }

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, tick(), waitForResult(m.results))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tickMsg:
		m.cur = m.snapshot()
		return m, tea.Batch(tick(), m.bar.SetPercent(m.percent()))

	case resultMsg:
		m.done = true
		m.settled = true
		m.result = Result(msg)
		// Drive the bar to full so the last frame does not contradict the
		// success message the caller prints next. SetPercent only schedules one
		// spring frame, so this lands the target rather than animating all the
		// way to it — the run is over, and stalling the exit for the spring to
		// settle would be worse.
		return m, tea.Sequence(m.bar.SetPercent(1), tea.Quit)

	case progress.FrameMsg:
		bar, cmd := m.bar.Update(msg)
		m.bar = bar
		return m, cmd

	case spinner.TickMsg:
		sp, cmd := m.spin.Update(msg)
		m.spin = sp
		return m, cmd
	}
	return m, nil
}

// handleKey implements the one interaction the block offers: cancelling.
//
// Cancellation is refused from StageInstalling onward. What remains at that
// point is the extract-and-atomically-rename that the whole updater's safety
// rests on; interrupting it is the only way this feature could leave a user
// without a working binary. The caller's context is cancelled by the program
// exiting, so the guard has to live here.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() != "ctrl+c" && msg.String() != "esc" {
		return m, nil
	}
	// Read the live stage, not m.cur. m.cur is only refreshed on the 40ms poll,
	// and Apply reports StageInstalling immediately before extracting and
	// renaming — a stale read would accept a cancel inside exactly the window
	// this guard exists to close.
	if m.liveStage() >= update.StageInstalling {
		return m, nil
	}
	// Deliberately not setting done: the run did not finish, so the frame must
	// not paint every row as complete.
	m.result = Result{Err: ErrCancelled}
	m.settled = true
	return m, tea.Quit
}

// liveStage samples the update directly, falling back to the last polled value
// if no snapshot was supplied.
func (m Model) liveStage() update.Stage {
	if m.snapshot == nil {
		return m.cur.Stage
	}
	return m.snapshot().Stage
}

// percent is the download's completion fraction. A Total of 0 means the server
// sent no Content-Length, so there is no honest ratio to show — hold at zero
// and let the spinner carry the activity.
func (m Model) percent() float64 {
	if m.cur.Stage != update.StageDownloading || m.cur.Total <= 0 {
		if m.cur.Stage > update.StageDownloading {
			return 1
		}
		return 0
	}
	return float64(m.cur.Done) / float64(m.cur.Total)
}

// barView renders the bar, or a bare indeterminate track when the total size is
// unknown — never a fabricated percentage.
func (m Model) barView() string {
	if m.cur.Total <= 0 {
		return m.style(theme.RoleTextFaint).Render("· · ·")
	}
	return m.bar.View()
}

func tick() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func waitForResult(ch <-chan Result) tea.Cmd {
	return func() tea.Msg { return resultMsg(<-ch) }
}
