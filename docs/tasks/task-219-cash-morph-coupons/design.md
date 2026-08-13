# Cash Transformation (Morph) Coupons — Design

Task: task-219-cash-morph-coupons
PRD: [`prd.md`](./prd.md) (approved)
Status: Draft
Created: 2026-08-12

---

## 0. Scope recap

Three plumbing gaps, one per service, plus one packet codec:

| Layer | Gap | Change |
|---|---|---|
| `atlas-data` | cash reader drops `spec/morph` and `spec/hp` | two `SpecType` constants + two parses |
| `atlas-consumables` | no consume branch for classification 530; cash REST model drops the same two keys | two `SpecType` constants + `ConsumeMorphCoupon` + one routing branch |
| `atlas-channel` | no arm for classification 530 → terminal warn | one classification-gated arm |
| `libs/atlas-packet` | no sub-body codec for the (empty) case-40/41 arm | one codec + round-trip test |

No new REST endpoints, Kafka topics, message bodies, DDL, or template edits.

---

## 1. Evidence gathered during design

Everything below was read this session; nothing is carried from memory.

### 1.1 WZ data (FR-2.3 fixture values confirmed)

`Item.wz/Cash/0530.img.xml` (local corpus at `<wz-root>/Item.wz/Cash/0530.img.xml`) contains
exactly three items — `05300000`, `05300001`, `05300002`. Each `spec` node holds exactly three
children:

```xml
<int name="hp"    value="50"/>
<int name="time"  value="600000"/>
<int name="morph" value="1"/>   <!-- 2 and 3 for the other two items -->
```

`Morph.wz` carries `0001.img.xml`, `0002.img.xml`, `0003.img.xml`, so the three morph ids resolve.
The imgdir names are zero-padded to eight digits (`05300000`); `parseCashId` uses `strconv.Atoi`,
which accepts the leading zero, so ids materialise as `5300000`/`5300001`/`5300002`.

No `morphRandom` node is present on any of the three items, confirming the PRD's non-goal.

### 1.2 Open question 1 — RESOLVED: what clears the client's exclusive-request lock

**Answer: the non-silent INVENTORY_OPERATION packet the consume commit already produces. No
`EnableActions` is required, and adding one would be wrong.**

Evidence, GMS v83 `MapleStory_dump.exe.i64` (IDB session resolved by binary name):

- `CWvsContext::CanSendExclRequest` @ `0x485bf7` decompiles to
  `return !this[2089] && (a3 || …) && get_update_time() - this[2090] >= a2;` — the lock is the
  dword pair `this[2089]` (flag) / `this[2090]` (timestamp).
- `CWvsContext::OnGameStageChanged` @ `0xa0400e` — the known-good unlock on every `SET_FIELD` —
  clears it with exactly:
  `*&this[1].m_Cookie.szCookie[96] = 0; *&this[1].m_Cookie.szCookie[100] = get_update_time();`
- `CWvsContext::OnInventoryOperation` @ `0xa1ead9` opens with the **identical pair of stores**,
  guarded by the packet's leading byte:

  ```c
  if ( CInPacket::Decode1(iPacket) )              /*0xa1eaf4*/
  {
    *&this[1].m_Cookie.szCookie[96]  = 0;         /*0xa1eb02*/
    *&this[1].m_Cookie.szCookie[100] = get_update_time();  /*0xa1eb0d*/
  }
  ```

  That leading byte is `bOnExclRequest`.

Server side, `libs/atlas-packet/inventory/clientbound/change_batch.go:35` writes exactly that byte
as `w.WriteBool(!m.silent)`, and every emitter on the consume path passes `silent = false`:
`services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go:316` (quantity update,
the >1-coupon case) and `:428` / `:506` (remove entry, the last-coupon case).

So a successful `ConsumeItem` on the Cash compartment unlocks the client on its own — the same
mechanism that makes ordinary Use-tab potions work today despite nothing in the consumable path
calling `session.EnableActions`. **Design decision: emit no explicit unlock.** The failure paths
(item-not-found / template mismatch, and `ConsumeError`) send nothing, matching every neighbouring
arm; the client's own `CanSendExclRequest` timeout is what recovers there, exactly as it does for a
failed potion use today.

This also satisfies the "never unlock an outcome that warps" rule vacuously — a morph does not warp.

### 1.3 Existing shapes the implementation must match

