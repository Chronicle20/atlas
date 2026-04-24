# First-Job AP Rebalance — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-24
---

## 1. Overview

In MapleStory v83, beginners do not have manual control over ability-point (AP) allocation; the client auto-allocates AP on level-up until the character either completes the first job advancement or reaches Level 11. In vanilla v83 this auto-allocation deposits points into STR only (per existing atlas-character behavior: +5 STR per level for levels 2–5, +4 STR and +1 DEX per level for levels 6–10). As a result, a Level 10 beginner arrives at first-job advancement with roughly **STR 53, DEX 9, INT 4, LUK 4** — a distribution that cannot satisfy class-minimum stat requirements for any job except warrior (35 STR) or Aran (35 STR).

Atlas's current NPC conversation scripts enforce these minimums as **hard gates** (e.g., bowman NPC 1012100 rejects the advancement if DEX < 25), making non-warrior first-job advancement impossible on an unmodified v83 client. This does not match canonical Nexon v83 behavior: period gameplay footage of Aran and Pirate first-job advancements on two different versions confirms that the server **rebalances** AP at advancement — setting the class floor, returning reclaimed points to the unallocated pool, and performing no pre-advancement stat check.

This task redesigns first-job advancement so the class-minimum numbers currently encoded as check gates become **rebalance targets** applied by the server at advancement time. All five Explorer 1st-job NPCs, the Cygnus Knight 1st-job NPC, and the Aran 1st-job polearm interaction are updated. No tenant-configurable strict-gate override is included in this task — Atlas commits to vanilla v83 behavior.

## 2. Goals

Primary goals:
- A Level-10 beginner running vanilla v83 auto-allocation can complete first-job advancement for any Explorer class, Cygnus class, or Aran without needing client modification.
- Post-advancement stats match the canonical pattern observed in period gameplay: all non-target stats reset to base (4), target primary stat set to class floor, accumulated surplus AP returned as unallocated.
- The NPC conversation conversion spec documents a single canonical operation for first-job AP rebalance so future NPC conversions follow the same pattern.
- The currently-undocumented `reset_stats` operation is either documented or superseded by the new operation.

Non-goals:
- No changes to 2nd / 3rd / 4th job advancement. Those happen post-Level-11 in a world where the player has been manually allocating, so stat gates there remain legitimate.
- No changes to beginner auto-allocation logic. The existing v83-gated schedule in `atlas-character/character/processor.go:1295-1310` is correct and stays.
- No tenant-configurable strict-gate mode. If a tenant ships a client edit allowing early manual allocation, that tenant gets rebalance behavior anyway — the surplus is returned as unallocated AP, which is functionally equivalent.
- No new tenant configuration resource in atlas-tenants.
- No client-side changes.
- No UI changes to the stat allocation window (if the client auto-pops it on advancement, we rely on that; if it doesn't, that's a separate follow-up).
- No changes to the Resistance, Dual Blade, or post-v83 class advancement flows (out of scope for target versions).

## 3. User Stories

- **As a player** on a vanilla v83 client, I want to complete my first-job advancement without being blocked by a stat check I have no way to satisfy, so that I can play non-warrior classes.
- **As a player** whose stats are rebalanced at advancement, I want my stat sheet to reflect the new distribution immediately and correctly, so I am not surprised by unexplained changes to my weapon attack, accuracy, or magic attack.
- **As an NPC script author**, I want a single documented `rebalance_ap` operation I can place in first-job advancement scripts, so I don't have to encode the rebalance algorithm in every script.
- **As a future contributor** converting additional first-job NPC scripts (e.g., Cygnus, Aran variants), I want the conversion spec to describe exactly when to use `rebalance_ap` and with what parameters, so converted scripts are consistent.

## 4. Functional Requirements

### 4.1 Rebalance algorithm

Given a character at first-job advancement with current stats (STR, DEX, INT, LUK), current unallocated AP, and a target floor `(target_stat, floor_value)`, the rebalance produces new stats and new unallocated AP by the following deterministic procedure:

1. Compute the reclaimed pool:
   ```
   reclaimed = (STR − 4) + (DEX − 4) + (INT − 4) + (LUK − 4)
   ```
   Each per-stat contribution is clamped to `max(0, …)` — stats are guaranteed ≥ 4 by game invariants, but the clamp protects against malformed state.
2. Reset the four primary stats to the base value: `STR = DEX = INT = LUK = 4`.
3. Raise the target stat to its class floor: `new_stats[target_stat] = floor_value`. The cost is `floor_value − 4`.
4. Compute new unallocated AP:
   ```
   new_unallocated_AP = old_unallocated_AP + reclaimed − (floor_value − 4)
   ```
5. If there is an advancement-specific AP grant (see §9 Open Question 1), add it to `new_unallocated_AP` after step 4.

Expected values matching observed gameplay footage for a canonical Level 10 vanilla v83 beginner (STR 53, DEX 9, INT 4, LUK 4, unallocated = 0):

| Class | Floor | Post-advancement stats | Unallocated AP |
|---|---|---|---|
| Warrior | 35 STR | 35/4/4/4 | 54 − 31 = **23** |
| Magician | 20 INT | 4/4/20/4 | 54 − 16 = **38** |
| Bowman | 25 DEX | 4/25/4/4 | 54 − 21 = **33** |
| Thief | 25 DEX | 4/25/4/4 | 54 − 21 = **33** |
| Pirate | 20 DEX | 4/20/4/4 | 54 − 16 = **38** ✓ (matches video) |
| Aran | 35 STR | 35/4/4/4 | 54 − 31 = **23** |

The Pirate row matches the observed 20/38 from period gameplay footage and is the reference test case.

### 4.2 Rebalance scope within character stats

The rebalance operates on STR, DEX, INT, and LUK only. HP and MP (which can also be increased via AP in the existing `RequestDistributeAp` flow) are **not** touched. This matches observed behavior: HP/MP represent survivability and are governed by level-up grants and class-specific scaling, not the AP redistribution flow.

Skill points (SP) are not part of this task; first-job advancement separately grants SP via the existing conversation script flow and that is unchanged.

### 4.3 NPC conversation operation: `rebalance_ap`

A new operation is introduced to the NPC conversation state machine, invoked from action states. Parameters:

```json
{
  "operation": "rebalance_ap",
  "target_stat": "strength" | "dexterity" | "intelligence" | "luck",
  "floor": <integer>
}
```

Semantics: when executed in the context of an active conversation with a character, the server performs the rebalance algorithm (§4.1) on that character's stats using the supplied `target_stat` and `floor`, emitting the resulting stat updates through the existing stat-change event flow.

The operation is side-effecting and must be executed **before** `change_job` in the action sequence, so clients receive stat updates before the job-change broadcast (minimizing the window in which a client could display, e.g., a Magician with STR 53).

### 4.4 NPC conversation script changes

For each first-job advancement NPC listed in §4.5, the conversation JSON is updated as follows:

1. **Remove the class-minimum stat check** — the existing condition that branches to an "insufficient stats" state (e.g., `{"type": "dexterity", "operator": "<", "value": "25"}` in bowman NPC 1012100) is deleted along with the unreachable rejection state it pointed to.
2. **Retain the level check** — the Level 10 (Level 8 for Magician in some versions) requirement remains; it is not affected by this task.
3. **Add `rebalance_ap`** to the action sequence immediately before `change_job`, parameterized with the class's target stat and floor.
4. **Retain `reset_stats` only if it was serving a purpose** other than the now-removed gate; in most scripts it is redundant with the new `rebalance_ap` and should be removed.

### 4.5 Affected NPCs

All first-job advancement entry points in atlas-npc-conversations. NPC IDs to be confirmed during the plan phase; the canonical set is:

| Class | NPC | Floor parameter |
|---|---|---|
| Explorer Warrior | Dances with Balrog (Perion) | STR 35 |
| Explorer Magician | Grendel the Really Old (Ellinia) | INT 20 |
| Explorer Bowman | Athena Pierce (Henesys) — NPC 1012100 confirmed | DEX 25 |
| Explorer Thief | Dark Lord (Kerning City) | DEX 25 |
| Explorer Pirate | Kyrin (Nautilus) | DEX 20 |
| Cygnus Knight | (Chief Knight / Shinsoo interaction — to confirm during plan) | Per sub-class |
| Aran | Lilin-adjacent polearm click (Rien) | STR 35 |

The plan phase is responsible for enumerating exact NPC IDs and each JSON file path. The Cygnus row may split into multiple entries if v83 Cygnus advancement flows through multiple NPCs or has sub-class-specific floors.

### 4.6 Conversion spec updates

`docs/npc_conversation_conversion_spec.md` is updated to:
1. Document the new `rebalance_ap` operation with parameters, semantics, ordering constraints, and an example drawn from one of the updated NPCs.
2. Document the previously undocumented `reset_stats` operation (seen in npc_1012100.json:214), or explicitly mark it deprecated and point to `rebalance_ap`.
3. Add guidance: "For first-job advancement NPCs, use `rebalance_ap` with the class floor; do not use stat-minimum condition checks as advancement gates."

### 4.7 Player-facing notification

For v1 we rely on the existing stat-change event flow. atlas-character emits stat-change events (`statChangedProvider`, `processor.go:893`) that already include `TypeAvailableAP` and the primary stats. The client will receive updates for STR, DEX, INT, LUK, and available AP as part of the rebalance.

An explicit yellow chat notice is **not** included in v1; we will observe client behavior (the v83 client is believed to auto-open the stat window at first-job advancement per observed gameplay footage) and add an explicit notification only if testing reveals a gap.

## 5. API Surface

No new REST endpoints. No new public Kafka topics.

**Internal event flow** remains:
- atlas-npc-conversations, upon executing `rebalance_ap`, issues a command to atlas-character to perform the rebalance. The existing command transport between the two services is reused. Exact transport mechanism (REST call vs. Kafka command) is a plan-phase decision; it must match how `change_job` is currently wired from the same conversation flow.
- atlas-character emits stat-change events for STR, DEX, INT, LUK, and available AP via the existing `statChangedProvider`. No new event types are required.

## 6. Data Model

No schema changes. No new persistent entities.

The rebalance is a point-in-time recomputation of existing fields on the character entity (STR, DEX, INT, LUK, AP — all already present per `atlas-character/character/entity.go:40` and surrounding lines).

## 7. Service Impact

| Service | Change | Magnitude |
|---|---|---|
| `atlas-character` | New processor method (e.g., `RebalanceAPForJobAndEmit(targetStat, floor)`) implementing §4.1 algorithm; wired to emit stat-change events for all four primary stats plus available AP. | Medium — one new processor method, reuses existing event providers. |
| `atlas-npc-conversations` | Implement `rebalance_ap` operation handler invoking the new atlas-character method; update conversation JSON for all first-job NPCs per §4.4; remove unreachable rejection states from those JSON files. | Medium — one new action handler, ~6–10 JSON files edited. |
| `docs/npc_conversation_conversion_spec.md` | Document new `rebalance_ap` operation and resolve `reset_stats` documentation gap per §4.6. | Small — spec doc edits. |
| `atlas-tenants` | No changes. | None. |

## 8. Non-Functional Requirements

- **Performance**: Rebalance is an O(1) arithmetic operation on a single character. No measurable latency impact on conversation flow.
- **Multi-tenancy**: Existing tenant scoping on the character entity applies unchanged. The rebalance runs in the caller's tenant context, inherited from the active conversation. No cross-tenant access.
- **Observability**: The rebalance should log at info level with before/after stats and AP for the affected character ID, enabling post-hoc audit if a player disputes their post-advancement numbers.
- **Idempotency**: The rebalance is **not** idempotent — re-running it would reset stats to base and re-compute. The advancement flow must guarantee it runs at most once per advancement; existing single-fire semantics of the action state machine provide this.
- **Determinism**: Given identical inputs, the algorithm produces identical outputs. No RNG, no clock dependency.
- **Backwards compatibility**: Characters who have already completed first-job advancement under the old gated behavior are unaffected. No migration is needed for existing characters.

