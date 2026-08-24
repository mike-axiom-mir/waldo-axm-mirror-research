# AXM Local Workshop v0.1

A deliberately small local workspace for Waldo first and additional AI identities later.

## What exists now

- Local browser chat workspace.
- Waldo is the seeded identity; sessions are separate.
- A goal field per session.
- A visible heartbeat/pulse checkpoint per session.
- Explicit consent requests with approve/reject history.
- Three memory layers:
  - session memory;
  - AI identity memory;
  - shared vault memory.
- Creative Room image intake and per-session gallery.
- Images can be attached to chat requests for vision-capable models.
- Local JSON persistence plus local media files.
- An OpenAI-compatible local model adapter.

The workshop is intentionally not an IDE, agent framework, vector database, or security platform. v0.1 keeps the ownership boundary simple: the workspace owns state, memory, consent, and media; the model adapter generates replies.

## Run

From the repository root:

```bash
AXM_AI_BASE_URL=http://127.0.0.1:11434/v1 \
AXM_AI_MODEL=waldo \
go run ./cmd/workshop
```

Then open:

```text
http://127.0.0.1:7788
```

Optional environment variables:

- `AXM_AI_BASE_URL` — local OpenAI-compatible API base URL.
- `AXM_AI_MODEL` — model name exposed by that endpoint.
- `AXM_AI_KEY` — optional bearer key if the local endpoint requires one.
- `AXM_WORKSHOP_PORT` — defaults to `7788`.
- `AXM_WORKSHOP_DATA` — defaults to `~/.axm-workshop`.

## Storage

By default the workshop writes only to:

```text
~/.axm-workshop/
  state.json
  media/
```

Conversation history stays with its session. Explicit identity memory is visible to sessions for that identity. Vault memory is visible across identities. v0.1 does not silently promote chat content into durable memory.

## Consent boundary

Consent items are records, not hidden execution permissions. Approving an item records the decision; it does not secretly run a tool, change a file, connect an app, or write memory.

## Deliberate v0.1 limits

- The current Waldo research CLI is not claimed to be a chat API. The workshop adapter currently expects a local OpenAI-compatible endpoint; a native Waldo bridge is the next seam if Waldo remains CLI-only.
- Heartbeat is a visible/manual pulse, not a background autonomous loop yet.
- App connectors are not wired yet. They should enter through a small explicit adapter/consent boundary rather than turning the workshop into another large platform.
- No merge is performed by this branch.
