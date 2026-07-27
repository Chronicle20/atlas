# task-128 — Context for Execution

Companion to `plan.md`. Read this before executing any task.

## What this task builds

Three cash-item behaviors (Item Tag 5060000, Sealing Lock 5060001/5061000–3, Incubator 5060002) carved out of `CharacterCashItemUseHandleFunc`'s fall-through, plus a new asset `owner` string end-to-end and the `INCUBATOR_RESULT` clientbound packet for all supported versions.

## Branch / worktree state

- Worktree: `.worktrees/task-128-item-tag-seal-incubator`, branch `task-128-item-tag-seal-incubator`, branched from main at `38d4d0ba2` (behind origin/main only by dependency bumps as of 2026-07-02).
- **Sibling tasks 123–127 (megaphones, teleport rocks, mastery books, AP/SP resets, owl search) are NOT merged.** This worktree's `character_cash_item_use.go` has only three implemented arms (PetConsumable 30, Chalkboard 32, FieldEffect 16). Expect rebase conflicts in `character_cash_item_use.go`, `libs/atlas-saga` (action consts/payloads/unmarshal), the orchestrator handler/acceptance tables, and the seed templates when siblings land. Resolve at PR-time rebase; do not fork a new worktree.

## Corrections to design.md discovered during planning (already baked into plan.md)

1. **`DestroyAssetFromSlotPayload` exists but has NO `TemplateId`** (`libs/atlas-saga/payloads.go:101-108`). Design §7.1 assumed it carried one. Plan Task 5 adds the field; the compensator (Task 10) needs it to re-create the incubator sacrifice.
2. **No base+perStep saga timeout machinery exists anywhere** (design §12 cited task-086/126 scaling — absent in this worktree). All three sagas have fixed 2–4 steps, so plan uses `Timeout: 0` → orchestrator `DefaultSagaTimeout` (30 s, orchestrator `saga/model.go:263`). The flat-timeout bug class only applies to data-driven step counts.
3. **No Dockerfile COPY line is needed** for the tenants seed dir (design §8/§15 said one was). The root Dockerfile copies the whole `configurations/` dir in a loop (Dockerfile:135-142) + one `COPY --from=build-env /app/configurations /configurations` (:161).
4. **The orchestrator has a second, independent `UnmarshalJSON` switch** (`saga-orchestrator/saga/model.go:862-940`) — every new Action must be added in BOTH `libs/atlas-saga/unmarshal.go` and there; `unmarshal_completeness_test.go` enforces it (so Tasks 5 and 9 must land together, never Task 5 alone).
5. **No `EventKindAssetUpdated` exists** in the orchestrator acceptance table — the design's "complete on asset UPDATED" requires adding the EventKind, the acceptance entries, and a new `handleAssetUpdatedEvent` in the orchestrator's asset consumer (Task 9). Character-side precedent: `EventKindCharacterStatChanged` (event_acceptance.go:110-119).
6. **`INCUBATOR_RESULT` version delta is a tenant switch, not a constructor flag** (design §7.2 suggested `NewIncubatorResult(extended bool, …)`). The codebase idiom for tenant-version-dependent bodies is `tenant.MustFromContext(ctx)` inside `Encode` (`model/asset.go`), so Task 4 uses that; fixtures cover both shapes.
7. **Serverbound sub-bodies get round-trip + golden-byte tests, no per-version markers** — matching the chalkboard/field-effect convention (`item_use_chalkboard_test.go` has no packet-audit markers; sub-bodies have no matrix cells). The clientbound `INCUBATOR_RESULT` gets the full marker/evidence/matrix treatment (Task 19).
8. **atlas-storage and atlas-merchant mirror `AssetData` field-by-field** (storage persists per-field: `storage/asset/entity.go:23-24`). Without Task 14, a tagged equip loses its owner on a storage round-trip. All five mirrors: inventory (source), channel, storage, atlas-merchant, channel-merchant display, plus the orchestrator message mirror.
9. **gms_92 gets NO template rows** (design §10 said "include for completeness"). There is no verifiable v92 opcode for either the writer (STATUS.md has no v92 column) or the handler (no v92 IDB). An absent row is a safe no-op; a guessed opcode can crash the client. Unblocks when a v92 IDB exists.

