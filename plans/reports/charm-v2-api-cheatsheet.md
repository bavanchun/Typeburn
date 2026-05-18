# Charm v2 API Cheatsheet (verified against pinned versions)

Verified by `go doc` against: `charm.land/bubbletea/v2 v2.0.6`, `charm.land/lipgloss/v2 v2.0.3`, `charm.land/bubbles/v2 v2.1.0`, Go 1.26.2. **Module path is `charm.land/...` NOT `github.com/charmbracelet/...`.** Trust this over training-data v1 habits.

## bubbletea/v2 (`import tea "charm.land/bubbletea/v2"`)

- `tea.Model` interface: `Init() Cmd`, `Update(Msg) (Model, Cmd)`, `View() View`.
- `tea.View` is a **struct**, not a string. Fields: `Content string`, `Cursor *tea.Cursor`, `BackgroundColor color.Color`, `OnMouse func(MouseMsg) Cmd`. Construct via `tea.NewView(s string) View` or `tea.View{Content: s}`.
- Key events: `case tea.KeyPressMsg:` then `k := msg.Key()` → `tea.Key{ Code rune; Text string; Mod tea.KeyMod; ShiftedCode rune }`. (`tea.KeyReleaseMsg` also exists; ignore unless needed.)
  - Printable char: `k.Text` is non-empty and holds the character(s). Use `k.Text` for typed runes, not `k.Code`, for printables.
  - Special keys are rune constants: `tea.KeyTab, tea.KeyEnter, tea.KeyEsc, tea.KeyEscape, tea.KeyBackspace, tea.KeySpace, tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight, tea.KeyHome, tea.KeyEnd, tea.KeyPgUp, tea.KeyPgDown`.
  - Modifiers (bitmask `tea.KeyMod`): `tea.ModCtrl, tea.ModShift, tea.ModAlt`. Test exact-match `k.Mod == tea.ModCtrl` (the project's config.Keymap already does this — reuse `config.Keymap` + `Binding.Matches(k tea.Key)`).
  - ctrl+c arrives as `Code:'c', Mod:tea.ModCtrl`.
- Paste: `case tea.PasteMsg:` has **`.Content` (string)** — NOT `.Data` (verified in Phase 9). (Project decision: feed paste runes sequentially through the typing engine.)
- `tea.WindowSizeMsg{ Width, Height int }` — sent once at start and on every resize.
- Commands: `tea.Quit` (a Cmd-returning func; `return m, tea.Quit`). `tea.QuitMsg` is the message type (used in tests: `cmd().(tea.QuitMsg)`). `tea.Batch(cmds...)`, `tea.Tick(d time.Duration, fn func(time.Time) tea.Msg) tea.Cmd` — fn receives the fire time; use wall-clock deltas (`time.Now()`/the passed time), never tick counts.
- Program: `tea.NewProgram(model, opts...) *Program`; `p.Run() (Model, error)`. v2 Cursed Renderer handles flicker-free/alt-screen automatically — do NOT manually toggle alt screen.
- Update return type is `(tea.Model, tea.Cmd)`; when delegating to sub-models, type-assert back as needed. Sub-model pattern in this project: root `app.Model` holds screen sub-models; root `Update` routes by `m.screen`, calls sub-model `Update`, stores returned sub-model.

## lipgloss/v2 (`import "charm.land/lipgloss/v2"`)

- `lipgloss.Color(s string) color.Color` — **a constructor, not a type**. Accepts `"#RRGGBB"` or ANSI `"240"`. Map values must be typed `map[K]color.Color` (`import "image/color"`), NOT `map[K]lipgloss.Color`.
- `lipgloss.NewStyle() Style`; chain: `.Foreground(c color.Color)`, `.Background(c color.Color)`, `.Bold(bool)`, `.Faint(bool)`, `.Italic(bool)`, `.Underline(bool)`, `.Reverse(bool)`, `.Width(int)`, `.Align(...Position)`, `.Padding(...)`, `.Margin(...)`, `.Border(...)`. Render with `.Render(s string) string`.
- `lipgloss.Place(w, h int, hPos, vPos lipgloss.Position, s string, opts...) string` — centering.
- `lipgloss.Position` consts: `Top=0.0, Bottom=1.0, Left=0.0, Right=1.0, Center=0.5`.
- `lipgloss.JoinHorizontal(pos, strs...)`, `lipgloss.JoinVertical(pos, strs...)` for layout composition.
- Use the project's `theme.Theme.Style(role)` / `theme.Theme.Color(role)` — never hardcode colors in screens (role-based theming + NO_COLOR swap already handled).

## Reuse, don't reinvent (existing project APIs)

- `monkeytype-tui/internal/config`: `config.Mode` (`ModeTime/ModeWords/ModeQuote`), `config.LengthsFor(mode) []int`, `config.Settings`, `config.Defaults()`, `config.Keymap`, `config.DefaultKeymap()`, `Binding.Matches(tea.Key) bool`, `config.ConfigDir()/DataDir()`.
- `monkeytype-tui/internal/theme`: `theme.Role` consts, `theme.Theme.Style(Role) lipgloss.Style`, `.Color(Role) color.Color`, `theme.Load(name, noColor)`, `theme.Available()`, `theme.EnvNoColor()`.
- `monkeytype-tui/internal/typing`: `typing.Engine` (`New`, `Apply(r rune, nowMs int64)`, `Backspace(nowMs)`, `States() []CharState`, `Complete(nowMs) bool`, `Log() []Keystroke`, `Progress() (done,total int)`), `typing.CharState` (Untyped/Correct/Incorrect/IncorrectSpace/Extra/Current).
- `monkeytype-tui/internal/metrics`: `metrics.Compute(log, mode, endMs) metrics.Result` (NetWPM, RawWPM, Accuracy, Consistency, CPS, char counts, PerSecond[]).
- `monkeytype-tui/internal/words`: `words.NewGenerator(seed int64)`, `g.Words(n) string`, `g.TimeBuffer() string`, `words.ForMode(g, mode, length, ql) string`, `words.Quote(QuoteLen) Quote{Text,Source}`.

## Project conventions

- Files <200 lines, kebab-case names; tests `*_test.go` (Go-required underscore). Doc comment on every exported symbol. No mocks/fakes. `gofmt -w` before done. Verify with `go build ./... && go vet ./... && go test ./... -race && gofmt -l internal` (last must be empty).
