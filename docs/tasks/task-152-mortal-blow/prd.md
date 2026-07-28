# Mortal Blow (Ranger/Sniper) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Mortal Blow (Ranger 3110001 / Sniper 3210001) is a pre-Big-Bang bowman passive: when the player uses a **normal attack** with a bow/crossbow against a monster at point-blank range, the client — with a per-level success rate — converts the swing into a special close-range shot. When that shot lands on a monster whose HP is at or below a per-level threshold, the server rolls a per-level chance to kill the monster outright. Atlas currently ignores the server half entirely: the gap is the `// TODO Mortal Blow` marker in `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (line 421 at time of writing), inside `processAttack` after per-monster damage application.

The client half was IDA-verified for this PRD (see §4, "Client Behavior Contract"). The key finding: **the client sends the Mortal Blow proc as a standard ranged attack packet carrying skill id 3110001/3210001**, so the server does not need Cosmic's job-range heuristic — it can gate precisely on the attack's skill id, preserving the authentic trigger conditions (normal attack, point-blank, client-side success roll) for free because the client only ever tags an attack with these ids under those conditions.

Scope is S: one proc block in the ranged-attack path, a `Y()` accessor on the channel's skill-effect model, and the kill delivery through the existing monster damage path. Structural template: the MP Eater proc (task-049) and the drain-family heal (task-147), which work the same post-damage block of the same function.

## 2. Goals

Primary goals:
- A ranged attack arriving with skill id 3110001/3210001 applies, per damaged non-boss monster, the Mortal Blow instant-kill roll: monster HP ≤ `maxHP × x / 100` → roll 1–100 ≤ `y` → kill.
- `x` (HP threshold %) and `y` (kill chance %) come from the tenant's skill data at the character's owned skill level — no hard-coded values.
- The kill is delivered through the standard monster damage path so EXP and drops credit the attacker exactly as a normal kill (Cosmic parity: `map.damageMonster(player, monster, Integer.MAX_VALUE, …)`).
- Proc failures (effect lookup, monster snapshot fetch, emit errors) are logged and swallowed — the attack pipeline is never aborted or delayed (same policy as MP Eater, venom, and drain).

Non-goals:
- **HP/MP recovery on kill.** The backlog note claimed it, but Cosmic `AbstractDealDamageHandler.java:441-460` has no recovery, and the v83 WZ level data (verified: `Skill.wz/311.img.xml` / `321.img.xml`, fields `prop/damage/x/y` only) has no recovery attributes. HP/MP-on-kill is the post-Big-Bang redesign of Mortal Blow — out of scope.
- **Cosmic's job-range trigger.** Cosmic rolls the instakill on *every* attack by jobs 311–322 (any skill, any range — Hurricane included). That is a Cosmic simplification, not authentic behavior; this task keys on the attack's skill id instead (owner decision 2026-07-10: preserve the normal-attack-only trigger).
- **The client-side halves of the skill** — the point-blank detection, the `prop` success roll, and the shot's `damage%` — are client-owned (verified §4) and need no server work. The shot's damage arrives pre-computed in the attack packet like any other attack.
- **Projectile consumption changes.** The Mortal Blow shot consumes an arrow client-side (verified §4.5); the existing server-side consumption plan already matches. (FR-6 corrects a stale comment that claimed otherwise.)
- **New or changed packets.** v83 has no Mortal Blow-specific clientbound packet; the existing ranged-attack broadcast already carries the skill id to observers, whose clients render the effect themselves. (v95's `CMob::OnSpecialEffectBySkill` special-cases these ids, but post-BB Mortal Blow is a different mechanic — out of scope.)
- Server-side validation of attack distance / point-blank range (Atlas performs no such validation for any attack type today).
- The other TODOs in the post-attack block.

## 3. User Stories

- As a Ranger fighting at point-blank range with normal attacks, I want low-HP monsters to sometimes die instantly to my Mortal Blow shots so that the passive I invested SP in actually functions.
- As a Sniper, I want the same behavior keyed to my version of the skill (3210001).
- As a player killing a monster via Mortal Blow, I want full EXP and drops so that the instant kill is a reward, not a penalty.
- As a boss-fight participant, I want bosses immune to the instant kill so that the mechanic cannot trivialize boss content.
- As a player without the passive (or on a job that cannot own it), I want attacks falsely tagged with the skill id rejected so that the mechanic cannot be exploited by packet forgery.

## 4. Client Behavior Contract (IDA-verified)

Verified against GMS v83 (`MapleStory_dump.exe.i64`, ida-pro instance port 13342) on 2026-07-10. This section is the ground truth the server design must complement.

### 4.1 Trigger path (normal attack only)

`CUserLocal::HandleCtrlKeyDown` (0x94e256) — the attack-key handler — is the only path that arms the Mortal Blow check. For a ranged weapon it calls `CUserLocal::TryDoingMeleeAttack` (0x950921) with a point-blank out-flag armed; skill casts enter through `DoActiveSkill_*` and never arm it.

### 4.2 Proc decision (client-side `prop` roll)

Inside `TryDoingMeleeAttack`, when the point-blank flag is armed and `CMobPool::FindHitMobInRect` finds a mob in the melee rect:

1. Helper `sub_7656AA` (0x7656AA, sole caller: TryDoingMeleeAttack) resolves the passive: job track 311 → `CSkillInfo::GetSkillLevel(…, 3110001)`, track 321 → `3210001`; returns 0 if unowned.
2. It returns the level data's `nProp` (SKILLLEVELDATA offset +244 in v83; layout cross-mapped from the v95 typed struct — v83 `+304` = `nMobCount` anchors the mapping; v83 WZ `prop` = 33–90%).
3. The client rolls `CRand32::Random % 100 < nProp`. On failure the melee/shoot attack proceeds normally.
4. On success it calls `CUserLocal::TryDoingShootAttack` (0x9537d5) **with the Mortal Blow SKILLENTRY and level**, plus a flag that lands in the attack packet (§4.3).

### 4.3 Wire shape

The proc is sent as the standard serverbound ranged attack (v83 opcode 0x2D) with:
- `skillId` = 3110001 or 3210001 — the server's gating key.
- The Mortal Blow flag as **bit 2 of the option byte** (Atlas `AttackInfo.mask1`; currently folded into the `finalAfterSlashBlast` bits 0–2 decode). Redundant with the skill id — the server does not need it.
- Damage computed client-side per the skill's `damage%` (110–250%), like every attack.

v83 attack packets carry no skill level; the server resolves the level from the character's owned skills (the existing `processAttack` pipeline already does this and **destroys the session** if the character does not own the attack's skill — that guard covers the forgery story in §3).

