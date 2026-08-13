# Portal ENTER Double-Execute — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-07
Issue: [#1193](https://github.com/Chronicle20/atlas/issues/1193)
---

## 1. Overview

A single portal touch delivers the portal `ENTER` command twice on
`COMMAND_TOPIC_PORTAL_ACTIONS`, roughly 300 ms apart with identical payloads.
`atlas-portal-actions` executes the matched rule in full both times, so every
operation in that rule runs twice — including `warp`, which the player sees as a
double warp. Any portal script with a side effect is affected;
`play_portal_sound`, inventory operations, and `start_instance_transport` are all
exposed, and each touch starts its own saga.

The duplicate is not manufactured inside `atlas-portal-actions` — it arrives
already duplicated on the topic. But it is not a client defect either. The client
sends the second request **because the server told it to**:
`atlas-portal-actions` unconditionally sends `EnableActions` (a `STAT_CHANGED`
status event with `exclRequestSent=true`) at the end of every portal-enter
outcome, including outcomes whose matched rule just queued an asynchronous warp.
On the GMS v83 client that flag *is* the anti-duplicate gate. Clearing it while
the player is still standing inside the portal's collision rect — which they are,
because the warp is still in flight — makes the client legitimately re-fire.

The fix has three parts. Stop unlocking a client we are in the middle of moving.
Make warp saga steps sign off on the observed warp rather than hanging until a
30 s timeout, so that a saga `FAILED` becomes a trustworthy "the warp did not
land" signal that can safely drive the unlock. And add a short Redis dedupe gate
in front of the command handler as insurance against any future unlock-shaped
regression, since the client is *designed* to retry while a player stands in a
portal.

### 1.1 Verified root cause

Server side, on `main` as of `e0f5bd01d`:

- `services/atlas-portal-actions/atlas.com/portal/script/consumer.go:97-105` —
  both the `!result.Allow` branch and the final fallthrough call
  `character.EnableActions`. Every outcome unlocks the client.
- `services/atlas-portal-actions/atlas.com/portal/character/producer.go:20` —
  `EnableActions` emits `STAT_CHANGED` with `ExclRequestSent: true`.

Client side, GMS v83 (`MapleStory_dump.exe.i64`):

- `CUserLocal::CheckPortal_Collision` @`0x94dac6` — for a script portal
  (`PORTAL.type == 9`) the client sends the request **only if**
  `CWvsContext::CanSendExclRequest` passes, then sets `m_bExclRequestSent = 1`.
  The collision check runs every frame while the player overlaps the portal rect.
- `CWvsContext::CanSendExclRequest` @`0x485bf7` —
  `return !m_bExclRequestSent && … && get_update_time() - m_tLastExclRequest >= delay`,
  where `delay` is the portal's own re-trigger delay at `PORTAL+0x2C` (read from a
  WZ int property, default 0). A second send is impossible until the flag is
  cleared.
- `CWvsContext::OnGameStageChanged` @`0xa0400e` — called from `set_stage` on every
  `SET_FIELD`. It clears `m_bExclRequestSent` and resets `m_tLastExclRequest`.
  **A successful field change already unlocks the client on its own.**
- `CField::SendTransferFieldRequest` @`0x53035d` — the *non-scripted* portal path
  additionally refuses to send within 500 ms of the last request. The scripted
  path has no such floor, which is why this surfaced on a script portal.

The existing convention in the codebase already matches the client's contract:
`services/atlas-portals/atlas.com/portals/portal/processor.go:126,134` warp
**without** calling `EnableActions`, reserving it for the blocked / failed /
no-target paths (lines 76, 96, 122, 130). `atlas-portal-actions` is the outlier.

### 1.2 Why the saga change is a prerequisite, not a bonus

The obvious safety net for "we stopped unlocking, so unlock on failure instead"
does not work against current behaviour. `handleWarpToPortal`
(`saga/handler.go:991-1016`) fires the command and returns without calling
`StepCompleted`, and `saga/event_acceptance.go:219-220` lists both
`WarpToPortal` and `WarpToSavedLocation` as `{}` (no event advances them). The
step therefore never completes and **every** warp saga emits
`FAILED`/`SAGA_TIMEOUT` after 30 s — including warps that succeeded. That is the
`SAGA_TIMEOUT` on `warp-1` in the issue report. Wiring the unlock to `FAILED`
without fixing this would unlock (and send a failure message to) players who
arrived just fine.

