# GEMINI.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Typeburn is a Monkeytype-style terminal typing test: Go 1.26 + Bubble Tea v2 / Lip Gloss v2, single binary, no backend, local XDG persistence.

## Commands

```sh
make build       # ldflags-stamped binary → ./bin/typeburn
make run         # go run . (launches TUI)
make test        # go test ./...
make test-race   # go test ./... -race -count=1   ← the CI gate; must be GREEN
make lint        # gofmt -l check (must be empty) + go vet ./...
make version     # build then print the resolved --version banner

# Single test / package
go test ./internal/metrics/ -run TestCompute -count=1
go test ./internal/version/ -run TestResolve_LdflagsWin -v
```

`go test ./... -race -count=1`, `go vet ./...`, and an empty `gofmt -l .` are exactly what CI enforces — run all three before considering work done. UI tests use `teatest` golden files; pure packages are table-driven with real data (no mocks).

## Architecture

**Strict dependency layering — do not violate.** The *pure-logic* packages are UI-free and must stay that way (no `bubbletea`/`lipgloss` imports):

- Pure logic (no UI deps): `internal/typing` (keystroke state machine), `internal/metrics` (WPM/accuracy/consistency formulas), `internal/words` (embedded wordlist + quote pack), `internal/codetext` (Code-mode loader + normalization: `Load(path)` is the ONLY file/stdin I/O boundary; the exported pure `Normalize(string)` shares the same core for in-app paste — keeps words/typing I/O-free), `internal/storage` (atomic JSON persistence), `internal/version` (build stamp).
- Styling/input boundary (intentionally not reusable-core): `internal/config` binds Bubble Tea key types for its keymap, and `internal/theme` returns Lip Gloss styles/colors by design. These two depend on `charm.land` libs deliberately — they are the seam between pure logic and the TUI, not general-purpose libraries. Do not "fix" this by removing the imports; do not add new UI deps to the pure-logic packages above.
- `internal/ui` depends on the packages above and implements the six screen sub-models (Home, Typing, Result, Settings, History, CodePaste) + reusable components.
- `internal/app` is the root Bubble Tea Elm model that wires everything together.

**Elm message flow.** `app.Model` owns a `Screen` enum and six sub-models. Screen sub-models in `internal/ui` emit *domain* messages — `StartTestMsg`, `ResultMsg`, `AbortMsg`, `NavHistoryMsg`, `NavCodePasteMsg`, `CodePastedMsg` (defined in `internal/ui/messages.go`). `ScreenCodePaste` captures one `tea.PasteMsg`, validates via `codetext.Normalize`, and on success emits `CodePastedMsg`; esc is handled by the existing global Back handler (no cancel message). The root model's `Update` routes them, owns screen transitions, and is the *only* place that persists results (`AppendHistory`) and computes new-best. Sub-models never touch storage directly. To add a screen interaction, emit a message from the sub-model and handle routing/side-effects in `internal/app`.

**Metrics derive entirely from the keystroke log.** `typing.Engine` only records keystrokes (`Apply`/`Backspace`); nothing computes WPM live. `metrics.Compute(log, startMs, durationMs)` replays the log post-hoc. Never add live metric mutation — extend the log/replay path instead. `AFKTrim` (Time mode, >7s trailing idle) runs before compute.

**Theme is a semantic `Role` enum, never hex.** UI code asks for `theme.Style(RoleX)` / `theme.Color(RoleX)`. `NO_COLOR` and the `mono` (attribute-only) theme are first-class and must keep layouts identical (only attributes change). Adding a color means adding a `Role` and mapping it in every theme, not a literal in UI code.

**Storage is defensive.** Atomic temp-file + rename; any corrupt/missing file returns safe defaults and never panics; history is capped at the 200 newest records. Settings load once at startup (`app.NewFromDisk()`); history loads on demand and after each test.

**Versioning is hybrid.** `internal/version` reads ldflags-injected `Version/Commit/Date` (set by Makefile + GoReleaser); when empty (bare `go install`) it falls back to `debug.ReadBuildInfo()`, final fallback `"dev"`. `main.go`'s `decide()` is a pure, tested function returning `(printVersion bool, textPath string)`: `--version` short-circuits to the banner; `--text <file>`/`-` selects Code mode (loaded via `internal/codetext`, errors → Code shown disabled with a reason, never a crash); a `ContinueOnError` FlagSet with discarded output ensures unknown flags / `-h` / `-v` / positional args fall through to the TUI (no `os.Exit(2)`, no usage dump). `-v` is intentionally unbound (reserved for a future `--verbose`).

## Git Workflow (protected main — enforced)

`main` is protected. **Never commit or push directly to `main`.** This is hard-enforced two ways: GitHub branch protection (PR required, direct push denied, `ci.yml` must pass, linear history) and a local PreToolUse hook that blocks `git commit`/`git push` to main in this repo.

Every change — code, docs, config, release prep — follows:

1. Branch off `main`: `feat/…`, `fix/…`, or `chore/…`.
2. Commit on the branch (conventional commits, no AI references).
3. Push the branch; open a PR to `main`.
4. `ci.yml` must be green; **squash-merge** the PR (squash is the only enabled merge mode; branch auto-deletes).
5. Tags are cut on `main` **only after** the PR is merged. Tag pushes are allowed by the hook/protection (only branch refs are protected).

**Release runbook is now PR-based:** branch → commit phases → push branch → PR → squash-merge → `git tag` the merged commit on main → push tag → `release.yml`. The disposable-dry-run / fix-forward / never-re-tag invariants are unchanged (see `CONTRIBUTING.md`).

## Conventions & Constraints