## IDA-verified facts (from design.md §1 — do not re-derive)

- `USE_CASH_ITEM` serverbound: v83/v84 `0x4F` (already in templates), v87 `0x52`, v95 `0x55`, jms `0x47` (confirmed against `docs/packets/registry/*.yaml` during planning).
- v83 sub-body read orders: tag = `short slot` (+trailing `int updateTime`); seal/incubator = `int inventoryType`, `int slot` (+trailing updateTime). v87/v95/jms re-checked from IDA exports in Task 19 Step 1.
- `INCUBATOR_RESULT` (`CWvsContext::OnIncubatorResult`): v83 @0xa28298 = `int itemId`, `short count`; v95 @0xa00380 adds `int gachaponItemId`, `int bonusItemId`, `int bonusCount`. `itemId <= 0` → client "inventory full" dialog (used as the generic failure/no-op signal). Writer opcodes per STATUS.md row 89: 0x045/0x047/0x047/0x048/0x03F.
- WZ (`Item.wz/Cash/0506.img.xml`, v83): exactly 7 items; `protectTime` (days, `info` block) = 7/30/90/365 on 5061000–3; no 5062xxx in v83; 5062xxx in later dumps is the Miracle Cube family (unrelated → documented dead routing on type 74).
- Loaded IDA instances rotate — as of 2026-06-30: v83-dump port 13342, v95 port 13341, NO v84/v87/jms live instances. Use checked-in `docs/packets/ida-exports/` for those; unresolvable fname = stop-and-ask.

## Key architecture decisions (from design.md, confirmed against code)

