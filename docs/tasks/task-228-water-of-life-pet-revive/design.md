# Water of Life (Pet Revive) — Design

Task: task-228-water-of-life-pet-revive
Status: Draft
Created: 2026-08-14
Input: [`prd.md`](prd.md) (approved)

---

## 0. What this document decides

The PRD fixes *what* the feature does. This document fixes *how*, and closes the five open
questions it left. The load-bearing decisions are:

| # | Decision | Section |
|---|---|---|
| D1 | Pets are exempted at the **emitter** (atlas-asset-expiration), by classification, once for all three sweeps | §2 |
| D2 | An expired pet is **not summonable**, and is **not** force-despawned mid-summon | §3 |
| D3 | One empty-body codec covers all five versions — verified in five IDBs, not assumed | §4 |
| D4 | The handler sends **no** `EnableActions`: this path never arms the client's excl latch | §5 |
| D5 | The revive is a **two-step** saga; the pet↔inventory expiration pair is one transaction, not two saga steps | §7 |
| D6 | The channel sends **no expiration at all** — both downstream services derive it themselves | §8 |
| D7 | Idempotency is a `revive_transaction_id` column on `pets`, which also settles the double-water race | §9 |

---

## 1. Architecture at a glance

```
client double-click 5180000 (CASH tab)
  │  CDraggableItem::OnDoubleClicked  → get_etc_cash_item_type == 5
  │  → CWvsContext::SendEtcCashItemUseRequest → CWvsContext::SendWaterOfLife
  │  opcode 0x75/0x78/0x80/0x81 per version, ZERO body bytes
  ▼
atlas-channel  WaterOfLifeHandle
  │  resolve: does the character hold a classification-518 item?      (REST, atlas-inventory)
  │  resolve: most-recently-expired pet                               (REST, atlas-pets)
  │  reject → CharacterStatusMessageWriter / system message; nothing consumed
  ▼  accept
atlas-saga-orchestrator  saga PetReviveUse
  step 1  destroy_water_of_life   DestroyAsset(templateId, qty 1)     → asset DELETED
  step 2  revive_pet              RevivePet(petId, sourceTemplateId)  → pet REVIVED / REVIVE_FAILED
                                                                        (failure ⇒ compensate step 1: refund)
  ▼
atlas-pets  REVIVE command  (ONE database transaction + outbox)
  ├── read info/life from atlas-data for sourceTemplateId  → expiration = now + life days
  ├── write pets.expiration, pets.revive_transaction_id
  ├── buffer RESET_PET_EXPIRATION on COMMAND_TOPIC_COMPARTMENT   ← the cascade
  └── buffer REVIVED on EVENT_TOPIC_PET_STATUS
  ▼
atlas-inventory  RESET_PET_EXPIRATION
  ├── re-derive the cap from info/life itself; reject beyond it
  ├── resolve the cash asset by petId; asset.ExtendExpiration
  └── asset UPDATED
  ▼
atlas-channel  asset UPDATED consumer → InventoryChangeWriter add-entry → the slot's
                dateExpire is replaced; the tooltip stops saying "dried up"
```

The shape is deliberately the **evolve** shape, not a new one. `pet.Evolve` already mutates a
pet's expiration and cascades an in-place inventory mutation (`CHANGE_TEMPLATE`, keyed by petId)
from inside one transaction's outbox
(`services/atlas-pets/atlas.com/pets/pet/processor.go:881-951`,
`services/atlas-pets/atlas.com/pets/inventory/command.go:16`). Revive reuses that seam verbatim
and changes only which field it writes.

---

## 2. FR-1 — Pets survive their own expiration

### 2.1 Where the exemption goes

**Decision (D1): filter in `atlas-asset-expiration`, at the emitter, by classification.**

The sweep is the single choke point. `character/processor.go:57-125` runs three independent
scans — inventory, storage, cash-shop — and each emits an expire command to a *different*
service's topic (`COMMAND_TOPIC_COMPARTMENT`, storage, cash-shop). Exempting at the consumer
would mean the same rule written three times, in three services, each able to drift. Exempting
at the emitter is one predicate applied at three call sites in one file.