## 9. Open Questions

1. **The "+5 on advancement" grant.** Observed period footage shows Pirate post-advancement with DEX 20 + 38 unallocated AP, which matches the algorithm exactly without any additional grant. It is unclear whether an additional +5 AP is granted to the character as part of the advancement event itself (distinct from the rebalance). Plan phase should confirm against a reference v83 source (MapleRoyals or MapleLegends server logs, or a second video capture where the unallocated AP count is visible on-screen post-advancement). If the grant exists, add it as step 5 of the algorithm; if not, remove that step from §4.1.

2. **Cygnus first-job advancement flow.** The exact NPC path for v83 Cygnus Knights (Noblesse → Dawn Warrior / Blaze Wizard / Wind Archer / Night Walker / Thunder Breaker) requires enumeration during the plan phase. Each sub-class likely has its own floor (primary stat + 35 or +20 depending on class archetype). If the flow branches in a way that makes `rebalance_ap` placement ambiguous, plan phase must resolve.

3. **Does the v83 client auto-open the stat window after first-job advancement?** If yes, §4.7 is complete as written. If no, we will need either an explicit server-sent packet to open the window, or a yellow chat notice directing the player to open it. This is confirmed by testing the updated scripts in a running client; not blocking the PRD.

4. **`reset_stats` deprecation vs. documentation.** The operation is used in the current bowman NPC (`npc_1012100.json:214`). Plan phase decides whether it is superseded by `rebalance_ap` (remove from script, deprecate in spec) or retained for other legitimate uses (document in spec). A survey of all current uses of `reset_stats` across converted NPC scripts answers this.

## 10. Acceptance Criteria

The task is complete when all of the following are demonstrably true:

1. **Rebalance algorithm behavior** — Given a character with stats (53, 9, 4, 4) and unallocated AP = 0 at Level 10, executing `rebalance_ap` with `target_stat = dexterity, floor = 20` produces stats (4, 20, 4, 4) and unallocated AP = 38. Verified by unit test in atlas-character.
2. **Boundary cases** — Rebalance with target floor below the character's current target-stat value (e.g., warrior with STR 53, floor 35) correctly reduces the target stat to the floor and returns the surplus to unallocated AP. Warrior case post-advancement: stats (35, 4, 4, 4), unallocated = 23.
3. **HP/MP unchanged** — Character HP and MP values are bit-identical before and after the rebalance. Verified by unit test.
4. **All first-job NPC scripts updated** — Every first-job advancement JSON listed in §4.5 has the class-minimum stat-check condition removed, the unreachable rejection state removed, and `rebalance_ap` inserted before `change_job` with correct parameters.
5. **NPC conversation spec updated** — `docs/npc_conversation_conversion_spec.md` documents the `rebalance_ap` operation with parameters, semantics, an example, and guidance that first-job NPC conversions must use it. The `reset_stats` gap is resolved (either documented or marked deprecated per Open Question 4).
6. **End-to-end test** — A scripted integration test (or documented manual test) walks a Level 10 beginner through first-job advancement with each of the six classes in §4.5 and confirms that: advancement succeeds, post-advancement stats match the §4.1 table, the stat-change event is emitted, and the player receives the SP grant that the script already awards.
7. **Observational video match** — Post-advancement stats for a Pirate match the reference video: DEX 20, unallocated AP = 38 (± any confirmed +5 grant from Open Question 1).
8. **No regression** — Existing 2nd/3rd/4th job advancement flows produce no behavior change. Existing `RequestDistributeAp` flow produces no behavior change. Existing beginner auto-allocation on level-up produces no behavior change.
