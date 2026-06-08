---
title: "Audit Hardening — Bug Fixes + Test Coverage"
description: >-
  Fix 2 accepted bugs (itoa infinite loop, modeIdx reset) and add test coverage
  to 8 high-churn files identified by code review + git retro audit.
status: completed
priority: P1
effort: 4h
branch: main
tags:
  - bugfix
  - test
  - hardening
blockedBy: []
blocks: []
created: '2026-06-08'
---

# Audit Hardening — Bug Fixes + Test Coverage

## Overview

Execute the 6 action items from the code review + git retro audit report.
Two accepted bugs must be fixed first (Phase 1), then test coverage is added
to high-churn files across 3 parallel tracks (Phases 2–4).

## Source

- [Code Review + Retro Report](../../.gemini/antigravity-cli/brain/0888844e-0299-4e85-b8db-0e787b81ba56/code-review-and-retro-report.md)
- [Git Retro](./../../plans/reports/retro-260608-alltime.md)

## Dependency Graph

```
Phase 1 (bugs) ─┬─→ Phase 2 (storage tests)  ← parallel
                 ├─→ Phase 3 (app tests)       ← parallel
                 └─→ Phase 4 (typing+ui tests) ← parallel
```

## File Ownership Matrix

| File | Phase |
|------|-------|
| `internal/storage/new_best.go` | 1 |
| `internal/app/model_settings.go` | 1 |
| `internal/ui/screen_home.go` | 1 |
| `internal/storage/new_best_test.go` | 2 |
| `internal/storage/history_store_test.go` | 2 |
| `internal/app/model_view_test.go` (new) | 3 |
| `internal/app/model_settings_test.go` (new) | 3 |
| `internal/app/model_history_test.go` (new) | 3 |
| `internal/typing/completion_test.go` (new) | 4 |
| `internal/ui/screen_typing_actions_test.go` (new) | 4 |

## Phases

| Phase | Name | Status | Depends On |
|-------|------|--------|------------|
| 1 | [Fix Accepted Bugs](./phase-01-fix-accepted-bugs.md) | ✅ Completed | — |
| 2 | [Storage Test Hardening](./phase-02-storage-test-hardening.md) | ✅ Completed | Phase 1 |
| 3 | [App Test Hardening](./phase-03-app-test-hardening.md) | ✅ Completed | Phase 1 |
| 4 | [Typing + UI Test Hardening](./phase-04-typing-ui-test-hardening.md) | ✅ Completed | Phase 1 |
