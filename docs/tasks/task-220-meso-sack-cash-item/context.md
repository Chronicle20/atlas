# task-220 Context — Meso Sack Cash Item

Companion to [`plan.md`](./plan.md). Everything here was read from source in this
worktree during planning; file:line references are to the branch as of the plan
commit.

---

## 1. What is broken today

`CharacterCashItemUseHandleFunc`
(`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:36`)
has ~14 `if it == CashSlotItemType…` arms. Classification 520 resolves to type
`19` (`character_cash_item_use.go:894`) and **no arm matches it**, so the request
falls through to the trailing `l.Warnf` at line 641 and returns. The item is not
consumed, no mesos are awarded, and — because nothing announces an
enable-actions packet — the client stays behind its exclusive-request gate until
it times out.

Two things are missing beneath the handler: the payout amount is not parsed from
WZ, and `RequestChangeMeso`'s overflow guard returns `ErrMesoOverflow` with no
emission at all, which would strand the saga's award step until the 30s timeout
backstop.

---

## 2. Key files, by service

### `atlas-data`
| File | Role |
|---|---|
| `cash/reader.go:53-140` | `Read` — walks `0520.img`-style XML; the `info` scalar block at lines 76-80 is where `meso` is parsed. `GetIntegerWithDefault(name, 0)` already tolerates an absent node. |
| `cash/rest.go:38-50` | `RestModel` — the JSONB document shape (`GetName() == "cash_items"`). |
| `cash/reader_test.go` | Per-feature tests, each with its own `const test…XML` fixture + a `model.CollectToMap` assertion. `TestReaderProtectTime` (line 843) is the closest template. |

**Verified WZ ground truth (GMS 83.1 `Item.wz/Cash/0520.img.xml`):** `05200000`
→ `meso 1000000`, `05200001` → `5000000`, `05200002` → `10000000`. The node set
for these entries is only `icon`/`iconRaw`/`meso`/`cash` — no `slotMax`, no
`spec`, no `tradeBlock`.

### `atlas-channel`
| File | Role |
|---|---|
| `data/cash/rest.go` | Deliberately **partial** mirror of atlas-data's resource — four fields today. Adding one is the established pattern. |
| `data/cash/requests.go` | `DATA_SERVICE_URL` + `data/cash/items/{id}`. |
| `socket/handler/character_cash_item_use.go` | The dispatcher. Pre-branch guard at lines 52-57 resolves the CASH-slot template and asserts it equals the claimed `itemId` — this is why the amount can be derived from a server-resolved id. `CashSlotItemType` const block at line 645. `cashItemInSlotFunc` package-var seam at ~line 683. |
| `socket/handler/character_cash_item_use_point_reset.go` | **The model for the new arm**: `enableActions` closure, destroy-first saga, `saga.NewProcessor(l, ctx).Create(...)`. |
| `socket/handler/note_send.go:20` | **The model for testability**: a pure `buildNoteSendSaga(...)` extracted so the unit test asserts saga shape without touching Kafka (`note_send_test.go`). |
| `saga/model.go` | Type-alias + const re-exports of `libs/atlas-saga`. |
| `saga/processor.go:33` | `Create` emits straight to `saga.EnvCommandTopic` — no seam, hence the pure-builder test pattern. |
| `kafka/message/saga/kafka.go:15-27` | The message-layer **string copies** the consumer compares against (`SagaTypePointReset`, `ErrorCodeNotEnoughMesos`, …). Separate declarations from the `Type` constants; drift is silent. |
| `kafka/consumer/saga/consumer.go:232-360` | `handleFailedEvent` — resolves the session by `e.Body.CharacterId`, then a chain of per-saga-type arms. Point-reset arm at ~line 348 is the model. |