### 4.4 Instant-kill is server-side

Nothing in the client path reads the skill's `x` or `y`. The v83 binary references 3110001/3210001 in exactly two functions: the trigger helper (§4.2) and `is_shoot_skill_not_switched_to_melee_attack` (v83 `sub_7668B7`; name from the v95 twin at 0x6edd62). The HP-threshold/kill roll must therefore be implemented server-side, per the WZ semantics: threshold `x`% of max HP (20–50 by level), kill chance `y`% (1–10 by level), master level 20.

### 4.5 Projectile is consumed

3110001/3210001 do **not** appear in the client's not-consuming-bullet predicate (only in the two functions above), so the Mortal Blow shot decrements an arrow client-side like a normal shot. The server's existing consumption plan already matches; no change. This **contradicts** the `TODO(task-007)` comment in `character_attack_projectile.go` (~line 118) which lists Mortal Blow as a passive no-consume mechanic — see FR-6.

### 4.6 Version notes

- **v84**: identical trigger helper (`sub_7879E2` @ 0x787a14 in `GMS_v84.1_U_DEVM`, port 13345), same `+244` read. Behavior identical to v83.
- **v87/v92**: pre-BB, expected identical; not individually verified. The server design is version-agnostic (keyed on incoming skill id + tenant skill data), so it works wherever a client sends the id and is inert where none does.
- **v95/JMS (post-BB)**: the client trigger helper does not exist (Mortal Blow was redesigned); a v95 client never tags an attack with these ids, so the feature is naturally inert for those tenants.

## 5. Functional Requirements

### FR-1 — Gating

The proc runs in `processAttack` (ranged path), per damaged monster, **only** when `ai.SkillId()` equals `skill.RangerMortalBlowId` (3110001) or `skill.SniperMortalBlowId` (3210001), referenced from `libs/atlas-constants/skill` (DOM-21; both constants already exist). No job-range check is added: skill ownership (already enforced — unowned skill ids destroy the session) is the authoritative guard, and only the 311/321 tracks can own these skills.

### FR-2 — HP threshold

For each damaged monster, fetch the channel's monster snapshot (`monster.Processor.GetById`, as MP Eater does) and evaluate:

```
mon.Hp() ≤ mon.MaxHp() × x / 100
```

where `x` is the skill effect's per-level X value (`se.X()` — `se` is already resolved at the character's owned level by the existing pipeline). Integer arithmetic, truncating division (Cosmic parity: `(getStats().getHp() * getX()) / 100`).

**Timing semantics:** the snapshot reflects pre-attack HP in practice — damage propagates to atlas-monsters asynchronously via Kafka, so the snapshot read in the post-damage block has not (reliably) absorbed the current attack's damage. This matches Cosmic, whose Mortal Blow check (line 441) runs *before* its damage application (line 521). Pre-damage threshold evaluation is the specified behavior; eventual-consistency drift within one attack is acceptable and inherent to the architecture. *(Correction for the record: the spec interview stated Cosmic checks post-damage; that was wrong — verified pre-damage on 2026-07-10.)*

### FR-3 — Kill roll

On threshold pass, roll a uniform integer 1–100; the kill procs when `roll ≤ y`, where `y` is the skill effect's per-level Y value. Both `x` and `y` must come from the tenant's skill data — no numeric literals for thresholds or chances.

### FR-4 — Kill delivery and boss exclusion

