---
phase: 5
title: "Publish and verify v2.5.1 fix-forward"
status: done
effort: "L"
---

# Phase 5: Publish and verify v2.5.1 fix-forward

## Objective

Publish one immutable stable tag on the proven SHA, verify every channel, and reconcile the old recovery plan.

## State Inventory

| Surface | Required proof |
|---|---|
| Git tag and release | annotated v2.5.1, exact SHA/run, stable/latest |
| Assets | seven files, checksums, archive member, banner |
| Installer, Homebrew, updater | pinned install, Cask, latest discovery agree |
| Go proxy and sumdb | `/v2` list/info plus exact and latest installs |
| Plans and docs | corrective completion and old-plan supersession |

## Implementation Steps

1. Assert local/remote v2.5.1 absence, `origin/main` still contains frozen SHA,
   tap rollback readiness, and clean disposable state. Create annotated tag,
   prove `v2.5.1^{commit}=RELEASE_SHA`, push only its tag ref, then re-read remote
   tag object and peeled commit.
2. Capture run ID/attempt and assert workflow, push event, exact tag, head SHA,
   creation window, and all required job conclusions.
3. Verify latest, notes, seven assets, checksums, archive members, executable name, and version.
4. Verify pinned installer, Homebrew Cask, and updater discovery.
5. Poll proxy list, info, and mod endpoints within a documented deadline/backoff;
   require v2.5.1 metadata and `/v2` module directive. Record first success time.
6. Use two independent temp environments for exact and latest. Set proxy-only
   `GOPROXY`, sumdb, `GOWORK=off`, `GOENV=off`; empty private/no-proxy/no-sumdb
   variables; use distinct module, compile, and binary caches. Assert environment,
   module resolution v2.5.1, exact executable set `{typeburn}`, runtime v2.5.1,
   and `go version -m` command path, module path, and version.
7. Complete metadata through a merged post-release docs PR, refetch remote truth,
   then close the old plan using supported `completed` status with an explicit
   Outcome/Supersession record. Leave its impossible v2.5.0 criterion unchecked.

## Failure Classification

| Failure | Action |
|---|---|
| Proxy 404 within bound | wait and retry; never retag |
| Semantic path, name, or version failure | block completion; fix forward v2.5.2 |
| Channel mismatch after stable mutation | block completion; fix forward v2.5.2 |
| Pre-tag failure | repair PR and repeat Phase 4 |

## Partial-Publish Containment

If workflow fails after GitHub/tap mutation: capture evidence; contain a bad
GitHub release as draft when possible; stop announcements and install checks;
revert the exact tap commit with the verified human credential. Retry unchanged
transient infrastructure only when content/source is identical; otherwise
publish v2.5.2. Never move or delete the v2.5.1 tag.

## Dependencies

- Requires Phase 4 frozen SHA and clean disposable state.
- Unblocks old plan only after public proxy and all channels pass.

## Success Criteria

- [x] Annotated v2.5.1 and run `29159099750` point to frozen main SHA `8307ee6c`.
- [x] GitHub, installer, Homebrew, updater, seven assets, and banners agree.
- [x] Proxy-only exact and latest installs create lowercase typeburn v2.5.1.
- [x] v2.5.0 stays immutable; old plan closes as superseded and recovered.
- [x] Completion metadata and journal prepared for post-release documentation PR.
