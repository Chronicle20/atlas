# Combo Attack Orb Gain/Consume (Crusader/Hero, Dawn Warrior) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-09
---

## 1. Overview

Crusader/Hero (and the Dawn Warrior mirror) Combo Attack is currently display-only in Atlas. Casting Combo Attack (1111002 / 11111001) applies a static `COMBO = 1` temporary stat (`services/atlas-data/atlas.com/data/skill/reader.go:239-240`), so the buff icon and empty orb ring appear — but orbs never accumulate on hits, finisher skills (Panic/Coma families) never consume them, and Advanced Combo's double-orb proc never happens. The attack pipeline has an explicit placeholder: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:404` — `// TODO apply combo orbs (add or consume)`.

This task implements orb-count truth on the server and its broadcast to the owner and observers. The client renders orbs from the `ComboCounter` temporary stat value carried by the existing GIVE_BUFF / GIVE_FOREIGN_BUFF packets (verified in the v83 client: the `CTS_ComboCounter` temporary stat drives orb display; no dedicated orb packet exists for Crusaders). The client also computes combo-scaled damage itself from its local orb count, so no server-side damage multiplier is needed — Atlas accepts client damage via `DamageInfo` as it does today.

> **Note (line references drift):** the placeholder cited above as `character_attack_common.go:404` is at `:500` on current `main` (the file grew since this PRD was written); the unique marker text `// TODO apply combo orbs (add or consume)` still resolves it. Skill-constant citations elsewhere in this doc have likewise shifted by a few lines but all resolve by name.

Orb state lives where all other temporary-stat state lives: in atlas-buffs, as the value of the COMBO stat on the active Combo Attack buff. atlas-buffs gains a new capability to update a stat value on an existing buff (today it only supports Apply/Cancel), and atlas-channel re-broadcasts the buff on every change.

### Verified reference semantics (Cosmic, `CloseRangeDamageHandler.java:83-141`, `Character.java:6046-6053`)

- The COMBO stat **value = orb count + 1**. Cast applies value 1 (zero orbs). Atlas's existing `COMBO = 1` cast mapping is already correct.
- **Gain** (per attack, not per mob): on a close-range attack with ≥1 monster hit, where the skill is not a finisher and not Shout (1111008), and the COMBO buff is active: if `value < x + 1`, increment value by 1. `x` comes from the governing effect: Advanced Combo's effect if the character has any Advanced Combo level, otherwise Combo Attack's effect at the character's level.
- **Double-orb proc**: if Advanced Combo is learned, roll its `prop`; on success add a second orb, but only if the pre-roll incremented value is `≤ x` (net cap stays `x + 1`).
- **Consume**: when the attack skill is a finisher — Panic Sword/Axe (1111003/1111004), Coma Sword/Axe (1111005/1111006), Dawn Warrior Panic/Coma (11111002/11111003) — reset value to 1 and re-broadcast. The finisher fires regardless of current orb count; fewer orbs just means the client computed less damage. No server-side rejection.
- **Re-broadcast**: on every value change, send GIVE_BUFF to the owner with the **remaining** duration (original duration minus elapsed time since the buff started) and GIVE_FOREIGN_BUFF to other sessions in the map.
- **Not applicable**: Cosmic's `isComboReset()` (`StatEffect.java:1747-1749`) is Aran-only (Combo Barrier / Combo Drain reset Aran's combo counter) — despite the backlog idea listing it, it has no Crusader/Dawn Warrior behavior and is out of scope.

### Verified WZ data (Skill.wz 111.img / 112.img, v83 dump)

| Skill | Attribute | L1 → Lmax |
|---|---|---|
| Combo Attack 1111002 | `x` (orb cap basis) | 3 → 5 (L30); `time` 100s → 200s |
| Advanced Combo 1120003 | `x` | 6 → 10 (L30) |
| Advanced Combo 1120003 | `prop` (double-orb %) | 31 → 60 (L30) |

Atlas-data already parses `x`, `prop`, and `time` per level; no WZ reader changes are expected for these attributes (to be confirmed in design against the effect REST model).

### Verified client behavior (IDA, v83 `MapleStory_dump.exe`)