- **Channel = composition root**: decode → validate (via inventory REST reads) → build saga. All mutation in atlas-inventory. Template arm: FieldEffect (`character_cash_item_use.go:60-106`).
- **Destroy-first step ordering** for double-use safety; the incubator adds a channel-side capacity pre-check so the common inventory-full case consumes nothing; compensation (reverse-walk re-create, PetEvolution pattern at `compensator.go:1053-1140`) is the race safety net.
- **Dedicated `SET_OWNER`/`APPLY_LOCK` commands**, not `MODIFY_EQUIPMENT` (which replaces the full stat block).
- **Lock rides the asset `expiration` field**; the expire flow branches inventory-side in `compartment/processor.go ExpireAsset` (:920) on `a.Locked()` — unlock+persist instead of destroy. atlas-asset-expiration is untouched (verified: it only compares expiration vs now).
- **Owner is a name snapshot** captured at saga-build time (renames don't retag). It is a new string; the existing numeric `ownerId` is the pickup/trade owner — a different concept.
- **`INCUBATOR_RESULT` via dedicated fire-and-forget event** (`EVENT_TOPIC_INCUBATOR_RESULT`), Precedent A / `handleEmitGachaponWin`-style (`handler.go:2475`), because the generic saga completed-event body lacks the rolled item. Failure paths: inline announce in the handler (validated no-ops) + `handleFailedEvent` saga-type branch (saga failures).
- **Seal guard**: unlocked asset with non-zero expiration is rejected (launder prevention); timed lock on an already-locked asset extends from the current expiration.
- **Weighted roll in the channel at saga-build time** — concrete itemId/quantity baked into both AwardAsset and IncubatorResult steps.

## Load-bearing code references

| What | Where |
|---|---|
| Handler + arms template | `services/atlas-channel/.../socket/handler/character_cash_item_use.go` (FieldEffect arm :60-106; imprint routing :149-175; TODO to remove :108) |
| Sub-body codec template | `libs/atlas-packet/cash/serverbound/item_use_chalkboard.go` |
| Writer + marker template | `libs/atlas-packet/door/clientbound/remove.go` + `remove_test.go:29-52` |
| Owner encode sites | `libs/atlas-packet/model/asset.go:209,261,287,332` |
| Inventory mutation persister style | `asset/administrator.go` (`updateSlot` :54, `updateEquipmentStats` :66) |
| UPDATED event emitter | `asset/producer.go:116 UpdatedEventStatusProvider` |
| Expire flow | `compartment/processor.go:920 ExpireAsset` → `asset/processor.go:159 Expire` |
| Inventory processor test setup (sqlite) | `compartment/processor_test.go:47` |
| Orchestrator command producer template | `saga-orchestrator/compartment/producer.go:35 RequestDestroyAssetCommandProvider` |
| Acceptance gate | `saga/event_acceptance.go` (`acceptanceTable` :96, self-completing block :171-200) + `saga/processor.go:362 AcceptEvent` |
| Fire-and-forget handler template | `saga/handler.go:1651 handleSendMessage`, `:2475 handleEmitGachaponWin` |
| Compensation template | `saga/compensator.go:1053 compensatePetEvolution` / `:1105 DispatchPetEvolutionRollbacks` |
| Tenants resource template (vessels) | `tenants/configuration/{rest,provider,processor,resource,kafka,seed}.go` + `mock/processor.go` + `configurations/vessels/*.json` |
| Channel config REST client template | `atlas-transports/.../transport/config/{requests,rest,processor}.go` |
| Channel event consumer template | `atlas-channel/kafka/consumer/gachapon/consumer.go` (registration: `main.go:215`, `:545`) |
| Channel asset projection | `atlas-channel/asset/{model,builder,rest}.go`, `kafka/consumer/asset/consumer.go` (`buildAssetFrom*Body` :119/:157/:195), `socket/model/asset.go:13 NewAsset` |
| protectTime read site | `atlas-data/.../cash/reader.go:75` (info block), test fixtures `reader_test.go:413-487` |
| Writer registration | `atlas-channel/main.go:608 produceWriters` → opcode resolution `libs/atlas-opcodes/producer.go:14-35` |
| Verification playbook | `docs/packets/audits/VERIFYING_A_PACKET.md`; matrix row `docs/packets/audits/STATUS.md:89` |

## Task dependency graph

```
T1 constants ─┐
T2 pkt owner ─┼─► T13 channel projection ─┐
T3 sub-bodies ┤                            │
T4 writer ────┤                            ├─► T16 handler arms ─► T17 consumer/fail ─► T18 templates ─► T19 verify ─► T20 suite
T5 saga lib ──┼─► T9 orch actions ─► T10 orch incubator/compensation ─┘
              ├─► T6 inv owner ─► T7 commands ─► T8 expire
              │                 └─► T14 mirrors
T11 tenants ──┴─► T15 channel client ──────► T16
T12 protectTime ───────────────────────────► T16
```

T5 and T9 MUST land together (completeness test). T1–T5, T11, T12 are parallelizable; everything channel-side funnels through T16.

## Open items / risks

- **gms_92**: writer + handler rows deliberately omitted (no verifiable opcode; no IDB). Documented in the deploy runbook. Unblocks when a v92 IDB exists.
- **JMS `INCUBATOR_RESULT` body**: no local JMS IDB instance; the checked-in jms export is the source. If `CWvsContext::OnIncubatorResult` is absent from it, escalate (do not guess) — Task 19 Step 1.
- **v87/v95/jms sub-body read orders**: expected to match v83 (shared client send-site code); verified from exports in Task 19 Step 1 before the branch is called done.
- **Capacity pre-check race**: a concurrent inventory fill between pre-check and AwardAsset is compensated (sacrifice + incubator re-created) and the failed-saga path emits `INCUBATOR_RESULT(0)`.
- **`FlagSpikes == FlagKarmaUse == 0x02`** collision exists in `libs/atlas-constants/asset/flag.go` but `FlagLock 0x01` is unambiguous.
- **Trade paths**: trade moves assets by inventory-internal operations (no AssetData mirror found outside the five listed), so owner survives; if a reviewer finds another mirror, extend Task 14's recipe to it.
- **Baseline publish/restore**: NOT affected — `assets` is not in atlas-data's `DumpTables` (`baseline/dump.go:20-27`), verified during planning.

## Verification gates (per CLAUDE.md — non-negotiable)

`go test -race ./... -count=1`, `go vet ./...`, `go build ./...` per changed module; `docker buildx bake` for atlas-inventory, atlas-saga-orchestrator, atlas-tenants, atlas-data, atlas-channel, atlas-storage, atlas-merchant; `tools/redis-key-guard.sh` from repo root (no GOWORK=off prefix); `packet-audit matrix --check` with no new problems; code review (`superpowers:requesting-code-review`) before any PR.