- **File size:** keep every Go file < 200 LOC. Split by concern (`screen_x.go` / `screen_x_view.go` / `screen_x_actions.go` / `screen_x_test.go`). Core logic uses `snake_case` filenames; small utility/output modules use `kebab-case`.
- **Module path is case-sensitive:** `github.com/bavanchun/Typeburn` (capital `T`). ldflags `-X` targets and `go install` both depend on this exact casing.
- **No new dependencies** for core behavior without strong justification — the app deliberately uses only Bubble Tea / Lip Gloss + stdlib.

## Release Engineering (read before touching release files)

- **GoReleaser is pinned to exactly `v2.15.4`** in three places that must stay in lockstep: `.goreleaser.yaml` validation, `release.yml` `version:`, and `CONTRIBUTING.md`. Bump all three together, deliberately.
- **`ci.yml` does NOT trigger on tag pushes** (branches/PR only). The tagged commit therefore has zero CI unless `release.yml` provides it — that is why `release.yml` runs its own least-privilege `test` job that gates the `contents: write` publish job. Keep `ci.yml` byte-identical when working on release infra.
- **Non-obvious GoReleaser trap:** `changelog.disable: true` *also* makes GoReleaser ignore `--release-notes` and publish an empty release body. This repo deliberately uses a `changelog.filters.exclude: ['.*']` filter instead, with curated notes supplied via `.github/release-notes.md`. Do not "simplify" this back to `disable: true`.
- **Tags are immutable / fix-forward only.** The Go module proxy + sumdb are append-only: never delete-and-re-tag a version that was pushed (it becomes permanently uninstallable). A defect in a release is fixed by cutting the next patch (`v1.0.1`). The only safe delete-and-retry is the unadvertised disposable `v0.0.0-rc.test` dry-run tag.
- **Release procedure:** push `main`, run the disposable pre-release tag dry-run through `release.yml`, verify assets/notes/checksums, delete the dry-run (`gh release delete --cleanup-tag`), then annotate `v1.0.0` on the exact dry-run-proven SHA and push the tag in a *separate* push (never `git push --follow-tags`). Expected published assets = 7 (6 archives + `checksums.txt`); `release.yml` asserts this.
- Release binaries are **unsigned** by design (v1 scope); integrity = HTTPS + `checksums.txt` only (see `SECURITY.md`). Homebrew is a documented prose TODO in `CONTRIBUTING.md`, intentionally not a dead YAML block.

## Documentation

`docs/` is the source of truth: `codebase-summary.md` (per-package detail), `system-architecture.md`, `project-roadmap.md` (M2 new-best-precision is the tracked fast-follow), `code-standards.md`, `project-overview-pdr.md`. Update them when behavior or structure changes. Plans live in `plans/`; session journals in `docs/journals/`.

---

---

---

---

---

---

---

---

---

## Config

# GEMINI.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Typeburn is a Monkeytype-style terminal typing test: Go 1.26 + Bubble Tea v2 / Lip Gloss v2, single binary, no backend, local XDG persistence.

## Commands

```sh
make build       # ldflags-stamped binary → ./bin/typeburn
make run         # go run . (launches TUI)
make test        # go test ./...
make test-race   # go test ./... -race -count=1   ← the CI gate; must be GREEN
make lint        # gofmt -l check (must be empty) + go vet ./...
make version     # build then print the resolved --version banner

# Single test / package
go test ./internal/metrics/ -run TestCompute -count=1
go test ./internal/version/ -run TestResolve_LdflagsWin -v
```

`go test ./... -race -count=1`, `go vet ./...`, and an empty `gofmt -l .` are exactly what CI enforces — run all three before considering work done. UI tests use `teatest` golden files; pure packages are table-driven with real data (no mocks).

## Architecture

**Strict dependency layering — do not violate.** The *pure-logic* packages are UI-free and must stay that way (no `bubbletea`/`lipgloss` imports):

- Pure logic (no UI deps): `internal/typing` (keystroke state machine), `internal/metrics` (WPM/accuracy/consistency formulas), `internal/words` (embedded wordlist + quote pack), `internal/codetext` (Code-mode loader + normalization: `Load(path)` is the ONLY file/stdin I/O boundary; the exported pure `Normalize(string)` shares the same core for in-app paste — keeps words/typing I/O-free), `internal/storage` (atomic JSON persistence), `internal/version` (build stamp).
- Styling/input boundary (intentionally not reusable-core): `internal/config` binds Bubble Tea key types for its keymap, and `internal/theme` returns Lip Gloss styles/colors by design. These two depend on `charm.land` libs deliberately — they are the seam between pure logic and the TUI, not general-purpose libraries. Do not "fix" this by removing the imports; do not add new UI deps to the pure-logic packages above.
- `internal/ui` depends on the packages above and implements the six screen sub-models (Home, Typing, Result, Settings, History, CodePaste) + reusable components.
- `internal/app` is the root Bubble Tea Elm model that wires everything together.

**Elm message flow.** `app.Model` owns a `Screen` enum and six sub-models. Screen sub-models in `internal/ui` emit *domain* messages — `StartTestMsg`, `ResultMsg`, `AbortMsg`, `NavHistoryMsg`, `NavCodePasteMsg`, `CodePastedMsg` (defined in `internal/ui/messages.go`). `ScreenCodePaste` captures one `tea.PasteMsg`, validates via `codetext.Normalize`, and on success emits `CodePastedMsg`; esc is handled by the existing global Back handler (no cancel message). The root model's `Update` routes them, owns screen transitions, and is the *only* place that persists results (`AppendHistory`) and computes new-best. Sub-models never touch storage directly. To add a screen interaction, emit a message from the sub-model and handle routing/side-effects in `internal/app`.