### `atlas-character`
| File | Role |
|---|---|
| `character/processor.go:824-859` | `RequestChangeMeso`. The `rejectEmit` closure idiom: set inside the tx, fire after rollback. `ErrNotEnoughMeso` swallows the error (`return nil`); overflow must keep returning it. |
| `character/producer.go:168-181` | `notEnoughMesoErrorStatusEventProvider` — the exact shape to mirror. |
| `character/producer.go:238` | `statChangedProvider` hard-codes `ExclRequestSent: true`. **This is why the success path needs no unlock packet.** |
| `kafka/message/character/kafka.go:244` | `StatusEventErrorTypeNotEnoughMeso`; `StatusEventMesoErrorBody` at line 326-332 (`{error, amount}`). |
| `character/meso_outbox_test.go` | Test harness: `outboxTestDb`, `createTestCharacter`, `outboxRowCount`. `TestMain` (`character/testmain_test.go`) installs `producertest.InstallNoop()`, so a test wanting to assert emissions must swap in `producertest.InstallCapturing()` and restore. |

### `atlas-saga-orchestrator`
| File | Role |
|---|---|
| `kafka/consumer/character/consumer.go:166` | `handleCharacterMesoErrorEvent` — drops `Body.Error`. Line 193 (`handleCharacterApTransferErrorEvent`) shows the `StepCompletedWithResult` threading idiom. |
| `saga/event_acceptance.go:132` | `AwardMesos: {MesoChanged, MesoError}` — **already covers the new saga's award step**. Line 327: `EventKindCharacterMesoError → OutcomeFailure`. No edit needed. |
| `saga/compensator.go:40-125` | The `Compensator` interface — every bespoke compensator and every `Dispatch…Rollbacks` is declared here. |
| `saga/compensator.go:234-330` | `CompensateFailedStep` — the per-type dispatch chain. |
| `saga/compensator.go:1485-1585` | `compensatePointReset` + `pointResetFailureFields` + `DispatchPointResetRollbacks` — the shape to mirror. |
| `saga/compensator.go:1585-1660` | `compensateNoteSend` — documents the characterId-0 trap and works around it with `EmitSagaFailedByIds`. |
| `saga/producer.go:138-190` | `ExtractCharacterCreationIds` (returns 0 without a `CreateCharacter` step) and `EmitSagaFailed` (with its `MtsOperation` arm — the precedent for a per-type id-resolution arm). |
| `saga/character_extractor.go` | `ExtractCharacterId(step)` — already handles `AwardMesosPayload` and `DestroyAssetPayload`. |
| `saga/timer.go:169-250` | `reverseWalkSagaTypes`, `noReverseWalkSagaTypes`, `allSagaTypes`, `dispatchTimeoutRollbacks`. **The design missed this file.** |
| `saga/producer_testseam.go` | `SetEmitSagaFailedForTest` — `//go:build test`. |
| `saga/point_reset_compensation_test.go` | The compensation-test template (`compmock.ProcessorMock`, `testTenantContext()`, driving `Dispatch…Rollbacks` directly to avoid Kafka). |

### `libs/atlas-saga`
| File | Role |
|---|---|
| `model.go:30-44` | The `Type` const block. |
| `payloads.go:68-77` | `AwardMesosPayload` — note `Amount int32` and `ShowEffect bool`. |

---

## 3. Decisions carried in from design.md

| Question | Resolution | Source |
|---|---|---|
| Does the wire carry a sub-body? | **No**, on all ten versions. Per-version send-fn + case-arm addresses are tabulated in design §3. `libs/atlas-packet` and all nine templates are untouched. | design §3 |
| `DestroyAsset` or `DestroyAssetFromSlot`? | **`DestroyAsset`** (template-keyed). The orchestrator's inverse for it is a plain `RequestCreateItem`; every sibling arm consumes the cash item itself by template; the slot was already proven by the pre-branch guard; a refund to the first free CASH slot matches every other refund path. | design §4.3 |
| Unlock ordering on success? | **No handler-side unlock.** The award's `STAT_CHANGED{Meso}` carries `ExclRequestSent: true`, so the packet that renders the new balance *is* the unlock. | design §5.1 |
| Meso-ceiling copy? | Server-authored — the client owns no such string (its `0x31D`-family strings are the pre-use confirmation prompt). `You cannot hold any more mesos.` for `MESO_OVERFLOW`, generic text otherwise. | design §5.2 |
| Why not version-gate the `19`? | The type is derived from the server-resolved template id and never rides the wire; no other classification maps to 19 in Atlas's table. The v48 (17) / v61 (18) client tables are irrelevant. | design §3.1(a) |

