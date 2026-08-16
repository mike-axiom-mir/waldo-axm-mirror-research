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

The v0.6 experimental stack merges the exact OpenWALDO head
`451e029abd1f74fd77625984526a1980c48fb477`. The merge and its downstream
compatibility work are recorded in ADR 9009 and
`research/openwaldo-upstream-alignment-2026-08-16.json`; neither upstream
`main` nor this fork's `main` is changed. The inherited v0.6 organ-map overlay
records the upgraded corpus witness and keeps the still-missing continuity and
specialist-growth surfaces explicit.

## Implemented witness slices

The stacked `axm/mirror-waldo-experiment-*` branches add a fork-only Go package
and a small separate CLI. The current deterministic path is:

```mermaid
flowchart TD
    A["WALDO corpus BOM"] --> B["Corpus evidence lens"]
    B --> C["RUN-BOM.json + RUN.json"]
    C --> D["Training-run witness"]
    E["WALDO model/release BOM"] --> F["Origin anchor + identity lock"]
    D --> G["Witnessed behavior-evidence seal"]
    F --> G
    H["Evaluation inventories"] --> I["Contamination guard"]
    I --> G
    K["Typed Sensorium receipts"] --> L["Situated context"]
    M["Skill + backup manifests"] --> L
    N["AI-native seams + human advice"] --> L
    L --> G
    G --> J["Later behavior delta and dissent"]
    O["Bounded pixel recipe"] --> P["Inner asset foundry"]
    P --> Q["Portable unreviewed .axmasset candidate"]
    R["Self census + external capability snapshot"] --> S["Exact gap and handoff plan"]
    S --> T["Translation-loss receipt"]
    T --> U["Digest-bound return verification"]
```

Build after installing the Go version required by WALDO:

```bash
go build ./cmd/waldo-axm-mirror
```

Project a WALDO corpus BOM into the bounded evidence surface:

```bash
./waldo-axm-mirror lens-corpus examples/axm-mirror/corpus-bom.json /tmp/example.corpus-lens.json
```

Current output uses corpus-lens v0.2 and binds WALDO record-filter,
content-assessment, main-content-era writer, and privacy-redaction evidence.
The checked example remains a legacy schema-1 input on purpose, so its receipt
shows the absence of newer facts instead of inventing them. Legacy v0.1 lens
receipts remain validation-compatible.

Bind its immutable run plan to the current durable run record:

```bash
./waldo-axm-mirror witness-run examples/axm-mirror/run-bom.json examples/axm-mirror/run.json /tmp/example.run-witness.json
```

Anchor a WALDO-produced model or release BOM. All checked-in files are
contract-only examples with placeholder digests and observations, not a
trained model or claim that the named backend actually ran:

```bash
./waldo-axm-mirror anchor examples/axm-mirror/model-release-bom.json /tmp/example.anchor.json
```

Refuse silent answering-identity drift by comparing an expected anchor with a
freshly observed BOM:

```bash
./waldo-axm-mirror lock /tmp/example.anchor.json examples/axm-mirror/model-release-bom.json /tmp/example.lock.json
```

Check exact declared training/evaluation overlap:

```bash
./waldo-axm-mirror contamination examples/axm-mirror/evaluation-comparison.json /tmp/example.contamination.json
```

For a run-backed model or release, seal a draft only after the model, run, and
corpus identities all match their deterministic receipts:

```bash
./waldo-axm-mirror seal-witnessed /tmp/example.anchor.json /tmp/example.run-witness.json examples/axm-mirror/evidence-draft.json /tmp/evidence.sealed.json
```

`seal-anchored` remains the correct path for an origin-backed model that has no
WALDO training run. The original contract-only `seal` command also remains
available for isolated schema tests:

```bash
./waldo-axm-mirror seal-anchored /tmp/example.anchor.json examples/axm-mirror/evidence-draft.json /tmp/evidence.anchored.json
./waldo-axm-mirror seal examples/axm-mirror/evidence-draft.json /tmp/evidence.unanchored.sealed.json
```

Verify it later:

```bash
./waldo-axm-mirror verify /tmp/evidence.sealed.json
```

### Situated clone boundary

The Wave 2.1 path adds receipt-only situation around the existing provenance
and evaluation gates:

```text
intake-sensory + assess-skills + discover
  -> situated-context
  -> seal-situated
  -> verify
```

It recognizes all thirteen public Workshop Sensorium contracts without copying
their executors. It keeps passive organ knowledge, portable skill instructions,
and executable adapters as different inventory roles. It also preserves the
Mirror discovery asymmetry: AI-native seams are primary; human usefulness,
accessibility, craft, and product-soul review is explicit, secondary, advisory,
and unable to close native seams.