**Metrics derive entirely from the keystroke log.** `typing.Engine` only records keystrokes (`Apply`/`Backspace`); nothing computes WPM live. `metrics.Compute(log, startMs, durationMs)` replays the log post-hoc. Never add live metric mutation — extend the log/replay path instead. `AFKTrim` (Time mode, >7s trailing idle) runs before compute.

**Theme is a semantic `Role` enum, never hex.** UI code asks for `theme.Style(RoleX)` / `theme.Color(RoleX)`. `NO_COLOR` and the `mono` (attribute-only) theme are first-class and must keep layouts identical (only attributes change). Adding a color means adding a `Role` and mapping it in every theme, not a literal in UI code.

**Storage is defensive.** Atomic temp-file + rename; any corrupt/missing file returns safe defaults and never panics; history is capped at the 200 newest records. Settings load once at startup (`app.NewFromDisk()`); history loads on demand and after each test.

**Versioning is hybrid.** `internal/version` reads ldflags-injected `Version/Commit/Date` (set by Makefile + GoReleaser); when empty (bare `go install`) it falls back to `debug.ReadBuildInfo()`, final fallback `"dev"`. `main.go`'s `decide()` is a pure, tested function returning `(printVersion bool, textPath string)`: `--version` short-circuits to the banner; `--text <file>`/`-` selects Code mode (loaded via `internal/codetext`, errors → Code shown disabled with a reason, never a crash); a `ContinueOnError` FlagSet with discarded output ensures unknown flags / `-h` / `-v` / positional args fall through to the TUI (no `os.Exit(2)`, no usage dump). `-v` is intentionally unbound (reserved for a future `--verbose`).

## Git Workflow (protected main — enforced)

`main` is protected. **Never commit or push directly to `main`.** This is hard-enforced two ways: GitHub branch protection (PR required, direct push denied, `ci.yml` must pass, linear history) and a local PreToolUse hook that blocks `git commit`/`git push` to main in this repo.

Every change — code, docs, config, release prep — follows:

1. Branch off `main`: `feat/…`, `fix/…`, or `chore/…`.
2. Commit on the branch (conventional commits, no AI references).
3. Push the branch; open a PR to `main`.
4. `ci.yml` must be green; **squash-merge** the PR (squash is the only enabled merge mode; branch auto-deletes).
5. Tags are cut on `main` **only after** the PR is merged. Tag pushes are allowed by the hook/protection (only branch refs are protected).

**Release runbook is now PR-based:** branch → commit phases → push branch → PR → squash-merge → `git tag` the merged commit on main → push tag → `release.yml`. The disposable-dry-run / fix-forward / never-re-tag invariants are unchanged (see `CONTRIBUTING.md`).

## Conventions & Constraints

- **File size:** keep every Go file < 200 LOC. Split by concern (`screen_x.go` / `screen_x_view.go` / `screen_x_actions.go` / `screen_x_test.go`). Core logic uses `snake_case` filenames; small utility/output modules use `kebab-case`.
- **Module path is case-sensitive:** `github.com/bavanchun/Typeburn` (capital `T`). ldflags `-X` targets and `go install` both depend on this exact casing.
- **No new dependencies** for core behavior without strong justification — the app deliberately uses only Bubble Tea / Lip Gloss + stdlib.

## Release Engineering (read before touching release files)

- **GoReleaser is pinned to exactly `v2.15.4`** in three places that must stay in lockstep: `.goreleaser.yaml` validation, `release.yml` `version:`, and `CONTRIBUTING.md`. Bump all three together, deliberately.
- **`ci.yml` does NOT trigger on tag pushes** (branches/PR only). The tagged commit therefore has zero CI unless `release.yml` provides it — that is why `release.yml` runs its own least-privilege `test` job that gates the `contents: write` publish job. Keep `ci.yml` byte-identical when working on release infra.
- **Non-obvious GoReleaser trap:** `changelog.disable: true` *also* makes GoReleaser ignore `--release-notes` and publish an empty release body. This repo deliberately uses a `changelog.filters.exclude: ['.*']` filter instead, with curated notes supplied via `.github/release-notes.md`. Do not "simplify" this back to `disable: true`.
- **Tags are immutable / fix-forward only.** The Go module proxy + sumdb are append-only: never delete-and-re-tag a version that was pushed (it becomes permanently uninstallable). A defect in a release is fixed by cutting the next patch (`v1.0.1`). The only safe delete-and-retry is the unadvertised disposable `v0.0.0-rc.test` dry-run tag.
- **Release procedure:** push `main`, run the disposable pre-release tag dry-run through `release.yml`, verify assets/notes/checksums, delete the dry-run (`gh release delete --cleanup-tag`), then annotate `v1.0.0` on the exact dry-run-proven SHA and push the tag in a *separate* push (never `git push --follow-tags`). Expected published assets = 7 (6 archives + `checksums.txt`); `release.yml` asserts this.
- Release binaries are **unsigned** by design (v1 scope); integrity = HTTPS + `checksums.txt` only (see `SECURITY.md`). Homebrew is a documented prose TODO in `CONTRIBUTING.md`, intentionally not a dead YAML block.

## Documentation

`docs/` is the source of truth: `codebase-summary.md` (per-package detail), `system-architecture.md`, `project-roadmap.md` (M2 new-best-precision is the tracked fast-follow), `code-standards.md`, `project-overview-pdr.md`. Update them when behavior or structure changes. Plans live in `plans/`; session journals in `docs/journals/`.
---

## Rule: development-rules

# Development Rules

**IMPORTANT:** Analyze the skills catalog and activate the skills that are needed for the task during the process.
**IMPORTANT:** You ALWAYS follow these principles: **YAGNI (You Aren't Gonna Need It) - KISS (Keep It Simple, Stupid) - DRY (Don't Repeat Yourself)**

