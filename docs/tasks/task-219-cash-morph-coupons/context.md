# Cash Transformation (Morph) Coupons — Implementation Context

Task: `task-219-cash-morph-coupons`
Companion to [`plan.md`](./plan.md). PRD: [`prd.md`](./prd.md). Design: [`design.md`](./design.md).
Created: 2026-08-12

This is the orientation document for whoever implements the plan. Everything here was read from source, WZ data, or a decompiled binary during the design or planning phase — nothing is carried from memory.

---

## 1. What this feature is, in one paragraph

Cash transformation coupons (`Item.wz/Cash/0530.img.xml`, item classification 530) transform the character into a Morph.wz creature for ten minutes and heal 50 HP. Today they do **nothing**: the client sends the use request, `atlas-channel`'s cash-item handler falls through every arm to a terminal warn, no item is consumed and no effect applied. The morph effect itself already exists — task-140 landed the `TemporaryStatTypeMorph` applier for use-tab morph potions. What is missing is plumbing in three services plus one packet codec.

---

## 2. The three gaps, and why each one is a gap

| Layer | What's missing | Why it can't just reuse what's there |
|---|---|---|
| `atlas-data` | `cash/reader.go` never parses `spec/morph` or `spec/hp` | The values are dropped at ingest, so no downstream service can ever observe them regardless of what it does |
| `atlas-consumables` | no consume branch for classification 530; the cash REST model has no `morph`/`hp`/`time` spec keys | `ConsumeStandard` hard-codes `inventory2.TypeValueUse` (`processor.go:471`) and fetches from the **consumable** data resource (`:465`), where cash items do not exist |
| `atlas-channel` | no arm for classification 530 | — |
| `libs/atlas-packet` | no sub-body codec for the (empty) case-40/41 arm | — |

---

## 3. Key files, with the line numbers that matter

### `libs/atlas-packet/cash/serverbound/`
- `item_use.go:21-23` — `UpdateTimeFirst(t tenant.Model) bool` = `(GMS && major >= 87) || JMS`. The single source of truth for whether `updateTime` leads or trails. Never re-derive it.
- `item_use.go:26-45` — the `ItemUse` common header (`updateTime?`, `int16 source`, `int32 itemId`) and its IDA provenance per version.
- `item_use_pet_consumable.go` — **the template to copy.** An existing "empty body + trailing updateTime" codec, 46 lines. `item_use_pet_consumable_test.go` is the template for the round-trip test.
- `libs/atlas-packet/test/` — `pt.Variants` (13 tenant variants incl. GMS v48/61/72/79/83/84/86/87/92/95 and JMS v185), `pt.CreateContext(region, major, minor)`, `pt.RoundTrip(t, ctx, encode, decode, opts)`.

### `services/atlas-data/atlas.com/data/cash/`
- `rest.go:9-26` — the `SpecType` const block. Note `SpecTypeTime`'s comment currently says "Duration in minutes"; that is wrong for 0530 (600000 = ten minutes in **milliseconds**) and the plan corrects the comment.
- `reader.go:114-139` — the `spec` block. The `expR`/`drpR`/`time` parses at `:130-138` are the omit-when-zero pattern to follow; the `inc`/index parses above them are unconditional (they write zeros), which is why `morph`/`hp` must go in the lower group.
- `reader_test.go` — fixture pattern: a `const test<Name>XML` string, then `Read(l)(xml.FromByteArrayProvider([]byte(x)))` collected via `model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)`. `Identity` is defined at `:411`. `TestReaderExpCoupons` (`:618`) is the closest model.

