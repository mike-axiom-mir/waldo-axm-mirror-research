# ADR 0064: Expose stage reuse as run-anchored evidence

- Status: accepted
- Date: 2026-09-10

## Context

ADR 0063 makes completed-stage reuse an exact, all-or-nothing decision bound
to the current run-history suffix. Its first observable result represented that
stage decision as repeated corpus-path `skipped` entries. A caller could see
which logical paths were omitted, but not which completed run and immutable
BOM caused WALDO to omit them. Partial compose execution also dropped that
reuse information from JSON after the remaining stages trained.

## Decision

Return one derived evidence record per reused stage. Each record carries the
stage name, complete declared corpus set, matched run ID and ordinal, run-BOM
SHA-256, and corpus-BOM SHA-256. Both an all-reused result and a result that
trains later stages expose the same `reused` projection.

The projection is reconstructed from the already verified model inspection.
Run history and run BOMs remain authoritative; WALDO does not persist a second
completion fact or grant the result mutation, training, release, or publication
authority.

## Consequences

- Operators and automation can trace every saved trainer invocation to exact
  existing evidence instead of interpreting a logical path as proof.
- Multi-corpus stages remain visibly atomic rather than appearing to be a set
  of independent path skips.
- Partial compose runs retain evidence for their reused prefix.
- Reconstructing the projection is bounded by the number of reused stages and
  adds no model-state write or network dependency.
