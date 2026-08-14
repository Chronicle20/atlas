# task-221-miumiu-travel-store — Implementation Context

Companion to [`plan.md`](plan.md). Everything here was read in this worktree during the planning pass; file:line references are against the branch tip at plan time.

---

## 1. What the feature actually is

Cash item **5450000 (Miu Miu the Traveling Merchant)**, classification 545, opens **NPC 9090000**'s shop from any map and is consumed once — but only after the shop is confirmed open. `5451000 (Remote Gachapon Ticket)` shares the classification but **no audited client build emits it**, so it is a warn-and-unlock, not a feature.

Everything else in the task exists because the path is broken end-to-end: no shop data for 9090000, no `NPCShopHandle` on three templates, no saga step that can wait for a shop to open, and no way to correlate a shop event with a saga.

---

## 2. Key files, by subsystem

### atlas-channel — the entry point

| File | What matters |
|---|---|
| `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` | `CharacterCashItemUseHandleFunc` at `:35`. **Line 54-58 already re-validates cash-slot ownership for every arm** — arms must not repeat it. `GetCashSlotItemType` at `:997-1010` already classifies 545 → 37/38 and 59/60. Terminal catch-all warn at `:641`. The classification-first dispatch rationale is documented at `:503-507`. |
| `character_cash_item_use_point_reset.go` | The closest precedent for a saga-creating arm: `saga.Saga{TransactionId, SagaType, InitiatedBy, Steps}` built at `:57-77`, `enableActions` closure at `:28`. |
| `character_cash_item_use_megaphone.go` | The other file-per-arm precedent; also the model for a per-(version, tier) allow-list with IDA addresses in the comment. |
| `kafka/consumer/npc/shop/consumer.go` | `handleEnteredStatusEvent` `:59` announces `NPCShopWriter`; `handleErrorStatusEvent` `:88` writes `NPCShopOperationWriter`. `InitHandlers` `:35` registers both on `EVENT_TOPIC_NPC_SHOP_STATUS`. |
| `npc/shops/processor.go` `:48` | `EnterShop` produces the `ENTER` command today. |
| `npc/shops/producer.go` `:12` | `ShopEnterCommandProvider`. |
| `data/cash/rest.go` | Channel-side `cash.RestModel` — three fields today, **no `Npc`**. |
| `socket/init.go` `:56-66` | `SetDestroyer` — where per-character registries are cleared. |
| `shopscanner/registry.go`, `character/statreset/registry.go` | The two registry idioms to copy: singleton `sync.Once` + `sync.RWMutex`, `Key{Tenant, CharacterId}`, `ClearCharacter` on session destroy. |

### atlas-npc-shops — the shop service

| File | What matters |
|---|---|
| `kafka/message/shops/kafka.go` | The contract. `Command[E]{CharacterId, Type, Body}` `:12`, `StatusEvent[E]{CharacterId, Type, Body}` `:63`. **Neither carries a transaction id.** |
| `shops/processor.go` `:271-304` | `EnterAndEmit`/`Enter`. Shop-not-found → `return err`, **no event emitted**. `AddCharacter` `:284` overwrites unconditionally — no already-in-shop guard. `Exit` `:294` shows the `GetRegistry().GetShop(ctx, characterId) (uint32, bool)` signature. |
| `shops/producer.go` `:12-80` | Event providers. The doc comment at `:36-51` is the load-bearing one: `CShopDlg::OnPacket` @ `0x756da7` **throws `CDisconnectException`** when `CONFIRM_SHOP_TRANSACTION` arrives with no outstanding request. |
| `kafka/consumer/shops/consumer.go` `:52` | `handleEnterCommand`. |
| `shops/subdomain.go` `:28-29` | Seed subdomain: name `npc-shops`, path `npc-shops/shops`. |

### atlas-saga-orchestrator