## General
- **File Naming**: Use kebab-case for file names with a meaningful name that describes the purpose of the file, doesn't matter if the file name is long, just make sure when LLMs read the file names while using Grep or other tools, they can understand the purpose of the file right away without reading the file content.
- **File Size Management**: Keep individual code files under 200 lines for optimal context management
  - Split large files into smaller, focused components/modules
  - Use composition over inheritance for complex widgets
  - Extract utility functions into separate modules
  - Create dedicated service classes for business logic
- When looking for docs, activate `docs-seeker` skill (`context7` reference) for exploring latest docs.
- Use `gh` bash command to interact with Github features if needed
- Use `psql` bash command to query Postgres database for debugging if needed
- Use `ai-multimodal` skill for describing details of images, videos, documents, etc. if needed
- Use `ai-multimodal` skill and `imagemagick` skill for generating and editing images, videos, documents, etc. if needed
- Use `sequential-thinking` and `debug` skills for sequential thinking, analyzing code, debugging, etc. if needed
- **[IMPORTANT]** Follow the codebase structure and code standards in `.` during implementation.
- **[IMPORTANT]** Do not just simulate the implementation or mocking them, always implement the real code.

## Code Quality Guidelines
- Read and follow codebase structure and code standards in `.`
- Don't be too harsh on code linting, but **make sure there are no syntax errors and code are compilable**
- Prioritize functionality and readability over strict style enforcement and code formatting
- Use reasonable code quality standards that enhance developer productivity
- Use try catch error handling & cover security standards
- Use `code-reviewer` agent to review code after every implementation

## Pre-commit/Push Rules
- Run linting before commit
- Run tests before push (DO NOT ignore failed tests just to pass the build or github actions)
- Keep commits focused on the actual code changes
- **DO NOT** commit and push any confidential information (such as dotenv files, API keys, database credentials, etc.) to git repository!
- Create clean, professional commit messages without AI references. Use conventional commit format.

## Code Implementation
- Write clean, readable, and maintainable code
- Follow established architectural patterns
- Implement features according to specifications
- Handle edge cases and error scenarios
- **DO NOT** create new enhanced files, update to the existing files directly.

## Visual Aids
- Use ` --explain` when explaining unfamiliar code patterns or complex logic
- Use ` --diagram` for architecture diagrams and data flow visualization
- Use ` --slides` for step-by-step walkthroughs and presentations
- Use ` --ascii` for terminal-friendly diagrams (no browser needed to understand)
- Add `--html` to any generation flag for self-contained HTML output (opens in browser, no server needed)
- **Plan context:** Active plan determined from `## Plan Context` in hook injection; visuals save to `{plan_dir}`
- If no active plan, fallback to `plans/visuals/` directory
- For Mermaid diagrams, use `` skill for v11 syntax rules
- See `primary-workflow.md` → Step 6 for workflow integration
---

## Rule: documentation-management

# Project Documentation Management

### Roadmap & Changelog Maintenance
- **Project Roadmap** (`./docs/development-roadmap.md`): Living document tracking project phases, milestones, and progress
- **Project Changelog** (`./docs/project-changelog.md`): Detailed record of all significant changes, features, and fixes
- **System Architecture** (`./docs/system-architecture.md`): Detailed record of all significant changes, features, and fixes
- **Code Standards** (`./docs/code-standards.md`): Detailed record of all significant changes, features, and fixes

### Automatic Updates Required
- **After Feature Implementation**: Update roadmap progress status and changelog entries
- **After Major Milestones**: Review and adjust roadmap phases, update success metrics
- **After Bug Fixes**: Document fixes in changelog with severity and impact
- **After Security Updates**: Record security improvements and version updates
- **Weekly Reviews**: Update progress percentages and milestone statuses

### Documentation Triggers
The `project-manager` agent MUST update these documents when:
- A development phase status changes (e.g., from "In Progress" to "Complete")
- Major features are implemented or released
- Significant bugs are resolved or security patches applied
- Project timeline or scope adjustments are made
- External dependencies or breaking changes occur

### Update Protocol
1. **Before Updates**: Always read current roadmap and changelog status
2. **During Updates**: Maintain version consistency and proper formatting
3. **After Updates**: Verify links, dates, and cross-references are accurate
4. **Quality Check**: Ensure updates align with actual implementation progress

### Plans

### Plan Location
Save plans in `.` directory with timestamp and descriptive name.

**Format:** Use naming pattern from `## Naming` section injected by hooks.

**Example:** `plans/251101-1505-authentication-and-profile-implementation/`

#### File Organization

```
plans/
├── 20251101-1505-authentication-and-profile-implementation/
    ├── research/
    │   ├── researcher-XX-report.md
    │   └── ...
│   ├── reports/
│   │   ├── scout-report.md
│   │   ├── researcher-report.md
│   │   └── ...
│   ├── plan.md                                # Overview access point
│   ├── phase-01-setup-environment.md          # Setup environment
│   ├── phase-02-implement-database.md         # Database models
│   ├── phase-03-implement-api-endpoints.md    # API endpoints
│   ├── phase-04-implement-ui-components.md    # UI components
│   ├── phase-05-implement-authentication.md   # Auth & authorization
│   ├── phase-06-implement-profile.md          # Profile page
│   └── phase-07-write-tests.md                # Tests
└── ...
```

#### File Structure

##### Overview Plan (plan.md)
- Keep generic and under 80 lines
- List each phase with status/progress
- Link to detailed phase files
- Key dependencies

##### Phase Files (phase-XX-name.md)
Fully respect the `./docs/development-rules.md` file.
Each phase file should contain:

**Context Links**
- Links to related reports, files, documentation

**Overview**
- Priority
- Current status
- Brief description

**Key Insights**
- Important findings from research
- Critical considerations

**Requirements**
- Functional requirements
- Non-functional requirements

