# Water of Life (Pet Revive) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-14

---

## 1. Overview

In MapleStory a pet is a time-limited cash item. When its lifespan runs out the pet does not vanish — it
"dries up" and reverts to an inert doll that still occupies its cash-inventory slot. The **Water of Life**
(item `5180000`, `Item.wz/Cash/0518.img`) is the cash item that reawakens it. The client's own strings state
the contract on both sides:

> "A mysterious water gathered from the deepest valleys of Ellinia. Double click the Water of Life to
> reawaken your pet, which has turned into a doll."
> — `String.wz/Cash.img/5180000/desc` (GMS v83, local extract)

> "It was a cute little kitty, but with the \"Water of Life\" all dried up, it turned back into a doll.
> A little magic can bring it back to life..."
> — `String.wz/Pet.img/5000000/descD` (GMS v83, local extract)

Atlas has none of this. A grep for `water.?of.?life|WaterOfLife` over `services/` and `libs/` returns zero,
and the packet coverage matrix records `WATER_OF_LIFE` as `❌` on every version where it exists
(`docs/packets/audits/STATUS.md:666`). Worse, the precondition for the feature does not hold either:
`atlas-asset-expiration` sweeps every online character once a minute and emits an expire command for **any**
expired asset (`character/processor.go:71-84`), and `atlas-inventory`'s `ExpireAsset` **deletes** it
(`compartment/processor.go:999-1000`). Pets are cash-inventory assets, so an expired pet in Atlas today is
destroyed within sixty seconds. There is no doll to revive.

This task implements the whole feature: it stops pets from being reaped so they persist as dolls, adds the
serverbound codec and channel handler for the `WATER_OF_LIFE` opcode, resolves the target pet and the Water
of Life item server-side (the request carries neither), restores the pet's lifespan from WZ data, consumes
the Water of Life exactly once, and pushes the revived pet to the client without a relog. The item-expiration
extender work (task-222) is the structural precedent throughout — same trust-boundary reasoning, same
saga-driven consume-then-mutate ordering, same asset-update path back to the client.

## 2. Goals

Primary goals:

- A pet whose lifespan expires persists in the cash inventory as a dried-up doll instead of being deleted.
- A player who uses a Water of Life while owning at least one dried-up pet has that pet revived with a fresh
  lifespan, keeping its name, level, closeness and fullness, with the Water of Life consumed exactly once.
- The revived pet's new expiration is visible to the client immediately — the tooltip stops saying "The Water
  of Life has dried up" and the pet is summonable again — without relogging.
- A player who uses a Water of Life with no dried-up pet gets an explanatory message and keeps the item.
- Behavior is identical across every GMS version where the client sends the opcode (v83, v84, v87, v92, v95).

Non-goals:

- Cash-shop *purchase* of `5180000` — the generic cash-shop commodity path already covers acquiring the item.
- Pet evolution (classification 538), pet name tags (517), pet skills (519), pet food/consumables (524/546).
- Reviving or extending anything that is not a pet. Equip/consumable expiration extension is task-222 and
  must not be regressed.
- The other `SendEtcCashItemUseRequest` arms — store permit (type 4), emotion items (type 6), gachapon
  remote (type 57).
- Changing the expiry-and-delete behavior for any asset class other than pets.

## 3. User Stories

- As a player whose pet's ninety days ran out, I want the pet to stay in my cash inventory as a doll rather
  than disappear, so that I have something to revive.
- As a player, I want to double-click a Water of Life and have my dried-up pet come back to life with its
  name, level and closeness intact, so that I do not lose the pet I raised.
- As a player with two dried-up pets, I want a single Water of Life to revive one of them — the one that
  died most recently — and to be able to buy a second Water of Life for the other.
- As a player who double-clicks a Water of Life by mistake with no dried-up pet, I want a message explaining
  why nothing happened and I want to still have my Water of Life.
- As a player, I want the revived pet to be summonable and its tooltip to show the new dry-up date right
  away, without logging out.

## 4. Functional Requirements

### FR-1 — Pets survive their own expiration