The acceptance-table comment at `saga/event_acceptance.go:210-215` justifies the
`{}` by asserting that these actions "can warp WITHIN the current map … where no
map_changed fires". **That is false against current code.**
`warp.ProcessorImpl.ChangeMap`
(`services/atlas-maps/atlas.com/maps/character/warp/processor.go:74-96`) emits
`MAP_CHANGED` unconditionally — there is no `oldField == dest` guard there or in
`changeMapFromCommand` (`kafka/consumer/character/change_map.go:16-21`). A
portal-to-portal same-map warp emits `MAP_CHANGED` exactly like a cross-map one.

The correlation plumbing is already complete end to end:

| Hop | Location |
|---|---|
| Handler passes the saga transaction id | `saga/handler.go:1009` — `WarpToPortalAndEmit(s.TransactionId(), …)` |
| Command carries it | `saga-orchestrator/character/producer.go:18-32` — `ChangeMapProvider` sets `TransactionId` |
| atlas-maps forwards it | `atlas-maps/.../change_map.go:19` — passes `c.TransactionId` |
| Status event is stamped with it | `atlas-maps/.../kafka/producer/character.go:40-47` — `MapChangedStatusProvider` |
| Orchestrator consumes and completes | `saga-orchestrator/.../kafka/consumer/character/consumer.go:68-77` |

The only thing standing between today's behaviour and a real acknowledgement is
the two `{}` entries in the acceptance table.

## 2. Goals

Primary goals:

- One portal touch results in exactly one execution of the matched portal rule,
  and exactly one warp.
- A portal outcome that moves the character does not clear the client's
  exclusive-request flag; the resulting `SET_FIELD` does that.
- A portal outcome that does **not** move the character continues to unlock the
  client exactly as it does today.
- A warp that genuinely fails to land unlocks the player within a few seconds
  rather than freezing them.
- `WarpToPortal` and `WarpToSavedLocation` saga steps complete on the observed
  warp instead of hanging to a timeout, so `FAILED` means failed.
- A future portal operation that moves the character cannot silently
  reintroduce the double-execute.

Non-goals:

- Auditing or changing the other services that emit `exclRequestSent`
  (`atlas-npc-conversations`, `atlas-map-actions`, `atlas-buffs`,
  `atlas-consumables`). A cross-service convention sweep was considered and
  explicitly deferred.
- Any change to packet codecs, tenant socket-config templates, or the coverage
  matrix. No wire format is touched.
- Client-side behaviour. The client is doing what it was designed to do.
- Reworking the saga engine's event-correlation model beyond the targeted
  character-id guard in FR-1.3.
- Making `handleWarpPartyQuestMembers` itself acknowledge its N warps. It stays
  self-completing; FR-1.3 only prevents its late events from leaking into a
  later step.

## 3. User Stories

- As a **player**, I want touching a portal once to warp me once, so that I do
  not visibly bounce through two map transitions.
- As a **player**, I want a portal script's side effects (sounds, items, quest
  state, transport boarding) to happen once per touch, so that I am not charged
  or rewarded twice.
- As a **player**, when a warp genuinely fails I want control of my character
  back promptly with an explanation, rather than being frozen.
- As an **operator**, I want portal warp sagas to complete when the warp lands,
  so that `SAGA_TIMEOUT` in the logs is a real signal instead of routine noise
  on every successful portal warp.
- As a **developer**, I want adding a new character-moving portal operation to
  fail loudly at the point of definition if I have not declared it as moving,
  so I cannot reintroduce this bug by omission.

## 4. Functional Requirements

### FR-1 — Warp steps sign off on the observed warp (`atlas-saga-orchestrator`)

**FR-1.1** `acceptanceTable` in `saga/event_acceptance.go` MUST list
`EventKindCharacterMapChanged` for `sharedsaga.WarpToPortal` and
`sharedsaga.WarpToSavedLocation`, matching the existing
`sharedsaga.WarpToRandomPortal` entry.