**Architecture**
- System design
- Component interactions
- Data flow

**Related Code Files**
- List of files to modify
- List of files to create
- List of files to delete

**Implementation Steps**
- Detailed, numbered steps
- Specific instructions

**Todo List**
- Checkbox list for tracking

**Success Criteria**
- Definition of done
- Validation methods

**Risk Assessment**
- Potential issues
- Mitigation strategies

**Security Considerations**
- Auth/authorization
- Data protection

**Next Steps**
- Dependencies
- Follow-up tasks
---

## Rule: orchestration-protocol

# Orchestration Protocol

## Delegation Context (MANDATORY)

When spawning subagents via subtask delegation, **ALWAYS** include in prompt:

1. **Work Context Path**: The git root of the PRIMARY files being worked on
2. **Reports Path**: `{work_context}/plans/reports/` for that project
3. **Plans Path**: `{work_context}` for that project

**Example:**
```
Task prompt: "Fix parser bug.
Work context: /path/to/project-b
Reports: /path/to/project-b/plans/reports/
Plans: /path/to/project-b/plans/"
```

**Rule:** If CWD differs from work context (editing files in different project), use the **work context paths**, not CWD paths.
---

#### Sequential Chaining
Chain subagents when tasks have dependencies or require outputs from previous steps:
- **Planning → Implementation → Simplification → Testing → Review**: Use for feature development (tests verify simplified code)
- **Research → Design → Code → Documentation**: Use for new system components
- Each agent completes fully before the next begins
- Pass context and outputs between agents in the chain

#### Parallel Execution
Spawn multiple subagents simultaneously for independent tasks:
- **Code + Tests + Docs**: When implementing separate, non-conflicting components
- **Multiple Feature Branches**: Different agents working on isolated features
- **Cross-platform Development**: iOS and Android specific implementations
- **Careful Coordination**: Ensure no file conflicts or shared resource contention
- **Merge Strategy**: Plan integration points before parallel execution begins
---

## Subagent Status Protocol

Subagents MUST report one of these statuses when completing work:

| Status | Meaning | Controller Action |
|--------|---------|-------------------|
| **DONE** | Task completed successfully | Proceed to next step (review, next task) |
| **DONE_WITH_CONCERNS** | Completed but flagged doubts | Read concerns → address if correctness/scope issue → proceed if observational |
| **BLOCKED** | Cannot complete task | Assess blocker → provide context / break task / escalate to user |
| **NEEDS_CONTEXT** | Missing information to proceed | Provide missing context → re-dispatch |

### Handling Rules

- **Never** ignore BLOCKED or NEEDS_CONTEXT — something must change before retry
- **Never** force same approach after BLOCKED — try: more context → simpler task → more capable model → escalate
- **DONE_WITH_CONCERNS** about file growth or tech debt → note for future, proceed now
- **DONE_WITH_CONCERNS** about correctness → address before review
- If subagent fails 3+ times on same task → escalate to user, don't retry blindly

### Reporting Format

Subagents should end their response with:

```
**Status:** DONE | DONE_WITH_CONCERNS | BLOCKED | NEEDS_CONTEXT
**Summary:** [1-2 sentence summary]
**Concerns/Blockers:** [if applicable]
```
---

## Context Isolation Principle

**Subagents receive only the context they need.** Never pass full session history.

### Rules

1. **Craft prompts explicitly** — Provide task description, relevant file paths, acceptance criteria. Not "here's what we discussed."
2. **No session history** — Subagent gets fresh context. Summarize relevant decisions, don't replay conversation.
3. **Scope file references** — List specific files to read/modify. Not "look at the codebase."
4. **Include plan context** — If working from a plan, provide the specific phase text, not the entire plan.
5. **Preserve controller context** — Coordination work stays in main agent. Don't dump coordination details into subagent prompts.

### Prompt Template

```
Task: [specific task description]
Files to modify: [list]
Files to read for context: [list]
Acceptance criteria: [list]
Constraints: [any relevant constraints]
Plan reference: [phase file path if applicable]

Work context: [project path]
Reports: [reports path]
```

### Anti-Patterns

| Bad | Good |
|-----|------|
| "Continue from where we left off" | "Implement X feature per spec in phase-02.md" |
| "Fix the issues we discussed" | "Fix null check in auth.ts:45, root cause: missing validation" |
| "Look at the codebase and figure out" | "Read src/api/routes.ts and add POST  endpoint" |
| Passing 50+ lines of conversation | 5-line task summary with file paths |
---

## Agent Teams (Optional)

For multi-session parallel collaboration, activate the `` skill.
Not part of the default orchestration workflow. See `.agents/skills/team/SKILL.md` for templates, decision criteria, and spawn instructions.
---

## Rule: primary-workflow

# Primary Workflow

**IMPORTANT:** Analyze the skills catalog and activate the skills that are needed for the task during the process.
**IMPORTANT**: Ensure token efficiency while maintaining high quality.

#### 1. Code Implementation
- Before you start, delegate to `planner` agent to create a implementation plan with TODO tasks in `.` directory.
- When in planning phase, use multiple `researcher` agents in parallel to conduct research on different relevant technical topics and report back to `planner` agent to create implementation plan.
- Write clean, readable, and maintainable code
- Follow established architectural patterns
- Implement features according to specifications
- Handle edge cases and error scenarios
- **DO NOT** create new enhanced files, update to the existing files directly.
- **[IMPORTANT]** After creating or modifying code file, run compile command/script to check for any compile errors.

#### 2. Testing
- Delegate to `tester` agent to run tests on the **simplified code**
  - Write comprehensive unit tests
  - Ensure high code coverage
  - Test error scenarios
  - Validate performance requirements