- `libs/atlas-packet/cash/serverbound/item_use.go:21-23` — `UpdateTimeFirst(t)` is
  `(GMS && major >= 87) || JMS`.
- `libs/atlas-packet/cash/serverbound/item_use_pet_consumable.go` — an existing sub-body codec that
  is *already* "empty body + trailing updateTime": `if !updateTimeFirst { WriteInt/ReadUint32 }`.
- `services/atlas-channel/.../character_cash_item_use.go:503-511` — the established
  "Classification-FIRST dispatch" block, with a comment explaining that cash-slot type bytes collide
  and classification must win.
- `services/atlas-channel/.../character_cash_item_use.go:923-929` — 530 → type `41` (GMS ≥ 95) /
  `40` (otherwise); `:896-901` gachapon coupon → `40`/`39`; `:937-942` pet evolution → `42`/`41`. The
  collisions the PRD names are real and confirmed.
- `services/atlas-consumables/.../consumable/processor.go:454-479` (`ConsumeStandard`) and
  `:566-604` (`ConsumeCashPetFood`) — the two precedents for a consumer.
- `services/atlas-consumables/.../character/buff/processor.go:17` —
  `Apply(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []stat.Model) model.Operator[uint32]`.
- `services/atlas-consumables/.../character/processor.go:21` —
  `ChangeHP(f field.Model, characterId uint32, amount int16) error`.
- `libs/atlas-constants/item/constants.go:97` — `ClassificationTransformationCoupon = 530`.

### 1.4 One correction to the PRD

PRD §4.3 FR-3.3 cites `ConsumeCashPetFood` as the precedent to follow. It is the right precedent for
*reading cash data* and for the **first** `ConsumeError` call (`processor.go:572` uses
`inventory2.TypeValueCash`), but its later calls at `:592`, `:598`, `:602` pass
`inventory2.TypeValueUse` for what is unambiguously a Cash-compartment item. That looks like a
pre-existing defect in a neighbouring consumer. **This task does not fix it** (different item family,
different acceptance criteria, and no evidence gathered on its live impact), but `ConsumeMorphCoupon`
must use `inventory2.TypeValueCash` on *every* path — which is what FR-3.3/FR-3.4 already require. A
note is filed in §7.

---

## 2. Architecture

Data flows along the existing rails; nothing new is introduced.

```
client ── USE_CASH_ITEM ──▶ atlas-channel
                             CharacterCashItemUseHandleFunc
                               ├─ ownership check (existing)
                               ├─ classification == 530  ── new arm
                               │     └─ decode empty sub-body (trailing updateTime)
                               └─ consumable.RequestItemConsume(field, char, itemId, source, 1, updateTime)
                                        │  (existing Kafka command)
                                        ▼
                          atlas-consumables  RequestItemConsume
                               ├─ inventory2.TypeFromItemId(5300000) → Cash
                               ├─ RequestReserve (existing)
                               └─ ConsumeMorphCoupon  ── new ItemConsumer
                                     ├─ cash.GetById  ────────▶ atlas-data GET /cash-items/{id}
                                     │                            (spec now carries morph + hp)
                                     ├─ ConsumeItem(Cash, txn, slot) ─▶ INVENTORY_OPERATION
                                     │                                   (non-silent → unlocks client)
                                     ├─ character.ChangeHP(field, char, hp)
                                     └─ buff.Apply(field, char, -itemId, 0, time, [MORPH=morphId])
```

### 2.1 Boundary decisions

- **Effect application stays in `atlas-consumables`.** Settled at spec time; restated here because
  it is what makes the consume-then-apply ordering a single unit and keeps `atlas-channel` a pure
  socket boundary.
- **`atlas-channel` forwards, it does not interpret.** The arm decodes and delegates; it never reads
  cash data and never decides what a morph is.
- **`atlas-data` remains the only WZ interpreter.** The two new spec keys are data passthrough.

---

## 3. Component design

### 3.1 `libs/atlas-packet/cash/serverbound/item_use_morph_coupon.go` (new)

```go
type ItemUseMorphCoupon struct {
    updateTime      uint32
    updateTimeFirst bool
}
func NewItemUseMorphCoupon(updateTimeFirst bool) *ItemUseMorphCoupon
// Encode/Decode: if !updateTimeFirst { WriteInt / ReadUint32 } — nothing else.
```