**FR-1.2** The comment block at `saga/event_acceptance.go:202-215` MUST be
rewritten. Its current claim — that these actions can warp within the current map
where no `map_changed` fires — is false. The replacement MUST state the verified
behaviour and cite `warp.ProcessorImpl.ChangeMap` as the authority for
`MAP_CHANGED` being emitted unconditionally, so the next reader does not
re-derive the same wrong conclusion.

**FR-1.3** `AcceptEvent` MUST NOT complete a step using a `MAP_CHANGED` event
whose `CharacterId` differs from the character id of the current step's payload.
Concretely: when the accepted event kind is `EventKindCharacterMapChanged` and
`ExtractCharacterId(step)` returns a non-zero value, the event's `CharacterId`
MUST equal it, otherwise the event is skipped with an explicit skip reason and
the step is left pending.

This closes the `handleWarpPartyQuestMembers` fan-out
(`saga/handler.go:2894-2906`), which fires N warps under one transaction id and
then self-completes: without the guard, those N late `MAP_CHANGED` events could
spuriously complete whatever step became current next.

**FR-1.4** `ExtractCharacterId` (`saga/character_extractor.go`) MUST handle
`WarpToSavedLocationPayload`. It currently does not, which would leave FR-1.3
unable to guard that action.

**FR-1.5** A skip caused by FR-1.3 MUST be logged through the existing
`LogSkip` mechanism with a distinct skip reason, so a mismatch is diagnosable
rather than silent.

### FR-2 — Do not unlock a client that is being moved (`atlas-portal-actions`)

**FR-2.1** The set of operations that move a character MUST be declared as data
alongside the operation definitions rather than as a literal list inside the
Kafka consumer. The initial members are `warp`, `warp_to_saved_location`, and
`start_instance_transport`.

**FR-2.2** The declaration MUST be structured so that adding a new operation
without classifying it as moving or non-moving is a compile-time or
test-time failure, not a silent default. A new operation MUST NOT default to
"non-moving".

**FR-2.3** `handleEnterCommand`
(`services/atlas-portal-actions/atlas.com/portal/script/consumer.go`) MUST NOT
call `character.EnableActions` when the matched outcome's operations include at
least one moving operation. In that case the client is unlocked by the resulting
`SET_FIELD` (see §1.1).

**FR-2.4** All existing unlock paths MUST be preserved unchanged for outcomes
that do not move the character: `result.Error != nil`, `!result.Allow` with no
moving operation, `no_script`, `no_match`, and the allowed-with-no-operations
fallthrough.

**FR-2.5** When `handleEnterCommand` suppresses the unlock under FR-2.3, it MUST
register a `PendingAction` in the existing registry
(`portal/action/registry.go`) keyed by the created saga's transaction id, so the
existing `handleStatusEventFailed` path
(`portal/kafka/consumer/saga/consumer.go:78-115`) unlocks the player if the warp
does not land. This is sound only because FR-1 makes `FAILED` meaningful.

**FR-2.6** Sagas created by `executeWarp` and `executeWarpToSavedLocation`
(`portal/script/executor.go`) MUST set an explicit per-saga timeout of **5
seconds** via `Builder.SetTimeout` (`libs/atlas-saga/builder.go:46`) rather than
inheriting the 30 s default, so a genuinely failed warp returns control to the
player promptly.

**FR-2.7** The failure message selected by `resolveFailureMessage` MUST remain
appropriate when the failing action is a plain `warp` rather than a transport.
The current default — "Unable to board transport at this time." — is wrong for a
portal warp and MUST NOT be sent for one.

### FR-3 — Duplicate-command gate (`atlas-portal-actions`)

**FR-3.1** `handleEnterCommand` MUST drop an `ENTER` command that duplicates one
already handled within a short window. The gate MUST be evaluated before any
rule evaluation or operation execution.

**FR-3.2** The gate MUST be implemented with `atlas.Lock.AcquireWithTTL`
(`libs/atlas-redis/lock.go:60`, `SET NX` + TTL), which is atomic and correct
across replicas. It MUST NOT be an in-process map: partition reassignment would
silently drop the protection.

**FR-3.3** The gate key MUST be composed of tenant, `characterId`, `mapId`,
`instance`, and `portalId`. Tenant scoping MUST come from the standard
`libs/atlas-redis` tenant-namespacing rather than being hand-concatenated.

