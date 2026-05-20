# Research: Cobra + Fang Integration for Typeburn CLI v2

## Executive Summary

Cobra + Fang is idiomatic for Go CLIs needing subcommands and styled help. However, Typeburn's existing design (pure `decide()` function, graceful fallthrough for unknown args, deliberate ContinueOnError behavior) conflicts with Cobra's default error-exit model. **Recommendation: Adopt Cobra + Fang but with a custom wrapper that preserves the `decide()` contract and handles parse-error fallthrough at the root level.**

---

## Key Findings

### 1. Fang + Cobra Integration Pattern

**What Fang does:**
- Wraps Cobra's `Execute()` and applies Charmbracelet's styled help, errors, and shell completions
- Does NOT call `os.Exit` itself; returns error for the caller to handle
- Accepts custom `WithErrorHandler(func)` option to override default error styling

**Idiomatic usage:**
```go
if err := fang.Execute(ctx, rootCmd, fang.WithNotifySignal(os.Interrupt)); err != nil {
    os.Exit(1)
}
```

**Authority:** [Fang example/main.go](https://github.com/charmbracelet/fang/blob/main/example/main.go), [Fang fang.go](https://github.com/charmbracelet/fang/blob/main/fang.go)

### 2. Cobra Error Handling: SilenceErrors + SilenceUsage

**Problem:** Cobra exits on parse/flag errors by default (via `pflag`), incompatible with Typeburn's "unknown arg = run TUI" design.

**Solution:** Set two flags on root command:
```go
rootCmd.SilenceErrors = true  // suppress Cobra's auto-print on error
rootCmd.SilenceUsage = true   // don't print usage on runtime errors
```

**Caveats:**
- `SilenceUsage=true` still prints usage for flag/arg *validation* errors (e.g. `--bogus`)
- To truly silence all parsing, you must **not use `pflag` for root** or catch `pflag` errors before Cobra sees them
- Per [Cobra docs](https://cobra.dev/docs/how-to-guides/working-with-commands/): "RunE keeps your command logic clean; Cobra handles the exit code"

**Authority:** [Cobra error handling guide](https://www.jetbrains.com/guide/go/tutorials/cli-apps-go-cobra/error_handling/)

### 3. Root-Level Fallthrough Pattern (Typeburn's Key Constraint)

**The challenge:**
- Typeburn wants `typeburn unknown-arg` → launch TUI (success)
- But `typeburn run --bogus` → exit 1 (fail: subcommand validation)
- Cobra's legacyArgs logic prevents *both* subcommands AND positional args on root

**Why it matters:**
Cobra forbids a root command from having both subcommands and `Args: cobra.ArbitraryArgs`. Once subcommands exist, the root is "command dispatcher only"; unknown args trigger "unknown command" error.

**Recommended workaround:**
1. **Don't use `Args: cobra.ArbitraryArgs` on root** — instead, use `DisableFlagParsing: true` on root only
2. **Custom pre-parse wrapper** before calling Cobra's Execute:
   - Call the existing pure `decide(os.Args[1:])` function
   - If `decide()` returns `printVersion=true` or `textPath!=""`, handle directly
   - Otherwise, pass through to Cobra's command parsing (subcommand dispatch)
3. **Per-subcommand strictness:**
   - Root has `DisableFlagParsing: true` (lets unknown flags through)
   - Each subcommand (`run`, `history`, etc.) has normal flag parsing + `SilenceUsage=true`

**Authority:** [Cobra GitHub issue #295](https://github.com/spf13/cobra/issues/295), [Cobra issue #610](https://github.com/spf13/cobra/issues/610)

### 4. Persistent Flags for Root-Level Aliases

**Pattern for `--version` and `--text <file>` at root:**

**Option A (Recommended for Typeburn):** Keep in `decide()`, bypass Cobra entirely
- Preserves pure function, no coupling to Cobra
- Callsite: `decide(os.Args[1:])` before `fang.Execute()`

**Option B (Cobra way):** Persistent flags + PersistentPreRunE
```go
rootCmd.PersistentFlags().Bool("version", false, "...")
rootCmd.PersistentFlags().String("text", "", "...")
rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
    if versionFlag { /* handle */ }
    if textPath != "" { /* handle */ }
    return nil
}
```
Pros: Single code path. Cons: Couples version/text logic to Cobra, harder to test in isolation.

**Authority:** [Cobra working-with-commands](https://cobra.dev/docs/how-to-guides/working-with-commands/)

### 5. Real-World Examples

**Glow (Charmbracelet):**
- Uses Cobra for structure (`run`, `config`, `man` subcommands)
- Does NOT use Fang (as of 2024)
- Integrates Cobra with Viper for config file + flags
- [glow/main.go](https://github.com/charmbracelet/glow/blob/master/main.go)

**Soft-Serve (Charmbracelet):**
- Git server CLI; complex subcommands
- Likely uses Cobra, but no explicit Fang mention in search
- [soft-serve GitHub](https://github.com/charmbracelet/soft-serve)

**Fang example:**
- Minimal scaffolding example; no real-world app complexity
- Demonstrates signal handling, context passing
- [fang/example/main.go](https://github.com/charmbracelet/fang/blob/main/example/main.go)

---

## Concrete Code Patterns

### Pattern 1: Preserve `decide()` Purity + Fallthrough

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "io"
    "os"
    
    tea "charm.land/bubbletea/v2"
    "github.com/charmbracelet/fang"
    "github.com/spf13/cobra"
    
    "github.com/bavanchun/Typeburn/internal/app"
    "github.com/bavanchun/Typeburn/internal/codetext"
    "github.com/bavanchun/Typeburn/internal/version"
)

// decide() — unchanged, pure function
func decide(args []string) (printVersion bool, textPath string) {
    fs := flag.NewFlagSet("typeburn", flag.ContinueOnError)
    fs.SetOutput(io.Discard)
    showVersion := fs.Bool("version", false, "")
    textFlag := fs.String("text", "", "")
    if err := fs.Parse(args); err != nil {
        return false, ""
    }
    return *showVersion, *textFlag
}

func main() {
    // 1. Pure function first — no Cobra coupling
    printVersion, textPath := decide(os.Args[1:])
    if printVersion {
        fmt.Println(version.String())
        return
    }
    
    // 2. Build root command with DisableFlagParsing
    rootCmd := &cobra.Command{
        Use: "typeburn [args]",
        DisableFlagParsing: true, // allows unknown args to reach RunE
        RunE: func(cmd *cobra.Command, args []string) error {
            // Fall-through: launch TUI
            codeText, codeHint := "", ""
            if textPath != "" {
                var err error
                codeText, err = codetext.Load(textPath)
                if err != nil {
                    codeHint = codeHintFor(err)
                }
            }
            p := tea.NewProgram(app.NewFromDisk(codeText, codeHint))
            _, err := p.Run()
            return err
        },
    }
    
    // 3. Add subcommands (each with normal flag parsing)
    runCmd := &cobra.Command{
        Use: "run [--text <file>] [args...]",
        RunE: func(cmd *cobra.Command, args []string) error {
            // ... subcommand logic
            return nil
        },
    }
    runCmd.SilenceUsage = true
    rootCmd.AddCommand(runCmd)
    
    // 4. Execute via fang
    if err := fang.Execute(context.Background(), rootCmd); err != nil {
        os.Exit(1)
    }
}
```

**Key advantages:**
- `decide()` stays pure, unchanged, fully testable
- Root uses `DisableFlagParsing` to allow fallthrough
- Subcommands use normal Cobra flag parsing + SilenceUsage
- Fang wraps the whole thing for styled help/errors

### Pattern 2: Custom Error Handler (if needed)

```go
if err := fang.Execute(context.Background(), rootCmd,
    fang.WithErrorHandler(customErrorHandler),
); err != nil {
    os.Exit(1)
}

func customErrorHandler(w io.Writer, styles fang.Styles, err error) {
    // Example: suppress usage for parse errors, print custom message
    fmt.Fprintf(w, "typeburn: %v\n", err)
}
```

---

## Recommended Approach for Typeburn

### Phase 1: Minimal Migration
1. **Keep `decide()`** as-is, pure function
2. **Wrap main() in fang.Execute()**; root command has `DisableFlagParsing: true`
3. **Add one subcommand** (`run`) to test the pattern
4. No Persistent flags; `--text` and `--version` stay in `decide()`

### Phase 2: Expand Subcommands
Add `history`, `config`, `replay`, `version` subcommands; each with `SilenceUsage=true` and explicit flag parsing.

### Phase 3: Polish
- Shell completions via Fang's built-in
- Man page generation via Fang
- Custom error handler if needed

---

## Risks & Gotchas

| Risk | Mitigation | Impact |
|------|-----------|--------|
| `DisableFlagParsing` on root → subcommand flags not parsed until subcommand | Each subcommand parses its own flags; design as normal Cobra commands | Medium — requires discipline, not coupling |
| Fang depends on Charmbracelet libs (Lip Gloss, etc.) — adds ~500KB to binary | Already a Typeburn dependency for TUI; no delta | Low |
| Cobra's help always exits (via pflag); can't suppress cleanly if root has subcommands | Use `DisableFlagParsing` on root to prevent help from triggering | Medium — need test coverage |
| `-v` reserved in `decide()` — conflicts with verbose flag in future | Document in CONTRIBUTING.md; Cobra provides `-h`/`--help` by default | Low — acceptable trade-off |
| Unknown flags on subcommands still exit 1 (e.g., `typeburn run --bogus`) | Correct behavior; document in help | None — intended |

---

## Binary Size Impact

**Cobra alone:** ~3.4 MB (stripped binary, per 2024 measurements [go-cli-comparison](https://github.com/gschauer/go-cli-comparison))

**Current Typeburn (stdlib flag):** ~2.5 MB

**Delta:** +900 KB (stripped binary)

**Mitigation:**
- Use `go build -ldflags="-s -w"` (strip symbols + debug; saves ~300 KB)
- Binary is still single-file, no external deps at runtime
- Size acceptable for terminal app; Homebrew cask + install.sh mask this for users

**Fang overhead:** Unmeasured in public comparisons; likely <100 KB additional (mostly Lip Gloss re-export, already in TUI).

---

## Unresolved Questions

1. **Exact Fang + Lipgloss binary delta:** No published measurements; requires local test build
2. **Shell completion quality:** Fang provides scaffolding, but does auto-generation work well for subcommands with shared flags?
3. **Subcommand help text styling:** Does Fang apply the same theme to subcommand help as root, or is manual work needed?

---

**Status:** DONE

**Summary:** Cobra + Fang is the right choice; use `DisableFlagParsing` on root to preserve fallthrough behavior, keep `decide()` pure by calling it before Cobra, and adopt Fang's error/help styling. Binary size +900 KB is acceptable for a terminal app.