| File | What matters |
|---|---|
| `saga/processor.go` `:86`, `:420` | `AcceptEvent(transactionId uuid.UUID, kind EventKind, opts …)`. **Transaction-id-keyed only** — there is no character-keyed accept. This is why the npc-shop contract must gain a transaction id. |
| `saga/event_acceptance.go` `:14-112` | `EventKind` const block. `acceptanceTable` `:118`; `ShowStorage: {}` (self-completing) at `:169`; `AcceptToStorage: {…Accepted, …Error}` at `:174` is the two-entry shape to copy. |
| `saga/event_acceptance_test.go` `:14-52` | `allActions` — the coverage test fails if a new action has no table entry. Must be updated in the same change. |
| `saga/handler.go` `:849` | Dispatch switch. `handleShowStorage` `:1858` (self-completing, uses `h.storageP`). `handleIncubatorResult` `:1157-1172` — **the lighter precedent**: `producer.ProviderImpl(h.l)(h.ctx)(topic)(Provider(payload))`, no `HandlerImpl` field. `HandlerImpl` struct `:179-209` with ~15 `With*` builders is what the incubator shape avoids. |
| `saga/producer.go` | Providers + the `…Fn` seam idiom (`emitConversationRewardNoticeFn` etc.). `IncubatorResultEventProvider` `:346` is the template. |
| `saga/producer_testseam.go` | `//go:build test` setters. The emit funcs and their `…Fn` vars live in untagged `producer.go`; only the setters are tagged. Test files using them (`late_event_integration_test.go`, `point_reset_compensation_test.go`'s neighbours) carry the same tag, so **`go test ./...` without `-tags=test` silently skips them** — the orchestrator needs both runs. |
| `saga/point_reset_compensation_test.go` `:21-97` | The compensation-test template: `testTenantContext()`, `NewBuilder().SetTransactionId().SetSagaType().SetInitiatedBy().AddStep(id, status, action, payload).Build()`, `NewCompensator(logger, ctx).WithCompartmentProcessor(mock)`, then `Dispatch*Rollbacks` called directly (no broker in the test env). |
| `saga/model.go` `:131`, `:275`, `:1315` | Alias blocks and the private unmarshal switch. Note the orchestrator's model uses `s.action`/`s.payload` (lowercase), unlike the shared lib. |
| `saga/compensator.go` `:266-268`, `:1371`, `:1411-1483` | Saga-type routing to `compensateCashItemUse`, and `DispatchCashItemUseRollbacks`'s per-action inverse switch. |
| `kafka/consumer/storage/consumer.go` | The consumer shape to copy verbatim (curried `InitConsumers`/`InitHandlers`, `AcceptEvent` → `StepCompleted`). |
| `main.go` `:117`, `:167` | Where consumers and handlers get registered. |

### libs/atlas-saga

| File | What matters |
|---|---|
| `model.go` `:13-44` | `Type` consts (…`PointReset`, `MegaphoneUse`). `:57+` `Action` consts, grouped by subsystem. |
| `payloads.go` `:107-115` | `DestroyAssetFromSlotPayload` — `InventoryType` byte (**5 = cash**), `Slot int16`, `TemplateId` for compensator re-create. `:496-502` `ShowStoragePayload` is the field-comment style to match. |
| `unmarshal.go` `:312` | The `case ShowStorage:` arm to copy. |

### atlas-data

| File | What matters |
|---|---|
| `cash/reader.go` | Parses `slotMax`, `protectTime`, `tradeBlock`, `stateChangeItem`, `bgmPath`, `rate`, `time`, pet skills, the `spec` subtree — **not `npc`**. |
| `cash/rest.go` `:41-52` | `RestModel` — the cash domain has no separate `model.go`; the RestModel *is* the model. |
| `consumable/reader.go:75` | `m.Npc = uint32(i.GetIntegerWithDefault("npc", 0))` — the verbatim precedent. |
| `cash/resource.go:22-24` | Route `/data/cash/items/{itemId}`. Consumables are at `/data/consumables`. |

### Config / seed / packets

| File | What matters |
|---|---|
| `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` | `:520-536` the `NPCShopHandle` block to copy; `:4425-4457` the `NPCShop` (`0x131`) and `NPCShopOperation` (`0x132`) writer blocks with the 13-entry operations table. |
| `deploy/seed/gms/83_1/npc-shops/shops/shop-1001000.json` | The envelope: `{"data":{"attributes":{"commodities":[…],"npcId":…,"recharger":…},"id":"…","type":"npc-shop"}}`, keys alphabetical. |
| `docs/packets/audits/STATUS.md` | Header at `:22` (`v48 # | v48 | v61 # | v61 | …`). `OPEN_NPC_SHOP` `:381`, `CONFIRM_SHOP_TRANSACTION` `:383`, `NPC_SHOP` (serverbound) `:572`. |
| `tools/trade-contract-mirror-guard.sh` | The 72-line model for the new npc-shop mirror guard. |
| `.github/workflows/pr-validation.yml:85` | The `redis-key-guard` job to model the new CI job on. |

---

## 3. Verified current state

Counted in this worktree, not recalled:

```
NPCShopHandle present:  gms_48, 61, 72, 79, 83, 84        (absent: 12, 87, 92, 95)
NPCShop writer present: gms_87, gms_95                     (absent: 48, 92)
```

Matrix rows (STATUS.md):
- `NPC_SHOP` serverbound: every version ❌ (unverified), opcodes v48 `0x030` … v87 `0x040`, v92 `0x043`, v95 `0x042`, JMS `0x035`.
- `OPEN_NPC_SHOP`: v48 blank/⬜; v61-v87 and v95/JMS ✅; **v92 `0x164` ❌**.
- `CONFIRM_SHOP_TRANSACTION`: v48 blank/⬜; **v92 `0x165` ❌**; rest ✅.

`❌` on the serverbound row means *unverified*, not unregistered — gms_83 binds `0x3D` today and works.

Seed shop counts: `deploy/seed/gms/12_1/npc-shops/shops/` has 98 files, `95_1` has 99. No `shop-9090000.json` exists anywhere.

---

## 4. The four decisions that shape the implementation

1. **The saga needs a transaction id on the npc-shop contract.** `AcceptEvent` is transaction-id-keyed and the contract has no such field. Threading one through is unavoidable and touches three Go modules (atlas-npc-shops owns it; atlas-channel and the orchestrator mirror it). `uuid.Nil` on the NPC-talk path never matches a saga, so the existing flow is unaffected.

2. **Enter failures need their own status-event type.** Reusing `ERROR` would make atlas-channel write `NPCShopOperation`, which the client throws `CDisconnectException` on when no buy/sell/recharge is outstanding. Hence `ENTER_ERROR`, handled channel-side with **no packet write at all** — just the conditional unlock.

3. **The unlock is registry-conditional, never unconditional.** `CShopDlg::SetShopDlg` does not clear `m_bExclRequestSent` (design §1.2, OQ-2), so the server must send `EnableActions` — but only for remote-initiated opens, or the already-verified NPC-talk wire changes. A small in-memory `remotemerchant` registry is the condition; it holds presentation state only, and losing it costs an unlock, never an item.

4. **Compensation reuses the existing cash-item reverse-walk.** `compensateCashItemUse` + `DispatchCashItemUseRollbacks` already handle "consume-after-effect" sagas; `RemoteMerchant` joins the saga-type list and `OpenNpcShop` gets an inverse (`EXIT`) in the rollback switch.

---

## 5. Dependency order

```
1  atlas-data npc field
2  atlas-channel npc field           (needs 1's json contract)
3  npc-shop contract: txn id + ENTER_ERROR + mirror guard
4  atlas-npc-shops enter failures    (needs 3)
5  libs/atlas-saga action + payload
6  orchestrator aliases              (needs 5)
7  orchestrator handler + acceptance (needs 3, 6)
8  orchestrator consumer             (needs 3, 7)
9  orchestrator compensation         (needs 5, 7)
10 channel registry
11 channel unlock wiring             (needs 3, 10)
12 channel handler arm               (needs 2, 5, 10)
13 shop seed data                    (independent — can run any time)
14 templates gms_87/92/95            (independent)
15 template gms_48                   (independent; needs an IDA session)
16 packet verification               (needs 14, 15)
17 live tenant reconciliation        (needs 14, 15)
18 full verification + code review   (needs everything)
```

Tasks 13, 14 and 15 are independent of the Go work and of each other — parallelise if convenient.

---

## 6. External prerequisites

| Task | Needs |
|---|---|
| 13 | A reachable `atlas-data` with each version's WZ ingested, plus that version's tenant id, to run the commodity existence sweep. If a version is not ingested, record `not-ingested` and seed all 26 — do not guess. |
| 15 | An open v48 IDB (resolve by binary **name** from `idb_list`, pass as `database`). |
| 16 | `packet-audit` on PATH (or run from source per `docs/packets/PROCESS.md`) and the relevant IDBs. |
| 17 | A live environment with the four tenants provisioned. |
| 18 | Docker with buildx. |

---

## 7. Risks

**7.1 The `EnterShop` signature change ripples.** `EnterShop` gains a leading `transactionId`; every caller in atlas-channel and the mock must be updated. `go build ./...` catches all of them — but the channel `kafka/message/npc/shop` package clause also changes from `shop` to `shops` (so the mirror guard can compare `package` lines), which the compiler catches only for unaliased importers. Build the module, don't assume.

**7.2 `InitHandlers` in the channel npc/shop consumer has no `ctx`.** The TTL sweep needs one carrying the tenant. Thread the caller's context through the curried chain rather than fabricating a `context.Background()`.

**7.3 gms_48's `0xE5`/`0xE6` are IDB-derived, not fixture-verified.** The same reading procedure reproduces v83's known-correct `0x131`/`0x132`, and v48 has a distinct `CTrunkDlg` family so the dialog identity is not confused. The cells still only move off ⬜ through `/verify-packet`. If v48's body layout differs from v83's, that is a version gate found during verification, not a design change.

**7.4 gms_92's operations table is inherited, not derived.** Task 14 cross-checks gms_87 and gms_95 first and falls back to an IDB derivation if they disagree. Copying blind is `[[bug_gms_61_72_79_interaction_operations_wrong]]`.

**7.5 OQ-5 stays open until Task 13's sweep runs.** The design could not settle per-version commodity existence from this worktree — only a v83-era WZ set is available locally. Task 13 makes it an explicit, recorded step with a defined fallback.

**7.6 JMS is knowingly left inconsistent.** `GetCashSlotItemType` returns 37 for JMS where the client computes 36 (`get_cashslot_item_type` @ `0x49a1ee`). Inert today because nothing consumes the JMS value, and `remoteMerchantEnabled` returns false for JMS. Recorded in design §7.3 as a bounded follow-up (JMS type 36 + a JMS template/shop pass), not a silent gap.

**7.7 The npc-shop contract now has three copies.** `tools/npc-shop-contract-mirror-guard.sh` (added in Task 3, wired into CI) is what keeps them from drifting into a silently zero-valued body.

---

## 8. Cross-service seams that no compiler checks

Per `[[feedback_green_tests_miss_cross_service_seams]]`, these get event-level assertions, not stubs:

| Seam | Producer | Consumer | Asserted in |
|---|---|---|---|
| `COMMAND_TOPIC_NPC_SHOP` / `ENTER` | orchestrator `NpcShopEnterCommandProvider` | atlas-npc-shops `handleEnterCommand` | Task 7 producer test (body + txn id), Task 3 consumer test (round trip) |
| `EVENT_TOPIC_NPC_SHOP_STATUS` / `ENTERED` | atlas-npc-shops `enteredEventProvider` | orchestrator `handleEnteredEvent`, channel `handleEnteredStatusEvent` | Task 4 processor test, Task 8 consumer test |
| `EVENT_TOPIC_NPC_SHOP_STATUS` / `ENTER_ERROR` | atlas-npc-shops `enterErrorEventProvider` | orchestrator `handleEnterErrorEvent`, channel `handleEnterErrorStatusEvent` | Task 4, Task 8, Task 11 |
| `COMMAND_TOPIC_NPC_SHOP` / `EXIT` (compensation) | orchestrator `NpcShopExitCommandProvider` | atlas-npc-shops `handleExitCommand` | Task 9 compensator test |
| `cash_items` `npc` attribute | atlas-data `cash.RestModel` | atlas-channel `cash.RestModel` | Task 1 reader test, Task 2 decode test |

The mirror guard covers the struct-shape half of these; the tests cover the semantic half.