- `CUserLocal::RequestIncCombo` (0x9602f3, sends opcode 0xA3) is called only from `CMob::OnHit` (0x668b83), gated on `GetSkillLevel(Combo Ability 21000000, or 20000017 for job 2000) > 0 && weapon_type == 44` — i.e. **Aran with a polearm only**. `CUserLocal::OnIncComboResponse` (0x9602cb) decodes a 4-byte counter and calls `DrawCombo` (the Aran on-screen combo number). The SHOW_COMBO packet row in STATUS.md:296 is therefore the Aran counter and is **out of scope** for this task.
- Crusader/Dawn Warrior orb display is driven by the `CTS_ComboCounter` temporary stat delivered via the temporary-stat-set (GIVE_BUFF) path, which Atlas already implements (`CharacterBuffGiveWriter` / `CharacterBuffGiveForeignWriter`).

## 2. Goals

Primary goals:
- Orbs accumulate server-side on qualifying attacks and the count is visible to the owner and all observers in the map.
- Advanced Combo raises the orb cap and adds the chance-based double-orb proc.
- Finisher skills reset the orb count to zero (stat value 1) and the reset is broadcast.
- atlas-buffs owns orb state as the COMBO stat value on the existing buff, updated via a new stat-value-update capability.
- Works on all supported tenant versions — achievable because no new packets are introduced; the existing per-version buff writers carry the value. The current runtime set (`deploy/k8s/base/versions.json`) is **GMS v12, v48, v61, v72, v79, v83, v84, v87, v92, v95, and JMS v185** — the pre-Big-Bang legacy columns (v12/v48/v61/v72/v79) were added to `main` after this PRD's first draft and are in scope for this task. Version-independence is grounded, not assumed: the COMBO temporary stat registers unconditionally at bit21 of `mask.L` (bits 0–46), which every client reads, IDA-verified down to `GMS_v48` (`libs/atlas-packet/model/character_temporary_stat.go:101`, `legacyGmsMask` `:560-577`). See §8 for the per-version caveats.

Non-goals:
- Server-side combo damage multiplier or damage validation (client computes combo-scaled damage; Atlas continues to accept client `DamageInfo`).
- SHOW_COMBO / ARAN_COMBO_COUNTER packets (verified Aran-only).
- Aran combo systems (Combo Ability counter, Combo Barrier, Combo Drain, Combo Tempest — separate TODOs).
- Energy Charge (Marauder/Thunder Breaker) — different stat, separate branch in the same Cosmic handler.
- Server-side rejection of finishers cast with zero orbs.

## 3. User Stories

- As a Crusader/Hero player, I want orbs to appear around my character as I land attacks so that Combo Attack functions as designed.
- As a Crusader/Hero player, I want Panic/Coma to consume my orbs so that finisher gameplay (build orbs, spend orbs) works.
- As a Hero with Advanced Combo, I want a chance to gain two orbs per attack and a cap of up to 10 orbs so that my 4th-job upgrade is meaningful.
- As a Dawn Warrior, I want the same orb behavior via my skill IDs (11111001/11111002/11111003/11110005).
- As a bystander in the map, I want to see other players' orb counts update so that combat looks correct.

## 4. Functional Requirements

### FR-1: Orb gain on qualifying attack

In the melee attack pipeline (`character_attack_common.go`, at the existing TODO hook), after damage processing and the attack broadcast:

