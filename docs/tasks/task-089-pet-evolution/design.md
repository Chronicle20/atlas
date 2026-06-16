# Pet Evolution — Design

Status: Approved (architecture decisions confirmed)
Task: task-089-pet-evolution
PRD: `docs/tasks/task-089-pet-evolution/prd.md`

---

## 1. Summary & Confirmed Decisions

Dragon/Robo pets change item identity in two ways: **egg hatching** (automatic, on
summon) and **NPC-driven evolution / re-evolution** (consumes a Rock of Evolution +
mesos, rolls a random adult form). Both are implemented as an **in-place template
swap** of the inventory asset paired with an in-place mutation of the `atlas-pets`
pet row, so the pet record (`id`, `cashId`) and its cash linkage survive.

Three architectural decisions were confirmed before this design (they shape everything
below):

| # | Decision | Choice | Consequence |
|---|----------|--------|-------------|
| A | Who issues the inventory template-swap during evolution | **A1 — atlas-pets cascades** | atlas-pets rolls the outcome, mutates its own row, then emits the inventory `CHANGE_TEMPLATE` command itself (keyed by `petId`). The saga never threads the rolled id between steps. |
| B | Egg-hatch flow | **Direct cascade, no saga** | Hatch consumes nothing and has nothing to compensate, so it runs entirely inside the `SPAWN` path in atlas-pets — same cascade as evolution minus the resource steps. |
| C | Pet-level eligibility gate | **Enumeration filter + atlas-pets re-validate** | The conversation's evolvable-pet enumeration lists only pets meeting `evolReqPetLvl`; atlas-pets re-validates on `EVOLVE` as the authority. No new conversation condition type. |

A direct corollary of A1 + the existing code is that **atlas-channel needs no code
change** (Section 5.6): atlas-pets refreshes a summoned pet's appearance by re-emitting
the existing `DESPAWNED`/`SPAWNED` events, which the channel already consumes with no
version branching. atlas-channel is *verify-only*.

Net service impact (revised from PRD §7):

| Service | Change | Magnitude |
|---|---|---|
| `atlas-data` | Parse `evol*` WZ nodes; expose on pet REST resource | Small, additive |
| `atlas-pets` | `EVOLVE` cmd + `EVOLVED` event; egg hatch-on-spawn; cascade to inventory; appearance refresh | **Largest** |
| `atlas-inventory` | `CHANGE_TEMPLATE` cmd + `updateTemplate` administrator; reuse `UPDATED` event | Medium |
| `atlas-npc-conversations` | `enumerate_evolvable_pets` local op + `evolve_pet` remote op; thin atlas-data pet read; reference Garnox conversation | Medium |
| `atlas-saga-orchestrator` + `libs/atlas-saga` | `EvolvePet` action/payload/handler; reverse-walk compensation for the evolution saga type | Medium |
| `atlas-channel` | **None** — verify despawn/respawn refresh on v83 + v95 | Verify-only |
| `libs/atlas-constants` | Reuse 538 / `5380000`; **no new egg id constants** (data-driven) | None/trivial |

---

## 2. Egg vs. Evolution: the data-driven discriminator

Everything keys off WZ data read through atlas-data. The single most important design
rule is that **services never hard-code egg or adult template ids**; they classify a
pet template by the shape of its evolution data:

- **Egg (auto-hatch on summon):** evolution data present with **no requirements** —
  `evolReqItemID == 0` and `evolReqPetLvl == 0` — and a single deterministic target
  (`evol1`, `evolNo == 1`). Dragon Egg / Robo Egg fall here.
- **Baby / Adult (NPC evolution):** evolution data present **with** an item requirement
  (`evolReqItemID != 0`, normally Rock of Evolution `5380000`) and usually a level
  requirement and **multiple weighted candidates** (`evolNo > 1`).
- **Non-evolvable:** no evolution data. Reads succeed unchanged (FR-1.2).

