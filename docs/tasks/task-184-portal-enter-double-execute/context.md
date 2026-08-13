# Portal ENTER Double-Execute — Implementation Context

Companion to [plan.md](plan.md). Read this first if you are picking the task up cold.

Sources: [prd.md](prd.md) · [design.md](design.md) · issue [#1193](https://github.com/Chronicle20/atlas/issues/1193)

---

## 1. The bug in one paragraph

A single portal touch delivers the portal `ENTER` command twice on
`COMMAND_TOPIC_PORTAL_ACTIONS`, ~300 ms apart, identical payloads.
`atlas-portal-actions` executes the matched rule in full both times, so every
operation runs twice — the player sees a double warp. The duplicate is not
manufactured server-side and it is not a client defect: the client re-fires
**because the server told it to**. `atlas-portal-actions` unconditionally sends
`EnableActions` (a `STAT_CHANGED` with `exclRequestSent=true`) at the end of
*every* outcome, including one that just queued an asynchronous warp. On the GMS
v83 client that flag **is** the anti-duplicate gate. Clearing it while the player
is still standing inside the portal's collision rect — which they are, because
the warp is still in flight — makes the client legitimately re-fire.

Client-side evidence (GMS v83, `MapleStory_dump.exe.i64`), from PRD §1.1:

| Symbol | Address | What it establishes |
|---|---|---|
| `CUserLocal::CheckPortal_Collision` | `0x94dac6` | For a script portal (`PORTAL.type == 9`) the request is sent only if `CanSendExclRequest` passes, then `m_bExclRequestSent = 1`. The check runs **every frame** the player overlaps the rect. |
| `CWvsContext::CanSendExclRequest` | `0x485bf7` | `!m_bExclRequestSent && … && get_update_time() - m_tLastExclRequest >= delay`. A second send is impossible until the flag clears. |
| `CWvsContext::OnGameStageChanged` | `0xa0400e` | Called from `set_stage` on every `SET_FIELD`. Clears the flag. **A successful field change already unlocks the client on its own.** |
| `CField::SendTransferFieldRequest` | `0x53035d` | The *non-scripted* path additionally refuses to re-send within 500 ms. The scripted path has no such floor — which is why this surfaced on a script portal. |

`atlas-portals` already follows the correct convention:
`services/atlas-portals/atlas.com/portals/portal/processor.go:126,134` warp
*without* `EnableActions`, reserving it for the blocked / failed / no-target
paths. `atlas-portal-actions` is the outlier.

---

## 2. Why the saga layer must change first

The obvious safety net for "we stopped unlocking, so unlock on failure instead"
does not work against current behaviour:

- `saga/handler.go:991-1016` — `handleWarpToPortal` fires the command and returns
  **without** `StepCompleted`.
- `saga/event_acceptance.go:219-220` — `WarpToPortal` and `WarpToSavedLocation`
  are both `{}` (no event advances them).

So the step never completes and **every** warp saga emits `FAILED`/`SAGA_TIMEOUT`
after 30 s — including warps that succeeded. Wiring the unlock to `FAILED`
without fixing this would unlock (and send a failure message to) players who
arrived just fine.

The comment justifying the `{}` claims these actions can warp within the current
map "where no `map_changed` fires". **That claim is false.**
`warp.ProcessorImpl.ChangeMap`
(`services/atlas-maps/atlas.com/maps/character/warp/processor.go:74-96`) emits
`MAP_CHANGED` unconditionally — no `oldField == dest` guard there or in
`changeMapFromCommand` (`kafka/consumer/character/change_map.go:16-21`).

The correlation plumbing is already complete end to end; only the two `{}`
entries stand between today's behaviour and a real acknowledgement:

| Hop | Location |
|---|---|
| Handler passes the saga tx id | `saga/handler.go:1009` — `WarpToPortalAndEmit(s.TransactionId(), …)` |
| Command carries it | `saga-orchestrator/character/producer.go:18-32` |
| atlas-maps forwards it | `atlas-maps/.../change_map.go:19` |
| Status event stamped with it | `atlas-maps/.../kafka/producer/character.go:40-47` |
| Orchestrator consumes and completes | `saga-orchestrator/.../kafka/consumer/character/consumer.go:68-77` |

---

## 3. Layer map

| Layer | Service | Plan tasks | Standalone failure mode |
|---|---|---|---|
| A — warp acknowledgement | `atlas-saga-orchestrator` | 1–3 | None. Strictly removes spurious `SAGA_TIMEOUT`. |
| B — conditional unlock | `atlas-portal-actions` | 4–7 | Without A, a suppressed unlock leans on a `FAILED` that still fires spuriously at 30 s. |
| C — dedupe gate | `atlas-portal-actions` | 8–9 | None. Pure defence in depth. |

**B is the actual fix. A is its prerequisite. C exists because the client is
*designed* to re-fire while the player stands in a portal rect**, so any future
unlock-shaped regression anywhere in the outcome path re-opens the same hole. A
non-zero C drop rate after B ships is itself the signal that some outcome path is
still unlocking a moving character.

---

## 4. Key files, verified against the worktree

### `atlas-saga-orchestrator` (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`)

| File:line | What matters |
|---|---|
| `saga/event_acceptance.go:202-215` | The false comment to rewrite (FR-1.2) |
| `saga/event_acceptance.go:219-220` | The two `{}` entries to move (FR-1.1) |
| `saga/event_acceptance.go:386-393` | `SkipReason*` const block — where `SkipReasonCharacterIdMismatch` goes |
| `saga/event_acceptance.go:~398` | `LogSkip(l, fields, reason)` |
| `saga/processor.go:63` | `AcceptEvent` interface declaration |
| `saga/processor.go:397-441` | `AcceptEvent` impl; the guard goes last, after the `StepAcceptsEvent` block |
| `saga/processor.go:~448` | `maybeWarnUnmatchedEvent` — deliberately NOT called on the new guard path |
| `saga/character_extractor.go:43-44` | Has `WarpToPortalPayload`, lacks `WarpToSavedLocationPayload` |
| `saga/character_extractor.go:~79` | `default: return 0` — the "unconstrained" signal |
| `saga/mock/processor.go:42,206` | Mock must track the new variadic signature |
| `saga/handler.go:2894-2906` | `handleWarpPartyQuestMembers` — N warps under one tx id, then self-completes. **The reason the guard is required, not speculative.** |
| `kafka/consumer/character/consumer.go:68-77` | `handleCharacterMapChangedEvent` — the only `ForCharacter` caller |
| `kafka/message/character/kafka.go:244-250` | `StatusEvent[E]` envelope carries **both** `TransactionId` and `CharacterId` |
| `saga/accept_event_test.go:25-36` | `newAcceptEventTestProcessor` / `putAcceptEventSaga` helpers to reuse |
| `saga/late_event_integration_test.go:98` | The `logtest.Hook` + `e.Data["reason"]` assertion idiom |

`AcceptEvent` has ~60 call sites across the service. The variadic
`opts ...AcceptOption` means none of them change.

### `atlas-portal-actions` (`services/atlas-portal-actions/atlas.com/portal/`)

| File:line | What matters |
|---|---|
| `script/consumer.go:86-105` | Three unlock sites: error branch (`:90`), `!Allow` (`:99`), unconditional fallthrough (`:105`) |
| `script/executor.go:39-86` | The 45-line dispatch `switch` replaced by `opTable` |
| `script/executor.go:88-96` | `ExecuteOperations` — gains the `movedCharacter` return |
| `script/executor.go:121-165` | `executeWarp` — mints no tx id today |
| `script/executor.go:433-475` | `executeStartInstanceTransport` — **already** mints `uuid.New()` and registers a `PendingAction`; the model for the two warp methods |
| `script/executor.go:562-588` | `executeWarpToSavedLocation` |
| `script/model.go:75-81` | `ProcessResult` — gains `CharacterMoved` |
| `script/processor.go:160-178` | Both `Process` returns that populate it |
| `script/processor.go:50-63` | `OperationExecutor` is constructed **per request** — nothing to race |
| `action/registry.go:16-21` | `PendingAction` — gains `Kind` |
| `action/registry.go:43-46` | `Add` uses `Put` with **no expiry** — a dropped `COMPLETED` leaks the key forever |
| `kafka/consumer/saga/consumer.go:78-115` | `handleStatusEventFailed` — the path that unlocks on a real failure |
| `kafka/consumer/saga/consumer.go:118-137` | `resolveFailureMessage` — only the `default:` arm changes |
| `kafka/consumer/saga/consumer.go:51,69,97` | "transport saga" wording that now also covers warps |
| `main.go:49-51` | `atlas.Connect(l)` → `action.InitRegistry(rc)`; `dedupe.InitGate(rc)` goes here |
| `action/registry_test.go:15-30` | `miniredis` + tenant-context test setup to reuse in `dedupe` |

### Shared libraries — **no change required in any of them**

| File:line | Contract relied on |
|---|---|
| `libs/atlas-redis/lock.go:60-67` | `AcquireWithTTL` is `SET NX` + TTL — atomic, correct across replicas |
| `libs/atlas-redis/lock.go:45` | `lockKey = namespacedKey(namespace, "_lock", key)` — **no tenant segment** |
| `libs/atlas-redis/keys.go:33,55` | `TenantKey(t)` and `CompositeKey(parts...)`, exported for exactly this |
| `libs/atlas-redis/tenant_registry.go:117` | `PutWithTTL` |
| `libs/atlas-saga/builder.go:44-48` | `SetTimeout`; zero/negative means "orchestrator default" |
| `libs/atlas-saga/payloads.go:38-46,829-` | `WarpToPortalPayload`, `WarpToSavedLocationPayload` — both carry `CharacterId` |
| `libs/atlas-script-core/operation/` | `Model.Type()`, `Model.Params()`, `NewBuilder().SetType().SetParams().Build()` |

`saga-orchestrator/saga/model.go:316-341` defines `DefaultSagaTimeout = 30s`,
scheduled at `saga/processor.go:302`.

---

## 5. Decisions already made — do not re-litigate

| Decision | Why | Rejected alternative |
|---|---|---|
| `opTable` with an invalid zero `opClass`, validated in `init()` | FR-2.2 requires that omitting a classification *fail*. Go offers no exhaustiveness check over a `switch`, and no reflection over its arms — a parallel `movingOps` list plus a test would need a third hardcoded list, giving three things to drift instead of two. | Keep the switch, add a `movingOps` set + a test (design §6.3) |
| `init()` panic **and** a unit test | The panic fires in every test binary, in tooling that loads the package, and at startup. Statically determined by a composite literal — unreachable from any runtime input. The test makes the intent visible in test output, not only in a panic string. | Test only |
| Condition is "successfully **dispatched**", not "declared" | On the error path the operations list is only a *prefix* of what ran. A `warp` that failed before creating its saga would otherwise suppress the unlock and freeze the player with no saga to fail and release them. | Classify from `ProcessResult.Operations` in the consumer (design §4.2) |
| Variadic `AcceptOption` on `AcceptEvent` | The doc comment declares `AcceptEvent` "the single gate at which a saga-tagged Kafka event is matched against the saga's pending step", and the `LogSkip`/`SkipReason*` vocabulary lives with it. Splitting the gate leaves two places to check. Variadic keeps all ~60 other call sites untouched. | Compare character ids in `handleCharacterMapChangedEvent` after `AcceptEvent` returns (design §6.4) |
| `ForCharacter` passed **only** at the `map_changed` consumer | The mechanism is generic; scoping the *use* means no other event family's behaviour shifts under this task. | Apply it everywhere a payload has a character id |
| `TenantKey` + `CompositeKey` composed into the lock key | `Lock` is not tenant-aware. A `TenantLock` type in `libs/atlas-redis` is the tidier long-term shape but expands the blast radius from two services to a shared library consumed by fourteen, for zero behavioural difference. Follow-up candidate if a second tenant-scoped lock site appears. | Add `TenantLock` to `libs/atlas-redis` (design §6.2) |
| The dedupe lock is **never released** — TTL expiry is the release | A successful portal entry should keep the gate closed for the full 2 s window. There is no `Release` call anywhere on this path. | Release on completion |
| `warpSagaTimeout = 5s` on the two warp sagas only | ~16× headroom over the ~300 ms observed end-to-end warp; absorbs a Kafka rebalance or a cold consumer. `start_instance_transport` keeps the 30 s default: it does strictly more work (route lookup, capacity check, instance provisioning) and has its own failure path. No evidence justifies shortening it. | Shorten transport too (PRD open question 4) |
| `pendingActionTTL = 60s` | Must exceed the 5 s saga timeout by a wide margin so `handleStatusEventFailed` can still find the entry. | Match the saga timeout |
| Empty `Kind` defaults to the **transport** message | Registry entries written by a pre-deploy replica must keep resolving to today's text. | Default to warp |
| `handleWarpPartyQuestMembers` stays self-completing | PRD §2 non-goal. FR-1.3 only prevents its late events from leaking into a later step. | Make it acknowledge its N warps |
| `atlas-maps` gets **no** code change | It is the authority for unconditional `MAP_CHANGED` emission. The rewritten comment in `event_acceptance.go` is the only coupling record. | Add a same-map short-circuit / an explicit guard |

---

## 6. Resolved PRD open questions

**1 — Sagas chaining after a warp step (design §3.4).** Every in-repo construction
of a step with these two actions was enumerated:

| Producer | Position | Effect |
|---|---|---|
| `atlas-channel/.../respawn/processor.go:255-278` | `warp_to_spawn` appended last | Terminal. Saga now completes instead of timing out. |
| `atlas-portal-actions/.../script/executor.go` `executeWarp`, `executeWarpToSavedLocation` | Single-step sagas | Terminal. |
| `atlas-npc-conversations/.../conversation/operation_executor.go:1362-1371, 2481-2488` | Via `createSagaForOperations` (multi-step batch, `:1013`) | **Only non-terminal case.** |

The batch path is driven by conversation scripts that live in the **database**, so
a static in-repo enumeration cannot fully close this. Grep of all committed JSON
found no `warp_to_portal` / `warp_to_saved_location` outside
`atlas-portal-actions/docs/portal_script_schema.json`; the only occurrences in
`atlas-npc-conversations` are `.bruno` request fixtures.

**Resolution:** where such a script exists, the operations after the warp step are
*already dead* — the batch stalls at the warp step until the 30 s timeout fails
the saga. Layer A makes them run, which is what the author wrote them to do. Same
fix task-124 applied to `WarpToRandomPortal` for the teleport-rock `consume_rock`
chain. Residual risk (a script authored around the broken behaviour) is assessed
as negligible — the broken behaviour also produced a `SAGA_TIMEOUT` and a `FAILED`
on every invocation, which is not a stable base to build on. **Call this out in
the PR body so it is reviewable against live tenant data.**

**2 — Residual party-quest overlap (design §3.5). Accepted as a documented
residual.** The uncovered case: one saga in which `WarpPartyQuestMembersToMap`
warps character X and a *later* step is a `WarpToPortal` also naming X. Those two
events agree on both correlation keys. Closing it would require a `stepId` on
`CHANGE_MAP` echoed on `MAP_CHANGED` — PRD §5 declares no Kafka field is added
and §2 lists reworking the correlation model as a non-goal. No such saga exists.
The mitigation, should it ever occur, is cheap and already available: give the
follow-on warp its own saga. Recorded in the rewritten comment block.

**3 — `start_instance_transport` double registration. There is none.** Each
`execute*` method mints its own `uuid.New()` and registers under that id, so a
rule containing both `warp` and `start_instance_transport` produces two sagas with
two distinct tx ids and two independent registry entries. If both fail,
`EnableActions` is sent twice — harmless; it is an idempotent unlock and the
second arrives with the flag already clear.

**4 — 5 s timeout headroom.** Resolved: 5 s for warps, 30 s default retained for
transport. See §5.

---

## 7. Deviations from the PRD (design §10, plus one from planning)

Recorded so plan-adherence review reads them as intentional, not omissions.

| PRD says | Implementation does | Why |
|---|---|---|
| FR-2.3/2.4 — condition is "the matched outcome's operations include a moving operation"; error branch preserved verbatim | Condition is "a moving operation was **successfully dispatched**", applied to the error branch too | Safer in both directions: a warp that failed to dispatch still unlocks; a dispatched warp followed by an unrelated operation error does not double-unlock. Plan Task 7. |
| FR-2.5 — `handleEnterCommand` registers the `PendingAction` | `executeWarp` / `executeWarpToSavedLocation` register it | The saga tx id is minted there and the consumer never sees it. Mirrors `executeStartInstanceTransport`, which already does this. Plan Task 5. |
| FR-3.3 — tenant scoping comes from `libs/atlas-redis` namespacing | Composed from `redis.TenantKey` + `redis.CompositeKey` into the lock key | `Lock` is not tenant-aware (`lock.go:45`). A `TenantLock` type was rejected as out-of-scope shared-lib churn. Plan Task 8. |
| Design §5.1 sketches `Gate.Allow(ctx, k) bool` | `Gate.Allow(l logrus.FieldLogger, ctx context.Context, k Key) bool` | FR-3.5 requires the drop to be logged at `Debug` with the key components, and the gate is a process singleton with no logger of its own. Passing the request logger keeps the log correlated with the rest of the command's trace. Plan Tasks 8–9. |

---

## 8. Rollout, compatibility, observability

**Order: `atlas-saga-orchestrator`, then `atlas-portal-actions`.** No wire or
schema change in either direction, so any order *works*; this order avoids a
window in which a suppressed unlock is backed by a `FAILED` that still fires
spuriously at 30 s. In that window a player whose warp succeeded would get a
failure message and be re-unlocked 30 s later — annoying, not stuck.

**Redis compatibility.** `PendingAction` gains `Kind`; entries written by an old
replica decode with `Kind == ""` and resolve to the current transport message.
The `portal-enter` lock namespace is disjoint from anything existing.

**After the change:**

- `SAGA_TIMEOUT` on a portal warp step becomes a genuine incident signal. It is
  currently emitted on *every* portal warp, success included.
- `character_id_mismatch` skips at `Debug` quantify the party-quest fan-out
  overlap.
- `portal-enter` gate drops at `Debug` should trend to zero once layer B ships.

---

## 9. Verification

Full checklist is plan.md Task 10. Summary:

1. `go test -race ./...` + `go vet ./...` in `services/atlas-portal-actions/atlas.com/portal` and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`
2. `go build ./...` in both
3. `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` from the repo root
4. `docker buildx bake atlas-portal-actions` / `atlas-saga-orchestrator` **only if** a `go.mod` changed — none is expected; verify with `git diff --stat main...HEAD -- '*/go.mod' '*/go.sum'`
5. `superpowers:requesting-code-review` before opening the PR
6. Live check on GMS 83.1 with the `undodraco` portal from #1193: one touch → one `ENTER` executed, one `MAP_CHANGED`, no `SAGA_TIMEOUT`

**Not in play:** no packet, tenant socket-config template, or coverage-matrix
work — no wire format is touched. `tools/template-*-guard.sh`,
`tools/service-registration-guard.sh`, and the packet matrix are not required.
Confirm with `git diff --stat main...HEAD` before skipping them.

---

## 10. Follow-up candidates (out of scope, do not do them here)

- `executeStartInstanceTransport` keeps `Registry.Add` (no expiry). A dropped
  `COMPLETED` leaks its key forever. Migrating it to `AddWithTTL` is a behaviour
  change to transports with no evidence behind it in this task.
- `TenantLock` in `libs/atlas-redis`, mirroring `TenantRegistry`, if a second
  tenant-scoped lock site ever appears.
- The cross-service `exclRequestSent` convention sweep — `atlas-npc-conversations`,
  `atlas-map-actions`, `atlas-buffs`, `atlas-consumables` all emit it. Explicitly
  deferred by PRD §2.
- PRD Appendix A: the `undodraco` seed declares `"mapId": 200090500` while the
  portal fires with `mapId: 200090510`. Script matching is by portal **name**
  (`ByPortalIdProvider(portalName)`, `portal/script/processor.go:125`), so the
  seed's `mapId` is not load-bearing for dispatch. Misleading as written, causes
  no incorrect behaviour. Recorded so the next investigator does not re-open it.
