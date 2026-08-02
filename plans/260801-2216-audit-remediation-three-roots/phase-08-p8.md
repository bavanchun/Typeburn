---
phase: 8
title: "Supply Chain And CI Gates"
status: done
priority: P2
effort: "4h"
dependencies: []
---

# Phase 8: Supply Chain And CI Gates

## Overview

Close three open holes: a known vulnerability with no scanner, a CI job that is
not actually a required check, and a `.gitignore` pattern that silently drops new
source files. Runs in the first group so everything else rebases on the `go.mod`
change once.

## Requirements

- Functional
  - `govulncheck` runs in CI under a **defined stdlib policy**.
  - Every `ci.yml` job is a required status check on `main`.
  - `git add .` cannot silently skip a new file under `cmd/typeburn/`.
- Non-functional
  - No behaviour change. `ci.yml`'s existing checks keep their intent — CLAUDE.md
    requires care here because `release.yml` is coupled to it.

## Architecture

### Correction: the vulnerability is NOT on the render path

The plan previously claimed `GO-2026-5970` is "called on every frame render".
**That was wrong.** Verified:

```
go mod why -m golang.org/x/text
  -> cmd/typeburn → github.com/charmbracelet/fang → golang.org/x/text/cases
go list -deps charm.land/lipgloss/v2 | grep -c '^golang.org/x/text'   -> 0
```

x/text enters only through fang's help-text title-casing. The govulncheck trace
through `lipgloss.Style.Render → norm.Form.Properties` is a sound
over-approximation across interface dispatch, not a real call path.

The finding is real and worth fixing (`v0.24.0` → `v0.40.0`), and the *absence of
any scanner* is the actual defect — nothing in this repo would ever tell the
maintainer. But it is a routine dependency bump, not an emergency, and the golden
churn risk from the bump is near zero. Phase 5's original mitigation ("if a
golden fails, that is the signal") will never fire — which is not the same as
being safe, so still run the suite before regenerating anything.

### The stdlib policy must be decided before the gate lands

`govulncheck ./...` currently reports **six** findings: one module and **five
stdlib** (crypto/tls, net/textproto, crypto/x509, net, net/http), all tied to the
local toolchain. It exits non-zero on stdlib findings too. CI builds go 1.25.x,
whose own advisory set is unmeasured and turns over roughly every six weeks.

So a naive `govulncheck ./...` step will **red-wall `main` on merge day**, and
blocking in-flight PRs from every other phase — while this phase is scheduled to
merge first. Pinning the *tool* does not help; the advisory database is queried
live.

`continue-on-error` is not an option: it recreates the hole this phase exists to
close.

**Decided: fail on non-stdlib findings only; report stdlib findings as a warning.**

Rationale: a stdlib advisory is fixed by bumping the toolchain — which GoReleaser
and the workflow control, not the author of an unrelated PR. Failing their build
is noise, and noise is what gets a gate disabled.

```
govulncheck -json ./... > vulns.json
# fail the job when any finding's module is not the Go standard library
# surface stdlib findings in the job summary without failing
```

Implementation notes:
- Parse the `-json` stream rather than grepping the human output — the text
  format is not a stable contract.
- The warning must be **visible**, not buried: write it to the job summary so a
  stdlib advisory still prompts a deliberate toolchain bump instead of being
  silently swallowed. A warning nobody reads is `continue-on-error` with extra
  steps.
- Record the policy and its rationale in `CONTRIBUTING.md`, so the next person
  who sees a stdlib advisory pass CI knows it was a decision, not an oversight.

### One third of CI is not a required check

```
gh api repos/bavanchun/Typeburn/branches/main/protection \
  --jq '.required_status_checks.contexts'
-> ["Build & Test (ubuntu-latest)", "Build & Test (macos-latest)"]
```

The `installer` job — `name: install.sh & release config` (`ci.yml:47`) — is
absent. CLAUDE.md states "`ci.yml` must pass" as a hard gate; it does not.

Consequence: `install.sh` is what `README.md:31` tells users to pipe into a shell,
and `goreleaser check` is the **only** pre-tag validation of the release config —
`release.yml` never runs it. A broken config publishes a partial release under an
immutable tag.

**Pre-state, recorded verbatim for rollback** (this is external state and is not
`git revert`-able):

```json
{"contexts":["Build & Test (ubuntu-latest)","Build & Test (macos-latest)"],
 "strict":true}
```

### `.gitignore` swallows new source files

```
$ sed -n 25p .gitignore
typeburn
$ git check-ignore -v cmd/typeburn/newfile.go
.gitignore:25:typeburn   cmd/typeburn/newfile.go
```

