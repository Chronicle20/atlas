# Follow-up: gms_61 serverbound opcode corruption (surfaced by task-125)

**Status:** deferred to a dedicated legacy-template-repair task. Not part of the task-125 PR beyond the four corrections listed under "Fixed in task-125" below.

## How this surfaced

Task-125 (skill/mastery books) needed to seed a `CharacterSkillBookUseHandle`
serverbound handler for gms_61 at its IDB-verified opcode `0x4B`. That slot was
already occupied by `PetFoodHandle` in `template_gms_61_1.json`. Investigating the
collision revealed that gms_61's serverbound opcode table is corrupted in the
`0x44`–`0x5A` item-use/skill cluster — several handler entries carry opcodes the
client sends for a *different* feature (the bindings look copied from gms_72,
which uses different values). A full 113-row audit was run against the gms_61
client IDB to bound the damage.

## IDB / method (for whoever resumes this)

- IDB: `GMS_v61.1_U_DEVM.exe.i64` (audit session id `965202bf` at audit time —
  match by binary NAME via `idb_list`, ports/sessions rotate).
- Ground truth for a serverbound handler = the integer literal passed to the
  `COutPacket::COutPacket(this, N)` constructor in the client's send function for
  that action. No serverbound opcode enum exists in this IDB — every value was
  read from a decompiled/disassembled ctor call site.
- Full per-row audit table (all 113 handlers, MATCH/MISMATCH/UNRESOLVED with
  evidence addresses) was written during task-125 to the scratch file
  `.superpowers/sdd/task-11-gms61-serverbound-audit.md`. That path is gitignored;
  the essential results are reproduced below so they are durable.

## Audit result summary (113 serverbound handlers)

- **MATCH: 38** — verified correct, do not touch.
- **MISMATCH (confirmed, corrected value known): 5**
- **UNRESOLVED: 70** — send path not pinned down in the audit pass (no guess made).
  Includes core handlers (player move `0x26`, the attack cluster `0x29`–`0x2C`,
  quest `0x62`, note `0x77`, the CashShop `0xC4` / ITC `0xD5`–`0xD7` submode
  families, etc.). These are UNVERIFIED, not confirmed-correct — a real repair
  task must resolve them before claiming the table is clean.

The corruption is **not** a uniform arithmetic shift. It is two defect types,
confined to the `0x44`–`0x5A` neighborhood (everything sampled outside it matched):

1. **Simple wrong-value:** the handler's true opcode is a free slot, no other
   handler claims it.
2. **Cross-wired:** the handler's *current* opcode is a real opcode the client
   sends — for an unrelated item-use sub-feature (Bridle / Shop-Scanner / Skill-
   Reset item). The handler's *true* opcode is elsewhere (free, or colliding).

### The 5 confirmed mismatches

| handler | template (wrong) | IDB truth | send fn @ addr → COutPacket(N) | note |
|---|---|---|---|---|
| `CharacterItemUseHandle` | 0x47 | **0x43** | `SendStatChangeItemUseRequest` @0x831880 → 67 | simple wrong-value |
| `PetFoodHandle` | 0x4B | **0x47** | `SendPetFoodItemUseRequest` @0x831DE9 → 71 | simple wrong-value |
| `MountFoodHandle` | 0x4C | **0x48** | `SendTamingMobFoodItemUseRequest` @0x831F44 → 72 | cross-wired: 0x4C is really Shop-Scanner item use |
| `CharacterUseSkillHandle` | 0x5A | **0x53** | `SendSkillUseRequest` @0x7BA213 → 83 | cross-wired: 0x5A is really the Skill-Reset item |
| `CharacterItemUseSummonBagHandle` | 0x4A | **0x46** | `SendMobSummonItemUseRequest` @0x831C83 → 70 | cross-wired: 0x4A is really the Bridle item; **0x46 collides** — see below |

Cross-reference: the new skill-book send is `SendSkillLearnItemUseRequest`
@0x8325D2 → 75 = `0x4B` (why skill-book must live at 0x4B).

## Fixed in task-125 (already in this PR)

Four IDB-verified corrections were applied to `template_gms_61_1.json` (user-
approved), because they are confidently verified, land on free serverbound slots,
and the ItemUse→PetFood→SkillBook chain is what frees `0x4B`:

- `CharacterItemUseHandle`  0x47 → 0x43
- `PetFoodHandle`           0x4B → 0x47
- `MountFoodHandle`         0x4C → 0x48
- `CharacterUseSkillHandle` 0x5A → 0x53
- (+ new) `CharacterSkillBookUseHandle` @ 0x4B, `CharacterSkillLearnItemResult` writer @ 0x30

No new handler/writer opcode collision was introduced by these edits.

## Deferred to this follow-up task

1. **`CharacterItemUseSummonBagHandle` (0x4A → 0x46) — BLOCKED.** The true Summon
   Bag opcode `0x46` collides with `CharacterInventoryMoveHandle`, whose own true
   opcode is UNRESOLVED (its send function was not located). Resolving this needs
   the real inventory-move send opcode first; do not move SummonBag to 0x46 until
   InventoryMove is verified and relocated.
2. **The 70 UNRESOLVED handlers** — verify each against the gms_61 IDB and correct
   any further mismatches. Until done, gms_61's serverbound table is only partially
   validated (~46% checked).
3. **Pre-existing gms_61 writer (clientbound) opcode collisions** at `0x00` (×4),
   `0x0A` (×2), `0xF2` (×2), `0x40` (×2) — present before task-125, unrelated to
   skill-books, likely part of the same template-corruption pattern on the writer
   side. A parallel clientbound audit is warranted.

## Suggested approach for the repair task

Treat it as a legacy-version bring-up for gms_61's serverbound (and writer) columns
— follow `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` methodology to
resolve the 70 unresolved handlers systematically, then reconcile the whole
`template_gms_61_1.json` socket config to IDB ground truth in one vetted pass.
Live gms_61 tenants will then need a socket-config PATCH + channel restart for the
corrected opcodes (same rollout mechanics as task-125 §Task 15).