> ⚠️ **Verification gate (CLAUDE.md "Verify against WZ"):** the exact node values for
> Dragon Egg `5000028`, Baby Dragon `5000029`, Robo Egg `5000047`, Baby Robo `5000048`
> (presence/absence of `evolReqItemID`, `evolReqPetLvl`, `evolNo`, the `evolProb*`
> weights) MUST be confirmed against local WZ data during planning/execution before the
> discriminator above is finalized. If real data contradicts the "no-requirement = egg"
> heuristic, fall back to a small data-resolved egg flag. This design does not cite WZ
> values from memory.

---

## 3. Data: `atlas-data` evolution exposure (FR-1.1, FR-1.3)

### 3.1 WZ parsing
`pet/reader.go` (`Read`, currently lines ~44–76) parses `info/` scalars (`hungry`,
`cash`, `life`) and the nested `interact/` skill list. Extend it to also read, from
`info/`:

- `evol` (flag) — `GetBool("evol", false)`
- `evolNo` (int) — candidate count
- `evol1..evolN` (int) — candidate target template ids, iterated `1..evolNo`
- `evolProb1..evolProbN` (int) — per-candidate weight, paired with `evol{i}`
- `evolReqPetLvl` (int) — `GetIntegerWithDefault("evolReqPetLvl", 0)`
- `evolReqItemID` (int) — `GetIntegerWithDefault("evolReqItemID", 0)`

Use the existing `xml.Node` accessors (`ChildByName`, `GetIntegerWithDefault`,
`GetBool`). Iterate `i := 1; i <= evolNo` reading `evol{i}`/`evolProb{i}`; tolerate
gaps. All fields absent → non-evolvable, no error.

