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
| `Say.img` | **Dialogue** the NPC speaks (`0`=start convo, `1`=complete convo; `yes`/`no`/`stop`/`ask` branches) | fully populated for all four |

Key facts established from the v83 WZ:

- **`Say.img` is the dialogue source.** e.g. `8185/0/0` = *"I can see that you truly care for
  your Dragon. Now, shall I help you with your Baby Dragon's evolution?"*, with a `yes` branch
  referencing `#t5380000#` (Rock of Evolution). The dialogue is **not** invented by us — it is
  authoritative per-version WZ text.
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
> from authoritative inputs and verify: **Say.img** (dialogue, per-version), **Check.img/Act.img**
> (requirements/rewards), and the **Cosmic `.js`** (script logic — present for 8185/8189/4659,
> absent for 8184), cross-checked against the actual evolution mechanic. This is the project's
> "Verification Over Memory" rule applied to quest data.

---

## 4. Design

### 4.1 atlas-quest — enforce `pet` and `pettamenessmin` (the only code gap)

atlas-quest **parses** `Pet []uint32` and `PetTamenessMin int16` (`data/quest/rest.go:62-63`) but
**never enforces** them — they are absent from `buildStartConditions`/`buildEndConditions`
(`data/validation/processor.go`), and `validation/model.go:16-25` has no corresponding condition
constants. Add two **general** condition types (pet evolution is just the first consumer):

1. `validation/model.go`: `PetCondition = "pet"`, `PetTamenessMinCondition = "petTamenessMin"`.
2. `validation/processor.go`: emit these conditions from `req.Pet` (character must own/has-summoned
   one of the listed pet ids) and `req.PetTamenessMin` (the qualifying pet's tameness ≥ N) in both
   `buildStartConditions` and `buildEndConditions` (these quests gate on both start and complete).
3. **query-aggregator** must be able to evaluate the new condition types — i.e. resolve a
   character's pets and their tameness/closeness. **OPEN:** confirm atlas-pets exposes pet
   closeness and that query-aggregator can read it (the `closeness` grep on `pets/model.go` came
   back empty — verify in planning; closeness is referenced elsewhere in the evolution engine, so
   the field exists somewhere, but the read path for query-aggregator must be confirmed).
4. Tests for both condition types (start and complete), following the existing validation tests.

**Semantics decision to confirm:** "tameness" here is the *summoned* pet's tameness (the quest
checks the active pet). Whether the check targets the summoned pet or any owned pet affects the
query-aggregator query — confirm against Cosmic behavior.

### 4.2 Author the quest conversations (data)

Author `quest-{id}.json` for **8185, 8189, 4659** (8184 needs none — see 4.4). Each:

- **`startStateMachine`** — dialogue from `Say.img/{id}/0` (the offer + yes/no). On accept, a
  `start_quest` operation transitions the quest to STARTED in atlas-quest. (For these quests the
  start machine is mostly dialogue; the real requirement gating is enforced by atlas-quest 4.1.)
- **`endStateMachine`** — dialogue from `Say.img/{id}/1`, then the operation chain that *is* the
  endscript:
  1. `destroy_item` Rock `5380000` (and for 4659, `destroy_item` 50× `4000111`),
  2. `evolve_pet` targeting the qualifying pet (see 4.3),
  3. `complete_quest` → atlas-quest COMPLETED.
  All three are saga-backed; the `PetEvolution` saga already compensates (refund) on failure.

Dialogue text is taken verbatim from the version's `Say.img`; operation logic is cross-checked
against the Cosmic `q8185e.js` / `q8189e.js` / `q4659e.js` for faithfulness, **not** copied blindly
from the inherited conversions.

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

1. **query-aggregator pet/closeness support (4.1.3)** — biggest unknown. If it can't read pet
   closeness, enforcing `pettamenessmin` needs additional plumbing (atlas-pets read path).
2. **Tameness vs level** — confirm 1642 closeness is the gate (it is, per WZ) and that atlas tracks
   closeness on the live pet.
3. **Quest availability/offer** — how the player initiates: does Garnox auto-offer the quest when
   conditions are met, and does clicking him emit `QuestActionScriptStart`? Verify the channel
   path actually fires for a quest NPC (distinct from the shop/generic path we saw).
4. **Faithfulness** — dialogue from Say.img is authoritative; script logic from Cosmic `.js` for
   8185/8189/4659 (none for 8184). Inherited conversions are structure-only references.
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
