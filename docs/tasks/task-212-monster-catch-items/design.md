# Monster Catch Items — Design

Task: task-212-monster-catch-items
Status: Draft
Created: 2026-08-10
Input: [`prd.md`](prd.md) (approved)

---

## 1. Purpose and shape of this document

The PRD settled *what* to build. This document settles *how*: the service
topology and message contracts, the placement of each decision (validate /
resolve / render), the codec work including several coverage-matrix
corrections that the design-phase IDA pass turned up, and the resolution of
the PRD's seven open questions.

Everything asserted about client behaviour below is cited to a decompiled
address in a named IDB. Where a value cannot be derived from the client
(because the client never computes it), the document says so and records the
choice as a numbered assumption with its revisit trigger.

---

## 2. Design-phase IDA findings

Before the architecture, four findings that change the codec scope. All were
produced during this design pass; none were known when the PRD was written.

### F-1 — The per-mob `uniqueId` prefix is universal, not legacy

`CMobPool::OnMobPacket` consumes a leading `Decode4` (the mob object id, fed
to `GetMob`) **before** dispatching to any `CMob::On*` handler, on every
version in the matrix:

| Version | IDB | `OnMobPacket` | `Decode4` first | CATCH_MONSTER case | CATCH_MONSTER_WITH_ITEM case |
|---|---|---|---|---|---|
| gms_v48 | `GMS_v48_1_DEVM.exe` | `0x559390` | yes | `172` = `0x0AC` | `173` = `0x0AD` |
| gms_v61 | `GMS_v61.1_U_DEVM.exe` | `0x5d48f3` | yes | (`0x0BE`, matrix) | (`0x0BF`, matrix) |
| gms_v72 | `GMS_v72.1_U_DEVM.exe` | — | — | (`0x0DF`, matrix) | (`0x0E0`, matrix) |
| gms_v79 | `GMS_v79_1_DEVM.exe` | `0x646d46` | yes | `229` = `0x0E5` | `230` = `0x0E6` |
| gms_v83 | `MapleStory_dump.exe` | `0x67936d` | yes | `0x0FB` | `0x0FC` |
| gms_v84 | `GMS_v84.1_U_DEVM` | (symbol) | yes | `257` = `0x101` | `258` = `0x102` |
| gms_v87 | `GMSv87_4GB.exe` | (symbol) | yes | `0x10B` | `0x10C` |
| gms_v92 | `GMS_v92_1_DEVM.exe` | `0x64a6c0` | yes | `291` = `0x123` | `292` = `0x124` |
| gms_v95 | `GMS_v95.0_U_DEVM.exe` | `0x6570b0` | yes | `299` = `0x12B` | `300` = `0x12C` |
| jms_v185 | `MapleStory_dump_SCY.exe` | `0x6f8732` | yes | `0x10C` | `0x10D` |

(v61 and v72 were confirmed by symbol presence of `CMob::OnEffectByItem` /
`CMob::OnCatchEffect` plus the v61 `OnMobPacket` at `0x5d48f3`; their opcode
columns are taken from the existing registry rather than re-derived, because
this design does not change them.)

Consequences:

- `CatchMonster`'s `legacyMobPoolPrefix` gate (`GMS && !MajorAtLeast(83)`,
  `libs/atlas-packet/monster/clientbound/catch_monster.go:78`) is **wrong for
  v83+**. Its own source comment already suspected this. The prefix must be
  unconditional.
- `CatchMonsterWithItem` has **no `uniqueId` field at all**
  (`catch_monster_with_item.go`) on any version, yet it is dispatched through
  the same `OnMobPacket`. Its ✅ cells on v79/v83/v84/v87/v95/jms are
  round-trip-fixture passes over a layout the client cannot consume — the
  known "✅ via round-trip fixture ≠ client-validated" failure mode. It needs
  the field added on every version.

Neither packet has an emitter today, so correcting them regresses nothing
live. FR-8's "no wire regressions on a version already recorded ✅" is
satisfied in spirit: the recorded ✅ is what is wrong, and the corrected bytes
are what the client actually reads.