The exact public contract sources and claim ceilings are pinned in
`research/mirror-situated-knowledge-sources-2026-08-15.json`. The 115-organ
archive remains inert knowledge and no Workshop or Mirror JavaScript is
imported or executed.

### Inner asset foundry

The Wave 2.2 path adds one deliberately small creation substrate without
copying the Workshop or full Asset Factory:

```text
strict pixel recipe
  -> deterministic Go compiler
  -> normalized recipe + resolved grid + primary PNG + preview PNG + atlas
  -> candidate-only portable .axmasset
  -> full digest verification and deterministic recompilation
```

```bash
./waldo-axm-mirror forge-asset examples/axm-mirror/inner-asset-recipe.json /tmp/witness-orb.axmasset
./waldo-axm-mirror verify-asset /tmp/witness-orb.axmasset
```

Recipes contain only bounded pixel primitives, explicit palettes, frames,
semantic/presentation layers, canvas constraints, and a deterministic seed.
They contain no code, prompt, path, URL, external generator, or Asset Hand. A
READY candidate still has `visual_status: UNREVIEWED`; technical PASS is not
visual approval, usefulness, shared-vocabulary admission, promotion, or CANON.

The exact public Asset Factory and visual-contract knowledge sources are pinned
in `research/asset-foundry-knowledge-sources-2026-08-15.json`. ADR 9007 defines
the smaller WALDO-native implementation and its authority ceiling.

### Portable capability spine

The Wave 2.3 path lets the specialist describe a missing capability and prepare
an inspectable external contract without putting the Workshop inside WALDO:

```text
compiled self capability census
  + source-attributed external inventory with explicit freshness
  + exact capability/schema/data-class gap request
  -> review-only handoff plan
  -> explicit translation-loss receipt
  -> digest-bound returned-artifact receipt
```

```bash
./waldo-axm-mirror census-capabilities examples/axm-mirror/self-capability-census-request.json /tmp/waldo-self-capabilities.json
./waldo-axm-mirror intake-capabilities examples/axm-mirror/external-capability-snapshot.json /tmp/platform-capabilities.json
./waldo-axm-mirror plan-handoff /tmp/waldo-self-capabilities.json /tmp/platform-capabilities.json examples/axm-mirror/capability-gap-request.json /tmp/capability-handoff.json
```

The checked external snapshot is deliberately `SOURCE_ONLY`, so the last
command writes `HOLD_EXTERNAL_CAPABILITY_SOURCE_ONLY` and returns nonzero. That
is the honest result: the public platform contracts are known, but no live
Workshop runtime or provider was observed. Tests separately prove a synthetic
fresh LIVE unique-provider path through return verification.

The planner uses exact capability, schema, and data-class matches. It never
selects or invokes a provider, inherits a permission, generates an adapter, or
turns source declarations into readiness. Translation receipts say
`NO_DECLARED_LOSS`, not “lossless,” and returned digests are not content or
visual approval. The source contracts and ceilings are pinned in
`research/portable-capability-spine-knowledge-sources-2026-08-15.json`; ADR
9008 defines the smaller WALDO-native contract.

Receipt writes are atomic and refuse to replace an existing output path. Use a
new output name, or remove an old disposable example deliberately before
rerunning a command.

## Truth boundary

These slices do **not** train a Mirror clone yet. They complete deterministic
provenance, evaluation, and situated-evidence boundaries needed before local
model experiments can be recorded honestly.

They do not grant capture or tool access, install or restore skills, run a
model, verify artifact bytes merely by reading BOM entries, certify safety or
legal usability, convert discovery or dissent into a score, expose hidden
reasoning, invoke or select an external capability provider, infer schema
semantics or permissions, approve or install an asset, or make any AXM result
CANON.

## Next experimental rungs

1. Bind a real WALDO-produced corpus, run, and model/release chain to a behavior
   record; the checked-in chain is intentionally synthetic.
2. Add a small public-safe Mirror evaluation fixture set without private memory
   or hidden reasoning.
3. Run a local model through bounded scenarios and preserve outputs by digest.
4. Compare exact releases without flattening behavior, verification, discovery,
   and dissent into one score.
5. Add append-only dissent continuity and a Reproducibility Twin.
6. Let a bounded specialist propose pixel recipes against the inner foundry and
   evaluate the resulting candidates under explicit human visual review.
7. Add resource leases and real provider transport only as separately
   permissioned contracts; keep provider selection and execution outside the
   learned clone.
8. Add continuity/replay capsules and artifact-byte or visual verification as
   separate evidence, not upgrades to handoff metadata.
9. Only then evaluate whether a small WALDO-trained Mirror research clone is
   technically and legally appropriate for the selected corpus.

Large or upstream-facing changes stay out until this fork has something tested
and useful to show.
