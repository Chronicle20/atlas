# task-130 Vega's Spell — Implementation Context

Companion to `plan.md`. Everything below was verified against the worktree source during planning (2026-07-02); file:line references are anchors, not guarantees — re-check before editing.

## Inputs

- PRD: `docs/tasks/task-130-vegas-spell/prd.md`
- Design (authoritative): `docs/tasks/task-130-vegas-spell/design.md` — includes live IDA verification of v83 + v95 (addresses cited inline)
- Plan: `docs/tasks/task-130-vegas-spell/plan.md` (13 tasks)

## Architecture in one paragraph

The channel's cash-item-use handler gains a category-561 arm that decodes the new six-int32 `ItemUseVegaScroll` sub-body (trailing updateTime on EVERY version — design §2.1) and emits a new `REQUEST_VEGA_SCROLL` command. atlas-consumables validates everything up front (vega item at cash slot, scroll at use slot, scroll's natural rate exactly 10/60, equip via a dual-sign resolver, `ValidateScrollUse`), then chains two single-item reservations (CASH vega → USE scroll) via item-id-keyed once-listeners; on the second confirmation `ConsumeVegaScroll` re-validates, rolls at the boosted rate (30/90) through the shared `applyScrollCore` extracted from `ConsumeScroll`, consumes both items, and emits `VEGA_SCROLL{success,cursed}`. The channel consumer answers with VegaScroll start(outcome) + result(outcome) back-to-back (no server delay), the map-broadcast ItemUpgrade (legendary=false, white=false), and enable-actions; rejections emit `VEGA_INVALID` → VegaScroll INVALID packet (0x42, required to unwedge the dialog) + enable-actions.

## Key files (verified)