### `services/atlas-consumables/atlas.com/consumables/`
- `cash/rest.go` — `SpecType` set (indexes + `inc` only; no `time`, `morph` or `hp`), `RestModel`, and `Extract(RestModel) (Model, error)`. `Extract` is **exported**, which is how tests build a `cash.Model` without a test-only constructor.
- `cash/model.go:11-14` — `GetSpec(SpecType) (int32, bool)`. The `bool` is what makes "absent" distinguishable from "zero".
- `cash/processor.go` — `Processor` interface is a single method: `GetById(itemId uint32) (Model, error)`.
- `consumable/processor.go:66` — `type ItemConsumer func(l logrus.FieldLogger) func(ctx context.Context) error`.
- `consumable/processor.go:109-123` — `usesStandardConsumer`. 530 must **not** be added here.
- `consumable/processor.go:149-227` — `effectPlan` / `computeEffectPlan`. The pure-planner precedent; `:156` documents `duration` as "WZ `time` spec in ms" and `:218-224` records that a prior `/1000` made every timed consumable buff expire ~1000x too early.
- `consumable/processor.go:232-260` — `ApplyItemEffects`. `:258` is the exact `Apply` call shape to mirror: `bp.Apply(f, characterId, -int32(itemId), byte(0), plan.duration, plan.statups)(characterId)`.
- `consumable/processor.go:262-319` — `RequestItemConsume`. The classification if/else chain is `:274-308`. **The reward-table fallback at `:288` is the trap**: it queries the consumable data resource, fails for a cash item, and falls through to `ConsumeBare`, which destroys the coupon and applies nothing. The new branch must precede it.
- `consumable/processor.go:373-393` — `ConsumeError`: cancels the reservation and emits the client error event. The inventory type argument decides which compartment's reservation is released.
- `consumable/processor.go:454-480` — `ConsumeStandard`: the parallel-read-then-commit-then-apply ordering to mirror.
- `consumable/processor.go:566-603` — `ConsumeCashPetFood`: the precedent for reading cash data. **Contains a live inconsistency** — see §7.
- `character/processor.go:21` — `ChangeHP(f field.Model, characterId uint32, amount int16) error`.
- `character/buff/processor.go:17` — `Apply(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []stat.Model) model.Operator[uint32]`.
- `map/character/processor.go:14` — `GetMap(characterId uint32) (field.Model, error)`.
- `compartment/processor.go` — `ConsumeItem(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error`.

**Mocks that exist and are currently unused by the `consumable` package** (all with `<Method>Func` fields and a `var _ X.Processor = (*ProcessorMock)(nil)` assertion): `cash/mock`, `map/character/mock`, `character/mock`, `character/buff/mock`, `compartment/mock`. The plan's Task 5 is the first thing in this package to use them.

### `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
- `:36-58` — handler entry, header decode, `updateTimeFirst`/`updateTime`/`source`/`itemId` locals, then the **ownership check** (`cashItemInSlotFunc` + template-id equality). Every arm below inherits it by position.
- `:62-69` — the pet-consumable arm: the exact delegation shape the new arm copies.
- `:503-639` — the "Classification-FIRST dispatch" block. `category := item.GetClassification(itemId)` is at `:507`. Its leading comment explains why classification must beat the type byte.
- `:639-641` — **the insertion point.** The megaphone block closes at 639; the terminal warn is 641.
- `:641` — the terminal warn that must no longer fire for classification 530.
- `:681-692` — `cashItemInSlotFunc`, the package-var test-seam precedent (the other is `useRockFunc` in `teleport_rock_use.go:35`).
- `:708-713` — `viciousHammerCashSlotItemType`.
- `:715-…` — `GetCashSlotItemType`; the three relevant arms are `:896-901` (gachapon), `:924-929` (transformation), `:937-942` (pet evolution).

- `character_cash_item_use_test.go` — `mustTenant` (`:20`), `installCashItemInSlotSeam` (`:33`), `newCashItemUseTestSession` (`:48`, builds a **v83** GMS session so `updateTimeFirst` resolves false), `cashItemUsePrefix` (`:73`). `teleport_rock_use_test.go:46` has `installUseRockSeam`, the capture-slice seam pattern.

### `services/atlas-channel/atlas.com/channel/consumable/processor.go`
- `:43` — `RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error`. Note this is the **channel-side** signature; the consumables-service method of the same name has a completely different one (`processor.go:262`: `(c channel.Model, characterId uint32, slot int16, itemId item2.Id, quantity int16, petId uint64)`). Don't mix them up.
- `:50` — `updateTime` is logged only; it is **not** forwarded past `RequestItemConsumeCommandProvider`.

### `libs/atlas-constants/item/constants.go`
- `:39` `ClassificationConsumableTransformation = 221` (use-tab morph potions — routes to `ConsumeStandard`, untouched)
- `:91` `ClassificationGachaponCoupon = 522`
- `:97` `ClassificationTransformationCoupon = 530` ← the gate
- `:101` `ClassificationPetEvolution = 538`

---

## 4. Verified facts (with provenance)

### WZ data — re-verified during planning, from two independent local corpora
`Item.wz/Cash/0530.img.xml` contains exactly three items. Every one:

```xml
<int name="hp"    value="50"/>
<int name="time"  value="600000"/>
<int name="morph" value="1"/>   <!-- 2 for 05300001, 3 for 05300002 -->
```

`info` carries `cash=1`, `price=100`, `slotMax=200`, `tradeBlock=1`. **No `morphRandom` node on any item** — the weighted selector in `consumable/morph.go` stays unwired on this path. The imgdir names are zero-padded to eight digits (`05300000`); `parseCashId` uses `strconv.Atoi`, which accepts the leading zero, so ids materialise as `5300000`/`5300001`/`5300002`.