Doc comment carries `// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest` and records
the case-40 arm span `0xa0caf0–0xa0cb37` (no `Encode*`; three client-side predicates, the first
`sub_A0ECCD` @ `0xa0eccd` being literally `itemId / 10000 == 530`; then `play_item_sound(nItemID,
0x29)` @ `0xa0cb30` and the shared send tail).

**Alternatives considered.**

- *(A, chosen)* A dedicated per-type codec, byte-identical in behaviour to `ItemUsePetConsumable`.
  The package convention is one file per case arm — twenty-plus such files exist, several of them
  trivially small — and the call site reads as the wire reads. Cost: ~20 duplicated lines.
- *(B)* Reuse `ItemUsePetConsumable` directly at the new call site. Zero new code, but the type name
  would then be a lie about which client arm it models, and a future divergence on either arm would
  force an unpick under time pressure. Rejected.
- *(C)* Extract a shared `ItemUseTrailingUpdateTime` and alias both. Renaming/retyping an
  already-verified codec churns its tests and the packet-audit surface for a 20-line saving on a
  package whose whole point is one struct per wire shape. Rejected — but if a *third* empty-body arm
  appears, C becomes the right call and should be revisited then.

**Honest note on what this codec buys.** `updateTime` is only logged on the channel side
(`consumable/processor.go:49`) and is not forwarded past `RequestItemConsumeCommandProvider`, and the
socket reader is discarded after the handler returns. So decoding the trailing int32 has no
functional effect today. It is still worth doing: it pins the layout with a test (FR-1.2), keeps the
arm structurally identical to its neighbours, and makes the debug log truthful on v83/v84. The design
does not claim more than that.

### 3.2 `atlas-channel` — the arm

Placed inside the existing "Classification-FIRST dispatch" block (`character_cash_item_use.go:503`),
after the megaphone branch:

```go
if category == item.ClassificationTransformationCoupon {
    sp := cashsb.NewItemUseMorphCoupon(updateTimeFirst)
    sp.Decode(l, ctx)(r, readerOptions)
    if !updateTimeFirst {
        updateTime = sp.UpdateTime()
    }
    _ = consumable.NewProcessor(l, ctx).RequestItemConsume(
        s.Field(), character.Id(s.CharacterId()), itemId, source, 1, updateTime)
    return
}
```

- Gates on classification, never on `it` (FR-1.3). Pre-95, type `40` is also gachapon coupon; on
  ≥95, type `41` is also pet evolution. Neither reaches this arm.
- No `530` literal — `item.ClassificationTransformationCoupon`.
- No region/version literal. `updateTimeFirst` is the pre-existing, IDA-verified split.
- Inherits the ownership check by position (FR-4.2).
- No `EnableActions` (§1.2).

Placement rationale: the classification-first block already exists precisely for type-byte
collisions, and this is one. Putting the arm above it (with the `it ==` arms) would work today — no
current `it ==` arm uses 40 or 41 — but would file the code under the wrong organising principle.

### 3.3 `atlas-data` — cash reader

`cash/rest.go`: `SpecTypeMorph = SpecType("morph")`, `SpecTypeHp = SpecType("hp")`.

`cash/reader.go`, inside the existing `spec` block, next to `expR`/`drpR`/`time`:

```go
if morph := s.GetIntegerWithDefault(string(SpecTypeMorph), 0); morph != 0 {
    m.Spec[SpecTypeMorph] = morph
}
if hp := s.GetIntegerWithDefault(string(SpecTypeHp), 0); hp != 0 {
    m.Spec[SpecTypeHp] = hp
}
```

Omit-when-zero, matching the three neighbours. This is why FR-3.7's "absent or zero" cases collapse
to a single `ok && val > 0` test downstream.

`spec/time` already parses at `reader.go:136-138` — unchanged.

FR-2.4 is structurally satisfied (two additive keys, both gated on non-zero) but is still pinned by a
regression fixture over an existing 0521 EXP-coupon shape, because "structurally satisfied" is not
evidence.

### 3.4 `atlas-consumables` — REST mirror

`cash/rest.go` gains the same two constants. `Extract` copies the whole `Spec` map already, so no
change there.