- Tests verify the FINAL code that will be reviewed and merged
- **DO NOT** ignore failing tests just to pass the build.
- **IMPORTANT:** make sure you don't use fake data, mocks, cheats, tricks, temporary solutions, just to pass the build or github actions.
- **IMPORTANT:** Always fix failing tests follow the recommendations and delegate to `tester` agent to run tests again, only finish your session when all tests pass.

#### 3. Code Quality
- After testing passes, delegate to `code-reviewer` agent to review clean, tested code.
- Follow coding standards and conventions
- Write self-documenting code
- Add meaningful comments for complex logic
- Optimize for performance and maintainability

#### 4. Integration
- Always follow the plan given by `planner` agent
- Ensure seamless integration with existing code
- Follow API contracts precisely
- Maintain backward compatibility
- Document breaking changes
- Delegate to `docs-manager` agent to update docs in `.` directory if any.

#### 5. Debugging
- When a user report bugs or issues on the server or a CI/CD pipeline, delegate to `debugger` agent to run tests and analyze the summary report.
- Read the summary report from `debugger` agent and implement the fix.
- Delegate to `tester` agent to run tests and analyze the summary report.
- If the `tester` agent reports failed tests, fix them follow the recommendations and repeat from the **Step 3**.

#### 6. Visual Explanations
When explaining complex code, protocols, or architecture:
- **When to use:** User asks "explain", "how does X work", "visualize", or topic has 3+ interacting components
- Use ` --explain <topic>` to generate visual explanation with ASCII + Mermaid
- Use ` --diagram <topic>` for architecture and data flow diagrams
- Use ` --slides <topic>` for step-by-step walkthroughs
- Use ` --ascii <topic>` for terminal-friendly output only
- **HTML mode** (add `--html` for self-contained HTML pages, opens directly in browser):
  - ` --html --explain <topic>` — publication-quality HTML explanation
  - ` --html --diagram <topic>` — interactive HTML diagram with zoom controls
  - ` --html --slides <topic>` — magazine-quality slide deck
  - ` --html --diff [ref]` — visual diff review
  - ` --html --plan-review` — plan vs codebase comparison
  - ` --html --recap [timeframe]` — project context snapshot
- **Plan context:** Visuals save to plan folder from `## Plan Context` hook injection; if none, uses `plans/visuals/`
- **Markdown mode:** Auto-opens in browser via markdown-novel-viewer with Mermaid rendering
- **HTML mode:** Opens directly in browser — self-contained, no server needed
- See `development-rules.md` → "Visual Aids" section for additional guidance
---

## Rule: review-audit-self-decision

# GEMINI.md Rules Summary — Review/Audit + Comments + Self-Decision

## 1. Verified Decisions Are Sticky — Audit Does Not Auto-Reverse

- Once a decision is verified (read source, ran tests, empirical experiment), lock it with a source note: `verified by {file:line}` or `verified by test {name}`.
- An audit/red-team counter-argument **alone is insufficient** to revise. Only revise when:
  - Audit finds a **new** issue the verification missed (state the issue + why prior check missed it).
  - Or context changed since verification (codebase moved, business decision changed).
- After verifying, prune stale risk rows / unresolved questions from reports.
- Surface contradictions on verified decisions to user as: "audit says X, but Y is verified by {source} — does the audit bring new data to justify a reverse?" Do NOT silently flip.

## 2. Validate Audit Findings Against Real Threat Model

Before applying a finding flagged "too narrow / too loose / risky":

1. Identify what the code **actually stores/protects**.
2. Walk each scenario the reviewer flagged through that lens. Does it actually produce the bad outcome? Often "theoretically yes, practically no".
3. Separate real risks from abstract worries. Real → fix; non-real → document rationale; borderline → ask user.
4. Look for the failure mode the reviewer missed — often the real bug sits one step away from what was flagged.

Anti-pattern: accepting every "widen/harden/add-check" recommendation without tracing it to a real failure mode.

## 3. Guard User Decisions Against Audit/YAGNI Drift

**NEVER silently reverse decisions the user has already confirmed.**

Before any cut/change from audit:

1. **Trace before cutting:** check if user explicitly chose that value/design.
2. **Categorize:**
   - ✅ Safe to apply: cuts of items Claude proposed that user never explicitly confirmed.
   - ⚠️ Must confirm first: anything touching user's explicit answer — thresholds, scope, library, schema, phase, feature inclusion/exclusion.
   - 🚫 Never auto-reverse: business decisions (pricing, timing, scope boundaries, compliance).
3. **Surface reversals:** present (a) user's original decision verbatim, (b) audit reasoning, (c) trade-off, (d) explicit ask "keep / change per audit / hybrid?". Do NOT apply.
4. **Document drift:** annotate cuts with reason + user-confirmation trail.
5. **Auditor bias awareness:** audits lean YAGNI/minimalism — they are **input to user**, not orders to Claude.

Red flags: changing a numeric threshold user picked, removing a column/field user mentioned, swapping library, moving scope across phases, cutting a feature user confirmed.

Rule of thumb: if unsure whether a cut reverses user intent, ask. Cost of 1 clarifying question ≪ cost of silent regression at demo.

## 4. Scout-First, Ask-Second (Confidence Score)

For any question answerable by grep/read on the codebase:

1. **Scout first:** grep, read live code, check current state.
2. **Self-rate confidence (0–100%):**
   - **≥ ~85%** → answer directly with `path:line` citation.
   - **< 85%** → ask user.
3. **Only ask when:**
   - Confidence < 85% (missing data, ambiguous).
   - Real conflict between 2+ sources (not a stale note already verified).
   - Anomaly requiring user judgment (business decision, UX trade-off, scope expansion).
   - High-reversibility risk (destructive op, breaking change, deploy-affecting).