### Client contract — GMS v83 `MapleStory_dump.exe`, IDA-verified during design
`CWvsContext::SendConsumeCashItemUseRequest` @ `0xa0a63f`:
- Header: `Encode2(nPOS)` @ `0xa0a6bf`, `Encode4(nItemID)` @ `0xa0a6cb`, then a 58-case jump table keyed on `type - 12`.
- **The case-40 arm (`0xa0caf0`–`0xa0cb37`) contains no `Encode*` call at all.** Three client-side predicates (the first, `sub_A0ECCD` @ `0xa0eccd`, is literally `itemId / 10000 == 530`), then `play_item_sound(nItemID, 0x29)` @ `0xa0cb30`, then the shared send tail. **The sub-body is empty.**
- Shared tail: `CanSendExclRequest(0x1F4, 0)` @ `0xa0a8fa`; on success `loc_A0EA53` appends `Encode4(get_update_time())` @ `0xa0ea5c`, calls `SendPacket`, then `SetExclRequestSent(1)` @ `0xa0ea6f`.

### The exclusive-request lock — RESOLVED, no unlock needed
PRD open question 1, answered in design §1.2 with IDA evidence:
- `CanSendExclRequest` @ `0x485bf7`: `return !this[2089] && (a3 || …) && get_update_time() - this[2090] >= a2;` — the lock is the dword pair `this[2089]` (flag) / `this[2090]` (timestamp).
- `OnGameStageChanged` @ `0xa0400e` (the known-good `SET_FIELD` unlock) clears it with `*&this[1].m_Cookie.szCookie[96] = 0; *&this[1].m_Cookie.szCookie[100] = get_update_time();`
- `OnInventoryOperation` @ `0xa1ead9` opens with the **identical pair of stores**, guarded by `if (CInPacket::Decode1(iPacket))` @ `0xa1eaf4` — that leading byte is `bOnExclRequest`.
- Server side: `libs/atlas-packet/inventory/clientbound/change_batch.go:35` writes it as `w.WriteBool(!m.silent)`, and every emitter on the consume path passes `silent = false` (`atlas-channel/kafka/consumer/asset/consumer.go:316`, `:428`, `:506`).

**Conclusion: a successful `ConsumeItem` on the Cash compartment unlocks the client on its own** — the same mechanism that makes ordinary Use-tab potions work today despite nothing in the consumable path calling `session.EnableActions`. Emit no explicit unlock. Failure paths send nothing (matching every neighbouring arm); the client's own `CanSendExclRequest` timeout recovers there, exactly as it does for a failed potion use. This also satisfies "never unlock an outcome that warps" vacuously — a morph does not warp.

### The cash-slot type-byte collision matrix — read from `GetCashSlotItemType`

| Classification | GMS ≥ 95 | otherwise |
|---|---|---|
| 522 gachapon coupon | **40** | 39 |
| 530 transformation coupon | 41 | **40** |
| 538 pet evolution | 42 | **41** |

Byte 40 = transformation pre-95, gachapon on ≥95. Byte 41 = pet evolution pre-95, transformation on ≥95. **The collision is cross-version, not same-tenant** — the PRD's FR-1.3 says "pre-95, gachapon also maps to 40", which is wrong (pre-95 gachapon is 39). The classification gate is still exactly the right fix, and the real collision is nastier than described: a type-byte-keyed arm would silently change meaning at a version bump. Verified by grep that no `it ==` arm in the handler uses 39, 40 or 41 today.

---

## 5. Decisions already made — do not relitigate

From the PRD interview:
- Effect application lives in `atlas-consumables`, not `atlas-channel` (keeps consume and effect in one unit).
- Live re-ingest and REST verification of `5300000` is an **operational follow-up**, not an acceptance criterion.
- Re-use while morphed replaces the morph and restarts the timer — no "already morphed" rejection.
- No version literals; classification gating only.
- The serverbound sub-body is empty.

From the design phase:
- The exclusive-request lock resolves to "no explicit unlock" (§4 above).
- A dedicated per-type codec (`ItemUseMorphCoupon`), not a reuse of `ItemUsePetConsumable` and not a shared extracted type. If a *third* empty-body arm appears, extracting a shared `ItemUseTrailingUpdateTime` becomes right — revisit then.
- A small dedicated pure planner (`computeMorphCouponPlan`), not inline application and not a shared planner with `computeEffectPlan` (the two spec vocabularies are different types with different key sets; the adapter would exceed the planner it saves).