`cash/rest.go` gains **three** constants, not two: `SpecTypeMorph`, `SpecTypeHp`, **and
`SpecTypeTime`**. The PRD's FR-3.1 names only the first two, but `atlas-consumables`' cash
`SpecType` set has no `SpecTypeTime` at all (unlike atlas-data's, which defines `rate`/`expR`/
`drpR`/`time`), and `ConsumeMorphCoupon` reads the duration from `time` — so FR-3.6 is unreachable
without it. This is a strict superset of FR-3.1, called out explicitly so it is not read as scope
creep. The other atlas-data-only keys (`rate`, `expR`, `drpR`) are **not** mirrored: nothing on this
path consumes them.

### 3.5 `atlas-consumables` — routing and the consumer

Routing, appended to the classification chain in `RequestItemConsume` (`processor.go:283-292`):

```go
} else if item2.GetClassification(itemId) == item2.ClassificationTransformationCoupon {
    itemConsumer = ConsumeMorphCoupon(transactionId, characterId, slot, itemId)
}
```

Deliberately **not** added to `usesStandardConsumer`: `ConsumeStandard` hard-codes
`inventory2.TypeValueUse` (`:471`) and fetches from the *consumable* data resource (`:465`), where
5300000 does not exist.

Consumer, following `ConsumeCashPetFood` for cash-data access and `ConsumeStandard` for
consume-then-apply ordering:

```go
func ConsumeMorphCoupon(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
    return func(l logrus.FieldLogger) func(ctx context.Context) error {
        return func(ctx context.Context) error {
            p := NewProcessor(l, ctx)

            // Independent reads → parallel group, as ConsumeStandard does.
            pg, _ := model.NewGroup(ctx)
            fm := model.Submit(pg, func() (field.Model, error) { return character2.NewProcessor(l, ctx).GetMap(characterId) })
            fi := model.Submit(pg, func() (cash.Model, error) { return cash.NewProcessor(l, ctx).GetById(uint32(itemId)) })
            if err := pg.Wait(); err != nil {
                return p.ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, err)
            }
            f, ci := fm.Get(), fi.Get()

            plan := computeMorphCouponPlan(ci)

            if err := compartment.NewProcessor(l, ctx).ConsumeItem(characterId, inventory2.TypeValueCash, transactionId, slot); err != nil {
                return p.ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, err)
            }

            if plan.hp > 0 {
                _ = character.NewProcessor(l, ctx).ChangeHP(f, characterId, plan.hp)
            }
            if len(plan.statups) > 0 {
                _ = buff.NewProcessor(l, ctx).Apply(f, characterId, -int32(itemId), byte(0), plan.duration, plan.statups)(characterId)
            }
            return nil
        }
    }
}
```

Ordering: every fallible read happens **before** `ConsumeItem`, so a data failure returns the coupon
(FR-3.3). Effect failures after the commit are logged, not rolled back — the `ApplyItemEffects`
convention (PRD §8, failure semantics).

`duration` is `plan.duration`, the raw `time` spec in milliseconds, unscaled (FR-3.6). No `*1000`,
no `/1000`; `tools/buff-duration-guard.sh` will confirm.

### 3.6 Open question 2 — RESOLVED: a small pure planner, not a shared one

```go
type morphCouponPlan struct {
    hp       int16
    statups  []stat.Model  // at most one: MORPH
    duration int32
}

func computeMorphCouponPlan(ci cash.Model) morphCouponPlan
```

- *(A)* Inline application, no planner. Smallest diff, but FR-3.7's four zero/absent permutations
  then require a mocked buff processor, a mocked character processor and a mocked cash processor per
  case. Rejected.
