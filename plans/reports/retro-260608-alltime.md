# Retrospective Report — All-Time

**Period:** 2026-05-18 → 2026-06-08
**Generated:** 2026-06-08
**Repo:** bavanchun/Typeburn
**Authors:** 1 active (2 emails, same person)

---

## Velocity

| Metric | Value |
|--------|-------|
| Total commits | 69 |
| Commits / day | 3.3 (`69 / 21 days`) |
| Active days | 9 / 21 (42.9%) |
| Files changed (unique) | 389 |

### Commits per Day

```
2026-05-18  █████████████████████████████████ 33
2026-05-19  ██████████ 10
2026-05-20  ████  4
2026-05-21  ████  4
2026-05-22  ██  2
2026-05-25  █████  5
2026-05-29  █████  5
2026-05-30  █████  5
2026-06-08  █  1
```

> [!NOTE]
> 62% of all commits (43/69) landed in the first 2 days — classic "greenfield burst" pattern.
> 12 of 21 calendar days had zero commits. Activity clusters around May 18-22 and May 25-30.

---

## Code Health

| Metric | Value |
|--------|-------|
| LOC added | 50,436 |
| LOC removed | 1,677 |
| Net LOC | +48,759 |
| Churn rate | 1.07x (`(50436 + 1677) / 48759`) |
| Test-to-code ratio | 17.8% (`138 test-file changes / 777 total file changes`) |

> [!TIP]
> Churn rate of 1.07x is healthy — nearly all code added is net-new (greenfield project). Very little rework.

---

## Commit Distribution

```
docs       ████████████████████ 20 (29%)
feat       ███████████████████  19 (28%)
chore      ████████████         12 (17%)
fix        ███████               7 (10%)
ci         ██                    2  (3%)
build      ██                    2  (3%)
refactor   ██                    2  (3%)
other      ████                  5  (7%)
```

> [!NOTE]
> `docs` is the #1 commit type — reflects heavy plan/changelog/README maintenance alongside feature work.

---

## File Hotspots ⚠️

**Primary deliverable.** Top files by change frequency across the full project history.

### All Files — Top 15

| Rank | File | Changes | Category |
|------|------|---------|----------|
| 1 | `docs/project-roadmap.md` | 21 | docs |
| 2 | `CHANGELOG.md` | 16 | docs |
| 3 | `internal/app/model.go` | **14** | **source** |
| 4 | `.github/release-notes.md` | 14 | docs |
| 5 | `README.md` | 13 | docs |
| 6 | `plans/…/plan.md` | 11 | plan |
| 7 | `docs/codebase-summary.md` | 11 | docs |
| 8 | `docs/system-architecture.md` | 8 | docs |
| 9 | `internal/ui/screen_typing.go` | **7** | **source** |
| 10 | `internal/ui/screen_typing_actions.go` | **7** | **source** |
| 11 | `internal/app/model_view.go` | **7** | **source** |
| 12 | `CLAUDE.md` | 7 | config |
| 13 | `main.go` | **6** | **source** |
| 14 | `internal/ui/screen_result_view.go` | **6** | **source** |
| 15 | `internal/ui/messages.go` | **6** | **source** |

### Source-Only Hotspots — Top 20

| Rank | File | Changes | Has Test? |
|------|------|---------|-----------|
| 1 | `internal/app/model.go` | 14 | ✅ `model_test.go` |
| 2 | `internal/ui/screen_typing.go` | 7 | ✅ `screen_typing_test.go` |
| 3 | `internal/ui/screen_typing_actions.go` | 7 | ❌ |
| 4 | `internal/app/model_view.go` | 7 | ❌ |
| 5 | `main.go` | 6 | ❌ |
| 6 | `internal/ui/screen_result_view.go` | 6 | ❌ |
| 7 | `internal/ui/messages.go` | 6 | ❌ |
| 8 | `internal/app/model_settings.go` | 6 | ❌ |
| 9 | `internal/app/model_history.go` | 6 | ❌ |
| 10 | `internal/ui/word_stream_renderer.go` | 5 | ❌ |
| 11 | `internal/ui/screen_typing_view.go` | 5 | ❌ |
| 12 | `internal/ui/screen_result.go` | 5 | ✅ `screen_result_test.go` |
| 13 | `internal/ui/screen_history_view.go` | 5 | ❌ |
| 14 | `internal/typing/completion.go` | 5 | ❌ |
| 15 | `internal/storage/new_best.go` | 5 | ✅ (package tests) |
| 16 | `internal/metrics/compute.go` | 5 | ✅ (package tests) |
| 17 | `internal/config/settings.go` | 5 | ✅ (package tests) |
| 18 | `internal/ui/screen_home.go` | 4 | ✅ (via smoke) |
| 19 | `internal/typing/engine.go` | 4 | ✅ (package tests) |
| 20 | `internal/ui/typing_log_helpers.go` | 4 | ❌ |

---

## High-Churn Files With No Test Coverage

> [!WARNING]
> These files have ≥4 changes in git history **and** no corresponding `_test.go` file. Top hardening candidates.