From the planning phase:
- `atlas-consumables`' cash `SpecType` set gains **three** constants (`morph`, `hp`, **`time`**), not the two FR-3.1 names. It has no `SpecTypeTime` at all, and FR-3.6 is unreachable without it. Strict superset, stated so it isn't read as scope creep.
- `ConsumeMorphCoupon` splits into a `morphCouponDeps`-taking core plus a thin exported wrapper. This is a **new pattern** in the `consumable` package — see plan §"Corrections", item 3, for the reasoning and the trade-off a reviewer may want to push back on.
- The new consumer lives in a new file (`consumable/morph_coupon.go`), not appended to the 71 KB `processor.go`. Package precedent: `morph.go`, `skill_book.go`, `vega.go`, `reward.go`, `processor_catch.go`.

---

## 6. Non-goals — explicitly out of scope

- The other fifteen unimplemented cash slot-item types (meso sacks 520, jukebox 510, pet name tag 517, Duey 533, name change / world transfer 540, Maple Life 543, MiuMiu 545, expiration extenders 550, karma scissors 552, store permit, cosmetic coupons, emotes, pet skill items). Each is its own task.
- `morphRandom` on cash items — no 0530 item carries one in any inspected corpus.
- Registering `CharacterCashItemUseHandle` in `template_gms_12_1.json`. It is the one template of eleven that omits the handler, so gms_12 is a documented no-op for the **entire** cash-item-use family, not a regression from this task.
- Any new clientbound packet writer. Morph rides the existing temporary-stat pipeline.
- Re-ingesting live tenant WZ data (operational follow-up).
- Anti-cheat for attacking while morphed (ruled out in task-140's PRD).

---

## 7. Traps and known limitations

1. **The reward-table fallback at `consumable/processor.go:288`.** It queries the *consumable* data resource; a cash item's lookup fails and it drops through to `ConsumeBare` — coupon destroyed, nothing applied. The new routing branch must sit **before** it. This is the single most likely way to implement the whole feature and still ship a silent no-op.

2. **Stale ingested data.** The reader change materialises `morph`/`hp` only for **newly ingested** WZ. Tenants ingested before this lands serve a `spec` without them, so the coupon consumes and does nothing (the "both absent" row). The consumer logs a warning naming this exact cause; the operational re-ingest is a tracked follow-up.

3. **`ConsumeCashPetFood`'s compartment-type inconsistency.** `processor.go:591-599` passes `inventory2.TypeValueUse` on three paths for what is unambiguously a Cash-compartment item, while its first `ConsumeError` at `:576` correctly passes `TypeValueCash`. That looks like a live defect in a neighbouring consumer. **This task does not fix it** — different item family, different acceptance criteria, no evidence gathered on its runtime impact — but `ConsumeMorphCoupon` must use `TypeValueCash` on *every* path. Worth its own task with its own evidence.

4. **Two `RequestItemConsume` methods with different signatures** (channel-side vs consumables-side). See §3.

5. **`tools/lint.sh --check` false-fails without nvm** on the atlas-ui half. Run `nvm use 22` first if it trips with no TS files changed — but never declare it clean without seeing exit 0.

6. **`tools/buff-duration-guard.sh`** exists because the millisecond contract has been flipped three times in prose alone. It fingerprints json tag sets, not type names. Do not silence it with `//buffdurationguard:allow`.

---

## 8. Verification commands

Per CLAUDE.md, from the worktree root.

Modules to verify: `libs/atlas-packet`, `services/atlas-data/atlas.com/data`, `services/atlas-consumables/atlas.com/consumables`, `services/atlas-channel/atlas.com/channel` — each `go test -race ./...`, `go vet ./...`, `go build ./...`.

Repo-root guards: `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/buff-duration-guard.sh`, `tools/lint.sh --check`.

**Not required, but confirm rather than assert:** `docker buildx bake` (only if a `go.mod` is touched — not expected); the four template guards and `service-registration-guard.sh` (no seed template, `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work` or `db-bootstrap.sh` change). Plan Task 8 Steps 2-3 include the `git diff` commands that prove both claims.

---

## 9. Task dependency graph

```
Task 1 (packet codec) ──────────────┐
Task 2 (atlas-data reader) ─────────┤
Task 3 (consumables REST mirror) ───┤
      └─▶ Task 4 (planner + predicate)      │
              └─▶ Task 5 (consumer)         │
                      └─▶ Task 6 (routing)  │
                                            ▼
                                  Task 7 (channel arm)
                                            │
                                            ▼
                                  Task 8 (verify + docs)
```

Tasks 1, 2, 3 are mutually independent and may run in parallel. Task 4 needs Task 3's `SpecType` constants. Task 7 needs Task 1's codec. Task 8 is last.
