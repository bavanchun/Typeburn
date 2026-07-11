# Go v2 Command Contract Research

- Official Go guidance requires v2 module paths to end in `/v2` and all
  intra-module imports to use that prefix.
- `cmd/go` names a root executable after the last non-major path component, so
  root `github.com/bavanchun/Typeburn/v2` installs as case-sensitive `Typeburn`.
- A disposable experiment confirmed root install creates `Typeburn`, while
  `./cmd/typeburn` creates lowercase `typeburn`.
- Selected contract: root module `github.com/bavanchun/Typeburn/v2`, command
  package `cmd/typeburn`, remote install `/v2/cmd/typeburn@v2.5.1`.
- Root-level v2 module layout is valid. A physical `v2/` directory is useful for
  parallel major maintenance but adds unnecessary corrective scope here.
- Official sources: https://go.dev/doc/modules/major-version,
  https://go.dev/doc/modules/gomod-ref, https://pkg.go.dev/cmd/go,
  https://go.dev/doc/modules/publishing.

## Verified Decision

Move the entrypoint to `cmd/typeburn`; this is required to keep the established
lowercase binary across Go install and release archives.