**FR-3.4** The TTL MUST be **2 seconds** — comfortably above the client's own
500 ms floor for non-scripted portals, and comfortably below any interval at
which a player could legitimately intend to re-enter the same portal in the same
map instance.

**FR-3.5** A dropped duplicate MUST be logged at `Debug` with the full key
components, so the gate's activity is observable without being noisy in normal
operation.

**FR-3.6** A Redis failure while acquiring the gate MUST fail **open** — the
command is processed. Losing Redis must not make every portal in the game
unusable; FR-2 remains the primary fix and stands on its own.

### FR-4 — Regression coverage

**FR-4.1** A test MUST assert that an outcome containing a moving operation does
not emit `EnableActions`, and that an outcome without one does.

**FR-4.2** A test MUST assert that a second identical `ENTER` command inside the
TTL window performs no rule evaluation and executes no operations.

**FR-4.3** A test MUST assert that `WarpToPortal` and `WarpToSavedLocation`
steps complete on a `MAP_CHANGED` event carrying the saga transaction id.

**FR-4.4** A test MUST assert the FR-1.3 guard: a `MAP_CHANGED` for character A
does not complete a current step whose payload names character B.

**FR-4.5** A test MUST assert that a same-map warp (`oldMapId == targetMapId`)
still completes the step, since that is the case the old comment wrongly
believed was unsupported.

## 5. API Surface

No REST endpoints are added, removed, or modified. No JSON:API resources change.

No Kafka topic, message type, or field is added or removed. The `MAP_CHANGED`
status event, the `CHANGE_MAP` command, the portal `ENTER` command, and the
`STAT_CHANGED` status event all keep their current shapes.

Two behavioural contract changes are worth recording even though no schema moves:

1. `WarpToPortal` and `WarpToSavedLocation` saga steps change from "never
   complete, always time out" to "complete on `MAP_CHANGED`". Any saga that
   chains a step after one of these actions will begin executing that step for
   the first time. No such saga is believed to exist today; the design phase MUST
   confirm this by enumerating saga definitions that contain these actions in a
   non-terminal position.
2. Portal warp sagas created by `atlas-portal-actions` carry an explicit 5 s
   timeout instead of the 30 s default.

## 6. Data Model

No database entities, columns, indexes, or migrations.

One new Redis key space is introduced for FR-3, allocated through the standard
`libs/atlas-redis` namespacing so it is tenant-scoped by construction. Keys are
ephemeral with a 2 s TTL and carry no payload beyond the lock token; nothing
needs to survive a restart, and no cleanup job is required.

## 7. Service Impact

**`atlas-portal-actions`** — the bulk of the change.
`portal/script/consumer.go` gains the dedupe gate and the conditional unlock.
`portal/script/executor.go` gains the moving-operation classification and the
explicit saga timeout, and registers pending actions for warp operations.
`portal/kafka/consumer/saga/consumer.go` needs its failure-message selection
widened so a plain warp failure does not report a transport error. A Redis lock
must be initialised at startup alongside the existing `action.InitRegistry`.

**`atlas-saga-orchestrator`** — small but load-bearing.
`saga/event_acceptance.go` gains two acceptance entries and a corrected comment.
`saga/processor.go` gains the character-id guard in `AcceptEvent`.
`saga/character_extractor.go` gains the `WarpToSavedLocationPayload` case.

**`atlas-maps`** — no change. It is cited as the authority for unconditional
`MAP_CHANGED` emission and must not regress; the design phase should note that
adding a same-map short-circuit there would silently break FR-1.

**`atlas-channel`, `atlas-portals`, `libs/atlas-saga`, `libs/atlas-redis`** — no
change. `SetTimeout` and `AcquireWithTTL` already exist.

## 8. Non-Functional Requirements

**Multi-tenancy.** The dedupe key MUST be tenant-scoped via the standard
`libs/atlas-redis` namespacing. Two tenants with the same character id, map, and
portal MUST NOT share a gate.

**Performance.** One additional Redis round trip per portal `ENTER` command.
Portal entry is a human-initiated action at human frequency; this is negligible.
The saga change *reduces* load — warp sagas currently linger for 30 s each and
emit a spurious failure event.

