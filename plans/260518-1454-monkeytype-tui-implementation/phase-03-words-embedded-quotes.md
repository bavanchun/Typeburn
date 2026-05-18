---
phase: 3
title: "Words & embedded quotes"
status: pending
priority: P1
effort: ~3h
dependencies: [2]
---

# Phase 3: Words & embedded quotes

## Overview

`internal/words`: deterministic-seedable random word-list generator for Time/Words modes, plus an embedded quote pack (`go:embed`) bucketed short/medium/long for Quote mode. Pure, unit-tested. Feeds the typing engine's target string in Phase 4.

Refs: researcher-02 §6 (per-mode); design §8.2 (length options).

## Requirements

### Functional
- Word generator: returns a space-joined string of N common English words; optional seed for deterministic tests.
- Time mode: generate a large buffer (overshoot — e.g. words to comfortably fill max duration; regenerate/extend if exhausted).
- Words mode: generate exactly N words (N ∈ 10/25/50/100).
- Quote pack: embedded text file(s); buckets short(<100 chars)/medium(100–250)/long(>250); selection API returns one quote (random or seeded) per bucket.

### Non-functional
- Words list embedded (no network). ASCII lowercase words (rune-safe regardless).
- Deterministic with seed; files <200 lines (word list data file exempt from logic-line rule but keep tidy).

## Architecture

Data flow: caller picks mode+length → `words.ForMode(...)` → target string → `typing.New(target, ...)`.

```go
// internal/words
type QuoteLen int
const ( QuoteShort QuoteLen = iota; QuoteMedium; QuoteLong )

//go:embed data/english-1000.txt
var wordListRaw string
//go:embed data/quotes.txt
var quotesRaw string

func NewGenerator(seed int64) *Generator   // seed=0 → time-seeded
func (g *Generator) Words(n int) string     // n space-joined words
func (g *Generator) TimeBuffer() string     // large overshoot buffer
func (g *Generator) Quote(l QuoteLen) Quote
type Quote struct { Text, Source string; Bucket QuoteLen }
func ForMode(g *Generator, mode typing.Mode, length int, ql QuoteLen) (target string)
```

Quote file format: one quote per line `text\tsource` (tab-sep); parsed once, bucketed by rune length at load.

## Related Code Files

Create:
- `internal/words/generator.go`, `internal/words/quotes.go`, `internal/words/for-mode.go`
- `internal/words/data/english-1000.txt` (≈1000 common words, embedded)
- `internal/words/data/quotes.txt` (≥30 quotes spanning all 3 buckets)
- `internal/words/generator_test.go`, `internal/words/quotes_test.go`

Modify: none. Delete: none.

## Implementation Steps

1. Add `data/english-1000.txt` (common English words, one per line) and `data/quotes.txt` (`text\tsource`, ≥30, covering short/medium/long).
2. `generator.go`: `*rand.Rand` from seed (0 → `time.Now().UnixNano()`); `Words(n)` picks n words; `TimeBuffer()` returns enough words for 120s at high WPM (~600 words) and is safely re-extendable.
3. `quotes.go`: parse `quotesRaw` once, bucket by `utf8.RuneCountInString`; `Quote(l)` random/seeded pick within bucket; fallback to nearest non-empty bucket if a bucket is empty (assert non-empty in test instead).
4. `for-mode.go`: dispatch mode→target (Time→TimeBuffer, Words→Words(n), Quote→Quote(ql).Text).
5. Tests: same seed → identical output (determinism); `Words(25)` → exactly 25 space-separated tokens; each quote bucket non-empty & length-correct; `ForMode` returns non-empty per mode.
6. `go test ./internal/words/... -race`; build/vet/gofmt.

## Success Criteria

- [ ] Same seed → byte-identical word output (deterministic).
- [ ] `Words(N)` returns exactly N words for N ∈ {10,25,50,100}.
- [ ] All 3 quote buckets non-empty and length-classified correctly.
- [ ] Embedded data loads with no filesystem/network access at runtime.
- [ ] `go test ./... -race` passes; build/vet/gofmt clean.

## Risk Assessment

| Risk | L×I | Mitigation |
|---|---|---|
| `go:embed` path mismatch | L×M | Embed dir co-located under package; build catches it |
| Quote bucket empty | L×M | Curate ≥30 quotes across buckets; test asserts each non-empty |
| Time buffer exhausted on fast 120s run | L×M | Overshoot ~600 words + re-extend hook used by Phase 4 |
