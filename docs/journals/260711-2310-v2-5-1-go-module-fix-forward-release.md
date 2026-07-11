---
title: Typeburn v2.5.1 Go Module Fix-Forward Release
date: 2026-07-11
status: complete
release: v2.5.1
---

# Typeburn v2.5.1 Go Module Fix-Forward Release

## Context

The immutable `v2.5.0` release shipped working archives but exposed an invalid Go v2 install path. The corrective release migrated the module to `github.com/bavanchun/Typeburn/v2`, moved the command entrypoint to `cmd/typeburn`, and preserved the lowercase `typeburn` executable across all install channels.

## What Happened

- PR [#59](https://github.com/bavanchun/Typeburn/pull/59) merged into `main` as `8307ee6c9384dceb27aaa639bdac980b43906e0b`.
- Disposable release run [29158988295](https://github.com/bavanchun/Typeburn/actions/runs/29158988295) completed successfully from the merge SHA.
- Immutable release run [29159099750](https://github.com/bavanchun/Typeburn/actions/runs/29159099750) published stable `v2.5.1` successfully.
- The release contains checksums plus six Darwin, Linux, and Windows archives for amd64 and arm64.

## Verification Evidence

- Public Go proxy exact lookup and `@latest` both resolve `v2.5.1` to `8307ee6c`.
- `go install github.com/bavanchun/Typeburn/v2/cmd/typeburn@v2.5.1` resolves through the public module channel and installs lowercase `typeburn`.
- The installer resolves and verifies the `v2.5.1` release archive.
- Homebrew cask now declares version `2.5.1` with release-specific hashes for all supported macOS and Linux architectures.
- The updater discovers `v2.5.1` as the latest stable release and preserves package-manager guidance.

## Reflection

Release archives alone were insufficient proof of a healthy Go release. A major-version module must be validated through the public proxy, exact and latest queries, installed command name, and binary build metadata. Disposable release execution caught packaging risk before the immutable tag was published.

## Decisions

- Keep `v2.5.0` immutable; repair the distribution contract only through `v2.5.1`.
- Use `/v2` only for Go module and import paths; GitHub repository, raw-content, and release URLs remain unchanged.
- Keep `cmd/typeburn` as the canonical Go command path so `go install` produces the established lowercase binary name.
- Treat release tags and published assets as immutable. Any future defect receives another fix-forward version.

## Outcome

The corrective release is complete. GitHub Release, public Go proxy, installer, Homebrew, and updater all converge on `v2.5.1` at merge SHA `8307ee6c`.

## Next Steps

None. No unresolved questions.
