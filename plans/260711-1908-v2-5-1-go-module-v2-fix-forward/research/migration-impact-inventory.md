# Migration Impact Inventory

- `go.mod` has the old module directive.
- 279 safe old internal-import occurrences span 133 tracked Go files: app 20,
  cli 22, config 2, metrics 12, runner 2, storage 3, theme 1, typing 9, ui 57,
  update 1, version 1, words 2, and root main 1.
- Root `main.go` is the only main package and moves to `cmd/typeburn/main.go`.
- `Makefile` needs the command package and `/v2/internal/version` linker prefix.
- `.goreleaser.yaml` needs `main: ./cmd/typeburn` and three v2 linker symbols.
- Existing workflow `./...` commands already discover the new command; no
  workflow change is currently justified.
- Runtime commands occur in `internal/update/selfpath.go` and
  `internal/cli/cmd_version.go`; both need exact regression assertions.
- Evergreen documentation impact includes README, CLAUDE, CONTRIBUTING,
  code standards/summary, architecture, CLI reference, overview, roadmap,
  changelog, and release notes.
- Repository, raw, API, release, installer, and security URLs must not be
  mechanically changed. Only module/import/install/linker paths gain `/v2`.