### F-2 — v48 `CATCH_MONSTER_WITH_ITEM` is one field shorter

`sub_551481` (v48, dispatched at case `173`) reads **only** `Decode4` and
passes it to `sub_54E82D`, which reads the mob's `x`/`y` and spawns the
effect. There is no trailing result byte. Every other version's
`CMob::OnEffectByItem` reads `Decode4` then `Decode1` — confirmed on v61
(`0x5cc793`), v79 (`0x63c937`), v92 (`0x630c50`), and by the existing codec
comment for v83/v84/v87/v95/jms.

So the layout is: `uniqueId(4)` + `itemId(4)` + `result(1)`, with the
`result` byte **absent on gms_v48 only**.

### F-3 — v48 has no `BRIDLE_MOB_CATCH_FAIL` handler

`CWvsContext::OnPacket` @ `0x70d215` in the v48 IDB is a complete switch over
cases 25–70 with no bridle-fail arm. The four unnamed arms were decompiled and
ruled out: `sub_72025D` (58, a delegating stub), `sub_713202` (62),
`sub_721481` (68, a two-byte + optional-string notice), `sub_7215EA` (70,
same shape). The v48 IDB is symbol-rich (mangled MSVC names throughout
`CWvsContext`), and `SendBridleItemUseRequest` @ `0x70e0c5` *is* named while
`OnBridleMobCatchFail` is absent from the class entirely.

Conclusion: the matrix's ⬜ n-a for `BRIDLE_MOB_CATCH_FAIL` on gms_v48 is
**correct** and stays. v61 has it (`0x8307f3`, named). PRD Open Question 5 is
resolved: v48 sends catch requests but has no server-driven failure notice.
The design must therefore not depend on the fail packet existing everywhere.

### F-4 — `CATCH_MONSTER`'s `result` byte selects between two effect images

`CMob::ShowCatchEffect` (v83 `0x66926e`) forwards the byte to
`CAnimationDisplayer::Effect_Catch` @ `0x438eb6`, which branches on it:
non-zero loads `StringPool` id `3687` under `SP_968_EFFECT_BASICEFFIMG`, zero
loads id `3688` and runs the fade/`Animate` path. It is a two-way selector
between animations, not a numeric result code. The design uses `1` for a
successful capture and never sends `CATCH_MONSTER` on failure (see §6.3).

---

## 3. Matrix and registry corrections

Derived from §2. These are in scope for this task.

| Row | Version | Current | Corrected | Evidence |
|---|---|---|---|---|
| `USE_CATCH_ITEM` | v48 | ⬜ | `0x03F` serverbound | PRD FR-1.2 (`push 3Fh` @ `0x70e2cc`) |
| `USE_CATCH_ITEM` | v61 | ⬜ | `0x04A` | PRD FR-1.2 |
| `USE_CATCH_ITEM` | v72 | ⬜ | `0x050` | PRD FR-1.2 |
| `USE_CATCH_ITEM` | v79 | ⬜ | `0x04F` | PRD FR-1.2 |
| `USE_CATCH_ITEM` | v83–jms | ❌ | ✅ | codec + fixtures (§5) |
| `CATCH_MONSTER` | v48 | ⬜ | `0x0AC`, ✅ | `OnMobPacket` `0x559390` case 172 → `sub_5511F4` (one `Decode1`) |
| `CATCH_MONSTER` | v92 | ❌ @ `0x123` | ✅ | `0x64a6c0` case 291 → `sub_630C30` (one `Decode1`) |
| `CATCH_MONSTER` | v79/v83/v84/v87/v95/jms | ✅ | re-verify after F-1 | prefix now unconditional |
| `CATCH_MONSTER_WITH_ITEM` | v48 | ⬜ | `0x0AD`, ✅ | case 173 → `sub_551481` (`Decode4` only, F-2) |
| `CATCH_MONSTER_WITH_ITEM` | v92 | ⬜ (no opcode) | `0x124`, ✅ | `0x64a6c0` case 292 → `CMob::OnEffectByItem` `0x630c50` |
| `CATCH_MONSTER_WITH_ITEM` | v61/v72/v79 | 🟡ᶠ | ✅ | fixtures while the family is open |
| `CATCH_MONSTER_WITH_ITEM` | all ✅ cells | ✅ | re-verify after F-1 | `uniqueId` added |
| `BRIDLE_MOB_CATCH_FAIL` | v48 | ⬜ | ⬜ (**affirmed**) | F-3 |
| `BRIDLE_MOB_CATCH_FAIL` | v61/v72/v79/v87 | 🟡ᶠ | ✅ | fixtures |

