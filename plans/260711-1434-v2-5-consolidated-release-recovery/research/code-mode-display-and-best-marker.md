# Research: Code Mode Display and Best Marker

## Summary

Both UI formatters predate Code mode and map their default branch to Quote.
Storage persists Code correctly. Separate scout evidence shows History computes
stars for all records even though `IsNewBest` excludes Code and Strict.

## Findings

- `history_table.go` and `result_render_helpers.go` duplicate mode formatting.
- One string-based UI formatter preserves forward-compatible raw history modes.
- `BestWPMPerBucket` and History row comparison need the same eligibility rule
  used by `IsNewBest`; bucket filtering alone is insufficient when WPM values tie.
- Focused new test files avoid growing existing screen tests beyond 200 lines.

## Recommendation

Tests first. Add one UI formatter and one storage eligibility predicate. Reuse
each at all production call sites. Keep schema, record length and PB buckets intact.

## Unresolved Questions

None.
