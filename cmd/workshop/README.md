# AXM Local Workshop v0.1

A deliberately small local workspace for Waldo first and additional AI identities later.

## What exists now

- Local browser chat workspace.
- Waldo is the seeded identity; sessions are separate.
- A goal field per session.
- A real opt-in background goal runtime per session.
- Start / pause controls and selectable heartbeat interval.
- Background heartbeats write visible progress notes into the session.
- No overlapping heartbeat calls for the same session.
- Runtime state survives process restarts; active sessions resume when the workshop is running again.
- Explicit consent requests with approve/reject history.
- Three memory layers:
  - session memory;
  - AI identity memory;
  - shared vault memory.
- Creative Room image intake and per-session gallery.
- Images can be attached to chat requests for vision-capable models.
- Local JSON persistence plus local media files.
- An OpenAI-compatible local model adapter.

The workshop is intentionally not an IDE, giant agent framework, vector database, or security platform. The workspace owns state, memory, consent, media, and heartbeat scheduling; the model adapter generates replies.

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

The workshop process is the required runtime. Background heartbeats only happen while this process is running. Closing it stops all runtime activity. Session/runtime state remains on disk and is restored on the next start.

Optional environment variables:

- `AXM_AI_BASE_URL` — local OpenAI-compatible API base URL.
- `AXM_AI_MODEL` — model name exposed by that endpoint.
- `AXM_AI_KEY` — optional bearer key if the local endpoint requires one.
- `AXM_WORKSHOP_PORT` — defaults to `7788`.
- `AXM_WORKSHOP_DATA` — defaults to `~/.axm-workshop`.

## Background goal runtime

1. Enter a goal in the session.
2. Pick a heartbeat interval: 30 seconds, 1 minute, 5 minutes, 15 minutes, or 1 hour.
3. Press **Start background**.
4. The first heartbeat is scheduled almost immediately; later heartbeats use the selected interval.
5. Each successful heartbeat adds a visible `Goal heartbeat` assistant message to the session.
6. Press **Pause** to prevent future heartbeats.

The scheduler runs one heartbeat at a time per session. If a model response takes longer than the configured interval, another request is not stacked on top of it. A failed model call is recorded visibly and retried on the next normal interval while the session remains active.

## Storage

By default the workshop writes only to:

```text
~/.axm-workshop/
  state.json
  media/
```

Conversation history stays with its session. Explicit identity memory is visible to sessions for that identity. Vault memory is visible across identities. Chat content is not silently promoted into durable memory.

## Consent boundary

Consent items are records, not hidden execution permissions. Approving an item records the decision; it does not secretly run a tool, change a file, connect an app, or write memory.

The background runtime can think, continue a goal, and produce a progress note. It does not gain separate tool/app/file execution authority from being active.

## Verification performed for this change

- `go test ./...` passes locally.
- End-to-end local runtime probe passed using a fake OpenAI-compatible model endpoint:
  - goal saved;
  - runtime activated;
  - background heartbeat fired without a user chat request;
  - model response was written into the session;
  - pulse count incremented;
  - next heartbeat was scheduled.

## Deliberate v0.1 limits

- The current Waldo research CLI is not claimed to be a chat API. The workshop adapter still expects a local OpenAI-compatible endpoint; a native Waldo bridge remains the next seam if Waldo remains CLI-only.
- App connectors are not wired yet. They should enter through the existing explicit adapter/consent boundary.
- The workshop does not install itself as an operating-system service yet. For always-on use, the workshop process must be kept running by the user or later wrapped as a local service.
- No merge is performed by this branch.