### 3.2 REST resource
The pet `RestModel` (`pet/rest.go`) is a mutable struct (no Builder here — that is the
service's existing convention for the WZ catalog; do not introduce a Builder). Skills
are expressed as a JSON:API to-many relationship. Evolutions have **no identity** and
are read-only, so expose them as **plain attributes**, not a relationship:

```go
type EvolutionRestModel struct {
    TemplateId  uint32 `json:"templateId"`
    Probability uint32 `json:"probability"`
}

// added to pet RestModel:
ReqPetLevel uint32               `json:"reqPetLevel"` // 0 when non-evolvable
ReqItemId   uint32               `json:"reqItemId"`   // 0 when none
Evolutions  []EvolutionRestModel `json:"evolutions"`  // empty when non-evolvable
```

`api2go/jsonapi` marshals a json-tagged nested slice as a normal attribute array — no
`GetReferences`/`SetReferencedStructs` plumbing needed (that machinery stays for
`skills`). No route changes: `GET /api/data/pets` and `/{itemId}` gain the attributes
automatically.

---

## 4. Sequences

### 4.1 Egg hatch (automatic, on summon) — Decision B, direct cascade

```
client double-clicks Dragon Egg
  → atlas-channel emits pet SPAWN command (existing path)
  → atlas-pets  consumer/pet  handleSpawnCommand → Processor.Spawn(petId, actorId, lead)
       1. load pet row (ownership check)                                  [existing]
       2. load atlas-data pet data for pet.templateId (existing data client, now w/ evol)
       3. IF egg-shaped (Section 2):
            a. baby := evol1
            b. IF character already owns `baby` (inventory REST check) → REFUSE:
                 emit player-facing notice, DO NOT spawn, DO NOT swap        (FR-2.2)
            c. mutate pet row IN PLACE: templateId egg→baby; reset stats to
               defaults (level 1, closeness 0, fullness default); preserve
               id, cashId, name?*, slot stays unspawned                      (FR-2.3)
            d. emit inventory CHANGE_TEMPLATE { characterId, petId, newTemplateId: baby }
            e. RETURN without spawning (egg is consumed; player re-summons baby) (FR-2.1)
       4. ELSE: normal spawn (existing slot logic + SPAWNED event)
```

\* Egg→baby is a fresh creature: default stats per FR-2.3. Name handling follows
upstream (baby takes its default catalog name); confirm during planning whether a
hatched baby keeps a player-assigned egg name (upstream: no carried stats, default
name). The asset's expiration is preserved by the in-place swap (Section 5.3).

Inventory emits `UPDATED` (full `AssetData`, new templateId). atlas-pets' asset consumer
ignores `UPDATED` (it only acts on `DELETED`), so the pet row is not destroyed (FR-3.5).
atlas-channel's asset consumer refreshes the cash inventory icon.

### 4.2 NPC evolution / re-evolution — saga + Decision A1 cascade

```
Garnox conversation:
  [eligibility node] → local op enumerate_evolvable_pets → writes context list
        ├─ 0 eligible  → "come back when stronger" branch
        ├─ 1 eligible  → auto-select, store {context.selectedPetId}
        └─ >1 eligible → list-selection menu → store {context.selectedPetId}
  [confirm node] → remote ops, IN ORDER, batched into ONE evolution saga:
        1. destroy_item        (Rock of Evolution 5380000, qty 1)   [resource]
        2. award_mesos         (negative cost)                       [resource]
        3. evolve_pet          ({context.selectedPetId})             [last]
```

Saga executes steps in order. On step 3 (`EvolvePet`) the orchestrator calls
`atlas-pets.EvolveAndEmit(transactionId, petId)`, which:

```
EvolveAndEmit:
  1. load pet (+ ownership)                                            [authority]
  2. load atlas-data evol data for pet.templateId
  3. re-validate: evolvable AND pet.level >= evolReqPetLvl
        └─ fail → return error → saga marks step Failed → COMPENSATE   (FR-5.4)
  4. roll one outcome from evolProb* weights (injectable rng for tests)  (decision #1)
  5. mutate pet row IN PLACE: templateId→rolled; expiration→now+2160h;
     preserve id, cashId, name, level, closeness, fullness, slot, flag, excludes (FR-3.2.4)
  6. emit inventory CHANGE_TEMPLATE { characterId, petId, newTemplateId } (cascade A1)
  7. emit EVOLVED { petId, slot, oldTemplateId, newTemplateId, transactionId }
  8. IF pet currently summoned (slot >= 0): refresh appearance
        → re-emit DESPAWNED then SPAWNED (new templateId) via existing events
          using position from the TemporalRegistry                      (FR-3.4)
```

**Step ordering rationale (correctness-critical):** resources are consumed *before* the
irreversible roll. If `EvolvePet` fails, only the resource steps need undoing — there is
no "un-roll." If a resource step failed first, there is nothing evolved to undo.

### 4.3 Compensation (FR-5.4)

The evolution saga uses **reverse-walk compensation** (the pattern the orchestrator
already implements for `CharacterCreation`): on a failed step, walk completed steps in
reverse and dispatch each one's inverse.

- `destroy_item` (Rock) inverse → award the Rock back.
- `award_mesos` (negative) inverse → award the mesos back.
- `evolve_pet` — only the *final* step; if it fails it produced no committed mutation to
  undo (it returns the error before/at validation, or the in-place mutation+cascade is
  treated atomically inside atlas-pets so a thrown error leaves the row untouched).

Extension point: the compensator currently reverse-walks only when
`SagaType() == CharacterCreation` (`compensator.go`). Add the evolution saga type to the
reverse-walk branch (or generalize the predicate) and ensure the inverse handlers for
the consume/award-mesos actions exist (they are already used by CharacterCreation
rollback). A failed evolution therefore consumes **neither** the Rock nor mesos.

---

## 5. Component designs

### 5.1 `atlas-pets` — command, event, processor

**Command** (`kafka/message/pet/kafka.go`): add `CommandPetEvolve = "EVOLVE"`. The
`Command[E]` envelope already carries `TransactionId`, `ActorId`, `PetId`, `Type`; the
`EvolveCommandBody` is empty (identity is in the envelope). Register
`handleEvolveCommand` in `consumer/pet/consumer.go` mirroring `handleSpawnCommand`.

**Status event** (`kafka/message/pet/kafka.go` + `pet/producer.go`): add
`StatusEventTypeEvolved = "EVOLVED"` with body:

```go
type EvolvedStatusEventBody struct {
    Slot          int8      `json:"slot"`
    OldTemplateId uint32    `json:"oldTemplateId"`
    NewTemplateId uint32    `json:"newTemplateId"`
    TransactionId uuid.UUID `json:"transactionId"`
}
```
plus an `evolvedEventProvider`. `EVOLVED` is the semantic/audit event; the *visual*
refresh rides the existing `DESPAWNED`/`SPAWNED` events (Section 5.6).

**Processor** (`pet/processor.go`): add `EvolveAndEmit(transactionId, petId)` and
`Evolve(mb)` following the `AwardCloseness` pair (load → `Clone(pe).Set…().Build()` →
administrator persist → buffer events). New administrator method
`updateOnEvolve(tx)(petId, newTemplateId, expiration)` updating only `TemplateId` +
`Expiration` columns (mirrors `updateCloseness`). The weighted roll lives behind an
injectable function (processor option, like `WithTransaction`) so tests are
deterministic — **no hard-coded probability table** (NFR data-driven).

**Egg hatch** (`pet/processor.go` `Spawn`): insert the egg branch after the
ownership/load step and before slot assignment, per Section 4.1. Add a `Hatch(mb)` helper
that performs the in-place template+stats reset and the inventory `CHANGE_TEMPLATE`
emission, returning early so the egg never reaches slot assignment.

**Ownership-of-baby check (FR-2.2)** and **baby-owned refusal message:** atlas-pets must
ask atlas-inventory whether the character already holds the hatch-target template. Use an
inventory REST read (add a thin client if none exists) or an existing ownership query.
The refusal surfaces to the player via the existing channel notice / pet
`COMMAND_RESPONSE` path — exact event chosen in planning; no new opcode.

**atlas-data client** (`data/pet`): extend the data `Model` + `Extract` with
`evolutions`, `reqPetLevel`, `reqItemId` so the processor can read weights and
requirements.

### 5.2 Inventory cascade emission (atlas-pets → atlas-inventory)
atlas-pets does not know the inventory cash slot, but it owns `petId`/`cashId`/`ownerId`.
The `CHANGE_TEMPLATE` command is therefore **keyed by `petId`** (Section 5.3). atlas-pets
emits it from both the hatch and evolve paths via the message buffer.

### 5.3 `atlas-inventory` — `CHANGE_TEMPLATE` (FR-4.1–4.4, OQ-3)

Confirmed by exploration: **no in-place template path exists** — must add one.

**Command** (`kafka/message/compartment/kafka.go`): add `CommandChangeTemplate =
"CHANGE_TEMPLATE"`. Body:

```go
type ChangeTemplateCommandBody struct {
    CharacterId   uint32    `json:"characterId"`
    PetId         uint32    `json:"petId"`         // asset selector (cascade key)
    AssetId       uint32    `json:"assetId"`       // optional alt selector
    NewTemplateId uint32    `json:"newTemplateId"`
    TransactionId uuid.UUID `json:"transactionId"`
}
```
Resolve the asset by `(characterId, petId)` → the cash-compartment asset whose `petId`
matches (unchanged across the swap). Register `handleChangeTemplate` in
`consumer/compartment/consumer.go`.

**Administrator** (`asset/administrator.go`): add
`updateTemplate(db, id, newTemplateId)` selecting **only** `TemplateId` (preserve slot,
compartment, `cashId`, `petId`, `expiration`, `quantity`). Model `UpdateEquipmentStats`'
explicit `Select(...)` pattern (which deliberately omits `TemplateId`).

**Processor** (`asset/processor.go`): add `ChangeTemplate(mb)` that loads the asset
(validate exists, owned by character, is a pet asset — `IsPet()`),
`Clone(a).SetTemplateId(new).Build()`, calls `updateTemplate`, and buffers the existing
**`UPDATED`** status event (full `AssetData`, new templateId). Failure surfaces as a
saga-compensatable error, not a silent no-op (FR-4.3).

Because the event is `UPDATED` (never `DELETED`), atlas-pets' asset consumer is inert
(FR-3.5) and atlas-channel's asset consumer refreshes the inventory icon.

### 5.4 `atlas-saga-orchestrator` + `libs/atlas-saga`

- `libs/atlas-saga/model.go`: add `EvolvePet Action = "evolve_pet"`; add an evolution
  saga `Type` (e.g. `PetEvolution`).
- `libs/atlas-saga/payloads.go`: `EvolvePetPayload { TransactionId, CharacterId, PetId }`
  (no target id — the roll is owned by atlas-pets).
- `saga/handler.go` `GetHandler`: add `case EvolvePet: return h.handleEvolvePet, true`;
  `handleEvolvePet` calls `h.petP.EvolveAndEmit(s.TransactionId(), payload.PetId)`.
- `saga/compensator.go`: register the `PetEvolution` saga type for reverse-walk
  compensation (Section 4.3).

### 5.5 `atlas-npc-conversations`

**`enumerate_evolvable_pets` (local op):** call `petP.GetPets(characterId)` for summoned
pets; for each, read evol data via a thin atlas-data pet client (new — reuse the item
client pattern) and keep those that are evolvable **and** `pet.level >= evolReqPetLvl`.
Write the eligible list to conversation context (pet ids + display labels) for the
list-selection state. Drives the 0/1/>1 branch (FR-5.2, OQ-4). Eligibility lives here so
the NPC never offers an ineligible pet (Decision C).

**`evolve_pet` (remote op):** params = pet selector (`{context.selectedPetId}` or slot).
Resolve `petId` (reuse `GetPetIdBySlot`), emit a `PetEvolution` saga step
`(stepId, Pending, EvolvePet, EvolvePetPayload{...})`. The reference conversation lists,
in order, `destroy_item` (Rock `5380000`) → `award_mesos` (negative) → `evolve_pet`, so
all three batch into one saga with `evolve_pet` last (Section 4.2). Resource gating
(`has item`, `has mesos`) uses existing conditions; **no new condition type** (Decision C).

**Reference Garnox conversation (FR-5.5):** one JSON definition: eligibility node
(enumerate) → optional selection menu → confirm → consume + evolve → success message,
with the ineligible branch. Authored data, not code.

### 5.6 `atlas-channel` — verify-only (FR-3.4, FR-5)

No code change. The pet spawn writer (`libs/atlas-packet/pet/clientbound/activated.go`)
encodes `templateId` straight from the `SPAWNED` event with **no version branching**, and
the channel pet consumer already handles `DESPAWNED`/`SPAWNED`. atlas-pets refreshing a
summoned pet via re-emitted despawn+spawn (Section 4.2 step 8) makes the client redraw the
new form for free. **Verification only:** confirm appearance refresh on one v83 and one
v95+ tenant (NFR version coverage). The inventory icon updates via the existing asset
`UPDATED` handler.

### 5.7 `libs/atlas-constants`
Reuse `ClassificationPetEvolution` (538) and Rock of Evolution `5380000`. **No new egg
template-id constants** — egg detection is data-driven (Section 2), satisfying FR-6.1's
"prefer data-driven" without adding ids.

---

## 6. Error handling & idempotency

- **Failed evolution** (ineligible pet, atlas-data unreachable): `EvolvePet` returns an
  error → saga reverse-walk refunds Rock + mesos. No partial state.
- **In-place swap atomicity:** atlas-pets mutates its row and emits `CHANGE_TEMPLATE`
  within one message-buffer transaction; the inventory update is a separate service hop
  but is idempotent (re-applying the same `newTemplateId` is a no-op). If the inventory
  swap is lost, the pet row and asset diverge by `templateId` only — observable and
  re-drivable; log at error.
- **Hatch refusal** (baby owned): no mutation, no swap, player notice. Idempotent.
- **Asset consumer safety (FR-3.5):** guaranteed by event-type choice (`UPDATED`, never
  `DELETED`) — verified by exploration of `consumer/asset/consumer.go`.

---

## 7. Multi-tenancy & version coverage (NFR)

All commands/events/REST/DB writes are tenant-scoped via existing context/header
propagation and GORM tenancy. Evolution data resolves per-tenant through atlas-data
(version/region headers). No `MajorVersion`/`Region` branching is introduced anywhere —
the mechanic is inventory mutation + event-driven respawn, which the channel already
renders version-agnostically. Acceptance verifies on a v83 and a v95+ tenant.

---

## 8. Testing strategy

- **atlas-data:** table test on `Read` with fixtures: egg (no requirements, single
  target), baby (multi-candidate + Rock requirement), non-evolvable (all absent →
  success). REST marshal test for the new attributes.
- **atlas-pets:** `Evolve` with an **injected deterministic roll** asserting outcome,
  in-place preservation (`id`/`cashId`/stats), expiration reset, `CHANGE_TEMPLATE` +
  `EVOLVED` emitted, and (summoned) despawn+spawn refresh. Eligibility rejection test
  (level < req → error, no emissions). Egg-hatch test incl. baby-owned refusal.
- **atlas-inventory:** `ChangeTemplate` preserves slot/`cashId`/`petId`/expiration,
  emits `UPDATED`; rejects non-owned / non-pet assets.
- **saga-orchestrator:** `handleEvolvePet` dispatch; reverse-walk compensation refunds
  Rock + mesos on `EvolvePet` failure.
- **npc-conversations:** `enumerate_evolvable_pets` 0/1/>1 branching; `evolve_pet` emits
  the ordered three-step saga.
- Full gate per CLAUDE.md: `go test -race`, `go vet`, `go build`,
  `docker buildx bake atlas-<svc>` for every changed module, `tools/redis-key-guard.sh`.

---

## 9. Out of scope (per PRD §2)

Item-use evolution; player-NPC spawning; breeding; full production Garnox script set
(one reference definition only); closeness/level/hunger mechanic changes; the Rock's
`CashSlotItemType` UI-tab categorization in atlas-channel.

---

## 10. Open items carried to planning

1. **WZ verification gate (Section 2)** — confirm real `evol*` node shape for
   `5000028/5000029/5000047/5000048` against local WZ; validate the egg discriminator.
2. **Hatched-baby name** — confirm upstream behavior (default catalog name vs. carried
   egg name) when reading WZ/Cosmic reference.
3. **Baby-owned ownership check + refusal message path** — pick the exact
   atlas-inventory read and the player-notice event (no new opcode).
4. **Compensator generalization** — confirm the existing consume/award-mesos inverse
   handlers are reusable for the `PetEvolution` reverse-walk, or add them.
5. **npc-conversations atlas-data pet client** — reuse existing data-client pattern vs.
   new thin client for the evol fields.

---

## 11. Acceptance criteria mapping (PRD §10)

| PRD acceptance item | Design section |
|---|---|
| atlas-data parses evol nodes; non-evolvable still reads | §3 |
| Egg hatch in place, refused if baby owned | §4.1, §5.1 |
| NPC evolves eligible pet; level gate; Rock+mesos; data-weighted roll in atlas-pets | §4.2, §5.1, §5.5 |
| Pet keeps id/cashId/stats; templateId rolled; expiration reset 90d | §4.2 step 5, §5.1 |
| Asset templateId swapped in place; pet not deleted by asset consumer | §5.3, §6 |
| Summoned pet visually becomes new form (v83 + v95) | §4.2 step 8, §5.6 |
| Re-evolving an adult re-rolls via same path | §2, §4.2 (data-driven) |
| Failed evolution refunds Rock + mesos | §4.3, §6 |
| >1 evolvable → menu; exactly one → auto-select | §4.2, §5.5 |
| EVOLVE/EVOLVED + inventory swap cmd/event exist | §5.1, §5.3 |
| No hard-coded probability tables; data-driven | §5.1, §5.7 |
| Build/test/vet/bake/redis-guard clean | §8 |
