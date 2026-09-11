# 0069: Make unset-language inclusion explicit

Status: accepted

## Context

Many legacy and primitive-format corpus rows have no language value even when
their source manifest declares a language. Treating an unset row as a declared
source language would invent per-record evidence. Conversely, silently making
every language inclusion filter accept unset rows would weaken its meaning.

## Decision

Language value filters accept an optional `include_unset: true`. With this
setting, rows whose language is empty pass alongside rows matching `include`;
known nonmatching languages remain excluded. The default is `false`.

The option is invalid for license and source value filters, and it requires a
nonempty `include` list. It is persisted in the corpus BOM and therefore
participates in training and preflight identity.

## Consequences

Composes can deliberately retain older unclassified rows without claiming
that those rows are English. Strict language filtering remains the default,
and changing this policy invalidates cached preflight selection as expected.