1. Qualifying conditions (all must hold):
   - The attacker has an active COMBO temporary stat (Combo Attack buff applied).
   - At least one monster was hit (`len(ai.DamageInfo()) > 0` with ≥1 target — match Cosmic's `numAttacked > 0`).
   - The attack skill is not a finisher (FR-3 list) and not Shout (1111008).
2. Determine the governing effect: if the character has Advanced Combo (Hero 1120003 or Dawn Warrior 11110005) at level > 0, use its effect at that level; otherwise use Combo Attack's effect (Crusader 1111002 or Dawn Warrior 11111001) at the character's skill level.
3. If `currentValue < x + 1`: increment by 1. If Advanced Combo is learned and a roll against its `prop` succeeds and the incremented value is `≤ x`, increment by 1 again.
4. If the value changed, emit the stat-value update (FR-4).

Increment is **per attack cast**, not per monster hit (user decision #5).

### FR-2: Skill-ID mapping

Use `libs/atlas-constants/skill` constants exclusively (all verified present, `constants.go:2937-2947`, `constants.go:3302-3306`):

| Role | Adventurer | Cygnus |
|---|---|---|
| Combo Attack | `CrusaderComboAttackId` 1111002 | `DawnWarriorStage3ComboAttackId` 11111001 |
| Advanced Combo | `HeroAdvancedComboAttackId` 1120003 | `DawnWarriorStage3AdvancedComboId` 11110005 |
| Finishers | `CrusaderPanicSwordId` 1111003, `CrusaderPanicAxeId` 1111004, `CrusaderComaSwordId` 1111005, `CrusaderComaAxeId` 1111006 | `DawnWarriorStage3PanicId` 11111002, `DawnWarriorStage3ComaId` 11111003 |

Branch selection (adventurer vs Cygnus constants) follows the character's job, matching Cosmic's `isCygnus()` split.

### FR-3: Orb consume on finisher

When the attack skill is one of the six finisher IDs and the attacker has an active COMBO stat: reset the stat value to 1 (zero orbs) and emit the stat-value update (FR-4). This happens whether or not any monster was hit (Cosmic consumes on finisher use unconditionally) and regardless of the current orb count (user decision #6). Do not reject the attack.

### FR-4: Stat-value update capability in atlas-buffs

atlas-buffs gains a command to change the value of one stat on an existing buff:

- New command type on `COMMAND_TOPIC_CHARACTER_BUFF` (existing envelope in `kafka/message/character/kafka.go`): update the COMBO stat value for a character's buff identified by source id.
- Semantics: absolute set (the channel computes the new value from its read of the current value) — design phase decides set-vs-delta; requirement is that concurrent attacks cannot lose updates (registry mutation must be atomic within atlas-buffs).
- If the character has no matching active buff, the command is a logged no-op (buff may have expired between attack and command processing).
- On success, atlas-buffs emits a status event on `EVENT_TOPIC_CHARACTER_BUFF_STATUS` carrying the updated stat set and the buff's **remaining duration**, so consumers can re-broadcast without recomputing expiry.

### FR-5: Re-broadcast on value change

atlas-channel's buff status consumer (`kafka/consumer/buff/consumer.go`) handles the new status event by announcing:
- `CharacterBuffGiveWriter` to the owner's session with the updated stat and remaining duration (client restarts its local timer, matching Cosmic's remaining-duration re-send), and
- `CharacterBuffGiveForeignWriter` to all other sessions in the map.

No new packet writers, opcodes, or tenant template changes — this is the existing APPLIED broadcast shape with updated values.

### FR-6: Buff expiry / cancel unchanged

Orb state dies with the buff. Expiry and cancel paths in atlas-buffs are untouched; a re-cast of Combo Attack re-applies `COMBO = 1` via the existing atlas-data statup (zero orbs), which matches classic behavior.

## 5. API Surface

No REST changes.

Kafka (atlas-buffs, `COMMAND_TOPIC_CHARACTER_BUFF`):
- New command type (working name `UPDATE_STAT_VALUE`): body carries `sourceId` (the buff's skill id), stat `type` (`"COMBO"`), and the new `value`. Tenant headers as per existing commands.

Kafka (atlas-buffs, `EVENT_TOPIC_CHARACTER_BUFF_STATUS`):
- New status type (working name `STAT_UPDATED`): body mirrors the APPLIED event shape (character id, source id, stat changes, remaining duration) so the channel consumer can reuse its announce path.

Exact naming/shape is a design-phase decision; the contract requirements are: idempotent-safe no-op on missing buff, remaining duration included, stat changes as a list (same `stat.Model` shape as APPLIED).

## 6. Data Model

No database changes. Orb count is in-memory state in the atlas-buffs registry (buff `changes` stat value), scoped per tenant/character as today. The COMBO stat type already exists (`character.TemporaryStatTypeCombo` in atlas-constants; used by `atlas-data` reader and the buff writers).

## 7. Service Impact

- **atlas-channel**: implement the TODO hook in `character_attack_common.go` — orb gain (FR-1), finisher consume (FR-3); read current COMBO value and skill levels (Combo/Advanced Combo) from existing character/buff lookups; emit the new buff command; extend the buff status consumer for the new event (FR-5). The melee handler is the only attack type that gains orbs (close-range attacks; matches Cosmic's `CloseRangeDamageHandler`).
- **atlas-buffs**: new command handling + registry stat-value mutation + new status event (FR-4). Mock updates in `character` package tests as required by the Processor interface change.
- **atlas-data**: expected no change (cast statup already correct; `x`/`prop` already parsed — design confirms the effect REST model exposes them to atlas-channel).
- **libs/atlas-packet, tenant templates**: no change (no new opcodes).

## 8. Non-Functional Requirements

- **Multi-tenancy**: commands/events carry tenant headers as per existing atlas-buffs message flow; registry keys already tenant-scoped.
- **Concurrency**: rapid attacks may race (channel reads value N, emits set N+1, twice). The design must ensure atlas-buffs serializes mutations per character (its registry is already `sync.RWMutex`-guarded); acceptable residual: a lost single increment under same-millisecond attacks, unacceptable: value exceeding cap or going below 1.
- **Failure isolation**: buff-command emission failures are logged and swallowed — the attack pipeline must not fail or retry because orb bookkeeping failed (same pattern as MP Eater / projectile emits in the same function).
- **Version coverage**: all supported tenant versions (GMS v12/v48/v61/v72/v79/v83/v84/v87/v92/v95, JMS v185). Since only existing writers are used, this is a verification obligation, not new per-version code. Grounding for the legacy columns added after first draft:
  - **Wire encoding is version-safe.** COMBO is at bit21 of `mask.L`, read by all clients including v48 (IDA-verified); the re-broadcast's "remaining duration" (FR-5) holds on legacy too because the legacy duration path (`legacyDurationUnits`, `character_temporary_stat.go:688`) and the modern path (`:666`) both compute the value relative to `time.Now()`.
  - **The gain gate is self-protecting.** Emission is gated channel-side on *owning Combo Attack at level > 0*; on any version where the class/skill does not exist (e.g. Dawn Warrior on pre-Cygnus v48/v61), the path is a no-op with no commands and no errors. Safe by construction, not by per-version code.
  - **Two items are unverified for legacy and must not be assumed working:** (a) whether the legacy clients (v12/v48/v61/v72/v79) actually *render* orbs from the COMBO stat (only the v83 client was reverse-verified), and (b) Dawn Warrior does not exist pre-Cygnus (v48/v61) — its absence is a safe no-op, not coverage. Verify (a) in-game per legacy column, or explicitly record it as deferred client-render verification.
- **No polling**: the channel must not query atlas-buffs REST per attack; current buff state should come from whatever cached/local view the channel already maintains, or the value computation moves into atlas-buffs (design decision).

## 9. Open Questions

1. Set-vs-delta command semantics (FR-4): absolute set computed by the channel vs. an `INCREMENT`/`RESET` command computed inside atlas-buffs. Delta-in-buffs is likely safer for concurrency (the roll for double-orb still happens channel-side where skill levels are known — or effect data moves with the command). Design phase decides.
2. Where the channel reads the current COMBO value and the character's Combo/Advanced Combo skill levels from without per-attack REST calls (existing session/character cache vs. pushing the gain logic into atlas-buffs).
3. Whether `x`/`prop` for the governing effect are already available to atlas-channel via its existing skill-effect lookups (they exist in atlas-data; confirm the REST model surfaces them).
4. GM behavior: Cosmic substitutes max skill level when a GM has no Combo levels. Decide whether to replicate or ignore (recommend ignore — no Atlas precedent for GM-specific combat paths).

## 10. Acceptance Criteria

- [ ] Casting Combo Attack applies COMBO=1 (unchanged); landing a non-finisher melee attack that hits ≥1 monster increments the orb display for the owner **and** for another client in the same map.
- [ ] Orb count never exceeds `x` orbs (stat value `x+1`) for the governing effect: 5 for maxed Combo Attack without Advanced Combo, 10 with maxed Advanced Combo.
- [ ] With Advanced Combo learned, double-orb gains occur (statistically, per `prop`), and never push the value past the cap.
- [ ] Casting Panic or Coma (any of the six finisher IDs) resets orbs to zero for owner and observers, even when orbs < max; the attack itself is never rejected.
- [ ] Shout (1111008) does not gain orbs.
- [ ] Attacks with zero monsters hit do not gain orbs.
- [ ] Dawn Warrior mirror works with the Cygnus skill IDs.
- [ ] Attacking without the COMBO buff active produces no buff commands and no errors.
- [ ] Buff expiry mid-combo clears orbs client-side with no server error on the next attack (no-op command path).
- [ ] The re-broadcast carries remaining duration — the buff does not extend its lifetime on each orb gain.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel and atlas-buffs; `docker buildx bake atlas-channel atlas-buffs` clean; `tools/redis-key-guard.sh` clean.
- [ ] Verified in-game on a v83 tenant (orb gain, double proc, finisher consume, foreign visibility).
- [ ] On each supported version, a combo-capable attacker with the buff active either shows orbs updating **or** the version is recorded as deferred client-render verification — no version silently assumed working. Pre-Cygnus columns (v48/v61) confirmed no-op for Dawn Warrior IDs (no commands, no errors).