- **FR-1.1** `atlas-asset-expiration` MUST NOT emit an expire command for an asset whose template is a pet
  (`item.ClassificationPet`, 500). The exclusion applies to the inventory sweep, the storage sweep and the
  cash-shop sweep, since a dried-up pet may sit in any of the three.
- **FR-1.2** The exclusion MUST be by classification, not by an id allowlist, so that every present and
  future pet template is covered.
- **FR-1.3** No other asset classification changes behavior. Equips, consumables and cash items other than
  pets continue to expire and be removed exactly as they do today.
- **FR-1.4** A pet whose expiration has passed remains a normal asset in every other respect: it occupies its
  slot, moves, and is included in character data. It is "expired" purely by virtue of its expiration
  timestamp being in the past.
- **FR-1.5** A pet whose expiration has passed MUST NOT be summonable. The pet-spawn path MUST reject a spawn
  request for an expired pet rather than summoning a dead pet.

### FR-2 — Serverbound `WATER_OF_LIFE` codec

- **FR-2.1** A new serverbound codec MUST be added to `libs/atlas-packet` for the `WATER_OF_LIFE` operation.
- **FR-2.2** The packet body is **empty**. `CWvsContext::SendWaterOfLife` (v83 IDB `MapleStory_dump.exe`,
  `0xa1dce6`) constructs `COutPacket(0x75)` and sends it with no `Encode` calls at all. The codec MUST decode
  zero fields and MUST NOT invent any.
- **FR-2.3** The codec MUST provide both `Encode` and `Decode`, per the packet playbook, so a round-trip
  fixture can be written.
- **FR-2.4** The op applies to GMS v83, v84, v87, v92 and v95 only. It is `n-a` on gms_12, v48, v61, v72, v79
  and jms_v185 (`docs/packets/audits/support/*.md`), and the codec MUST NOT be routed there.
- **FR-2.5** The opcode is resolved from tenant configuration, never hard-coded. Per registry
  (`docs/packets/registry/gms_v{83,84,87,92,95}.yaml`) the values are:

  | Version | Opcode | fname |
  |---|---|---|
  | gms_v83 | `0x075` (117) | `CWvsContext::SendWaterOfLife` |
  | gms_v84 | `0x075` (117) | `CWvsContext::SendWaterOfLife` |
  | gms_v87 | `0x078` (120) | `CWvsContext::SendWaterOfLife` |
  | gms_v92 | `0x080` (128) | `CWvsContext::SendWaterOfLife` |
  | gms_v95 | `0x081` (129) | `CWvsContext::SendWaterOfLife` |

### FR-3 — Channel handler and template routing

- **FR-3.1** A new handler, `WaterOfLifeHandle`, MUST be added to `atlas-channel`'s socket handler set. It is
  a top-level opcode handler, **not** an arm of `CharacterCashItemUseHandleFunc`. The client reaches it via
  `CWvsContext::SendEtcCashItemUseRequest` (v83 IDB `0xa1dc5b`), which switches on `get_etc_cash_item_type`
  (`0x486845`) and, for classification 518 → type 5, calls `SendWaterOfLife()` — a distinct opcode with no
  body, not a `CASH_ITEM_USE` sub-body. `CharacterCashItemUseHandleFunc` will never observe this item.
- **FR-3.2** The handler MUST be registered in the five applicable seed templates
  (`services/atlas-configurations/seed-data/templates/template_gms_{83,84,87,92,95}_1.json`) with a non-empty
  `validator` (`LoggedInValidator`), the correct per-version `opCode` from FR-2.5, the `fname`
  `CWvsContext::SendWaterOfLife`, and `services: ["channel"]`. A handler entry with a missing or empty
  validator is silently dropped at dispatch-map build time.
- **FR-3.3** Entries MUST be inserted at their sorted `opCode` position, not appended next to a
  semantically-related entry (`tools/template-opcode-order-guard.sh`).
- **FR-3.4** The handler MUST NOT be registered in gms_12, v48, v61, v72, v79 or jms templates.

### FR-4 — Target resolution (the request carries nothing)

Because the packet body is empty, the server derives every operand itself.

- **FR-4.1** The server MUST locate a Water of Life in the requesting character's cash inventory by
  classification (518). If the character holds none, the request is rejected per FR-6 and nothing is
  consumed or mutated.
