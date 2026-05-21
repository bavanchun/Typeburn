# Brainstorm Summary — typeburn update-check CLI (v2.1.0)

**Date:** 2026-05-21
**Branch target:** `feat/update-check-v2.1`
**Release target:** `v2.1.0` (separate from v2.0.0 currently in flight via PR #18)
**Status:** Approved by user; ready for `/ck:plan --deep`.

---

## 1. Problem statement

Typeburn has 3 documented install paths (`install.sh`, `go install`, Homebrew cask) but the binary itself has no way to tell the user a newer release is available. End-users have to remember to check GitHub Releases manually. Add a **check-and-notify** path inside the CLI — no self-upgrade.

## 2. Locked requirements (5 mandatory)

1. **Expected output**
   - Explicit flag: `typeburn version --check-update` — sync GET to GitHub Releases API, prints upgrade hint (or "up to date").
   - Opportunistic check: on TUI launch when `update_check=on`, render hint on **Result screen footer** (chosen over Home footer; least disruptive, session-boundary).
   - Config key: `update_check` (bool), default `false`, exposed through existing `config set/get/list/path/reset`.
   - Both surfaces support `--json` flag (mirrors v2.0.0 convention).

2. **Acceptance criteria**
   - Never blocks > 1.5s (HTTP timeout); never crashes the TUI on network error; silent-degrade on every error path.
   - 24h cache at `$XDG_STATE_HOME/typeburn/update-check.json`, atomic write (reuse `internal/storage` pattern).
   - Skips entirely when `version.Resolve()` returns `"dev"` or empty (un-stamped builds).
   - Prints all 3 upgrade commands verbatim — no install-method detection.
   - JSON schema is stable and documented.

3. **Out of scope**
   - No self-upgrade / binary replacement.
   - No install-method auto-detection (no `os.Executable()` inspection).
   - No first-run interactive prompt.
   - No telemetry beyond a single GET to `api.github.com/repos/bavanchun/Typeburn/releases/latest` with a `User-Agent: Typeburn/<ver>` header.
   - No background daemon, no scheduled cron.
   - No pre-release / draft tag tracking (latest stable only).

4. **Non-negotiable constraints**
   - Stdlib + existing dep allowlist only (`net/http`, `encoding/json`, `os`, `time`, `path/filepath` — all stdlib). **Zero new `go.mod` entries.**
   - Each file < 200 LOC.
   - `internal/update/` package stays UI-free (no `bubbletea`/`lipgloss` imports). Matches the strict layering rule.
   - Reuse `internal/version` for current; reuse `internal/storage` atomic write helper; reuse `internal/cli/output/` for human/JSON rendering.
   - Semver comparison via internal pure comparator — do NOT add `golang.org/x/mod/semver` (one-call use; comparator is ~30 LOC).
   - `make test-race`, `make lint`, `make size-check`, `notui-noexit-check` must remain green.

5. **Touchpoints**
   - **NEW** `internal/update/client.go` — `FetchLatest(ctx, ver) (Release, error)`; UA header; 1.5s timeout via `http.Client{Timeout}`.
   - **NEW** `internal/update/compare.go` — `Compare(a, b string) int`; strips `v`; ignores prereleases per locked scope.
   - **NEW** `internal/update/cache.go` — XDG state path; atomic JSON write; 24h TTL; `Load()`/`Save(Result)`.
   - **NEW** `internal/update/check.go` — `Check(ctx, currentVer string, force bool) (*Result, error)` orchestrator; honors cache, dev-skip, and the network/cache I/O boundary.
   - **NEW** `internal/update/*_test.go` — table-driven; client tested via `httptest.Server` (no real network in CI).
   - **MODIFY** `internal/cli/cmd_version.go` — add `--check-update` flag wiring + JSON shape.
   - **MODIFY** `internal/config/` — add `UpdateCheck bool` field, default `false`; backwards-compat load (missing field → false).
   - **MODIFY** `internal/cli/cmd_config.go` — register new key in set/get/list (likely zero code if config keys are reflection-driven; otherwise one line).
   - **MODIFY** `internal/cli/cmd_run.go` — synchronous pre-TUI call to `update.Check(...)` when `cfg.UpdateCheck && ver!="dev"`; stash result on the app model via a new `app.WithUpdateHint(*update.Result)` option or via a new `tea.Cmd` that emits `UpdateHintMsg`.
   - **MODIFY** `internal/ui/screen_result.go` (+ view) — render a footer line when an upgrade hint is present.
   - **MODIFY** `internal/app/messages.go` (or `internal/ui/messages.go`) — new domain message `UpdateHintMsg{Latest, URL string}` flowing through root model.
   - **DOCS** `README.md` (Install section: mention `--check-update` + config opt-in), `docs/cli-reference.md` (new flag + config key + JSON schema), `CHANGELOG.md` `[Unreleased]` → moved to `[2.1.0]` at release time, `docs/codebase-summary.md` (new package paragraph).

---

## 3. Evaluated approaches (decision rationale)

### Trigger model
| Option | Trade-off | Verdict |
|---|---|---|
| Explicit flag only | Smallest surface, zero hidden network | ❌ Misses passive discoverability |
| Opportunistic only | Discoverable, but no on-demand path for CI / scripts | ❌ |
| **Both, opt-in default-off** | Largest surface but covers all use cases honestly | ✅ Chosen |

### Data source
| Option | Trade-off | Verdict |
|---|---|---|
| **GitHub Releases API** | Authoritative; one call returns tag + URL + assets; 60 req/hr/IP unauth — fine with cache | ✅ Chosen |
| Static `latest.txt` | Cheaper, no rate limit | ❌ Adds release-pipeline plumbing for no real win |
| `git ls-remote` | No HTTP code | ❌ Assumes `git` binary; brittle for brew/install.sh users |

### Install-method hint
| Option | Trade-off | Verdict |
|---|---|---|
| **Print all 3 verbatim** | Honest, simple, no false claims | ✅ Chosen |
| Detect via `os.Executable()` | More user-friendly but lots of edge cases (custom prefix, symlinks, GOBIN overrides) | ❌ Scope creep |
| Print nothing | Forces user to figure out their own path | ❌ Worse DX |

### Module layout
| Option | Trade-off | Verdict |
|---|---|---|
| **New `internal/update/`** package | Clean boundary; `version` stays pure build-stamp resolver | ✅ Chosen |
| Extend `internal/version/` | One less package, but pollutes a UI-free pure resolver with HTTP/cache I/O | ❌ Breaks layering |

### Opportunistic invocation point
| Option | Trade-off | Verdict |
|---|---|---|
| In `internal/app` model `Init()` | Couples update-check into Elm loop, complicates pure-logic boundary | ❌ |
| Fire-and-forget goroutine | Avoids 1.5s worst-case startup; complicates lifecycle (when/where to print) | ❌ |
| **Synchronous before `tea.NewProgram(...).Run()`** | Predictable; 1.5s worst case → <1ms on cache hit (24h TTL covers ~all launches) | ✅ Chosen |

### Default state
| Option | Trade-off | Verdict |
|---|---|---|
| **Opt-in (OFF by default)** | Aligns with local-only/no-backend posture; no surprise outbound traffic | ✅ Chosen |
| Opt-out (ON by default) | Better discoverability; first-launch network call is a UX/privacy surprise | ❌ |
| First-run prompt | Honest but adds first-run flow complexity | ❌ |

---

## 4. Output shape

### Human (TUI Result footer + `version --check-update`)
```
typeburn v2.1.0 is available (you have v2.0.0).
Release notes: https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0
Upgrade with one of:
  brew upgrade typeburn
  curl -fsSL https://raw.githubusercontent.com/bavanchun/Typeburn/main/install.sh | sh
  go install github.com/bavanchun/Typeburn@latest
```

### JSON (stable schema)
```json
{
  "current": "v2.0.0",
  "latest": "v2.1.0",
  "upgrade_available": true,
  "release_url": "https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0",
  "checked_at": "2026-05-21T07:00:00Z"
}
```

When up-to-date: `upgrade_available=false`, `latest==current`. When network failed (explicit flag only, force=true): non-zero exit + stderr "could not check for updates" + JSON `{"error":"…"}` if `--json`.

---

## 5. Risks & mitigations

| # | Risk | Mitigation |
|---|---|---|
| R1 | GitHub API rate limit on shared NAT IPs | 24h cache + silent-degrade |
| R2 | TUI lifecycle coupling — hint must render *after* session, not mid-render | New `UpdateHintMsg` domain message; root model routes; Result screen renders |
| R3 | Network test flakiness | `httptest.Server`; lint enforces no real-network calls in tests |
| R4 | Semver edge cases (prerelease, +meta) | Comparator filters prerelease/draft tags from API response; never compares them |
| R5 | Scope creep magnet ("auto-update next?") | Lock in `docs/cli-reference.md`: "notify-only, manual install" |
| R6 | First-launch with no `XDG_STATE_HOME` set on macOS | Fall back to `$HOME/.local/state/typeburn/` (matches existing `internal/storage` pattern) |
| R7 | Time skew on `checked_at` (user clock wrong) | TTL comparison uses monotonic delta if available; otherwise just wall-clock — staleness is benign |
| R8 | Stamped binaries built between releases (`v2.0.0-7-gabc123`) | Comparator must accept `vX.Y.Z-N-gSHA` and compare on `vX.Y.Z` portion only |

---

## 6. Success metrics

- `typeburn version --check-update` returns ≤ 1.6s on cold path; ≤ 100ms on cache hit.
- New code: ~250 LOC across 4 source files, all <200 LOC each.
- `go.mod` diff = 0 (no new deps).
- All CI gates green (`test-race`, `lint`, `size-check`, `notui-noexit-check`).
- `make size-check` binary size delta < 100KB (stdlib `net/http` already linked — likely zero delta).
- Manual smoke: feature works against real `api.github.com/repos/bavanchun/Typeburn/releases/latest`.

---

## 7. Locked defaults (deferred to plan; user may override)

- **Flag spelling:** `--check-update` (long-only; no `-u` short form — matches `cmd_run.go` convention).
- **Hint render location:** Result screen footer only (not Home, not both).

---

## 8. Implementation considerations (handed to `/ck:plan --deep`)

Suggested phase decomposition (planner may refine):

1. **Phase 1 — `internal/update/` pure-logic package** (client, compare, cache, check) with table-driven tests via `httptest.Server`. Zero UI deps.
2. **Phase 2 — Config integration**: `UpdateCheck bool` field, backwards-compat load, config CLI surface tested.
3. **Phase 3 — Explicit flag**: `cmd_version.go --check-update` + JSON shape + tests.
4. **Phase 4 — Opportunistic wiring**: pre-TUI synchronous check in `cmd_run.go`; new `UpdateHintMsg`; Result-screen footer render; teatest golden update.
5. **Phase 5 — Docs sync + release-prep**: README, `docs/cli-reference.md`, `docs/codebase-summary.md`, CHANGELOG `[Unreleased]` block ready to land as `[2.1.0]` at tag time.

---

## 9. Dependencies & sequencing

- **Blocked on:** PR #18 (release-notes refresh) merged and v2.0.0 cut — this feature ships in **v2.1.0**, not v2.0.0.
- **Not blocked on:** anything else.
- **Branch off:** `main` once v2.0.0 is tagged (or off the latest `main` if user wants to start in parallel and rebase).

---

## 10. Unresolved questions

- None. (Locked defaults in §7 cover the two design choices the user did not explicitly answer; plan may revisit.)
