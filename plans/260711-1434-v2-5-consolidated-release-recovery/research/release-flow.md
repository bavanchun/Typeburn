# Research: Consolidated v2.5.0 Release Flow

## Summary

PR #57 is open, mergeable and CI-green. Protected main requires PRs and squash
merges. The tag-triggered release workflow self-tests, publishes six archives
plus checksums, and updates a separate Homebrew tap only for stable releases.

## Findings

- Extend PR #57 with Code/display and truth-doc changes; no force-push needed.
- Use a focused release-prep PR for metadata plus privileged tag-workflow
  hardening, then freeze its squash SHA.
- Local snapshot does not prove auth, upload, notes or tap behavior.
- A unique disposable prerelease must empirically prove full publish and cleanup.
- Real tags are immutable because Go proxy/sumdb are append-only; fix forward.

## Recommendation

Execute strict sequential gates. Match workflow evidence by tag/SHA, assert seven
assets and exact notes, verify prerelease isolation, then publish v2.5.0 on the
same proven SHA and validate every distribution channel.

## Unresolved Questions

None.