- The kill is delivered as max damage through the standard monster damage path (the existing `DamageCommandProvider` flow or a dedicated command — design phase decides), so atlas-monsters records the damage entry and the normal kill flow credits EXP and drops to the attacker (Cosmic parity: `Integer.MAX_VALUE` damage, "thanks Conrad for noticing reduced EXP gain from skill kill").
- **Boss monsters are excluded** (Cosmic `!monster.isBoss()`). The channel's monster snapshot does not carry a boss flag, so the authoritative guard must live where the flag lives — following the DrainMp precedent ("atlas-monsters re-checks all guards"). The design phase picks the mechanism (atlas-monsters-side guard on the command vs. a channel-side monster-info lookup); whichever is chosen, a Mortal Blow kill against a boss must be impossible even if the channel misfires.

### FR-5 — Failure isolation

Every failure in the proc (effect lookup, snapshot fetch, command emit) is logged and swallowed. The attack pipeline — damage application, broadcast, projectile emission — proceeds unaffected (same policy as MP Eater / venom / drain).

### FR-6 — Correct the stale task-007 comment

The `TODO(task-007)` comment in `character_attack_projectile.go` lists Mortal Blow among "passive no-consume mechanics." §4.5 disproves that for Mortal Blow. Remove Mortal Blow from that comment's list (leaving Expert Marksmanship / Claw Mastery untouched) so the next reader doesn't implement a desync.

### FR-7 — Expose `Y()` on the channel skill-effect model

`services/atlas-channel/atlas.com/channel/data/skill/effect` already receives `y` on its REST model (`rest.go` line 42) but the domain model exposes only `X()`. Thread `y` through Transform/Extract and add a `Y()` accessor (mirroring `X()`), with the doc comment noting Mortal Blow's use (kill chance %).

### FR-8 — Tests

- Unit tests for the proc decision: skill-id gating (non-Mortal-Blow ranged attacks never roll), threshold boundary (HP exactly at threshold procs-eligible; one above does not), roll boundary (roll == y procs; roll == y+1 does not), boss exclusion, and failure-swallowing.
- Test setup uses the project's Builder pattern (no `*_testhelpers.go` constructors).
- Randomness must be injectable for tests (match however MP Eater/venom handle their rolls; if they use a package-level rand, follow the existing seam or add the minimal injectable one).

## 6. API Surface

- No new or changed REST endpoints.
- No new or changed client packets (serverbound or clientbound).
- Kafka: at most one addition — if the design chooses a dedicated kill/damage command to atlas-monsters instead of reusing `DamageCommandProvider`, it follows the existing `monster2.Command[...]` envelope. No consumer-facing schema changes otherwise.

## 7. Data Model

None. `x` and `y` already exist in the tenant skill data served by atlas-data (the channel's effect REST model already deserializes `y`); no migrations, no new entities.

## 8. Service Impact

| Service | Change |
|---|---|
| `atlas-channel` | Proc block in `processAttack` ranged path (replacing the `// TODO Mortal Blow` marker); `Y()` accessor on the skill-effect model; comment fix in `character_attack_projectile.go`; tests. |
| `atlas-monsters` | Only if the design places the boss guard / kill command there (FR-4); otherwise none. |
| `libs/atlas-constants` | None (skill ids exist). |
| All others | None. |

## 9. Non-Functional Requirements

- **Multi-tenancy:** all skill values resolve from the tenant's skill data via the existing per-tenant effect pipeline; behavior is inert for tenants/versions whose clients never send the ids (§4.6).
- **Performance:** the proc adds at most one monster snapshot fetch and one effect lookup per damaged monster on Mortal Blow attacks only (mob count is 1 for this attack shape); non-Mortal-Blow attacks take a single integer comparison.
- **Observability:** Debug-level log on threshold-pass and on proc success (character id, monster id, skill id, roll), Error-level on swallowed failures — matching MP Eater's logging shape.
- **Determinism in tests:** RNG injectable per FR-8.

## 10. Open Questions

- None blocking. v87/v92 client triggers are unverified (§4.6) but do not affect the server design; if a discrepancy surfaces during those tenants' testing it is a client-data question, not a server one.

## 11. Acceptance Criteria

- [ ] Ranged attack with skill id 3110001/3210001 from a character owning the skill triggers the per-monster threshold check; all other attacks (including melee/magic and other ranged skills) never do.
- [ ] Monster at or below `maxHP × x/100` (per-level `x` from tenant data) is killed when the 1–100 roll is ≤ per-level `y`; above-threshold monsters and failed rolls take only normal attack damage.
- [ ] The killed monster's EXP and drops credit the attacker identically to a normal kill.
- [ ] Boss monsters are never killed by the proc, regardless of HP.
- [ ] Attack pipeline completes normally when any proc step fails (verified by test).
- [ ] Projectile consumption for Mortal Blow attacks is unchanged (one arrow), and the task-007 comment no longer lists Mortal Blow.
- [ ] `effect.Model.Y()` exists and is threaded from REST.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake atlas-channel` (and `atlas-monsters` if touched) clean from the worktree root; `tools/redis-key-guard.sh` clean.