- **FR-4.2** The server MUST enumerate the character's pets and select the **most-recently-expired** one:
  among pets whose expiration is strictly in the past, the one with the greatest (latest) expiration
  timestamp. Ties are broken deterministically (lowest pet id) so the operation is reproducible.
- **FR-4.3** If the character has no expired pet, the request is rejected per FR-6 and nothing is consumed or
  mutated. A character with only live pets is this case, not an error.
- **FR-4.4** Exactly one pet is revived per Water of Life consumed. Reviving multiple dolls requires multiple
  Waters of Life.
- **FR-4.5** Pet enumeration MUST cover pets wherever a dried-up pet can sit as a result of FR-1 — at minimum
  the character's cash inventory. Storage and the cash-shop locker are out of scope as *sources* for the
  revive target; only pets reachable from the character's own inventory are eligible.

### FR-5 — Revive semantics

- **FR-5.1** The new lifespan MUST be sourced from the Water of Life's own WZ data: `info/life` on the
  consumed item, in **days**. For `5180000` in GMS v83 this is `90`
  (`Item.wz/Cash/0518.img.xml`, `<int name="life" value="90"/>`). The value MUST NOT be hard-coded.
- **FR-5.2** The revived pet's expiration is **set to** `now + life days` — a reset, not an extension. The
  old expiration is in the past by definition, so adding to it would be wrong.
- **FR-5.3** `atlas-pets` is the authority for a pet's expiration. The revive MUST update the pet record
  there.
- **FR-5.4** The `atlas-inventory` cash asset's expiration MUST be updated to the same value. This is not
  bookkeeping: the client decides doll-versus-alive from the **item slot's** `dateExpire`, not from any
  separate pet state. `CUIToolTip::GetPetDeadDate` (v83 IDB `0x8ebfde`) reads the pet item slot and formats
  either `"The Water of Life has dried up"` (string id 678) or `"Water of Life dries up on %d/%d..."`
  (string id 679) from it. A pet revived only in `atlas-pets` would still read as a doll on the client.
- **FR-5.5** The pet's name, level, closeness and fullness MUST be preserved unchanged. The revive restores
  lifespan only.
- **FR-5.6** The pet MUST be left unspawned by the revive. The player summons it themselves afterwards.
- **FR-5.7** The Water of Life MUST be consumed exactly once. A failed or rejected revive consumes nothing.
- **FR-5.8** Consumption and revive MUST be atomic from the player's point of view: no outcome in which the
  Water of Life is gone and the pet is still a doll, or the pet is revived and the Water of Life is still in
  the inventory. This is the same guarantee task-222 obtained by driving consume-then-mutate through
  `atlas-saga-orchestrator` with a compensator that refunds the consumed item by template id on failure.

### FR-6 — Rejection feedback

- **FR-6.1** A rejected request MUST produce a message to the player explaining why. Silent no-ops are not
  acceptable — the client shows no UI of its own for this operation.
- **FR-6.2** The message MUST be delivered via the existing `CharacterStatusMessageWriter` with the
  system-message operation. This is derived, not assumed: the `MESSAGE` dispatcher
  `CWvsContext::OnMessage` (v83 IDB `0xa209d4`) has fourteen arms and **none** of them is water-of-life
  specific; arm `9` is `CWvsContext::OnSystemMessage` (`0xa21a78`), which decodes a single string and appends
  it to the chat log. Atlas already emits exactly this shape
  (`kafka/consumer/system_message/consumer.go:208`,
  `charpkt.CharacterStatusMessageOperationSystemMessageBody`).
- **FR-6.3** Distinct rejection reasons MUST produce distinct messages: no Water of Life held; no dried-up
  pet; and internal failure.
- **FR-6.4** A rejection MUST leave inventory and pet state untouched.

### FR-7 — Client refresh

- **FR-7.1** After a successful revive, the client MUST reflect the new expiration without a relog.
- **FR-7.2** The refresh rides the existing asset-update path: an asset `UPDATED` status event causes
  `atlas-channel` to re-announce the slot with a full add entry
  (`kafka/consumer/asset/consumer.go:289-296`, `invpkt.NewAddEntry`), which replaces the client's item slot —
  and with it the `dateExpire` that FR-5.4 depends on. No new clientbound codec is required.
