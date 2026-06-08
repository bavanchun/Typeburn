---
title: Architecture Performance Hardening
date: 2026-06-04
type: journal
---

# Architecture Performance Hardening

## Context

Implemented plan `plans/260604-1354-architecture-performance-hardening/plan.md`
after architecture/performance audit of Typeburn.

## What Happened

Added benchmark coverage for typing engine, metrics, TUI render, and storage
new-best detection. Baseline showed full TUI render is the real allocation/time
hotspot, while live metrics and storage were modest.

Applied small hot-path fixes: `typing.Engine.Typed()` exposes current buffer as
a copy, `ForwardKeystrokes()` lets the tick path compute live WPM without
copying/replaying the log, and `metrics.LiveWPMFromCount()` keeps that formula
pure.

Split shared mode definitions into `internal/mode`. `config.Mode` remains an
alias for compatibility, but production core packages now import `internal/mode`
instead of `internal/config`.

Centralized best-bucket comparison in `internal/storage` so persistence and UI
history badges use the same effective-WPM policy.

## Decisions

- Keep `config.Keymap` in `internal/config`; repo rules say this Bubble Tea
  key binding boundary is deliberate.
- Do not rewrite the renderer. Benchmarks identify render cost, but the planned
  scope was measurement plus low-risk allocation cleanup.
- Keep history JSON schema unchanged.

## Next

If future performance work is needed, target render-windowing or style batching
in code-mode rendering, using the new benchmarks as guardrails.