Registry files needing new/updated entries:
`docs/packets/registry/gms_v48.yaml` (three ops), `gms_v61.yaml`,
`gms_v72.yaml`, `gms_v79.yaml` (one op each), `gms_v92.yaml`
(`CATCH_MONSTER_WITH_ITEM` opcode). The n-a consistency gate must pass
afterward; the only surviving ⬜ in the family is v48
`BRIDLE_MOB_CATCH_FAIL`, justified by F-3.

Every promotion goes through the standard single-cell procedure
([`VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md)) —
this document does not restate it.

### SD-3 — Scope decision: the two sibling codecs with the same defect

`MonsterSpecialEffectBySkill` and `IncMobChargeCount` share
`legacyMobPoolPrefix` and are dispatched from the *same* `OnMobPacket` switch
(§2 F-1 tables show both arms on every version), so they carry the identical
missing-prefix defect on v83+. Their cells are currently ✅.

**Recommendation: include them.** The fix is the same one line each, the
evidence is already in hand, and leaving two knowingly-wrong ✅ cells behind is
worse than a slightly wider diff. Concretely: delete `legacyMobPoolPrefix`,
write the `uniqueId` unconditionally in all four codecs, and re-verify the
affected cells. If the reviewer prefers a narrower blast radius, the fallback
is to change only the two catch codecs and open a follow-up carrying the
`0x67936d` / `0x64a6c0` / `0x6570b0` citations — but the follow-up must be
opened in the same PR description, not left implicit.

---

## 4. Architecture

### 4.1 Topology

```
client ──USE_CATCH_ITEM──▶ atlas-channel
                              │  COMMAND_TOPIC_CONSUMABLE
                              │  REQUEST_CATCH_MONSTER {source, itemId, monsterUniqueId}
                              ▼
                         atlas-consumables
                              │  ① item validation (§4.3)
                              │  ② reserve the catch item
                              │  COMMAND_TOPIC_MONSTER  CATCH {characterId, itemId}
                              ▼
                          atlas-monsters
                              │  ③ species / HP / roll (§4.4), fail-closed
                              │  ④ atomic claim + removal (§4.5)
                              │
              ┌───────────────┴────────────────────────┐
              │ EVENT_TOPIC_MONSTER_CATCH              │ EVENT_TOPIC_MONSTER_STATUS
              │ CATCH_RESOLVED {characterId, itemId,   │ CAUGHT {characterId, itemId}
              │                 success, cause}        │ CATCH_FAILED {characterId, itemId, cause}
              ▼                                        │ DESTROYED (existing)
        atlas-consumables                              ▼
        ⑤ commit + grant `create`                 atlas-channel
           (or cancel reservation)                ⑥ effect / fail packets + EnableActions
```

### 4.2 Why the outcome is published twice

The catch outcome has two audiences with incompatible requirements.

*atlas-channel* needs the success effect to arrive **before** the monster's
`DESTROYED` despawn — `CMobPool::OnMobPacket` resolves the mob via `GetMob`
and silently drops the packet if the mob is already gone (§2 F-1). Monster
status events are keyed by `MapId`
(`monster/producer.go:36`), so emitting `CAUGHT` immediately before
`DESTROYED` on that topic makes the ordering a partition guarantee rather
than a hope. The channel already consumes this topic.

*atlas-consumables* must **not** join that topic: it carries a `DAMAGED`
event per hit, and every registered handler json-unmarshals every message.
Subscribing the consumables service to the combat firehose to learn about a
handful of catch attempts is the wrong trade. It gets a dedicated,
low-volume `EVENT_TOPIC_MONSTER_CATCH` instead.

Alternatives considered and rejected:

- **One event on the dedicated topic; suppress `DESTROYED` for catches and
  have the channel send the despawn packet itself.** Ordering-safe and single-
  emission, but it makes monster removal non-uniform — "caught monsters do
  not emit DESTROYED" is exactly the special case that breaks the next
  consumer of that event. Rejected.
- **One event on the status topic; consumables subscribes.** Simplest wiring,
  unacceptable cost (see above). Rejected.
- **atlas-monsters grants the reward directly.** Puts inventory and item
  semantics inside the monster service and duplicates the reservation
  machinery. Rejected.

The residual risk of the double emission is a partial publish: the monster is
gone but `CATCH_RESOLVED` never lands, leaving a dangling reservation. This is
the same exposure the existing reward-box flow carries (`ConsumeReward`'s
create request can fail to publish and dangle its once-handler), so this
design matches the codebase's accepted level rather than inventing a new
compensation mechanism. Emission order is `CATCH_RESOLVED` → `CAUGHT` →
`DESTROYED`, so the player-visible economic outcome is the first thing
attempted after the claim.

### 4.3 atlas-consumables — item validation and reservation

New processor entry point `RequestCatchMonster(f field.Model, characterId,
slot, itemId, monsterUniqueId)`, modelled on `RequestItemReward`
(`consumable/processor.go:1079`).

Pre-reserve checks (FR-3.2), each failing to a `CATCH_FAILED`-equivalent
client response without touching the inventory:

1. `item.GetClassification(itemId)` is the `227xxxx` catch class. Reuse the
   `libs/atlas-constants/item` classification rather than an ad-hoc
   `itemId/10000 == 227` (DOM-21).
2. The consumable data resolves and `Create() != 0`.
3. `useDelay` has elapsed for `(tenant, characterId, itemId)` — see §6.4.
4. `ip.CanAccommodate(characterId, [{Create(), 1}])` — the same merge-aware
   inventory verdict `RequestItemReward` uses. A full inventory fails here,
   before anything is reserved.

Then it registers the one-time reservation handler and calls
`cpp.RequestReserve(...)` exactly as the sibling flows do. The reserved item
is **not** consumed at this point — commit happens only on `CATCH_RESOLVED`
with `success = true`, which satisfies FR-3.9 (no consumption on failure) for
free.

On `CATCH_RESOLVED`:

- `success = true` → `compartment.ConsumeItem(...)` commits the reservation,
  then the `create` item is created for the character. This reuses the
  reward-box grant path (`grantRewardOnCreated` / `grantRewardOnFailed`), so
  a post-reserve creation failure cancels correctly.
- `success = false` → `CancelItemReservation(...)`. The item is untouched.
  No consumable error event is emitted; the channel already renders the
  failure from the monster-status event, and emitting both would send two
  unlock packets.

### 4.4 atlas-monsters — resolution, fail-closed

New `CommandTypeCatch = "CATCH"` on `COMMAND_TOPIC_MONSTER` with

```go
// catchCommandBody asks the processor to remove a monster as a bridle
// (catch-item) capture. Deliberately minimal: every handler on this shared
// topic unmarshals every message, so a field name whose type disagrees with a
// sibling body produces one spurious unmarshal error per message.
type catchCommandBody struct {
    CharacterId uint32 `json:"characterId"`
    ItemId      uint32 `json:"itemId"`
}
```

`characterId` and `itemId` are both `uint32` and both already appear with that
type in sibling bodies (`damageCommandBody.CharacterId`,
`drainMpCommandBody.SkillId`), so FR-3.7's collision hazard is avoided by
construction.

`Processor.Catch(uniqueId, characterId, itemId)` then, in order:

1. `GetMonsterRegistry().GetMonster` — missing or `!Alive()` → silent drop
   (a redelivery whose monster is already caught, or a concurrent kill).
2. Resolve the consumable via a **new** `data/consumable` REST client in
   atlas-monsters (`GET data/consumables/{itemId}` on atlas-data). The
   service already carries two atlas-data clients (`monster/information`,
   `monster/mobskill`), so this follows the established shape. A lookup error
   is a hard drop, not a catch failure — fail-closed, matching `Kill`'s boss
   lookup (`processor.go:1727`).
3. Species gate: `m.MonsterId() == ci.MonsterId()`. Mismatch → failure cause
   `SPECIES_MISMATCH`.
4. HP gate (FR-3.4): pass when `mobHP == 0` or
   `uint64(m.Hp()) * 100 <= uint64(m.MaxHp()) * uint64(mobHP)`. Written as a
   cross-multiplication precisely so integer truncation cannot let a full-HP
   monster through at `mobHP < 100`. Cause `HP_TOO_HIGH`.
5. Probability roll (§6.2). Cause `ROLL_FAILED`.
6. Atomic claim + removal (§4.5). Losing the claim → silent drop, exactly as
   in step 1.
7. Emit `CATCH_RESOLVED(success=true)`, `CAUGHT`, `DESTROYED`.

Any of causes 3–5 emits `CATCH_RESOLVED(success=false, cause)` and
`CATCH_FAILED(cause)` and removes nothing.

Per NFR-Observability, each cause is logged distinctly at debug — the wire
collapses all of them to a single byte (§6.3), so the log is the only place
the distinction survives.

### 4.5 Exactly-once removal

`Registry.RemoveMonster` (`monster/registry.go:547`) is `Get`-then-`Del` and
discards `Del`'s reply, so two concurrent catches can both observe the monster
and both "succeed" — two rewards for one mob. NFR-Race-safety is not met by
the existing primitive.

Fix: add to `libs/atlas-redis`

```go
// RemoveExisting deletes the key and reports whether it existed. Redis DEL is
// atomic and returns the number of keys removed, so exactly one concurrent
// caller can observe true — the primitive callers need when a removal must
// also be an exclusive claim.
func (r *TenantRegistry[K, V]) RemoveExisting(ctx context.Context, t tenant.Model, key K) (bool, error)
```

backed by `client.Del(ctx, rk).Result() == 1`, and a
`Registry.ClaimMonster(ctx, t, uniqueId) (Model, bool, error)` in
atlas-monsters that reads the model, calls `RemoveExisting`, and only on
`true` performs the map-index removal and id release. The catch path is the
only caller; `RemoveMonster` keeps its current signature and callers.

Idempotency (NFR) falls out of the same claim: a redelivered `CATCH` command
finds no monster (step 1) or loses the claim (step 6), and in both cases emits
nothing — no second removal, no second reward.

### 4.6 atlas-channel — handler and rendering

Handler `MonsterCatchItemUseHandleFunc` in `socket/handler/`, shaped like
`character_item_use.go`: decode, debug-log, forward. It performs no
validation and holds no state. The opcode is supplied by tenant configuration
via the template route (DOM-25); no opcode constant appears in Go.

Rendering, on the monster-status events:

- `CAUGHT` → broadcast to the map: `CatchMonster(uniqueId, result=1)` then
  `CatchMonsterWithItem(uniqueId, itemId, result=1)`; then, to the acting
  character's session only, `StatChanged([], true)` (EnableActions).
- `CATCH_FAILED` → to the acting character only:
  `BridleMobCatchFail(reason, itemId, 0)` then EnableActions.

The existing `DESTROYED` handler despawns the mob and evicts the three
channel-side mirrors; the catch path does not duplicate that.

EnableActions is unconditionally correct here: a catch never changes field, so
the "never unlock an outcome that warps" rule does not bite, and
`exclRequestSent` is the client's only duplicate gate for the exclusive
request the catch item sends (FR-2.4).

**v48 has no fail packet** (F-3). The writer is simply not routed in
`template_gms_48_1.json`; the channel attempts the announce, the writer
registry reports it unconfigured, and the handler still sends EnableActions.
The unlock is emitted from its own statement, never chained behind the fail
packet's error, so a missing route cannot wedge the client.

---

## 5. Codec work

### 5.1 New — `UseCatchItem` (serverbound)

`libs/atlas-packet/monster/serverbound/use_catch_item.go`, exporting
`MonsterCatchItemUseHandle` as the handler name for template routing.

```
updateTime      : 4 bytes
slot            : 2 bytes (int16, inventory position)
itemId          : 4 bytes
monsterUniqueId : 4 bytes
```

`Encode` and `Decode` both implemented (FR-1.3). **No version gate** — the
PRD verified the body byte-for-byte on v48/v61/v72/v79/v95 and observed no
divergence; a `MajorAtLeast` gate is introduced only if a remaining IDB proves
one.

Placement rationale: the request targets a field monster and carries its
object id, and `monster/serverbound/` already holds the mob-targeting requests
(`mob_drop_pickup_request.go`, `mob_damage_mob.go`). The matrix path becomes
`monster/serverbound/MonsterUseCatchItem`.

### 5.2 Changed — `CatchMonster`

Drop the `legacyMobPoolPrefix` gate; write `uniqueId` unconditionally. The
v95 two-byte `success` gate (`GMS && MajorAtLeast(95)`) is unchanged and
confirmed still correct: v92's arm (`sub_630C30` @ `0x630c30`) reads a single
`Decode1`.

### 5.3 Changed — `CatchMonsterWithItem`

Add `uniqueId uint32` as the leading field on every version. Gate the trailing
`result` byte off on gms_v48 only (F-2):

```go
// v48CatchByItemNoResult reports whether the tenant omits OnEffectByItem's
// trailing result byte. VERIFIED: v48 sub_551481 @0x551481 reads Decode4 and
// nothing else; every later version reads Decode4 + Decode1 (v61 0x5cc793,
// v79 0x63c937, v92 0x630c50, v83/v84/v87/v95/jms per the codec comment).
func v48CatchByItemNoResult(t tenant.Model) bool {
    return t.IsRegion("GMS") && !t.MajorAtLeast(61)
}
```

`writer/catch_monster_with_item.go` gains the `uniqueId` parameter.

### 5.4 Fixtures

Byte fixtures with `packet-audit:verify` markers for `USE_CATCH_ITEM` on all
ten versions, and for the three clientbound rows on every cell §3 promotes or
re-verifies. The v48 `CatchMonsterWithItem` fixture is the shape-divergence
regression test; the v92 `CatchMonster` fixture pins the single-`Decode1`
layout against the v95 two-byte one.

---

## 6. Resolutions of the PRD's open questions

### 6.1 `mobHP` — percentage of max HP (PRD Q1, carried forward)

Implemented as the cross-multiplied comparison in §4.4 step 4. **Assumption
A-1**, not a derivation: the client never performs this check, so it cannot be
read from any IDB. It rests on the observed value set across the thirteen
items (30, 40, 100, absent), where 100 is meaningful only as "any HP" under a
percentage reading. Revisit first if live testing shows catches succeeding or
failing at the wrong HP boundary.

### 6.2 `bridlePropChg` (PRD Q2)

`bridleProp` is the base success percentage. **Assumption A-2**: when
`bridlePropChg > 0` the effective chance is
`min(100, round(bridleProp * bridlePropChg))` — applied once, statelessly.
For the single item that carries it (`02270002`: 50 / 1.2) that is 60%.

This is a decision, not a derivation: both values are server-side WZ data the
client never reads, so no IDB can settle it. The rejected alternative is a
per-attempt escalation (`prop * chg^attempts`), which would require
per-`(character, monster)` attempt state that nothing else in the codebase
keeps and that no evidence supports. Ignoring the field entirely was also
rejected — it is present in the data and a stateless multiplier uses it
without inventing state. Blast radius is one item.

The roll uses `crypto/rand` via the same helper shape as `rollReward`
(`consumable/reward.go:14`); items without `bridleProp` are deterministic
successes once species and HP pass (FR-3.5).

### 6.3 Response packet selection (PRD Q3) — both always fire

`bridleMsgType` does **not** select between them. Neither
`CMob::OnCatchEffect` nor `CMob::OnEffectByItem` reads it from the wire — both
decode exactly the fields in §5.2/§5.3 — and the PRD's FR-5 already documents
its only use: a purely client-side selector for the local "no monster found"
message when `FindHitMobInRect` comes up empty.

The two packets render different things: `OnCatchEffect` plays the generic
capture animation at the mob position (`Effect_Catch`, F-4), and
`OnEffectByItem` plays the item-keyed effect (`ShowEffectByItem` reads the
mob's `x`/`y` and the item id). A successful catch sends both, in that order.
Neither is sent on failure.

`result` is `1` on both (F-4: a two-way image selector, not a status code).

### 6.4 Internal cause → wire reason (PRD Q4) and `useDelay` (PRD Q7)

The client branches on exactly two values (`CWvsContext::OnBridleMobCatchFail`
@ v95 `0x9d9a80`): `0` → string `0x110E`; `1` → the item's `delayMsg`, falling
back to `0x110F`; anything else renders nothing. So the server emits only 0 or
1.

Reason `1` renders the item's *delay* message ("You cannot use the Fishing Net
yet."). That is only coherent as a not-yet/try-again case, which makes
`useDelay` server-enforced — resolving Q7 in the affirmative. Without
enforcement, reason 1 would be unreachable and `delayMsg` dead data.

Mapping:

| Internal cause | Wire reason | Client shows |
|---|---|---|
| `USE_DELAY` (§4.3 check 3) | `1` | the item's `delayMsg`, else `0x110F` |
| `SPECIES_MISMATCH` | `0` | `0x110E` |
| `HP_TOO_HIGH` | `0` | `0x110E` |
| `ROLL_FAILED` | `0` | `0x110E` |
| `INVENTORY_FULL`, `NO_CREATE_ITEM`, item-not-held | `0` | `0x110E` |
| monster gone / claim lost | *(no packet)* | — |

`useDelay` is enforced in atlas-consumables over a per-`(tenant, characterId,
itemId)` Redis key with TTL `useDelay` milliseconds, set on every attempt
(success or failure) via `libs/atlas-redis`. The client's own 200 ms
`ExclRequest` floor is separate and is not replicated server-side.

The monster-gone / claim-lost path sends nothing but EnableActions: the
request was legitimate and lost a race, and the client should simply unlock.

### 6.5 v48 clientbound handlers (PRD Q5)

Resolved by F-3 and the §3 table: `CATCH_MONSTER` `0x0AC` and
`CATCH_MONSTER_WITH_ITEM` `0x0AD` exist and get implemented + verified;
`BRIDLE_MOB_CATCH_FAIL` genuinely does not exist and its ⬜ is affirmed with
the `0x70d215` citation recorded in the n-a justification.

### 6.6 gms_v12 (PRD Q6)

Out of scope, unchanged. `template_gms_12_1.json` is not a matrix column and
is not routed.

---

## 7. Multi-tenancy

Every hop carries the tenant through `tenant.MustFromContext(ctx)`; Kafka
headers propagate it (`consumer.TenantHeaderParser`, already configured on
both command topics). Consumable data is resolved per tenant at attempt time —
never cached across tenants — because `0227.img` differs by region and
version and the `create` reward ids differ with it. The new atlas-monsters
consumable client must therefore key any cache by tenant, or not cache at all;
given catch attempts are rare, **no cache** is the design's choice.

---

## 8. Template routing

`USE_CATCH_ITEM` gets a `handlers` entry in ten templates
(`template_gms_{48,61,72,79,83,84,87,92,95}_1.json`, `template_jms_185_1.json`)
at the FR-1.2 opcode, `"validator": "LoggedInValidator"`,
`"handler": "MonsterCatchItemUseHandle"`,
`"fname": "CWvsContext::SendBridleItemUseRequest"`, `"services": ["channel"]`,
each inserted at its **sorted** `opCode` position.

`writers` entries for `CatchMonster` and `CatchMonsterWithItem` are added
where §3 introduces a new opcode (v48, and v92 for `CatchMonsterWithItem`);
`BridleMobCatchFail` is added nowhere new and stays absent from v48.

Guards that must pass: `template-opcode-order-guard.sh`,
`template-duplicate-binding-guard.sh`. A handler with an empty validator is
silently dropped at load, so the validator is non-negotiable on every entry.

---

## 9. Service impact summary

| Service / module | Change |
|---|---|
| `libs/atlas-packet` | New `monster/serverbound/UseCatchItem`; `CatchMonster` prefix ungated; `CatchMonsterWithItem` gains `uniqueId` + v48 result gate; fixtures |
| `libs/atlas-redis` | `TenantRegistry.RemoveExisting` (atomic delete-and-claim) |
| `atlas-channel` | Catch handler; `REQUEST_CATCH_MONSTER` command emitter; `CAUGHT` / `CATCH_FAILED` renderers; writer signature update |
| `atlas-consumables` | `RequestCatchMonster`; `useDelay` gate; `EVENT_TOPIC_MONSTER_CATCH` consumer; grant + commit / cancel |
| `atlas-monsters` | `CATCH` command; `Processor.Catch`; `Registry.ClaimMonster`; new `data/consumable` client; two new event emissions |
| `atlas-configurations` | Ten handler routes + the new writer routes |
| `deploy/k8s` | `EVENT_TOPIC_MONSTER_CATCH` in the atlas-monsters, atlas-consumables and atlas-channel configmaps (both overlays) |
| `docs/packets` | Registry entries for v48/v61/v72/v79/v92; matrix promotions and corrections (§3) |
| `atlas-data` | None — every field already parses |

---

## 10. Testing

- **Codec** — round-trip plus byte fixtures per version, with the v48
  `CatchMonsterWithItem` short-body and the v92 vs v95 `CatchMonster`
  divergences each pinned by their own fixture.
- **atlas-monsters** — table tests over the resolution ladder: species
  mismatch, HP at and either side of the `mobHP` boundary (including
  `mobHP = 100` at full HP, the case integer truncation would break), roll
  pass/fail with a seeded roller, absent `bridleProp`, missing monster,
  dead monster, and a redelivered command. Built with the project Builder
  pattern; no `*_testhelpers.go`.
- **Exactly-once** — a concurrent test driving two `Catch` calls at one
  monster through `ClaimMonster` and asserting exactly one `CAUGHT`.
- **atlas-consumables** — validation rejections leave the inventory untouched;
  `CATCH_RESOLVED(false)` cancels the reservation; `CATCH_RESOLVED(true)`
  consumes once and creates the `create` item once; `useDelay` rejects inside
  the window and admits after it.
- **atlas-channel** — handler decode/forward; both renderers, including that
  EnableActions is emitted on every terminal path and that a missing
  `BridleMobCatchFail` route (v48) does not suppress it.
- **Guards** — the full CLAUDE.md build-and-verify list, notably
  `docker buildx bake atlas-channel atlas-consumables atlas-monsters`,
  the two template guards, `redis-key-guard.sh` (the new Redis work lives in
  `libs/atlas-redis`, which is where it must live), and `lint.sh --check`.

---

## 11. Assumptions register

| # | Assumption | Basis | Revisit trigger |
|---|---|---|---|
| A-1 | `mobHP` is a percentage of max HP | value set {30, 40, 100, absent}; client never checks | catches succeed/fail at the wrong HP |
| A-2 | `bridlePropChg` is a one-shot multiplier on `bridleProp` | server-side data, underivable from any client | item `02270002` catch rate feels wrong |
| A-3 | `result = 1` is the "captured" animation | `Effect_Catch` @ `0x438eb6` selects image 3687 (non-zero) vs 3688 (zero) | wrong animation plays on capture |
| A-4 | `useDelay` is server-enforced | otherwise wire reason 1 and `delayMsg` are unreachable dead data | spam-catching is possible, or the delay message never appears |