- **FR-7.3** The consumed Water of Life MUST disappear from the client's cash inventory in the same
  interaction.

### FR-8 — WZ data

- **FR-8.1** `atlas-data`'s cash reader MUST parse `info/life` (days). It currently parses `addTime` and
  `maxDays` only (`services/atlas-data/atlas.com/data/cash/reader.go:79-80`); `life` is read for *pets*
  (`data/pet/reader.go:58`) but not for cash items, so the Water of Life's own `life` is unavailable today.
- **FR-8.2** The parsed value MUST be exposed on the cash REST model so `atlas-inventory` can re-derive it
  (see NFR-2).
- **FR-8.3** A Water of Life whose `life` is absent or zero MUST be treated as a data error: the request is
  rejected per FR-6 and the item is not consumed.

### FR-9 — Constants

- **FR-9.1** `ClassificationWaterOfLife = Classification(518)` MUST be added to
  `libs/atlas-constants/item/constants.go`, alongside the neighbouring `ClassificationPetImprints` (517) and
  `ClassificationPetSkill` (519), and used everywhere in place of the literal.
- **FR-9.2** `GetCashSlotItemType`'s existing bare `if category == 518` branch
  (`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1053`) MUST be updated
  to use the new constant. Its return value (`CashSlotItemType(5)`) is correct and unchanged — it matches the
  client's `get_etc_cash_item_type` type 5 — and it stays in place for classification purposes even though
  the cash-item-use dispatcher is not the path this feature takes.

### FR-10 — Coverage matrix

- **FR-10.1** All five `WATER_OF_LIFE` matrix cells (v83, v84, v87, v92, v95) MUST be promoted from `❌` to
  `✅` by the single-cell verify procedure, with byte fixtures and pinned evidence records, in this change.
- **FR-10.2** The `n-a` cells (gms_12, v48, v61, v72, v79, jms_v185) MUST remain `n-a` and pass the n-a
  consistency gate.

## 5. API Surface

No new externally-facing REST resource is introduced. The changes are to existing service APIs.

**`atlas-data` — cash item data (modified)**

`GET /data/cash/{itemId}` gains one field on the existing JSON:API resource:

```json
{
  "data": {
    "type": "cash",
    "id": "5180000",
    "attributes": {
      "addTime": 0,
      "maxDays": 0,
      "life": 90
    }
  }
}
```

`life` is in **days** and is omitted when absent (`omitempty`), matching the existing `addTime` / `maxDays`
treatment.

**`atlas-pets` — revive (new)**

The revive is driven by a new command on the existing `COMMAND_TOPIC_PET`
(`services/atlas-pets/atlas.com/pets/kafka/message/pet/kafka.go:12`), joining `SPAWN`, `DESPAWN`, `EVOLVE`,
`AWARD_CLOSENESS`, `AWARD_FULLNESS`, `AWARD_LEVEL`, `EXCLUDE`, `SET_SKILL`. The command carries the target
pet, the transaction id, and the absolute new expiration. It emits the existing pet status event on success
and a failure response on rejection so the saga can compensate.

`atlas-pets` also needs a read path that lets the channel enumerate a character's pets with their
expirations; `GET /characters/{characterId}/pets` already exists
(`services/atlas-pets/atlas.com/pets/pet/resource.go:22`) and the model already exposes `Expiration()`
(`pet/model.go:52`), so FR-4.2's selection can be done by the caller without a new endpoint.

**`atlas-inventory` — pet expiration reset (new command)**

The existing `EXTEND_EXPIRATION` command (`kafka/message/compartment/kafka.go:36`) is **not** reusable as-is:
it re-derives its ceiling from the consumed item's `maxDays` and hard-rejects when `maxDays == 0`
(`compartment/processor.go:1131-1134`). The Water of Life has no `maxDays` — only `life`. A sibling command
on `COMMAND_TOPIC_COMPARTMENT` is required, carrying the target slot, the absolute new expiration, and the
consumed Water of Life's template id so the service can re-derive the cap itself.

