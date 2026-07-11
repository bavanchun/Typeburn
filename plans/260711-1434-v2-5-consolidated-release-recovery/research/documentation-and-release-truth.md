# Research: Documentation and Release Truth

## Summary

Runtime has eight themes, eight persisted/CLI keys and seven Settings UI rows.
`mono` is grayscale color; only `NO_COLOR` is attribute-only. Public stable is
v2.4.1; Strict is merged/unreleased and punctuation/numbers is PR #57/unreleased.

## Findings

- README, CLI reference, architecture, codebase summary, PDR and roadmap contain
  stale or ambiguous counts and theme semantics.
- CHANGELOG cut v2.5.0 prematurely; release notes correctly remain v2.4.1.
- Historical v1 statements must not be blindly rewritten as current product facts.
- Release documentation needs integration, release-prep and published states.

## Recommendation

Normalize all unreleased work under CHANGELOG Unreleased during integration.
Cut v2.5 and release notes only in the dedicated release-prep transaction.
Exclude journals from global drift replacement and staging.

## Unresolved Questions

None.
