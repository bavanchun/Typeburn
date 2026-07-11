# Phase 1 RED Contract Proof

## Reproduction

Reproduced on 2026-07-11 in a disposable detached worktree at the immutable
v2.5.0 commit `758dcb8c`. Only the Phase 1 test diff from `ef36f6e3` was applied;
production code remained at v2.5.0.

```text
go test ./internal/update ./internal/cli \
  -run 'TestInstructionFor|TestVersionCheckUpdate_UpgradeAvailable' -count=1

--- FAIL: TestInstructionFor_V2Contract
instructionFor(InstallGo) =
"go install github.com/bavanchun/Typeburn@latest",
want "go install github.com/bavanchun/Typeburn/v2/cmd/typeburn@latest"

FAIL
RED_TEST_EXIT=1
```

The CLI test diff also imports the future `/v2/internal/update` path, so it
correctly cannot compile against the unmodified v2.5.0 module. The focused
update-package failure independently proves the intended old-command mismatch.

## GREEN Evidence

At `ef36f6e3` and later, the focused tests and full race suite pass with the
new module path and lowercase command contract.
