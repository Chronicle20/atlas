# Design — Pet Evolution as Quest-Driven (supersedes the Garnox free-conversation approach)

Status: Draft
Created: 2026-06-15
Supersedes: `design.md` (free NPC conversation) and `design-multi-pet-chooser.md` (pickFromContext)
for the **trigger/UX layer only**. The evolution *engine* designed in those docs is retained.

---

## 1. Why we are pivoting

Task-089 shipped pet evolution as a **synthetic free-form NPC conversation** ("I am Garnox,
keeper of the Rock of Time…") with a custom `pickFromContext` multi-pet chooser, bound to an NPC.
Live testing surfaced that this does not match how the game actually drives evolution:

- The conversation was bound to NPC **1032102 ("Mar the Fairy")**; the real evolution NPC is
  **9102001 ("Garnox the Pet Scientist")** (verified against `String.wz/Npc.img`).
- Even on the right NPC, a free conversation is the wrong **mechanism**. Canonically, pet
  evolution is **quest-driven**: four quests at NPC 9102001 whose completion runs an
  `endscript` that performs the evolution. The free conversation bypasses the quest gates
  entirely (pet identity, tameness, quest-chain progression).
- The free-conversation path also collides with the channel NPC-click handler
  (`socket/handler/npc_start_conversation.go`), which routes a clicked NPC to *shop → generic
  conversation*, not the quest path.

**Decision:** drive evolution through the **existing quest-conversation (startscript/endscript)
mechanism**, reusing the evolution engine; back out the synthetic conversation.

---

## 2. Canonical mechanism (ground truth from WZ)

For each quest, `Quest.wz` carries four parallel trees, all keyed by quest id:

| WZ file | Role | For the pet quests |
|---|---|---|
| `QuestInfo.img` | Metadata (name, area) | names below |
| `Check.img` | Start (`0`) / complete (`1`) **requirements** + `startscript`/`endscript` names + `npc` | pet id, `pettamenessmin`, item, prereq quest |
| `Act.img` | Start/complete **rewards/effects** | **empty** for the script quests; populated for 8184 |
| `Say.img` | Partial/fallback **dialogue** (`0`=start, `1`=complete; `yes`/`no`/`stop`/`ask`) | **incomplete** for these script quests — fragments only |

Key facts established from the v83 WZ:

- **`Say.img` carries only fragments for script quests; it is NOT the authoritative dialogue
  source.** It does hold some text — e.g. `8185/0/0` = *"I can see that you truly care for your
  Dragon. Now, shall I help you with your Baby Dragon's evolution?"* — but research shows the full
  conversation/flow lives in the Cosmic **`q{id}s.js`/`q{id}e.js`** scripts; `Say.img` is a
  fallback the script may or may not use. **Dialogue is sourced from the Cosmic `.js`** (§4.2), with
  `Say.img` only as a secondary cross-reference for version-appropriate phrasing.
- **`startscript`/`endscript` are just names** (`q8185s`, `q8185e`). They flag the quest as
  script-driven so the client emits the script action; **atlas keys everything by `questId`, not
  by the script name** — there is no name→handler table to build.
- **`Act` is empty** for the rock quests because the evolution is the *script's* job, not a quest
  reward.

### The four quests (NPC 9102001, all `pettamenessmin = 1642`)

| Quest | Name | Kind | Start req | Complete req | Effect |
|---|---|---|---|---|---|
| **8184** | Pet's Evolution1 | **Simple item turn-in** (non-script, has Act) | pet 5000029 (baby dragon), tameness 1642 | + 50× `4000029`, 50× `4000023` | Act consumes the 2 items, awards 10× `2120000` (pet food). Unlocks 8185. |
| **8185** | Pet's Evolution2 | **Script** (`endscript=q8185e`) | quest 8184 complete, pet 5000029, tameness 1642 | + Rock `5380000` | Evolve baby dragon → random adult (5000030–33) |
| **8189** | Pet's Re-Evolution | **Script** | pet ∈ adults {5000030-33, 5000049-52}, tameness 1642 | + Rock `5380000` | Re-roll adult → different adult, same family |
| **4659** | Robo Upgrade! | **Script** (`endscript=q4659e`) | pet 5000048 (baby robo), tameness 1642 | + Rock `5380000`, 50× `4000111` | Evolve baby robo → random adult (5000049–53) |

Note the gate is **tameness ≥ 1642**, *not* pet level 15 — the original conversation used the
wrong gate (`evolReqPetLvl` from the pet WZ). Tameness = closeness.

---

## 3. Existing atlas machinery we reuse (and a caveat)

The quest-conversation subsystem already implements the startscript/endscript model end-to-end:

- **Data model** (`conversation/quest`): keyed by `questId`, with a required `startStateMachine`
  (acceptance) and optional `endStateMachine` (completion) — the start/end-script split.
  Seeded at `deploy/seed/<region>/<ver>/npc-conversations/quests/quest-{id}.json`
  (~219 already present per GMS version).
- **Trigger path**: client `QuestActionScriptStart`(4)/`QuestActionScriptEnd`(5) packet →
  channel `socket/handler/quest_action.go` → `StartQuestConversation(questId,npcId)` Kafka command
  (`COMMAND_TOPIC_QUEST_CONVERSATION`) → npc-conversations consumer →
  `GetStateMachineForCharacter(questId, characterId)` reads quest status from atlas-quest and
  routes NOT_STARTED→start machine, STARTED→end machine.
- **Operations** already available to a state machine include `start_quest`, `complete_quest`,
  `destroy_item`, **`evolve_pet`**, **`enumerate_evolvable_pets`** (the two we added stay).
- **Evolution engine** (unchanged, reused as the *body* of the end machine): atlas-pets `EVOLVE`
  (weighted roll + in-place mutate) + `EVOLVED` event, atlas-saga `PetEvolution` (reverse-walk
  refund), atlas-inventory `CHANGE_TEMPLATE` (in-place asset swap), atlas-data `evol*` parsing,
  and the egg-reader fix (`reader.go` tolerating no `interact` node).

> **Caveat (per the user): the ~219 inherited quest conversations were converted from Cosmic and
> are NOT a verified correctness oracle.** Use them only as a **structural/mechanical template**
> (JSON shape, how dialogue and operation states wire together). For *content* faithfulness, source
> from authoritative inputs and verify: the **Cosmic `q{id}s.js`/`q{id}e.js` scripts** (the
> **primary** source for both dialogue/flow **and** logic — present for 8185/8189/4659, absent for
> 8184), **Check.img/Act.img** (requirements/rewards), with **`Say.img` only as a secondary
> cross-reference** (it is incomplete for these script quests — §2). This is the project's
> "Verification Over Memory" rule applied to quest data.

---

## 4. Design

### 4.1 atlas-quest — enforce `pet` and `pettamenessmin` (the only code gap)

atlas-quest **parses** `Pet []uint32` and `PetTamenessMin int16` (`data/quest/rest.go:62-63`) but
**never enforces** them — they are absent from `buildStartConditions`/`buildEndConditions`
(`data/validation/processor.go`), and `validation/model.go:16-25` has no corresponding condition
constants. Add one **general** composite condition type (pet evolution is just the first consumer).

**The data pipeline already exists (verified) — this is contained, not a cross-service build:**

- query-aggregator already fetches the **full per-pet list** from atlas-pets via
  `pet.Processor.GetPets(characterId) → []Model` (`pet/processor.go:32`); each pet model carries
  **`Closeness uint16`, `TemplateId`, `Level`, `Slot`** (`pet/rest.go:12-19`). atlas-pets owns
  closeness (`CLOSENESS_CHANGED` events, `AwardCloseness`).
- `GetSpawnedPetCount` (`pet/processor.go:44`) **already iterates that list** and reduces it to a
  count; the `ValidationContext` keeps only `petCount` (`validation/context.go:37,697`;
  `processor.go:238`). So closeness/templateId is fetched today and then **discarded** — no new
  atlas-pets plumbing is needed.

Work (no new service integration):
1. **Retain per-pet detail in `ValidationContext`** — store the spawned-pet list (templateId +
   closeness + slot) instead of only the count, at the existing fetch site.
2. **Add `PetTamenessCondition = "petTameness"`** in `libs/atlas-saga` (the shared `sharedsaga`
   constants), the query-aggregator eval switch (`validation/model.go`), and atlas-quest's
   `validation/model.go`. Evaluate as: *a **spawned** pet exists whose `templateId ∈ Values` and
   `closeness ≥ Value`*. Reuse `ConditionInput.Values` for the pet-id set and `Value` for min
   closeness — this cleanly handles 8189's multi-id adult set (OR over ids) inside one AND-conjunct.
3. **atlas-quest `buildStartConditions`/`buildEndConditions`**: emit one `petTameness` condition
   from `req.Pet` (id set) + `req.PetTamenessMin` (these quests gate on both start and complete).
4. Tests in both services.

**Why one composite condition, not separate `pet` + `petTameness`:** the gate is "own a *summoned*
pet that is one of {ids} **and** has tameness ≥ N" — tameness binds to the *same* pet, and the id
requirement is an OR over the set. Two independent AND-conjuncts would wrongly pass if you had
pet-A (right id, low tameness) + pet-B (wrong id, high tameness). **Confirm** Cosmic checks the
*summoned* pet (slot ≥ 0), not merely owned.

### 4.2 Author the quest conversations (data)

Author `quest-{id}.json` for **8185, 8189, 4659** (8184 needs none — see 4.4). Each:

- **`startStateMachine`** — dialogue + branch structure from the Cosmic **`q{id}s.js`** start
  script. On accept, a `start_quest` operation transitions the quest to STARTED in atlas-quest.
  (For these quests the start machine is mostly dialogue; the real requirement gating is enforced
  by atlas-quest 4.1.)
- **`endStateMachine`** — dialogue + branch structure from the Cosmic **`q{id}e.js`** end script,
  then the operation chain that *is* the endscript:
  1. `destroy_item` Rock `5380000` (and for 4659, `destroy_item` 50× `4000111`),
  2. `evolve_pet` targeting the qualifying pet (see 4.3),
  3. `complete_quest` → atlas-quest COMPLETED.
  All three are saga-backed; the `PetEvolution` saga already compensates (refund) on failure.

**Dialogue source = Cosmic `.js`, not `Say.img`.** Research shows `Say.img` is **incomplete** for
these script quests (the canonical script supplies the real conversation text and flow; `Say.img`
holds only fragments / a fallback) — so the Cosmic `q{id}s.js` / `q{id}e.js` scripts are the
**primary** dialogue and logic source. Use `Say.img`/`String.wz` only as a secondary cross-reference
for version-appropriate phrasing of lines the script reuses. Do **not** copy from the inherited
atlas conversions (structure-only template). Cosmic has scripts for **8185/8189/4659**; if a start
script (`q{id}s.js`) is absent for a given quest, the start machine is a minimal accept dialogue.

Seed across all supported GMS versions (`12_1/83_1/84_1/87_1/92_1/95_1`), matching how the existing
quest conversations are distributed. Confirm each version's WZ actually defines the quest +
Say.img before seeding it there (versions differ).

### 4.3 Target-pet resolution — drop the multi-pet chooser

In the quest model the **quest Check already constrains the pet** (specific id + tameness), so a
"which pet?" chooser is moot and non-canonical. The end machine evolves the **qualifying summoned
pet**:

- Reuse `enumerate_evolvable_pets` to resolve the summoned pet that matches the quest's required
  template, store its id in context, and feed `evolve_pet petId={context...}`.
- **Remove `pickFromContext` entirely** (it existed only for the chooser).

**OPEN:** exact resolution when a player has multiple qualifying pets summoned — pick the single
required template per quest; confirm Cosmic picks the summoned pet.

### 4.4 Backout list (trigger/UX layer only)

Remove:
- `deploy/seed/gms/*/npc-conversations/npc/npc-9102001.json` (all 6 versions).
- `pickFromContext` **backend**: the state type/const, `PickFromContextModel` + builder, REST
  transform/extract, processor `processPickFromContextState` + `Continue` case + helpers, and the
  associated tests.
- `pickFromContext` **frontend**: revert the `conversation.ts` type, `stateMeta.ts`,
  `transitions.ts` additions (the UI crash fix becomes unnecessary once the type is gone).
- The `design-multi-pet-chooser.md` / `plan-multi-pet-chooser.md` chooser scope (mark superseded).

8184 needs **no conversation** — it is a standard item turn-in; once 4.1 enforces pet/tameness it
works through normal quest mechanics.

### 4.5 Retained (do not touch)

- Evolution engine: atlas-pets `EVOLVE`/`EVOLVED`, atlas-saga `PetEvolution`, atlas-inventory
  `CHANGE_TEMPLATE`, atlas-data `evol*` parsing.
- The **egg-reader fix** (`atlas-data/pet/reader.go` — interact optional) — needed for hatch and
  already merged on the branch; independent of this pivot.
- The `evolve_pet` and `enumerate_evolvable_pets` operations (now consumed by the end machine).

---

## 5. Open questions / risks

1. ~~query-aggregator pet/closeness support~~ — **RESOLVED** (verified 2026-06-15): query-aggregator
   already fetches per-pet `closeness` + `templateId` from atlas-pets (`pet/processor.go:32`,
   `pet/rest.go:12-19`) and just discards all but the count. No new atlas-pets plumbing — enforcing
   `pettamenessmin` only needs the ValidationContext to retain the per-pet list + the new
   `petTameness` condition (4.1). Contained to query-aggregator + atlas-quest.
2. **Tameness vs level** — confirmed 1642 closeness is the gate (per WZ); atlas-pets tracks closeness
   (`CLOSENESS_CHANGED`). Remaining: confirm the check targets the *summoned* pet (slot ≥ 0).
3. **Quest availability/offer** — how the player initiates: does Garnox auto-offer the quest when
   conditions are met, and does clicking him emit `QuestActionScriptStart`? Verify the channel
   path actually fires for a quest NPC (distinct from the shop/generic path we saw).
4. **Faithfulness** — dialogue **and** logic from the Cosmic `q{id}s.js`/`q{id}e.js` scripts
   (primary) for 8185/8189/4659 (none for 8184); `Say.img` is incomplete and only a secondary
   cross-reference; inherited atlas conversions are structure-only references. **Need:** locate the
   Cosmic script files for these quests (the actual `.js` text) during planning.
5. **Per-version WZ drift** — confirm each GMS version defines these quests + Say.img before
   seeding; don't blanket-seed.
6. **Re-evolution (8189)** repeatability — quests are normally one-shot; re-evolution implies the
   quest can be retaken. Check the quest's `interval`/repeat semantics.

---

## 6. Out of scope

- Building a *new* endscript engine — the mechanism exists; we extend quest *checks* and author
  *data*.
- Player-NPC spawning, breeding, non-dragon/robo pets.
- Re-verifying or fixing the broader inherited quest-conversation corpus.

---

## 7. What success looks like

Clicking Garnox (9102001) with a qualifying summoned pet (right template + tameness ≥ 1642) +
Rock offers/advances quest 8185/8189/4659; completing it consumes the Rock (+battery for 4659),
evolves the pet in place (random adult, stats preserved), and on failure the saga refunds. 8184
works as a normal item turn-in gated by pet + tameness. No synthetic free conversation, no
`pickFromContext`.