Anti-pattern: asking what grep can answer in 5s.
Good pattern: scout → "verified at `file:line`, confidence 95%, applying X" — verified + concise.

## 5. Code Comments & Artifact Naming — No Plan References

Code comments and file names (including SQL migrations) **must not reference plan artifacts**: phase numbers, finding codes (F1, F13, Y1, CU2…), audit labels (audit A4), red-team labels, brainstorm sections (§5.4), plan taxonomy.

**Why:** plan headers change/get renumbered/disappear → references become unresolvable noise. The *reason* for code (invariant, race, trade-off) must be stable and self-contained.

### Rules

- **Explain the why, not the origin.** Write "org-scoped advisory lock serializes concurrent reassigns" — NOT "per F13 advisory-lock fix".
- **Migration filenames:** domain slug only — `000003_polymorphic_permission_groups.up.sql` (NOT `000003_phase_0a_...`).
- **Test names:** describe scenario — `TestReassignPrimaryDept_Concurrent` (NOT `_F13`).
- **Commit messages:** describe the change, not the finding code.
- Plan refs belong in `plans/…XX-*.md` and PR descriptions, not in code.

### Allowed in code

- Function/symbol names in the same codebase.
- Stable external IDs: RFC numbers, PostgreSQL SQLSTATE, CVE IDs, durable issue numbers.
---

## Open Questions

None — all 5 rules are clear. Flag if any rule should be expanded further.
---

## Rule: skill-domain-routing

# Skill Domain Routing

When a user's task involves a specific domain, use these decision trees to pick the RIGHT skill based on user intent.

## Frontend / UI

```
User wants to...
├── Replicate a mockup, screenshot, or video    → /ck:frontend-design
├── Build React/TS components with best practices → /ck:frontend-development
├── Style with Tailwind CSS + shadcn/ui          → /ck:ui-styling
├── Choose colors, fonts, layout, design system  → /ck:ui-ux-pro-max
├── Audit existing UI for accessibility/UX       → /ck:web-design-guidelines
├── Apply React performance patterns             → /ck:react-best-practices
├── Build with Stitch (AI design generation)     → /ck:stitch
├── Create 3D / WebGL / Three.js experience      → /ck:threejs
├── Write GLSL shaders / procedural graphics     → /ck:shader
└── Build programmatic video with Remotion       → /ck:remotion
```

## Codebase Understanding

```
User wants to...
├── Quick file search, locate specific code     → /ck:scout
├── Onboard a new repo / dump codebase for LLM  → /ck:repomix
├── Semantic go-to-definition, find-usages      → /ck:gkg
└── Build a queryable knowledge graph from code → /ck:graphify
```

## Backend / API

```
User wants to...
├── Build REST/GraphQL API (NestJS, FastAPI, Django) → /ck:backend-development
├── Add authentication (OAuth, JWT, passkeys)        → /ck:better-auth
└── Integrate payments (Stripe, Polar, SePay)        → /ck:payment-integration
```

## Database

```
User wants to...
├── Design schemas, write SQL/NoSQL queries     → /ck:databases
├── Optimize indexes, migrations, replication   → /ck:databases
└── Add auth with database-backed sessions      → /ck:better-auth
```

## Infrastructure / Deployment

```
User wants to...
├── Deploy to Vercel, Netlify, Railway, Fly.io   → /ck:deploy
└── Docker, Kubernetes, CI/CD pipelines, GitOps   → /ck:devops
```

## Security

```
User wants to...
├── STRIDE/OWASP security audit with auto-fix    → /ck:security
├── Scan for secrets, vulnerabilities, OWASP patterns → /ck:security-scan
└── OSINT / CTI / threat-intel investigation     → /ck:cti-expert
```

## AI / LLM

```
User wants to...
├── Optimize context, agent architecture, memory → /ck:context-engineering
├── Generate llms.txt, LLM-friendly docs         → /ck:llms
├── Build AI agents with Google ADK              → /ck:google-adk-python
├── Generate/analyze images, audio, video with AI → /ck:ai-multimodal
└── Learn the autoresearch pattern / find the right family member → /ck:autoresearch
```

## MCP (Model Context Protocol)

```
User wants to...
├── Build a new MCP server                       → /ck:mcp-builder
├── Convert existing code into CLI/MCP server    → /ck:agentize
└── Discover and execute MCP tools               → /ck:use-mcp
```

## Testing / Browser

```
User wants to...
├── Run test suites, coverage reports, TDD          → /ck:test
├── Test strategy + Playwright/Vitest/k6 runner     → /ck:web-testing
└── Drive a live browser                            → /ck:agent-browser
```

## Media

```
User wants to...
├── Process video/audio (FFmpeg), images (ImageMagick) → /ck:media-processing
└── Generate AI images (Imagen, Nano Banana)           → /ck:ai-artist
```

## Documentation

```
User wants to...
├── Update project docs (codebase-summary, PDR)   → /ck:docs
├── Search library/framework docs (context7)      → /ck:docs-seeker
├── Discover skills by capability / "is there a skill" → /ck:find-skills
├── Build docs site with Mintlify                 → /ck:mintlify
├── Inline doc diagrams (Mermaid v11)             → /ck:mermaidjs-v11
├── Publish-grade SVG/PNG diagrams (architecture) → /ck:tech-graph
├── Read long-form docs / RFCs / specs in browser → /ck:markdown-novel-viewer
├── Generate session hand-off / EOD summary       → /ck:watzup
└── Sprint retrospective from git history         → /ck:retro
```

## Documents / Office Files

```
User wants to...
├── Create / edit / extract from .docx (Word)         → /ck:docx
├── Create / edit / extract from .pdf (forms, tables) → /ck:pdf
├── Create / edit / extract from .pptx (PowerPoint)   → /ck:pptx
└── Create / edit / extract from .xlsx (spreadsheets) → /ck:xlsx
```