**`atlas-saga-orchestrator` — revive saga (new step type)**

Mirroring `RequestExtendExpiration`
(`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/processor.go:135`), a new step
requests the pet revive, with the item-consume step ordered first and compensated by a refund on failure.

## 6. Data Model

No new persisted entity is introduced.

**`atlas-pets` `pet` entity** — `Expiration time.Time` already exists (`pet/entity.go:27`) and is already
mutated by the evolution path (`pet/processor.go:921`, `SetExpiration(time.Now().Add(2160 * time.Hour))` —
2160h = 90 days, the same lifespan the WZ `life` field encodes). The revive writes this same column. No
migration.

**`atlas-inventory` asset** — carries both `Expiration` and `PetId` (`asset/model.go:53,95`), and
`IsCash() && petId > 0` already identifies a pet asset (`asset/model.go:127`). This is the join between the
inventory slot and the pet record that FR-4 and FR-5.4 need. No migration.

**Multi-tenancy** — every read and write goes through `tenant.MustFromContext(ctx)`; the WZ `life` lookup is
tenant- and version-scoped like all other `atlas-data` reads, so a tenant whose WZ gives a different `life`
gets that value.

**Data ordering note** — the two expiration records (pet record, inventory asset) are updated as part of one
saga. `atlas-pets` is authoritative (FR-5.3); the inventory asset is the mirror that reaches the client
(FR-5.4). A divergence between them is a bug, and the acceptance criteria test for it explicitly.

## 7. Service Impact

| Service | Change |
|---|---|
| `libs/atlas-constants` | Add `ClassificationWaterOfLife = 518`. |
| `libs/atlas-packet` | New empty-body serverbound codec for `WATER_OF_LIFE`, gated to GMS v83/84/87/92/95. |
| `atlas-channel` | New `WaterOfLifeHandle` socket handler: resolve the Water of Life and the most-recently-expired pet, start the revive saga, or send a system message on rejection. Update `GetCashSlotItemType`'s 518 branch to the new constant. |
| `atlas-configurations` | Register the handler in the five applicable seed templates at the sorted opcode position with `LoggedInValidator`. |
| `atlas-data` | Parse `info/life` in the cash reader; expose it on the cash REST model. |
| `atlas-pets` | New revive command on `COMMAND_TOPIC_PET`: reset expiration, preserve name/level/closeness/fullness, leave unspawned; emit success/failure for the saga. Reject spawning an expired pet (FR-1.5). |
| `atlas-inventory` | New compartment command resetting a pet asset's expiration, re-deriving the cap from the consumed item's WZ `life` (the existing `EXTEND_EXPIRATION` cannot be reused — it requires `maxDays`). |
| `atlas-asset-expiration` | Exclude classification-500 assets from the inventory, storage and cash-shop expiration sweeps. |
| `atlas-saga-orchestrator` | New saga step type for the revive, with a compensator refunding the consumed Water of Life by template id. |
| `docs/packets` | Promote five matrix cells with fixtures and evidence records. |

## 8. Non-Functional Requirements

- **NFR-1 — Multi-tenancy.** Every path resolves tenant from context. The lifespan comes from the tenant's
  own WZ, so two tenants at different versions may legitimately grant different lifespans. No cross-tenant
  read or write.
- **NFR-2 — Trust boundary.** `atlas-channel` is not a trust boundary. The channel computes the new
  expiration, but `atlas-inventory` MUST re-derive the ceiling from the consumed item's own cash data before
  applying it, and reject (not clamp) a request beyond it — the reasoning and precedent are
  `compartment/processor.go:1109-1122`. Rejecting rather than clamping produces a full refund via the saga
  compensator instead of a silent partial grant.
- **NFR-3 — Idempotency.** The revive is keyed by transaction id. A redelivered command MUST NOT revive
  twice or consume twice; Kafka delivery is at-least-once and non-idempotent handlers have duplicated items
  in this codebase before.
- **NFR-4 — Concurrency.** Two Waters of Life used in quick succession MUST NOT both target the same pet.
  Target selection and consumption happen under the compartment lock that `atlas-inventory` already takes
  (`compartment/processor.go:1143-1145`).