- *(B, chosen)* A dedicated pure planner over `cash.Model`, mirroring `computeEffectPlan`'s stated
  rationale ("keeping the decision pure is what makes the morph/hp paths pinnable by plain unit
  tests", `processor.go:161-163`). ~20 lines, and FR-3.7/FR-3.8 become table tests with no mocks.
- *(C)* Share `computeEffectPlan` via an adapter from `cash.Model` to `consumable3.Model`. Rejected:
  the two spec vocabularies are different types with different key sets; the adapter would be larger
  than the planner it saves, and it would couple a cash-item change to every use-tab consumable's
  regression surface.

The planner emits no morph statup when `morph` is absent/zero and `hp = 0` when `hp` is
absent/zero — independently, per FR-3.7. It does **not** consult `morphRandom` (non-goal; no 0530
item carries one).

FR-3.8 (re-use while morphed) needs no code: the planner and the consumer are stateless, so the
second use issues a second `Apply` unconditionally. The test asserts the absence of a rejection
branch, not the presence of one.

---

## 4. Error handling

| Failure | Behaviour | Coupon |
|---|---|---|
| item not in slot / template mismatch (channel) | existing warn, nothing sent | kept |
| cash data fetch fails | `ConsumeError(…, TypeValueCash, slot, err)` — reservation released | kept |
| character/field fetch fails | same | kept |
| `ConsumeItem` fails | same | kept |
| `ChangeHP` fails after commit | logged, not rolled back | consumed |
| `buff.Apply` fails after commit | logged, not rolled back | consumed |
| `morph` absent/zero | no statup; hp + consume still happen | consumed |
| `hp` absent/zero | no HP change; morph + consume still happen | consumed |
| both absent/zero | consumes, does nothing | consumed |

The client's excl-request lock is released by the inventory operation on every *consumed* row above,
and by the client's own timeout on every *kept* row. No path emits `EnableActions`.

---

## 5. Testing strategy

Unit tests only; no integration harness exists for this path and none is added.

- **`libs/atlas-packet`** — round-trip `ItemUseMorphCoupon` for both tenants: trailing-version tenant
  consumes exactly four bytes and recovers the value; leading-version tenant consumes zero bytes
  (FR-1.2).
- **`atlas-data`** — reader fixtures shaped like `05300000`/`01`/`02` asserting `morph` 1/2/3,
  `hp` 50, `time` 600000 (FR-2.3); a 0521-shaped fixture asserting output identical to the current
  parse (FR-2.4).
- **`atlas-consumables`** —
  - `computeMorphCouponPlan` table test: full spec; `morph` missing; `hp` missing; both missing;
    `time` missing (FR-3.5, FR-3.7).
  - `RequestItemConsume` routing: 5300000 selects `ConsumeMorphCoupon`; asserts it does **not** take
    the `usesStandardConsumer` path (FR-3.2).
  - `ConsumeMorphCoupon` with mocked processors: asserts `ConsumeItem` is called with
    `inventory2.TypeValueCash` (FR-3.4); asserts `Apply` receives `sourceId = -itemId`,
    `duration = 600000`, one statup `MORPH` amount 1 (FR-3.5, FR-3.6); asserts a cash-fetch error
    yields `ConsumeError` and **no** `ConsumeItem` (FR-3.3); asserts two sequential uses issue two
    `Apply` calls (FR-3.8).
  - Cash `RestModel` JSON round-trip carrying `morph`/`hp`/`time` (FR-3.1).
- **`atlas-channel`** — handler test: a `5300000` request reaches the new arm and calls
  `RequestItemConsume`; a `ClassificationGachaponCoupon` id (which shares type byte 40 pre-95) does
  **not** (FR-1.3).

Test setup uses the project Builder pattern; no `*_testhelpers.go`.

---

## 6. Verification

Per CLAUDE.md, from the worktree root:

- `go test -race ./...`, `go vet ./...`, `go build ./...` in `atlas-data`, `atlas-consumables`,
  `atlas-channel`, `libs/atlas-packet`.
- `tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
  `tools/buff-duration-guard.sh` from the repo root.
- No `go.mod` is touched, so `docker buildx bake` is not required. If a plan step ends up touching
  one, the bake becomes mandatory.
- No seed template changes, so the four template guards are not implicated.

---

## 7. Known limitations and follow-ups

1. **Stale ingested data.** The reader change materialises `morph`/`hp` only for newly ingested WZ.
   Tenants ingested before this lands serve a `spec` without them, and the coupon consumes and does
   nothing (the FR-3.7 "both absent" row). The operational re-ingest plus a live
   `GET /cash-items/5300000` check per tenant is a follow-up, per PRD §6/§9.
2. **gms_12.** `template_gms_12_1.json` does not register `CharacterCashItemUseHandle`, so the entire
   cash-item-use family — not just this feature — is inert there. Documented, not fixed.
3. **`ConsumeCashPetFood` compartment-type inconsistency** (§1.4): three `ConsumeError`/`ConsumeItem`
   calls in that neighbouring consumer pass `TypeValueUse` for a Cash-compartment item. Not touched
   by this task; worth a separate look with its own evidence.
4. **`morphRandom` on cash items.** No 0530 item carries one in the inspected corpus; the weighted
   selector from task-140 stays unwired on this path.