All three REST models already carry what the predicate needs:

- `inventory.AssetRestModel.TemplateId` (`inventory/rest.go`)
- `storage.AssetRestModel.TemplateId` (`storage/rest.go`)
- `cashshop.ItemRestModel.TemplateId` (`cashshop/rest.go`)

so no new fetch, no new endpoint, and no per-asset REST round trip is introduced.

The predicate lives beside `IsExpired` in `expiration/checker.go`, which is already the service's
home for "should this asset be acted on":

```go
// IsReapable reports whether an expired asset may be destroyed. Pets are the
// sole exemption: an expired pet does not vanish, it dries up into a doll that
// keeps its cash-inventory slot until a Water of Life (5180000) revives it.
func IsReapable(templateId uint32) bool {
    return item.GetClassification(item.Id(templateId)) != item.ClassificationPet
}
```

`item.GetClassification` is `libs/atlas-constants/item/constants.go:133` — `floor(id/10000)` —
so the rule is by classification (FR-1.2), not an id allowlist, and every present and future
`5000xxx` pet is covered without further edits. `ClassificationPet = 500` already exists
(`constants.go:71`); nothing new is defined for FR-1.

Each of the three loops becomes `if expiration.IsExpired(...) && expiration.IsReapable(...)`.
Nothing else in the service changes, so FR-1.3 (every other classification behaves exactly as
today) holds by construction rather than by inspection.

### 2.2 Alternatives rejected

- **Exempt inside `atlas-inventory.ExpireAsset`.** Covers only the inventory sweep. A doll in
  storage or the cash-shop locker would still be destroyed, violating FR-1.1.
- **Allowlist of pet template ids.** Explicitly forbidden by FR-1.2, and stale the day a tenant
  ingests a new pet.
- **Emit the command but have consumers ignore it.** Three consumers, three chances to diverge,
  and a Kafka message per doll per minute forever for no effect.

---

## 3. FR-1.5 — An expired pet is not summonable

**Decision (D2): reject at spawn; do not force-despawn a pet that expires mid-summon.**

`pet.ProcessorImpl.Spawn` (`pet/processor.go:422`) gains an expiration guard alongside its
existing `ErrTooManySpawnedPets` / `ErrNeedMultiPetSkill` rejections:

```go
if !pe.Expiration().IsZero() && time.Now().After(pe.Expiration()) {
    return ErrPetExpired
}
```

This is the only gate that matters, because after §2 nothing in Atlas observes the *moment* a
pet expires any more — the once-a-minute sweep that used to notice it is exactly what FR-1
removes. Adding a new expiry watcher purely to force-despawn a running pet would reintroduce the
per-minute pet scan we just deleted, to serve a case the client has no concept of (there is no
clientbound "your pet just dried up" packet, and `CUIToolTip::GetPetDeadDate` re-reads the item
slot only when the tooltip is drawn). **Open question 4 is closed: not re-summonable, never
force-despawned.**

Note for the implementer: `pet.DespawnReasonExpired` (`kafka/message/pet/kafka.go:101`) is
declared and referenced by nothing in the repo. It stays unused — this design does not start
emitting it.

---

## 4. FR-2 — The codec

**Decision (D3): a single empty-body serverbound codec, gated to GMS v83/84/87/92/95.**

Open question 5 asked whether any version added a field. It did not. Verified by decompiling the
sender in each IDB rather than by inference from the shared `fname`:

| Version | IDB | Address | Body |
|---|---|---|---|
| gms_v83 | `MapleStory_dump.exe` | `0xa1dce6` | `COutPacket(0x75)` → `SendPacket`. No `Encode*`. |
| gms_v84 | `GMS_v84.1_U_DEVM` | `0xa68f85` | `COutPacket(117)` → `SendPacket`. No `Encode*`. |
| gms_v87 | `GMSv87_4GB.exe` | `0xab501c` | `COutPacket(0x78)` → `SendPacket`. No `Encode*`. |
| gms_v92 | `GMS_v92_1_DEVM` | `0x9c6f90` | `COutPacket(0x80)` → `SendPacket`. No `Encode*`. |
| gms_v95 | `GMS_v95.0_U_DEVM` | `0x9f28e0` | `COutPacket(129)` → `SendPacket`. No `Encode*`. |

Every opcode matches the checked-in registry (`docs/packets/registry/gms_v{83,84,87,92,95}.yaml`),
including v84's `117`, whose registry entry carries a `discover-ops` correction note — relevant
because of the known v84 clientbound table shift, which does not apply here.

The v84 and v92 sites were **unnamed** in their IDBs at the start of this task; both have been
renamed to `?SendWaterOfLife@CWvsContext@@QAEXXZ` so the next reader finds them by name. Unnamed
was not treated as absent.

The codec therefore has no fields. It still implements both `Encode` and `Decode` (FR-2.3) so the
verify step can write a round-trip fixture, and it is version-gated with the `MajorAtLeast` idiom
rather than a raw comparison. It is **not** routed to gms_12, v48, v61, v72, v79 or jms_v185,
whose matrix cells stay `n-a`.