## Content / Copy

```
User wants to...
├── Write landing page, email, headline copy     → /ck:copywriting
├── Brand identity, logos, banners               → /ckm:design
└── Create Excalidraw diagrams                   → /ck:excalidraw
```

## Frameworks

```
User wants to...
├── Next.js App Router, RSC, Turborepo           → /ck:web-frameworks
├── TanStack Start/Form/AI                       → /ck:tanstack
├── React Native, Flutter, SwiftUI               → /ck:mobile-development
└── Shopify apps, Polaris, Liquid templates       → /ck:shopify
```

## Usage Notes

- Pick ONE skill per distinct user intent
- If a task spans two domains (e.g. "build + deploy"), suggest the primary skill and mention the secondary
- Domain skills combine with core workflow: `` → domain skill → ``
- Skills not listed here are either core workflow skills (see `skill-workflow-routing.md`) or utility skills activated on demand (e.g. ``, ``, ``)
---

## Rule: skill-workflow-routing

# Skill Workflow Routing

When orchestrating multi-step tasks, consider these workflow sequences. Skills are listed in typical execution order.

## Core Development Workflow

```
/ck:plan → /ck:cook → /ck:test → /ck:code-review → /ck:ship → /ck:journal
```

| User Intent | Suggested Start |
|-------------|----------------|
| "implement feature X", "build X", "add X" | `` then `` |
| "execute this plan" | ` <plan-path>` |
| "quick implementation" | ` --fast` |

## Bugfix Workflow

```
/ck:scout → /ck:debug → /ck:fix → /ck:test → /ck:code-review
```

| User Intent | Suggested Start |
|-------------|----------------|
| "X is broken", "error in X", "bug in X" | `` (auto-scouts internally) |
| "CI is failing", "tests broken" | ` --auto` |
| "investigate why X happens" | `` then `` |

## Investigation Workflow

```
/ck:scout → /ck:debug → /ck:brainstorm → /ck:plan
```

| User Intent | Suggested Start |
|-------------|----------------|
| "understand how X works" | `` |
| "why is X happening" | `` |
| "explore options for X" | `` then `` |

## Post-Implementation Checklist

After completing implementation work, consider:
- `` — review changes before merging
- `` — run full shipping pipeline (tests, review, version, PR)
- `` — document decisions and lessons learned

## Setup Skills

Before starting implementation in a shared codebase:
- `` — create isolated worktree for the feature/fix
- `` — discover relevant files and code patterns
---

## Rule: team-coordination-rules

# Team Coordination Rules

> These rules only apply when operating as a teammate within an Agent Team.
> They have no effect on standard sessions or subagent workflows.

Rules for agents operating as teammates within an Agent Team.

## File Ownership (CRITICAL)

- Each teammate MUST own distinct files — no overlapping edits
- Define ownership via glob patterns in task descriptions: `File ownership: src/api/*, src/models/*`
- Lead resolves ownership conflicts by restructuring tasks or handling shared files directly
- Tester owns test files only; reads implementation files but never edits them
- If ownership violation detected: STOP and report to lead immediately

## Git Safety

- Prefer git worktrees for implementation teams — each dev in own worktree eliminates conflicts
- Never force-push from a teammate session
- Commit frequently with descriptive messages
- Pull before push to catch merge conflicts early
- If working in a git worktree, commit/push to the worktree branch — not main or dev

## Communication Protocol

- Include actionable findings in messages, not just "I'm done"
- Never send structured JSON status messages — use plain text

## CK Stack Conventions

### Report Output
- Save reports to `{CK_REPORTS_PATH}` (injected via hook, fallback: `plans/reports/`)
- Naming: `{type}-{date}-{slug}.md` where type = your role (researcher, reviewer, debugger)
- Sacrifice grammar for concision. List unresolved questions at end.

### Commit Messages
- Use conventional commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`
- No AI references in commit messages
- Keep commits focused on actual code changes

### Docs Sync (Implementation Teams Only)
- After completing implementation tasks, lead MUST evaluate docs impact
- State explicitly: `Docs impact: [none|minor|major]`
- If impact: update `docs/` directory or note in completion message

## Task Claiming

- Claim lowest-ID unblocked task first (earlier tasks set up context for later ones)
- Check `TaskList` after completing each task for newly unblocked work
- Set task to `in_progress` before starting work
- If all tasks blocked, notify lead and offer to help unblock

## Plan Approval Flow

When `plan_mode_required` is set:
1. Research and plan your approach (read-only — no file edits)
2. Send plan via `ExitPlanMode` — this triggers approval request to lead
3. Wait for lead's `plan_approval_response`
4. If rejected: revise based on feedback, resubmit
5. If approved: proceed with implementation

## Conflict Resolution

- If two teammates need the same file: escalate to lead immediately
- If a teammate's plan is rejected twice: lead takes over that task
- If findings conflict between reviewers: lead synthesizes and documents disagreement
- If blocked by another teammate's incomplete work: message them directly first, escalate to lead if unresponsive

## Shutdown Protocol

- Approve shutdown requests unless mid-critical-operation
- Always mark current task as completed before approving shutdown
- If rejecting shutdown, explain why concisely
- Extract `requestId` from shutdown request JSON and pass to `shutdown_response`

## Idle State (Normal Behavior)

- Going idle after sending a message is NORMAL — not an error
- Idle means waiting for input, not disconnected
- Sending a message to an idle teammate wakes them up
- Do not treat idle notifications as completion signals — check task status instead

## Discovery

- Read team config at `~/.claude/teams/{team-name}/config.json` to discover teammates
- Always refer to teammates by NAME (not agent ID)

