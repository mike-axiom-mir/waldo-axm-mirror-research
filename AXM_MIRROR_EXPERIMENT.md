# AXM Mirror research on a WALDO fork

**Status: EXPERIMENTAL / public-safe downstream research / not affiliated with or endorsed by OpenWALDO**

This repository is a GitHub-recognized fork of OpenWALDO's `waldo` project used
by AXM to test one narrow question:

> Can WALDO's inspectable model lineage be extended downstream with equally
> inspectable evidence about a model's concrete behavior, permissions,
> verification, dissent, and observed outcome?

This is not the full AXM Mirror system. The public AXM `mirror` branch contains
substantially more research, and additional local/private Mirror material is not
being copied here. This fork intentionally starts with a small public-safe slice.

## Upstream boundary

OpenWALDO/WALDO remains the upstream project and source of the corpus, training,
model, and release BOM architecture used here. AXM is not claiming affiliation,
partnership, endorsement, or ownership of OpenWALDO.

The upstream Apache-2.0 `LICENSE` and `NOTICE` remain intact.

## First implemented slice

The branch `axm/mirror-waldo-experiment-v0.1` adds a fork-only Go package and a
small separate CLI:

```text
WALDO corpus/run/model/release lineage
                 |
                 v
       AXM behavior-evidence draft
                 |
      validate closed authority
                 |
                 v
       SHA-256 sealed evidence
                 |
                 v
      later verification / dissent
```

Build after installing the Go version required by WALDO:

```bash
go build ./cmd/waldo-axm-mirror
```

Seal a draft:

```bash
./waldo-axm-mirror seal examples/axm-mirror/evidence-draft.json /tmp/evidence.sealed.json
```

Verify it later:

```bash
./waldo-axm-mirror verify /tmp/evidence.sealed.json
```

## Truth boundary

The first slice does **not** train a Mirror clone yet. It creates the downstream
provenance/evidence contract needed before local model experiments can be
recorded honestly.

It does not grant tool access, run a model, certify safety, convert dissent into
a score, expose hidden reasoning, or make any AXM result CANON.

## Next experimental rungs

1. Bind a real WALDO-produced model or release BOM to a behavior-evidence record.
2. Add a small public-safe Mirror evaluation fixture set without private memory
   or hidden reasoning.
3. Run a local model through bounded scenarios and preserve outputs by digest.
4. Record permission state, deterministic checks, unresolved dissent, and result.
5. Only then evaluate whether a small WALDO-trained Mirror research clone is
   technically and legally appropriate for the selected corpus.

Large or upstream-facing changes stay out until this fork has something tested
and useful to show.
