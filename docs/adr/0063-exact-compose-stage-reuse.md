# ADR 0063: Reuse only exact completed compose stages

- Status: accepted
- Date: 2026-09-09

## Context

When a compose targeted an existing model, WALDO removed every corpus path that
appeared in any completed run. Logical paths are selectors, not immutable data
identity. An authoritative corpus rebuild can replace manifests and shards at
the same path, so path-only reuse could silently treat new training data as
completed. It could also rewrite one declared multi-corpus stage into partial
work whose provenance no longer described the submitted compose.

## Decision

WALDO resolves each stage's current filtered OpenWALDO BOM before materializing
shards. It reuses a completed stage only when a completed run BOM matches the
stage name, type, objective, conversation transformation, exact corpus BOM
SHA-256, and normalized resolved training parameters. Epoch-derived stages use
the completed run's step count only as the deterministic resolution candidate;
the exact corpus BOM and declarative epoch parameters must still match.

Matches must form a prefix of the requested compose and a contiguous suffix
ending at the model's latest completed run. This binds reuse to the current
weights: an otherwise identical run earlier in history cannot suppress a stage
after a newer completed run changed the model state.

Reuse is all-or-nothing at the stage boundary. WALDO never removes individual
corpus selections or rewrites their weights. Failed and incomplete runs do not
qualify. Pending compose transactions retain their existing exact resume path
and do not use history reuse.

## Consequences

- An exact repeat avoids shard download and another trainer invocation.
- A corpus rebuild at the same path executes the complete declared stage.
- Parameter, objective, or conversation changes cannot inherit a path-only
  completion decision.
- Completed work behind newer model weights is not mistaken for current state.
- A stage that intentionally repeats old data alongside new data remains one
  auditable unit rather than being silently reduced.
- The existing run BOM contains every required identity, so no durable schema
  migration is required.