---

## 4. Traps

1. **Re-ingest is a hard prerequisite.** Cash items are JSONB documents; the new
   `meso` field is additive and **no existing document gains it on deploy**.
   Until each tenant's WZ is re-ingested, every sack use is a logged rejection —
   loud in logs, invisible to a smoke test that never uses a sack.
   `docs/tasks/task-220-meso-sack-cash-item/rollout.md` (Task 10) gates "live" on
   per-tenant field verification.
2. **`int32` truncation.** `AwardMesosPayload.Amount` is `int32`; the WZ value is
   `uint32`. Without the `math.MaxInt32` guard a large sack wraps negative and
   *deducts* mesos.
3. **characterId 0 on saga failure.** `EmitSagaFailed` resolves the id via
   `ExtractCharacterCreationIds`, which returns 0 for anything without a
   `CreateCharacter` step. The channel looks the session up by that id, so 0
   means silence *and* a still-locked client. Fixed by an arm in
   `EmitSagaFailed` (covers both the compensator and the timeout path).
4. **`saga/timer.go` is a third enumeration site.** Its doc comment records a
   prior incident where `TradeStaging` was added everywhere except there, and a
   timed-out staging saga destroyed the asset outright. `MesoSackUse` must land
   in `reverseWalkSagaTypes`, `allSagaTypes`, and the `dispatchTimeoutRollbacks`
   switch.
5. **Tag-gated tests.** `//go:build test` files (`saga/late_event_integration_test.go`,
   `producer_testseam.go`, and the new orchestrator tests) do **not** compile
   under a bare `go test ./...`. The orchestrator must be run twice:
   `go test -race ./...` **and** `go test -tags=test -race ./...`.
6. **Two `NOT_ENOUGH_MESO` spellings exist.** `atlas-character` emits
   `"NOT_ENOUGH_MESO"` (singular); the channel's storage error code is
   `"NOT_ENOUGH_MESOS"` (plural). They are unrelated strings — do not "unify"
   them. The new code uses `"MESO_OVERFLOW"` in both places, pinned by tests.
7. **v92/v95 random sacks show an amount-less prompt.** `sub_9A1AB0` on v92 is
   `get_cashslot_item_type(id) == 19 && id/1000 == 5202`; when true the client
   shows a "random amount" confirmation instead of the amount-bearing one. This
   task pays the flat `info/meso` value — an accepted, cosmetic-only divergence
   (both client branches converge on the same zero-byte send).
8. **JMS routes Maple Point sacks to type 49, GMS does not.** Atlas maps both to
   19 regardless, so on JMS a Maple Point sack enters our branch and is rejected
   by the zero-amount guard rather than by type. Same observable outcome; do not
   "correct" the table.

---

## 5. Verification commands

Changed modules: `services/atlas-data/atlas.com/data`,
`services/atlas-channel/atlas.com/channel`,
`services/atlas-character/atlas.com/character`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `libs/atlas-saga`.

Per module: `go build ./... && go vet ./... && go test -race ./...`
Orchestrator additionally: `go test -tags=test -race ./...`

From the worktree root: `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
`tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh`,
`tools/lint.sh --check` (fix with `tools/lint.sh`).

**Not required**, and confirm with `git diff --name-only main...HEAD` before
skipping: `docker buildx bake` (no `go.mod` changed), the three template guards
(no template changed), `tools/service-registration-guard.sh` (no
services.json/deploy/docker-bake/go.work change), `tools/trade-contract-mirror-guard.sh`
(no trade contract change).

Code review via `superpowers:requesting-code-review` before the PR — it dispatches
`plan-adherence-reviewer` + `backend-guidelines-reviewer` (Go only; no atlas-ui
change).