| File | Changes | Risk |
|------|---------|------|
| `internal/ui/screen_typing_actions.go` | 7 | 🔴 High — core typing input handler |
| `internal/app/model_view.go` | 7 | 🔴 High — main View() dispatch |
| `main.go` | 6 | 🟡 Medium — entry point, hard to unit test |
| `internal/ui/screen_result_view.go` | 6 | 🟡 Medium — render logic |
| `internal/ui/messages.go` | 6 | 🟡 Medium — message type definitions |
| `internal/app/model_settings.go` | 6 | 🔴 High — settings mutation logic |
| `internal/app/model_history.go` | 6 | 🔴 High — history state management |
| `internal/ui/word_stream_renderer.go` | 5 | 🟡 Medium — visual rendering |
| `internal/ui/screen_typing_view.go` | 5 | 🟡 Medium — visual rendering |
| `internal/ui/screen_history_view.go` | 5 | 🟡 Medium — visual rendering |
| `internal/typing/completion.go` | 5 | 🔴 High — test completion logic |
| `internal/ui/typing_log_helpers.go` | 4 | 🟡 Medium — helper utilities |
| `internal/ui/screen_home_view.go` | 4 | 🟡 Medium — visual rendering |

---

## Package Churn Analysis

### File Changes by Package

| Package | Total Changes | Test Changes | Non-Test Changes | Test Ratio |
|---------|---------------|--------------|------------------|------------|
| `internal/ui` | 135 | 34 | 101 | 25.2% |
| `internal/app` | 65 | 22 | 43 | 33.8% |
| `internal/update` | 40 | 18 | 22 | 45.0% |
| `internal/cli` | 36 | 13 | 23 | 36.1% |
| `internal/metrics` | 24 | 10 | 14 | 41.7% |
| `internal/storage` | 22 | 9 | 13 | 40.9% |
| `internal/typing` | 17 | 7 | 10 | 41.2% |
| `internal/theme` | 16 | 4 | 12 | 25.0% |
| `internal/config` | 13 | 4 | 9 | 30.8% |
| `internal/cli/notui` | 12 | 4 | 8 | 33.3% |
| `internal/words` | 9 | 4 | 5 | 44.4% |
| `main.go` (root) | 6 | 0 | 6 | 0.0% |
| `internal/codetext` | 4 | 2 | 2 | 50.0% |
| `internal/cli/output` | 4 | 2 | 2 | 50.0% |
| `internal/runner` | 3 | 1 | 2 | 33.3% |
| `internal/version` | 2 | 1 | 1 | 50.0% |
| `internal/mode` | 2 | 1 | 1 | 50.0% |

### LOC by Package

| Package | LOC Added | LOC Removed | Net |
|---------|-----------|-------------|-----|
| `internal/ui` | 6,035 | 471 | +5,564 |
| `internal/cli` | 2,444 | 58 | +2,386 |
| `internal/update` | 2,368 | 39 | +2,329 |
| `internal/app` | 2,133 | 291 | +1,842 |
| `internal/metrics` | 1,171 | 27 | +1,144 |
| `internal/storage` | 1,081 | 36 | +1,045 |
| `internal/typing` | 797 | 22 | +775 |
| `internal/cli/notui` | 747 | 28 | +719 |
| `internal/theme` | 678 | 10 | +668 |
| `internal/config` | 561 | 27 | +534 |
| `internal/words` | 493 | 7 | +486 |

> [!IMPORTANT]
> `internal/ui` dominates — 135 file changes (33% of all Go changes) and 6,035 LOC added.
> Its test ratio (25.2%) is the **lowest** among major packages. This is the #1 hardening target.

---

## Plan Progress

| Metric | Value |
|--------|-------|
| Completed tasks (`[x]`) | 51 |
| Open tasks (`[ ]`) | 515 |
| Completion rate | 9.0% |
| Issues closed (period) | N/A (gh CLI not used) |

> [!NOTE]
> Most plan files have 0 completed checkboxes — plans were executed via commits but checkboxes were not updated.
> The 51 completed tasks are concentrated in `20260519-distribution-v1.5-onramp` (45 tasks) and `phase-03-words-embedded-quotes` (5 tasks).

---

## Highlights

- ✅ **Healthy churn rate (1.07x)** — nearly all code is net-new; minimal rework/thrash
- ✅ **Strong test presence** — 138 test-file changes (17.8% of all file touches); several packages above 40% test ratio
- ✅ **Good package granularity** — 17 Go packages, each with a clear responsibility
- ⚠️ **Burst-then-idle pattern** — 62% of commits in 2 days, then long gaps. Suggests large AI-assisted batch sessions

---

## Recommendations

1. **Harden `internal/ui/screen_typing_actions.go`** — 7 changes, zero test coverage, handles all typing input. A regression here breaks the core user experience. Write targeted Update() tests with mock Msg inputs. (based on: #1 untested hotspot)

2. **Add tests for `internal/app/model_view.go` and `model_settings.go`** — View dispatch and settings mutation are high-churn (7 and 6 changes) with no tests. These are pure functions / switch statements — ideal for table-driven tests. (based on: 🔴 High-risk untested hotspots)

3. **Improve `internal/ui` test ratio from 25% → 35%+** — Largest package by LOC (6k lines) and file changes (135), but lowest test ratio among major packages. Focus tests on `word_stream_renderer.go`, `typing_log_helpers.go`, and `screen_history_view.go`. (based on: package test ratio analysis)

4. **Add test for `internal/typing/completion.go`** — 5 changes, no test file, core business logic (determines when a test is "complete"). A bug here silently corrupts WPM results. (based on: untested hotspot + business-critical logic)

5. **Update plan checkboxes or adopt a lighter tracking convention** — 515 open / 51 done (9% completion) doesn't reflect actual shipped state. Either mark completed tasks post-hoc or stop using checkboxes in favor of commit-based tracking. (based on: plan completion rate 9%)

---

*Generated by `ck:retro` — data sourced from git history only*
