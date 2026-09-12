# OpenWALDO refresh boundary for the Mirror/Waldo experiment

This repository is an ancestor laboratory for Walmi. It is intentionally allowed to remain experimental, irregular, and historically rich. It is **not** a release branch of Walmi and it is not a place where OpenWALDO is silently rewritten into an AXM-owned fork.

The refresh boundary exists so the lab can keep learning from current `openwaldo/waldo` while preserving the Mirror/Waldo experiment beside it.

## Ownership rule

For a refresh candidate:

1. The selected OpenWALDO commit owns every path present in that upstream tree.
2. Every upstream-owned path must remain byte-for-byte and mode-for-mode identical in the candidate.
3. AXM/Mirror material may survive only on explicitly classified additive paths that upstream does not own.
4. If a path that used to be AXM/Mirror-only becomes owned by upstream, refresh stops. A human or machine steward must inspect the collision; the tool never silently chooses a winner.
5. Historical branches remain evidence. Direct experiments against old WALDO runtime/CLI paths do not need to be erased merely because the current refresh body no longer carries them.

This means the lab can be messy in history while the refreshed working body has a clean, testable boundary.

## Refresh flow

`refresh.sh` reads `upstream.lock.json`, fetches the requested OpenWALDO ref, classifies the current additive overlay, checks for new ownership collisions, materializes the exact new upstream tree, restores only the allowed overlay, updates the lock, and proves the resulting upstream paths are exact.

The GitHub workflow `.github/workflows/axm-openwaldo-sync.yml` wraps that operation. It is manual by design. A successful refresh becomes a **draft learning PR**; it never merges itself and it never promotes anything into Walmi.

The candidate gates are:

- exact OpenWALDO path/mode identity;
- fail-closed overlay classification and collision detection;
- Go formatting, vet, and complete Go tests;
- direct live Mirror package/command tests;
- the retained v0.54 service-mode sidecar test;
- the inherited `testing/all.sh` lifecycle recorded as evidence. That inherited lifecycle can expose upstream flakiness; its result is surfaced on the draft PR instead of being disguised as an AXM change.

## Walmi promotion boundary

OpenWALDO refreshes are for **learning in this ancestor lab**. Nothing crosses into Walmi automatically.

A later Walmi integration should be a separate decision based on a concrete capability or lesson, with its own evidence, interfaces, state consequences, rollback story, and root-level review. Upstream novelty by itself is not a promotion reason.

## Current lock

The lock records the exact OpenWALDO commit/tree currently used as the working-body baseline. Updating it without running the refresh proof is not considered a valid refresh.