- **NFR-5 — Observability.** Log the resolved pet id, the consumed item id, the WZ `life` used, and the
  computed expiration at info level on success; log the rejection reason at warn level on failure. No new
  metrics.
- **NFR-6 — Performance.** The handler is a rare, player-initiated action. The only new recurring cost is a
  classification check per asset in the expiration sweep, which is negligible.
- **NFR-7 — Backwards compatibility.** Pets that were already deleted by the current reap behavior are gone
  and are not recoverable by this task. That is stated, not fixed.

## 9. Open Questions

1. **Pets already reaped.** No migration recovers pets destroyed by the pre-FR-1 behavior. Assumed
   acceptable; flag if any live tenant needs a data repair.
2. **Storage and cash-shop lockers.** FR-1.1 stops pets expiring in all three locations, but FR-4.5 makes
   only inventory pets eligible as revive targets. A pet dried up in storage therefore persists but cannot be
   revived until moved to the inventory. Assumed correct (it matches where the client can see it); confirm
   during design.
3. **The `life` unit for cash items.** `info/life` on `5180000` is `90`, and the pet lifespan constant in
   `atlas-pets` is `2160 * time.Hour` = 90 days, which corroborates **days**. `life` on *pet* templates is
   parsed by `atlas-data` already; design should confirm the two readers agree on the unit before sharing a
   field name.
4. **Spawn rejection scope (FR-1.5).** Whether an already-spawned pet should be force-despawned at the moment
   it expires, or simply not re-summonable, is a design call. The client has no concept of a pet expiring
   mid-summon.
5. **v92/v95 divergence.** The five versions share one `fname` and an empty body, so one codec should cover
   all five. Design must still confirm no version added a field, per the "unverified shared codec" failure
   mode.

## 10. Acceptance Criteria

**Pet persistence**
- [ ] A pet whose expiration passes is not deleted; it remains in its cash-inventory slot indefinitely.
- [ ] An expired equip, consumable, or non-pet cash item still expires and is removed exactly as before.
- [ ] An expired pet cannot be summoned.

**Codec and routing**
- [ ] `WATER_OF_LIFE` decodes an empty body and round-trips on v83, v84, v87, v92 and v95.
- [ ] The handler is registered in all five templates at the correct per-version opcode with a non-empty
      validator, and in none of the `n-a` templates.
- [ ] `tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh` and
      `tools/template-movement-types-guard.sh` are clean.

**Revive**
- [ ] Using a Water of Life with one dried-up pet revives it, consumes exactly one Water of Life, and sets
      both the pet record's and the inventory asset's expiration to `now + 90 days` (v83 data).
- [ ] The revived pet's name, level, closeness and fullness are byte-for-byte unchanged.
- [ ] The revived pet is unspawned and can then be summoned.
- [ ] With two dried-up pets, the one with the **later** past expiration is revived and the other is
      untouched.
- [ ] The new expiration is derived from WZ `info/life`, not a constant — a test with a modified `life`
      produces a correspondingly different expiration.

**Rejection**
- [ ] With no dried-up pet, no item is consumed, no state changes, and a system message is delivered.
- [ ] With no Water of Life held, no state changes and a system message is delivered.
- [ ] A Water of Life whose WZ `life` is zero or absent is rejected without consumption.
- [ ] `atlas-inventory` rejects a forged expiration beyond the cap it re-derives itself, and the saga refunds
      the item.

**Client**
- [ ] The pet's tooltip changes from "The Water of Life has dried up" to a future dry-up date without relog.
- [ ] The consumed Water of Life disappears from the cash inventory in the same interaction.

**Matrix and build**
- [ ] All five `WATER_OF_LIFE` cells are `✅` with byte fixtures and pinned evidence; the `n-a` cells still
      pass the consistency gate.
- [ ] `go test -race ./...` and `go vet ./...` clean in every changed module.
- [ ] `docker buildx bake atlas-<svc>` clean for every service whose `go.mod` was touched.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` and `tools/lint.sh --check` clean from the repo
      root.