**Availability.** FR-3.6 requires the dedupe gate to fail open. The system's
correctness rests on FR-2; FR-3 is defence in depth and must never become a
single point of failure for portal traversal.

**Observability.** Dropped duplicates log at `Debug` (FR-3.5). Character-id
guard skips log through the existing `LogSkip` path with a distinct reason
(FR-1.5). After this change, a `SAGA_TIMEOUT` on a portal warp step is a genuine
incident signal rather than routine noise, which is itself an observability win.

**Backward compatibility.** No wire or schema change, so services may be
deployed in any order. Deploying `atlas-portal-actions` before
`atlas-saga-orchestrator` would leave a window in which suppressed unlocks rely
on a `FAILED` that still fires spuriously at 30 s; the design phase should call
out the preferred ordering (orchestrator first).

## 9. Open Questions

1. **Sagas chaining after a warp step.** FR-1 makes previously-stranded steps
   execute. The design phase must enumerate every saga definition containing
   `WarpToPortal` or `WarpToSavedLocation` in a non-terminal position and
   confirm the newly-reachable steps are correct. Believed empty; unverified.
2. **Residual party-quest overlap.** The FR-1.3 character-id guard fully closes
   the cross-character case. It does not close the case where a party member
   warped by `handleWarpPartyQuestMembers` is *also* the subject of a subsequent
   `WarpToPortal` step in the same saga. No such saga is known to exist. Decide
   in design whether to accept this as a documented residual risk or add a
   sequence/step-id qualifier to the correlation.
3. **`start_instance_transport` unlock ownership.** That operation already
   registers a `PendingAction` and has its own failure path. Confirm during
   design that FR-2.5 does not double-register or conflict with it.
4. **5 s timeout headroom.** 5 s is proposed against observed warp latencies of
   roughly 300 ms end to end. Confirm this leaves adequate margin under load
   before committing, and whether `start_instance_transport` — which does more
   work — needs its own longer value.

## 10. Acceptance Criteria

- [ ] One touch of a scripted warp portal produces exactly one portal `ENTER`
      command execution and exactly one `MAP_CHANGED` transition, verified in a
      live environment on GMS 83.1 with the `undodraco` portal from the issue
      report.
- [ ] `atlas-portal-actions` emits no `EnableActions` for an outcome containing a
      moving operation, and still emits it for every non-moving outcome
      (error, `!Allow` without a move, `no_script`, `no_match`, allowed-with-no-ops).
- [ ] A moving operation added to the executor without being classified fails a
      build or a test rather than defaulting to non-moving.
- [ ] A duplicate `ENTER` for the same tenant/character/map/instance/portal
      within 2 s is dropped before rule evaluation and logged at `Debug`.
- [ ] A Redis outage does not prevent portal traversal.
- [ ] `WarpToPortal` and `WarpToSavedLocation` steps complete on `MAP_CHANGED`,
      including the same-map case; portal warp sagas no longer emit
      `SAGA_TIMEOUT` on success.
- [ ] A `MAP_CHANGED` for a different character does not complete a pending warp
      step, and the skip is logged.
- [ ] A warp that does not land unlocks the player within ~5 s with a message
      appropriate to a portal warp, not a transport boarding failure.
- [ ] `go test -race ./...` and `go vet ./...` clean in `atlas-portal-actions`
      and `atlas-saga-orchestrator`.
- [ ] `tools/lint.sh --check`, `tools/redis-key-guard.sh`, and
      `tools/goroutine-guard.sh` clean from the repo root.
- [ ] `docker buildx bake atlas-portal-actions` and
      `docker buildx bake atlas-saga-orchestrator` succeed if either `go.mod`
      changed.
- [ ] Code review completed via `superpowers:requesting-code-review` before the
      PR is opened.

## Appendix A — Noted non-issue

The issue report observes that the `undodraco` portal fires with
`mapId: 200090510` while its seed declares `"mapId": 200090500`. Portal script
matching is by portal **name** (`ByPortalIdProvider(portalName)` in
`portal/script/processor.go:125`), so the seed's `mapId` field is not
load-bearing for dispatch. It is misleading as written but causes no incorrect
behaviour, and is out of scope here. Recorded so the next investigator does not
re-open it as a lead.