A bare pattern with no leading slash matches any path component named `typeburn`
at any depth. Tracked files are unaffected (tracked beats ignore), which is why it
went unnoticed — but **a new file under `cmd/typeburn/` is silently skipped by
`git add .`**, builds locally, and vanishes from the pushed tree.

The rule exists only for `README.md:83`'s `go build -o typeburn ./cmd/typeburn`;
`make build` already writes to `./bin/`, ignored at `:2`. Fix: `/typeburn`.

### Also here

- `.github/dependabot.yml` for `gomod` + `github-actions` — absent today, so all
  16 outdated indirects and 6 outdated directs are manual.
- `Makefile:44-45` records a stale 5,302,642 B size baseline; actual is
  **9,002,530 B, 85.9% of the 10 MiB cap**. Update the comment so the next person
  sees real headroom.
- **C7 — no gate that `.github/release-notes.md` matches the tag.**
  `release.yml:103,111` passes `--release-notes` unconditionally; forget to update
  it and GoReleaser silently publishes the previous release's notes under a new,
  immutable tag. A three-line grep asserting `github.ref_name` appears in the
  file's first heading closes it.

## Related Code Files

- Modify: `go.mod`, `go.sum` (`golang.org/x/text` → v0.40.0)
- Modify: `.github/workflows/ci.yml` (govulncheck step)
- Modify: `.github/workflows/release.yml` (release-notes/tag assertion only)
- Create: `.github/dependabot.yml`
- Modify: `.gitignore` (line 25), `Makefile` (size comment), `CONTRIBUTING.md` (stdlib policy)
- External: branch-protection `required_status_checks.contexts`

## Implementation Steps

1. Decide the stdlib policy (open question 6). Nothing else here is blocked by it,
   but the CI step is.
2. `go get golang.org/x/text@v0.40.0 && go mod tidy`. Run the full suite **before**
   regenerating any golden.
3. Add the govulncheck step under the chosen policy, tool version pinned.
4. `.github/dependabot.yml`.
5. `.gitignore` line 25 → `/typeburn`. Verify `git check-ignore -v
   cmd/typeburn/newfile.go` returns nothing **and** a bare root `typeburn` binary
   is still ignored.
6. Release-notes/tag assertion in `release.yml`.
7. Add `install.sh & release config` to required checks; **verify by reading the
   protection API back**, not by assuming the write landed.
8. Update the Makefile size comment; document the stdlib policy in `CONTRIBUTING.md`.

## Success Criteria

- [ ] `govulncheck ./...` reports no non-stdlib findings
- [ ] govulncheck runs in CI at a pinned tool version, under a documented stdlib policy
- [ ] `.github/dependabot.yml` covers `gomod` and `github-actions`
- [ ] `git check-ignore cmd/typeburn/newfile.go` returns nothing; root `typeburn` still ignored
- [ ] `required_status_checks.contexts` includes all three job names, verified by reading the API back
- [ ] `release.yml` fails when `release-notes.md` does not mention the tag
- [ ] Full suite green after the bump; goldens unchanged (or the diff explained)
- [ ] `make size-check` passes; Makefile comment matches reality

## Risk Assessment

**Risk:** the govulncheck gate red-walls `main` and blocks every other phase.
**Mitigation:** step 1 decides the policy first; the gate does not land until it
is decided. This phase merges early precisely so the blast radius is known before
four other PRs are in flight.

**Risk:** the `x/text` bump changes text shaping and alters rendered output.
**Mitigation:** low, given x/text is not on the render path — but step 2 still
runs the suite before any golden is regenerated. A failing golden is a signal to
investigate, never to regenerate.

**Risk:** adding a required check blocks in-flight PRs until they re-run CI.
**Mitigation was wrong, corrected during execution.** "They re-run on push" only
holds for a check that *exists* on the branch. `govulncheck` is a new job, so a
branch cut before this phase merged never reports that context at all — the PR
does not go red, it waits forever. Re-running CI cannot fix it; only a rebase
onto a `main` containing the job can.

**Therefore the protection write is the last step of this phase, not part of the
code change**, and happens only after this phase's own PR merges. Order: merge
in-flight PRs → merge this phase → *then* add the required contexts. Any branch
cut afterwards inherits the job.

Required contexts are four, not three — the job matrix expands `Build & Test`
into one context per OS:
`Build & Test (ubuntu-latest)`, `Build & Test (macos-latest)`, `govulncheck`,
`install.sh & release config`.

## Rollback

Code changes revert with the commit. **The branch-protection change does not** —
it is external state. To roll back, restore the pre-state recorded verbatim above
via `gh api`. Record any further protection change the same way before making it.
