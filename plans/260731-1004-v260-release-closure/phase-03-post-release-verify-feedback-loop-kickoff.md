---
phase: 3
title: "Post-Release Verify + Feedback Loop Kickoff"
status: pending
priority: P2
effort: "1h"
dependencies: [2]
---

# Phase 3: Post-Release Verify + Feedback Loop Kickoff

## Overview

Verify every install/update channel actually delivers v2.6.0, then close the
delivery loop administratively: record completion and hand the roadmap's
"gather user feedback" item off as the next strategic (non-technical)
decision point.

## Requirements

- Functional: all four channels verified — release archive + checksums,
  `go install @v2.6.0`, `typeburn update --check` from an older binary,
  `install.sh`. Homebrew cask bump checked (tap repo may need the new
  version/sha).
- Non-functional: no repo changes except the completion-record PR; tolerate
  Go-proxy ingest lag (~1 h) before declaring the module channel broken.

## Architecture

Verification is read-only against published artifacts. The completion record
follows the repo's precedent (`docs(release): record v2.5.x completion` PRs):
roadmap release-history entry finalized + journal. Feedback kickoff is a
roadmap note, not a feature commitment — deciding WHAT feedback to gather and
through which channel (GitHub Discussions? issue template? README call-out?)
is the user's strategic call, explicitly out of scope here (YAGNI).

## Related Code Files

- Modify: `docs/project-roadmap.md` (release-history entry final; "Next"
  section points at feedback gathering as the active item)
- Add: `docs/journals/<ts>-v260-release-closure.md` (journal via /ak:journal)

## Implementation Steps

1. Download one release archive + `checksums.txt`; verify sha256 locally.
2. `GOBIN=$(mktemp -d) go install github.com/bavanchun/Typeburn/v2/cmd/typeburn@v2.6.0`
   → `typeburn --version` shows v2.6.0 (retry within ~1 h if proxy 404s).
3. From a v2.5.1 binary: `typeburn update --check` reports v2.6.0 available.
4. Run `install.sh` into a temp `BIN_DIR`; confirm sha verification + version.
5. Check `bavanchun/homebrew-tap-typeburn` cask version; bump if it pins
   versions manually (follow that repo's own convention).
6. Completion-record PR (`docs(release): record v2.6.0 completion`) +
   `/ak:journal`; archive this plan.

## Success Criteria

- [x] 4 channels verified delivering v2.6.0 (proxy-lag retries allowed).
- [x] Homebrew cask current or bumped.
- [x] Completion PR merged; journal written; plan archived.
- [x] Roadmap "Next" clearly frames feedback gathering as the user's next
      strategic decision — no new feature work started.

## Risk Assessment

- **Proxy lag mistaken for a broken module channel:** wait/retry up to ~1 h
  before escalating; the archive channel is immediate and independent.
- **Scope creep into feature work:** feedback-channel implementation (e.g.
  Discussions setup) only on explicit user instruction — this phase ends the
  loop, it does not start the next one.