Implementation of the codec, the nine-template routing sweep, and the promotion of the five
matrix cells follow [`docs/packets/IMPLEMENTING_A_PACKET.md`](../../packets/IMPLEMENTING_A_PACKET.md)
and, per cell, [`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md).
This document does not restate those procedures.

---

## 5. FR-3 — The channel handler

`WaterOfLifeHandle` is a **top-level opcode handler**, registered in the five applicable seed
templates at its sorted opcode position with `LoggedInValidator` and
`fname: CWvsContext::SendWaterOfLife`. It is not an arm of `CharacterCashItemUseHandleFunc`: the
client reaches `SendWaterOfLife` from `SendEtcCashItemUseRequest` (v83 `0xa1dc5b`), a sibling of
`SendCashSlotItemUseRequest` and `SendConsumeCashItemUseRequest`, so the cash-item-use dispatcher
never observes this item.

### 5.1 No `EnableActions` — and the evidence for it

**Decision (D4).** This is the one place where the natural reading of the codebase is wrong, so it
is recorded with evidence rather than left to be rediscovered.

The expiration-extender arm (task-222,
`socket/handler/character_cash_item_use.go:334-432`) carefully calls `enableActions` on every
rejection, because its comment records that `SendConsumeCashItemUseRequest` arms the client's
excl latch. That reasoning does **not** transfer:

- `CWvsContext::SetExclRequestSent` (v83 `0xa0ebbc`) has exactly **one** caller in the whole
  binary: `SendConsumeCashItemUseRequest` at `0xa0ea6f`. Confirmed by xref.
- `CWvsContext::CanSendExclRequest` (v83 `0x485bf7`) is **read-only** — it returns
  `!m_bExclRequestSent && … && get_update_time() - m_tLastExclRequest >= a2`. It sets nothing.
- `SendWaterOfLife` (`0xa1dce6`) constructs the packet and sends it. Nothing else.

So the CASH-tab double-click path gates on `CanSendExclRequest(500)`
(`CDraggableItem::OnDoubleClicked` `0x4efdf7`) but never latches. **No `EnableActions` is
required on any path of this handler, accepted or rejected.** Sending one would be inert rather
than harmful, but it would also be a lie in the code about what the client is doing.

The corollary matters for §9: the client self-throttles this request at 500 ms and nothing more.
A player can legitimately fire two Water-of-Life requests half a second apart.

### 5.2 Handler flow

```
1. resolve the character's cash compartment; find any asset whose classification is 518.
   none  → reject "You do not have a Water of Life."
2. resolve the character's pets (atlas-pets GET /characters/{id}/pets, already exposed;
   the channel's pet.Model already carries Expiration()).
   select: expiration non-zero AND strictly in the past;
           order by expiration DESC, then by pet id ASC.
   none  → reject "You have no pet that has dried up."
3. resolve cash data for the Water of Life's template id; life == 0 or absent
         → reject "The Water of Life has no effect." (FR-8.3 — nothing consumed)
4. create the saga (§7).
```

Step 3 is a *pre-flight* check only. The authoritative derivation happens in atlas-pets and
atlas-inventory (§8); the channel reads `life` solely so that a data error costs the player
nothing (FR-8.3 requires no consumption, and once the saga starts the item is gone).

Step 1 needs only *existence*, not a slot: `saga.DestroyAssetPayload` takes
`{CharacterId, TemplateId, Quantity}` and resolves the slot itself.

Tie-break by lowest pet id (FR-4.2) makes the choice reproducible when two dolls share an
expiration timestamp — which is the norm, not an edge case, for two pets bought in one
transaction.

### 5.3 FR-9 — constants

`ClassificationWaterOfLife = Classification(518)` is added to
`libs/atlas-constants/item/constants.go` between `ClassificationPetImprints` (517) and
`ClassificationPetSkill` (519), and the bare `if category == 518` in `GetCashSlotItemType`
(`character_cash_item_use.go:1053`) is rewritten in terms of it. That function's return value
(`CashSlotItemType(5)`) is correct and unchanged — it mirrors the client's
`get_etc_cash_item_type` type 5 — and stays in place for classification purposes even though the
cash-item-use dispatcher is not this feature's path.

---

## 6. FR-8 — `atlas-data` gains `info/life` for cash items

`cash/reader.go` reads `slotMax`, `protectTime`, `addTime`, `maxDays`, `meso`, `npc` … but not
`life`. One line is added next to `MaxDays`:

```go
m.Life = uint32(i.GetIntegerWithDefault("life", 0))
```

and one field to `cash.RestModel`:

```go
// Life is info/life in DAYS — the lifespan a pet-revival item (Water of Life,
// classification 518) grants. Same node, same unit as pet/reader.go's Life.
Life uint32 `json:"life,omitempty"`
```

**Open question 3 is closed: the unit is days, and the two readers agree.**
`data/pet/reader.go:58` already parses the same `info/life` node into a field of the same name,
and the local GMS 83.1 extract of `Item.wz/Cash/0518.img` contains exactly one item, `05180000`,
whose `info` block is `slotMax=1`, `cash=1`, `life=90` — no `maxDays`, no `addTime`. 90 days is
also what `pet.Evolve` hard-codes today as `2160 * time.Hour`
(`pet/processor.go:921`), corroborating the unit from a second direction.

The absence of `maxDays` on `0518.img` is not incidental: it is precisely why the existing
`EXTEND_EXPIRATION` command cannot be reused (§8.2).

`omitempty` matches the existing `addTime` / `maxDays` treatment, so "absent" and "zero" collapse
into one test downstream, which is what FR-8.3 wants.

---

## 7. The revive path — saga shape

**Decision (D5): two saga steps, not three.**

```
saga PetReviveUse
  step "destroy_water_of_life"  DestroyAsset{CharacterId, TemplateId, Quantity: 1}
  step "revive_pet"             RevivePet{CharacterId, PetId, SourceTemplateId}
```

`DestroyAsset` already exists and already accepts `{EventKindAssetDeleted,
EventKindAssetQuantityChanged}`. `RevivePet` is a new `sharedsaga.Action` in `libs/atlas-saga`,
modelled one-for-one on `EvolvePet`:

| | EvolvePet (existing) | RevivePet (new) |
|---|---|---|
| Action | `evolve_pet` | `revive_pet` |
| Payload | `EvolvePetPayload` | `RevivePetPayload{CharacterId, PetId, SourceTemplateId}` |
| Command | `COMMAND_TOPIC_PET` / `EVOLVE` | `COMMAND_TOPIC_PET` / `REVIVE` |
| Acceptance | `{EventKindPetEvolved}` | `{EventKindPetRevived, EventKindPetReviveFailed}` |

### 7.1 Why the inventory reset is a cascade, not a third step

The obvious three-step saga — consume → reset the pet record → reset the inventory asset — is
**rejected**. Its two mutation steps can half-apply, and a half-application is exactly the bug
the PRD's §6 data-ordering note and FR-5.4 name: a pet alive in `atlas-pets` and still a doll in
the item slot, or the reverse. A saga can only *compensate* that after the fact.

Instead, `atlas-pets` writes its own expiration and buffers the `RESET_PET_EXPIRATION` compartment
command **inside the same `database.ExecuteTransaction` + outbox emit**, which is verbatim what
`Evolve` does with `ChangeTemplate` (`pet/processor.go:926-934`). Either both the row update and
the outbox row commit, or neither does. The pet↔inventory pair becomes atomic at the database
level rather than eventually-consistent at the saga level, and the saga is left responsible only
for the one thing sagas are good at: the cross-service consume/refund pair.

### 7.2 Failure and compensation

`EvolvePet` emits nothing on failure and relies on the saga's flat timeout to roll back. That is
tolerable for evolve; it is not tolerable here, because by the time `revive_pet` runs the player's
Water of Life is already destroyed, and a timeout-length wait for a refund reads as a lost item.

So `REVIVE_FAILED` is a real status event on `EVENT_TOPIC_PET_STATUS`, accepted by the saga as a
terminal failure for the step, which compensates step 1 by refunding the item by template id —
the same compensator shape task-222 uses. The saga timeout stays as the backstop for the case
where atlas-pets never answers at all.

### 7.3 Rejection feedback, sync and async

FR-6 rejections split by where they are detected:

- **Synchronous** (no Water of Life; no dried-up pet; `life` absent or zero) — the handler has the
  session in hand and announces directly:
  `session.Announce(l)(ctx)(wp)(charcb.CharacterStatusMessageWriter)(charpkt.CharacterStatusMessageOperationSystemMessageBody(text))(s)`.
  This is the same call the system-message consumer makes
  (`kafka/consumer/system_message/consumer.go:208`), and the shape is derived, not assumed: the
  `MESSAGE` dispatcher `CWvsContext::OnMessage` (v83 `0xa209d4`) has no water-of-life arm; arm 9,
  `OnSystemMessage` (`0xa21a78`), decodes one string into the chat log.
- **Asynchronous** (`REVIVE_FAILED`) — `atlas-channel`'s existing pet-status consumer
  (`kafka/consumer/pet/consumer.go`) gains a handler that announces the "internal failure" text to
  the owner if present on this channel. The saga does the refund; the channel does the talking.
  This avoids teaching the saga orchestrator about world/channel routing for one message.

All three texts live in one place in the handler package so the sync and async paths cannot
drift, and every rejection leaves inventory and pet state untouched (FR-6.4) because the saga is
never created.

---

## 8. The two downstream commands

### 8.1 `atlas-pets` — `REVIVE`

New command on the existing `COMMAND_TOPIC_PET`, joining `SPAWN` … `SET_SKILL`:

```go
// ReviveCommandBody restores a dried-up pet's lifespan. It carries NO
// expiration: atlas-pets derives it from the consumed item's own WZ data, so a
// forged command cannot dictate a lifespan. SourceTemplateId names the consumed
// Water of Life (classification 518).
type ReviveCommandBody struct {
    SourceTemplateId uint32 `json:"sourceTemplateId"`
}
```

`Command[E]` already carries `TransactionId`, `ActorId` and `PetId`, so the body needs nothing
else.

`ProcessorImpl.Revive`, inside one transaction:

1. `GetById(petId)`; reject if the pet is not owned by `ActorId`.
2. Idempotency / liveness gate — §9.
3. Resolve `info/life` for `SourceTemplateId` from `atlas-data`. Zero or absent → `REVIVE_FAILED`.
   (`atlas-pets` gains a small cash-data client alongside its existing `data/` pet-data client;
   the shape is the same `requests.Provider` pattern.)
4. `expiration = time.Now().Add(life * 24h)` — a **set**, not an add (FR-5.2). Name, level,
   closeness, fullness, slot and flags are untouched (FR-5.5); slot stays whatever it was, so a
   doll (slot `-1`) stays unsummoned (FR-5.6).
5. Persist expiration + `revive_transaction_id`.
6. Buffer `RESET_PET_EXPIRATION` (§8.2) — the cascade.
7. Buffer `REVIVED` on `EVENT_TOPIC_PET_STATUS`, carrying `TransactionId` (the saga keys on it,
   exactly as `EvolvedStatusEventBody` does) and the new expiration.

**Deliberately not reused:** `updateOnEvolve` (`pet/administrator.go:93`) writes template id and
expiration together. Revive writes expiration only, so it gets its own narrow administrator
function rather than passing the pet's current template id back to a function that will rewrite
it.

### 8.2 `atlas-inventory` — `RESET_PET_EXPIRATION`

New command on `COMMAND_TOPIC_COMPARTMENT`:

```go
// ResetPetExpirationCommandBody sets a dried-up pet asset's expiration to an
// absolute instant. The asset is resolved by (CharacterId, PetId) — never by
// slot — mirroring ChangeTemplateCommandBody. SourceTemplateId names the
// consumed Water of Life so this service can re-derive the ceiling itself; the
// caller is not a trust boundary.
type ResetPetExpirationCommandBody struct {
    PetId            uint32    `json:"petId"`
    Expiration       time.Time `json:"expiration"`
    SourceTemplateId uint32    `json:"sourceTemplateId"`
}
```

`ProcessorImpl.ResetPetExpiration` is `ChangeTemplate`
(`compartment/processor.go:2090-2115`) with a different terminal call:

1. Take `LockRegistry().Get(characterId, inventory.TypeValueCash)` — the same lock, which is what
   makes NFR-4 hold without new machinery.
2. Re-derive the cap: `cashProcessor.GetById(sourceTemplateId)`; `life == 0` → reject;
   `serverCap = now + life days`; `expiration.After(serverCap)` → **reject, not clamp**. The
   reasoning is task-222's verbatim (`compartment/processor.go:1109-1122`): rejecting produces a
   full refund through the saga compensator, clamping produces a silent partial grant nobody can
   audit.
3. Walk the cash compartment for `a.IsPet() && a.PetId() == petId` (the existing join,
   `asset/model.go:127`).
4. `assetProcessor.ExtendExpiration(mb)(transactionId, characterId)(a, expiration)`
   (`asset/processor.go:356`) — reused as-is. Its three guards are exactly right here: it refuses
   a locked asset, refuses a permanent (zero-expiration) asset, refuses a backwards move, and
   **no-ops with an `UPDATED` emit when the value is already equal** — which is what makes the
   redelivery path in §9 terminate cleanly.
5. `UPDATED` on the asset status topic → FR-7.

**Why not reuse `EXTEND_EXPIRATION`.** It hard-rejects `maxDays == 0`
(`compartment/processor.go:1131-1134`) and `0518.img` has no `maxDays` node at all (§6). Relaxing
that guard to accept a second cap source would make one command mean two things and weaken the
extender's own ceiling. A sibling command is cheaper and safer than a conditional inside a
security check.

**Why keyed by petId and not slot.** The pet asset is identified by `petId` end-to-end;
`ChangeTemplate` already resolves that way; and it spares the channel a slot lookup it would only
perform in order to hand back a number the service can find itself.

### 8.3 D6 — the trust boundary

**The channel sends no expiration anywhere.** It sends `{petId, sourceTemplateId}` to atlas-pets;
atlas-pets derives the value from WZ and sends the absolute result plus `sourceTemplateId` on to
atlas-inventory; atlas-inventory re-derives the ceiling independently and rejects anything beyond
it. Two independent derivations from the same tenant-scoped WZ node, and a forged channel message
cannot name a lifespan at all.

This is strictly stronger than task-222, where the channel computes the expiration and the service
only bounds it, and it is the reason `RESET_PET_EXPIRATION` carries an absolute instant rather
than a duration: an absolute value makes redelivery a no-op instead of a second grant, the same
property `ExtendExpirationCommandBody` documents for itself.

---

## 9. NFR-3 / NFR-4 — idempotency and the double-water race

**Decision (D7): one nullable `revive_transaction_id` column on `pets`.**

Two hazards, one mechanism:

- **Redelivery.** Kafka is at-least-once and this codebase has duplicated items through
  non-idempotent handlers before.
- **The 500 ms race.** §5.1 established that the client throttles but does not latch, so two
  Water-of-Life requests can be in flight before either revive lands. Both channel-side selections
  would pick the *same* doll.

Naive guards fail one hazard each: "reject if the pet is already alive" refunds correctly for the
race but wrongly refunds a redelivery; "no-op if already alive" is idempotent but silently eats
the second player's item in the race. Distinguishing them needs the transaction id, so it is
stored:

```go
// Entity
ReviveTransactionId *uuid.UUID `gorm:"type:uuid"`
```

The gate in `Revive`, before any mutation:

| State | Action |
|---|---|
| `revive_transaction_id == command.TransactionId` | **Redelivery.** Re-buffer `RESET_PET_EXPIRATION` with the **stored** expiration (not a recomputed one) and re-emit `REVIVED`. No write. atlas-inventory's `ExtendExpiration` sees an equal value and no-ops with an `UPDATED`. |
| expiration in the future, different transaction id | **Live pet.** `REVIVE_FAILED` → saga refunds. The second Water of Life is returned to the player. |
| expiration in the past | **Revive.** Proceed. |

Re-cascading the stored expiration on redelivery (rather than skipping the cascade) is what makes
the pair converge even if the *first* delivery's cascade was itself lost.

The single AutoMigrate column is the whole cost. `Entity` already migrates via
`pet.Migration` (`pet/entity.go:13`), so no hand-written migration is needed.

---

## 10. FR-7 — client refresh

No new clientbound codec. The asset `UPDATED` event emitted by §8.2 step 4 is already consumed by
`atlas-channel` (`kafka/consumer/asset/consumer.go:270-296`), which re-announces the slot with a
full `invpkt.NewAddEntry` — replacing the client's item slot wholesale, and with it the
`dateExpire` the tooltip reads. `CUIToolTip::GetPetDeadDate` (v83 `0x8ebfde`) formats string 678
("The Water of Life has dried up") or 679 ("Water of Life dries up on …") from that slot alone,
which is precisely why FR-5.4 insists the inventory asset be updated and not just the pet record:
a pet revived only in `atlas-pets` would still read as a doll.

The consumed Water of Life disappears through the ordinary `DELETED` path of saga step 1
(FR-7.3).

---

## 11. Open questions — resolutions

| # | PRD question | Resolution |
|---|---|---|
| 1 | Pets already reaped by the current behavior | Not recoverable; no migration. Stated in NFR-7, unchanged. |
| 2 | Storage / cash-shop dolls | Confirmed as written. §2 stops the reap in all three locations, so a doll persists wherever it sits, but only pets in the character's own cash inventory are eligible revive targets — the request carries no target, and the client's pet UI reads the cash inventory. A doll in storage must be moved before it can be revived. |
| 3 | The `life` unit for cash items | **Days.** Same `info/life` node the pet reader already parses, same field name; `05180000` is `life=90` in the local GMS 83.1 extract; `pet.Evolve`'s `2160 * time.Hour` is the same 90 days. §6. |
| 4 | Force-despawn on expiry, or merely not re-summonable | **Not re-summonable; never force-despawned.** §3. |
| 5 | v92/v95 divergence | **No divergence.** All five senders decompiled; every one is `COutPacket(op)` + `SendPacket` with zero `Encode` calls. One codec covers all five. §4. |

---

## 12. Testing strategy

Unit, per service, using the project's Builder pattern (no `*_testhelpers.go`):

- **atlas-asset-expiration** — `IsReapable` over a table of classifications; and each of the three
  sweeps asserted to emit for an expired equip/consumable/non-pet-cash asset and to emit nothing
  for an expired `5000xxx`. This is the FR-1.3 regression guard and it must cover all three
  sweeps, not just inventory.
- **atlas-data** — cash reader over `0518.img` fixture XML yields `Life: 90`; over an `info` block
  with no `life` node yields `0` and the field is omitted from the JSON.
- **atlas-pets** — `Revive` sets expiration to `now + life days` and leaves name/level/closeness/
  fullness/slot byte-identical; `life` sourced from data, so a fixture with a different `life`
  produces a correspondingly different expiration (the FR-5.1 "not a constant" test); the three
  idempotency-gate rows of §9 each asserted, including that the redelivery row re-buffers the
  cascade with the stored expiration; `Spawn` refuses an expired pet.
- **atlas-inventory** — `ResetPetExpiration` resolves by petId; rejects when `life == 0`; rejects
  an expiration beyond the re-derived cap **without** mutating; no-ops idempotently on an equal
  value; takes the cash-compartment lock.
- **atlas-saga-orchestrator** — `RevivePet` appears in the acceptance table (its coverage test
  fails otherwise); `REVIVE_FAILED` drives compensation of the destroy step.
- **atlas-channel** — target selection: two dolls picks the later expiration; equal expirations
  picks the lower pet id; only-live-pets and no-water-held each reject without creating a saga and
  each emit a distinct message.
- **libs/atlas-packet** — round-trip byte fixture per version, with the `packet-audit:verify`
  marker, feeding the five matrix promotions.

Cross-service seams are the known blind spot: a green unit test on a stubbed seam is zero
coverage. The two seams that must be traced into their real consumers are (a) the pet REVIVE
cascade landing as a real `RESET_PET_EXPIRATION` on the compartment topic, and (b) the asset
`UPDATED` reaching the channel's add-entry announce.

---

## 13. Service impact and verification

| Service | Change | `go.mod` touched |
|---|---|---|
| `libs/atlas-constants` | `ClassificationWaterOfLife = 518` | no |
| `libs/atlas-packet` | empty-body `WATER_OF_LIFE` serverbound codec, gated v83/84/87/92/95 | no |
| `libs/atlas-saga` | `RevivePet` action + `RevivePetPayload` + unmarshal arm | no |
| `atlas-channel` | `WaterOfLifeHandle`; pet `REVIVE_FAILED` consumer arm; `GetCashSlotItemType` constant | no |
| `atlas-configurations` | handler in five seed templates at sorted opcode position | no |
| `atlas-data` | cash reader `info/life`; `Life` on the cash REST model | no |
| `atlas-pets` | `REVIVE` command, `REVIVED`/`REVIVE_FAILED` events, cash-data client, spawn gate, `revive_transaction_id` column | possibly |
| `atlas-inventory` | `RESET_PET_EXPIRATION` command + processor | no |
| `atlas-asset-expiration` | `IsReapable`; applied in all three sweeps | no |
| `atlas-saga-orchestrator` | `RevivePet` handler, pet processor request, acceptance-table entry, `REVIVE_FAILED` consumer arm | no |
| `docs/packets` | five matrix cells promoted with fixtures and pinned evidence | — |

Verification is CLAUDE.md's full list, with these specifically load-bearing here:

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module.
- `docker buildx bake atlas-<svc>` for any service whose `go.mod` moved (a shared-lib addition is
  invisible to `go build` against `go.work`).
- `tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh` and
  `tools/template-movement-types-guard.sh` — five templates change.
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check`.
- The packet `n-a` consistency gate, for the six versions that must stay `n-a`.