| Concern | File | Notes |
|---|---|---|
| Cash-item handler | `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` | Handler 25-112; `wp writer.Producer` currently discarded (`_`, line 25) — Task 10 un-discards; 561 branch at line 476 uses raw literal (68 pre-95 / 71 ≥95); fall-through warn line 110; line-108 updateTime TODO stays (task-126 shares it) |
| Outer prefix codec | `libs/atlas-packet/cash/serverbound/item_use.go` | `updateTimeFirst = GMS && ≥95` handled here (lines 47-56); v95 prefix re-verified: opcode 0x55, Encode4(updateTime), Encode2(slot), Encode4(itemId) |
| Sub-body codec pattern | `libs/atlas-packet/cash/serverbound/item_use_field_effect.go` | value-receiver Encode / pointer-receiver Decode; vega codec has NO updateTimeFirst flag (unconditional 6× int32) |
| Writer + body-func pattern | `libs/atlas-packet/storage/operation_body.go`, `libs/atlas-packet/party/clientbound/operation_body.go` | `atlas_packet.WithResolvedCode("operations", <FIXED KEY>, factory)`; `ResolveCode` in `libs/atlas-packet/resolve.go:27` (float64 or "0x.." string values; 99 fallback) |
| Fixture style | `libs/atlas-packet/party/clientbound/member_hp_test.go`, `storage/clientbound/show_test.go` | `// packet-audit:verify packet=<pkg>/<dir>/<Name> version=<v> ida=0x<addr>` markers; `pt.Variants`, `pt.CreateContext`, `pt.RoundTrip` |
| Scroll pipeline | `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` | `RequestScroll` 515-579; `ValidateScrollUse` 581-600; `ConsumeScroll` 606-738; `// TODO consume vega scroll` at 641 (deleted by Task 6); `PassScroll` 806 / `FailScroll` 859; `ConsumeError` 280-297 |
| Once-listener | `services/atlas-consumables/.../kafka/once/compartment/once.go:12` | `ReservationValidator(transactionId, itemId uint32)` — matches on `TransactionId` AND `ItemId`; item-id chain keying is collision-free (561xxxx vs 20xxxxx) |
| Compartment ops | `services/atlas-consumables/.../compartment/processor.go` | `RequestReserve(txn, charId, invType, []Reserves)` 33; `Consume(f)` wrapper 37; `ConsumeItem` 43; `DestroyItem` 47; `CancelItemReservation` 51; `Reserves{Slot,ItemId,Quantity}` |
| Inventory accessors | `services/atlas-consumables/.../inventory/model.go` | `Equipable()` 15, `Consumable()` 19, `Cash()` 31; compartment `FindBySlot(int16)` at `compartment/model.go:38` |
| Equipment (negative slots) | `services/atlas-consumables/.../equipment/model.go` + `equipment/slot/model.go` | `NewModel()` seeds all slots; `Get(slot2.Type)`; `slot.Model{Position, Equipable *asset.Model}` |
| Test builders | `consumable/processor_test.go` | `asset.NewBuilder(uuid, templateId)` (builder.go:92), `createTestEquipableAsset`/`createTestScrollAsset` helpers, `consumable3.RestModel{...}` + `Extract` (`makeCureModel` pattern); `character.NewModelBuilder()` model.go:365 with `SetInventory`/`SetEquipment`; `inventory.NewBuilder(charId)`; `compartment.NewBuilder(id, charId, type, cap)` + `AddAsset` |
| Consumable data | `services/atlas-consumables/.../data/consumable/model.go` | `SuccessRate()` 118, `CursedRate()`, `*Increase()` getters; RestModel fields `Success`, `Cursed`, `IncreaseSTR`... (rest.go:14-73) |
| Consumables command consumer | `services/atlas-consumables/.../kafka/consumer/consumable/consumer.go` | `InitHandlers` 27-45 (4 existing arms); arm pattern `handleRequestScroll` 58-66 |
| Consumables kafka contract | `services/atlas-consumables/.../kafka/message/consumable/kafka.go` | Command envelope HAS `TransactionId`; channel mirror does NOT (missing JSON field decodes zero — fine) |
| Channel consumable pkg | `services/atlas-channel/atlas.com/channel/consumable/{processor,producer}.go` + `kafka/message/consumable/kafka.go` | `RequestScrollUse`/`RequestScrollCommandProvider` are the templates for the vega twins |
| Channel status consumer | `services/atlas-channel/.../kafka/consumer/consumable/consumer.go` | `handleErrorConsumableEvent` 57-81 (enable-actions = empty `statpkt.NewStatChanged(..., true)` line 76); `handleScrollConsumableEvent` 83-101 (`_map.ForSessionsInMap` broadcast of `charpkt.NewItemUpgrade(charId, success, cursed, legendarySpirit, whiteScroll)`) |
| Announce-from-handler | `services/atlas-channel/.../socket/handler/character_skill_use.go:134` | `session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(...Encode)` applied to session |
| Writer registration | `services/atlas-channel/atlas.com/channel/main.go` | `produceWriters()` 609+ (cashcb aliases at 616-618); `handlerMap[cashsb.CharacterCashItemUseHandle]` at 867 |
| Seed templates | `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_92,gms_95,jms_185}_1.json` | writer-entry shape: `{"opCode": "0x135", "writer": "StorageOperation", "options": {"operations": {...ints...}}}`; handler shape `{"opCode": "0x4F", "validator": "LoggedInValidator", "handler": "CharacterCashItemUseHandle"}`; handler present ONLY in 83+84 today (task-126 adds 87/95/jms — skip-if-present) |
| Template gate | `tools/template-symbol-check.sh` | run per touched template |
| Registry rows | `docs/packets/registry/*.yaml` | VEGA_SCROLL: gms_v83:1707 (358=0x166, fname CUIVega::OnVegaResult), gms_v84:2299 (358 SUSPECT), gms_v87:1817 (0x17B), gms_v95:2020 (0x1AD), jms_v185:1914 (0x183); NO gms_v92 registry file |
| Audit tooling | `tools/packet-audit/` | `candidatesFromFName` cases in `cmd/run.go` (~1831 CWvsContext senders block — task-126 adds the ConsumeCashItemUseRequest case; SPLICE, don't duplicate); `matrix --check`; jms audit dir is `docs/packets/audits/jms_v185` (pass `--audit-dir` explicitly) |
| Playbook | `docs/packets/audits/VERIFYING_A_PACKET.md` | governing procedure for every Task 4 cell |

## Load-bearing decisions (frozen at plan time)

1. **Delivery = new `REQUEST_VEGA_SCROLL` command** on `COMMAND_TOPIC_CONSUMABLE` (design §3.1). `RequestScrollBody` and its producers untouched.
2. **Chained single-item reservations** CASH→USE (design §3.2). NEVER batch: inventory-side `RequestReserve` returns after the first entry (pre-existing bug, design §2.8 — owner-flagged, NOT fixed here). Stall envelope: reservation failures emit no event; 30s TTL self-heals (§2.9).
3. **Outcome-keyed operations** (`START_SUCCESS/START_FAILURE/RESULT_SUCCESS/RESULT_FAILURE/INVALID`) because v95's START byte carries the outcome (popup selection from `m_nRet1`, design §2.3) and modes shifted +4 v83→v95 (§2.2). v83 collapses both START keys to 0x40. Every `WithResolvedCode` call fixes its key as a named constant.
4. **Immediate resolution**: start+result sent back-to-back; no 3s Cosmic timer (owner decision). Both clients latch the result and animate on their own clock.
5. **Rejection surface = INVALID (0x42)** + enable-actions — required, not optional: the dialog is excl-request-blocked after sending; silence wedges it (resolves PRD open question 3).
6. **Dual-sign equip resolver on the vega path only** (design §2.7): positive slot → equip inventory `FindBySlot`; negative → classic `Equipment().Get`. The normal scroll path's inability to address positive slots is owner-flagged, not fixed.
7. **Core extraction** (design §4.6): `buildScrollChanges` (pure) + `applyScrollCore` (roll → changes → curse roll → ChangeStat), `successProb` parameterized. `ConsumeScroll` keeps identical rand-call order (success roll → chaos rolls → curse roll) and log lines. The `roll <= prob` comparator is inherited unchanged (PRD non-goal).
8. **Wire truth (IDA-verified in design)**: serverbound sub-body = 6 int32s with trailing updateTime on ALL versions (§2.1 — two more fields than Cosmic reads); v83 opcode 0x166 + modes 0x40/0x41/0x43/0x42; v95 opcode 0x1AD + modes 0x44|0x49 / 0x45/0x47 / 0x42. Cosmic's "0x39/0x45 crash" comment is wrong for this client — the else-arm is always the safe notice.
9. **Version gating**: v84's registry 0x166 presumed WRONG (csv carryover above the +2..+10 shift, §2.5) — wire only after re-verification; v87/jms verify-then-wire; v92 parked entirely (§2.6). Unverifiable version = BLOCKED + omitted wiring, never a guessed opcode.
10. **`flag` int32 (wire field 5)** read, logged via `String()`, ignored (v95 IDB calls it `m_nWhiteScrollUse` but always writes 1).

## Task dependency order

1 (constants), 2 (serverbound codec), 3 (clientbound writer), 6 (core extraction) are independent starters. 4 (IDA campaign) after 2+3. 5 (contract) after 1. 7 (vega path) after 1+5+6. 8 (consumer arm) after 5+7. 9 (channel mirror) after 1. 10 (handler arm) after 1+2+9. 11 (channel consumer) after 3+9. 12 (templates) after 4 (+ 3 for the symbol check). 13 last.

## Risks / escalation triggers

- **Task 4 (IDA)**: instance set rotates — `list_instances` and match binary NAME first (plan-time set: v83-dump 13342, v95 13341; NO v84/v87/jms). Unresolvable fname/opcode = STOP AND REPORT BLOCKED for that version; ship without its wiring. v95 START pairing (68 vs 73) pinned via string-pool templates 5417/5418; if reversed, swap the two values in the v95 fixture + template only.
- **Task-126 collisions** (same fname case in run.go, same USE_CASH_ITEM audit, same handler template entries, same line-108 TODO): whichever lands second splices/skips — check `git log` and the artifact before adding.
- **Marked verify-before-use sites**: `pt.RoundTrip` call shape (Tasks 2/3), `tenant.Create` signature (Task 10), `asset.ModelBuilder.SetSlot` spelling + a valid negative slot position from `slot.Slots` (Task 7), `candidate` literal shape (Task 4), signed-int helpers on response.Writer/request.Reader (Task 2).
- **Live tenants**: new writer/handler opcodes need config PATCH + channel restart (deployment.md, Task 12); seed templates only affect new tenants.
- **Symptom map for a bad rollout**: handler fall-through warn = missing handler entry; "Property [operations] missing ... 99" = missing writer options; dialog stuck open after rejection = INVALID packet not sent.

## Verification gates (Task 13)

Per-module `go test -race` / `go vet` / `go build` (atlas-constants, atlas-packet, atlas-consumables, atlas-channel, packet-audit); `go run ./tools/packet-audit matrix --check`; `go run ./tools/packet-audit dispatcher-lint`; `tools/redis-key-guard.sh` (from repo root, no GOWORK=off); `docker buildx bake all-go-services` (both shared libs changed); `tools/template-symbol-check.sh` per touched template; `superpowers:requesting-code-review` before PR.
