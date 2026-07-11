---
phase: 2
title: "Migrate module imports and command entrypoint"
status: done
effort: "L"
---

# Phase 2: Migrate module imports and command entrypoint

## Objective

Make the root a valid v2 module and preserve lowercase command identity across
local install, Make, and GoReleaser.

## File Inventory

| Files | Action | Purpose |
|---|---|---|
| `go.mod` | modify | append `/v2` module suffix |
| 133 Go files | scoped mechanical edit | rewrite 279 internal imports |
| `main.go` → `cmd/typeburn/main.go` | move/edit | lowercase command package |
| `Makefile` | modify | v2 linker prefix; command package |
| `.goreleaser.yaml` | modify | main path and v2 linker symbols |
| `internal/update/selfpath.go` | modify | canonical install advice |
| `internal/cli/cmd_version.go` | modify | canonical upgrade advice |
| `internal/version/version.go` | modify | accurate version example |

## Implementation Steps

1. Change module directive to `github.com/bavanchun/Typeburn/v2`.
2. Rewrite only Go self-imports; reject every Go import beginning with the repo
   prefix unless it begins with `/v2/`. Audit non-Go URLs separately.
3. Move the sole entrypoint to `cmd/typeburn/main.go`; remove root main.
4. Point Make and GoReleaser at `./cmd/typeburn`; update `/v2/internal/version`
   symbols while retaining output name `typeburn` and archive targets.
5. Implement Phase 1 runtime contracts and make focused tests green.
6. Run `go mod tidy -diff`, `go mod verify`, module/import/root-main guards,
   isolated local install, full quality gates, size check, and exact snapshot inspection.

## Function / Interface Checklist

- Move the current main body without refactoring: retain `cli.NewRoot`,
  `fang.Execute`, `fang.WithoutVersion`, and `cli.ExitCode`; only import changes.
- Version injection covers `Version`, `Commit`, and `Date` under `/v2`.
- Both update advice paths use the canonical command.
- Public CLI, config, storage, archive names, and GitHub URLs do not change.

## Test Matrix

| Gate | Assertion |
|---|---|
| Module | `go list -m` equals the `/v2` path |
| Imports | no old `/internal/` prefix in tracked Go files |
| Root command | no non-test root `package main`; sole command is `./cmd/typeburn` |
| Command | isolated `go install ./cmd/typeburn` creates only `typeburn` |
| Module hygiene | tidy diff empty, verify passes, `go.sum` unchanged absent evidence |
| Quality | gofmt, list, build all packages, vet, race suite |
| Release | exactly six OS/arch archives plus checksums; each archive has lowercase command and docs |

## Dependencies

- Requires Phase 1 red tests; supplies final paths to Phase 3.

## Success Criteria

- [ ] Module/import and lowercase command proofs pass.
- [ ] Full quality, size, and snapshot gates pass.
- [ ] Workflow files stay byte-identical unless a verified gap is documented.
- [ ] Commit: `fix(module): migrate Typeburn to the v2 module path`.
