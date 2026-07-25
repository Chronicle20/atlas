# Combo Attack Orb Gain/Consume — Design

Task: task-142-combo-attack-orbs
Status: Approved PRD → design
Inputs: `docs/tasks/task-142-combo-attack-orbs/prd.md`

## 1. Summary

Orb state lives in atlas-buffs as the amount of the `COMBO` stat on the active Combo Attack buff. atlas-channel decides *whether and by how much* the value should change (it owns skill levels, effect data, and the double-orb roll) and emits a new **delta-style** Kafka command; atlas-buffs applies the mutation *atomically at the state* (clamped increment or absolute set), and emits a new **STAT_UPDATED** status event carrying the buff's original `createdAt`/`expiresAt`. atlas-channel's buff consumer announces that event through the existing GIVE_BUFF / GIVE_FOREIGN_BUFF writers — the packet layer already encodes duration relative to `now`, so the remaining-duration requirement (FR-5) falls out with zero writer changes.

> **Version note (post-draft):** `main` now carries **two** duration-encoding paths in `libs/atlas-packet/model/character_temporary_stat.go` — the modern path writes raw ms (`v.ExpiresAt().Sub(time.Now())`, `:666`) and a legacy path writes int16 units for pre-modern clients (`legacyDurationUnits`, `:688`). **Both compute the value relative to `time.Now()`**, so the "remaining duration, buff never extends" property holds identically across the full version set (GMS v12/v48/v61/v72/v79/v83/v84/v87/v92/v95, JMS v185). The cite in the sentence above was `:623` at draft time. Likewise, the COMBO stat itself is version-independent — registered unconditionally at bit21 of `mask.L` (bits 0–46, read by every client incl. v48; IDA-verified via `legacyGmsMask`), so no per-version writer work is needed for the legacy columns added after this design's first draft.

No new packets, no REST changes, no DB changes, no atlas-data changes.

## 2. Decisions on the PRD's open questions

### Q1 — Set vs. delta command semantics: **delta, applied inside atlas-buffs**

Three candidates were considered:

**(a) Absolute set computed by the channel** — rejected. The channel would need the current value before every attack (a per-attack read the NFRs ban), and two in-flight attacks reading the same value would emit the same target (lost increment *and* a stale overwrite of a concurrent finisher reset).

**(b) Delta command, mutation computed inside atlas-buffs (chosen).** The command says "increment the COMBO stat by N, clamped to cap C" or "set it to N". The channel never reads orb state at all. Ordering is guaranteed end-to-end: buff commands are produced with `key = characterId` (`services/atlas-channel/atlas.com/channel/character/buff/producer.go:22`), so all commands for one character land on one partition and are consumed in order by the one atlas-buffs group member that owns it — the same serialization that already protects Apply/Cancel's Redis get-modify-put (`services/atlas-buffs/atlas.com/buffs/character/registry.go:68-113`). A clamped increment is also self-limiting under any residual race: the value can never exceed the cap or fall below 1, which is exactly the PRD's acceptable/unacceptable line.

**(c) Move all gain logic (skill levels, effects, the roll) into atlas-buffs** — rejected. atlas-buffs has no character skill data and no atlas-data client; importing both to compute a roll breaks the service boundary for no concurrency benefit over (b), since the roll's *outcome* travels fine in a command.

The Advanced Combo double-orb roll stays channel-side (where skill levels and `prop` are known); the command carries the resulting amount (1 or 2) and the cap (`x + 1`).

### Q2 — Where the channel reads COMBO state / skill levels: **it doesn't read buff state at all**

`processAttack` already fetches the character with `SkillModelDecorator` (`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:272`), so Combo Attack / Advanced Combo levels are in hand for free. The "is the COMBO buff active" gate is delegated to atlas-buffs: the command is a logged no-op when no matching buff exists (required by FR-4 anyway, since the buff can expire between attack and command processing).

Channel-side emission is gated on *owning* Combo Attack at level > 0 (adventurer or Cygnus variant), so the only characters that ever emit speculative commands are Crusaders/Heroes/Dawn Warriors — who in practice keep the buff up. Alternatives rejected:

- Per-attack REST read of atlas-buffs — banned by the NFRs.
- A channel-local projection of buff status events — a new cache with a new invalidation surface (missed/reordered events → permanently wrong gate) to save Kafka messages that are cheap and already idempotent-safe.

### Q3 — `x`/`prop` availability: **confirmed, already surfaced**

`effect.Model` in atlas-channel exposes `X()` and `Prop()`; the MP Eater passive in the same file already consumes both via `data/skill.NewProcessor(l, ctx).GetEffect(...)` (`character_attack_common.go:231-253`). atlas-data normalizes `prop` to `[0,1]` at parse time (`services/atlas-data/atlas.com/data/skill/reader.go:149`), so the roll is `rand.Float64() < prop`, identical to `mpEaterShouldProc`. The effect lookup is one REST GET to atlas-data per qualifying attack — the same cost and pattern as MP Eater, and only paid by combo-capable attackers.

### Q4 — GM max-level substitution: **not replicated**

No Atlas precedent for GM-specific combat paths; a GM without Combo levels simply gains no orbs. (PRD's recommendation, adopted.)

## 3. Kafka contract

### New command (topic `COMMAND_TOPIC_CHARACTER_BUFF`, existing envelope)

```
CommandTypeUpdateStatValue = "UPDATE_STAT_VALUE"

type UpdateStatValueCommandBody struct {
    SourceId  int32  `json:"sourceId"`  // buff key, e.g. 1111002 / 11111001
    StatType  string `json:"statType"`  // "COMBO"
    Operation string `json:"operation"` // "INCREMENT" | "SET"
    Amount    int32  `json:"amount"`    // INCREMENT: delta (1|2); SET: absolute value (1)
    Cap       int32  `json:"cap"`       // INCREMENT only: max resulting value (x+1)
}
```

- `INCREMENT`: `newAmount = min(current + Amount, Cap)`; no-op (no event) if already at/above cap.
- `SET`: `newAmount = Amount`; no-op (no event) if unchanged. Finisher consume = `SET 1`.
- Missing/expired buff, or buff lacking the stat type: logged no-op at Debug (normal expiry race).

The body is stat-generic (nothing COMBO-specific) so future counters can reuse it, but only these two operations are implemented — no speculative extras.

### New status event (topic `EVENT_TOPIC_CHARACTER_BUFF_STATUS`)

```
EventStatusTypeStatUpdated = "STAT_UPDATED"

type StatUpdatedStatusEventBody struct {
    SourceId  int32        `json:"sourceId"`
    Level     byte         `json:"level"`
    Duration  int32        `json:"duration"`
    Changes   []StatChange `json:"changes"`   // full post-update stat set of the buff
    CreatedAt time.Time    `json:"createdAt"` // original — NOT reset
    ExpiresAt time.Time    `json:"expiresAt"` // original — NOT reset
}
```

Same shape as `ExpiredStatusEventBody` (`kafka/message/character/kafka.go:83-90`); emitted only when the value actually changed.

**Why a new type instead of re-emitting APPLIED:** `EVENT_TOPIC_CHARACTER_BUFF_STATUS` is also consumed by atlas-effective-stats, atlas-mounts, and atlas-rates. Their handlers treat APPLIED as "a buff came into existence" (stat recompute, mount start, rate change). A value tick on an existing buff is a different fact; a distinct type means those three services ignore it by their existing `if e.Type != …` guards with zero changes, and nobody re-runs apply side effects per attack. (COMBO's amount affects no derived stats, so effective-stats correctly has nothing to do.)

## 4. atlas-buffs changes

`services/atlas-buffs/atlas.com/buffs/`

1. **`buff/model.go`** — add an immutable copy-mutator:
   ```
   func (m Model) WithStatAmount(statType string, amount int32) (Model, bool)
   ```
   Returns a copy with that stat's amount replaced (preserving `id`, `level`, `duration`, `createdAt`, `expiresAt`, other stats); `false` when the stat type isn't present. Keeps mutation logic next to the model instead of in the registry.

2. **`character/registry.go`** — new method:
   ```
   func (r *Registry) UpdateStatValue(ctx, characterId uint32, sourceId int32, statType string,
       operation string, amount int32, cap int32) (buff.Model, bool, error)
   ```
   Get character model → look up `buffs[srcKey(sourceId)]` (COMBO buffs are non-accumulate; accumulate keys are out of scope) → skip if absent or `Expired()` → compute new amount per operation → `WithStatAmount` → if changed, write back with `Put` and return `(updated, true, nil)`; otherwise `(Model{}, false, nil)`. Same get-modify-put shape as `Cancel`, protected by the same partition-ordering guarantee.

3. **`character/processor.go`** — add `UpdateStatValue(worldId, characterId, sourceId, statType, operation, amount, cap) error` to the `Processor` interface + impl: call the registry; when changed, `message.Emit` a `STAT_UPDATED` event via a new `statUpdatedStatusEventProvider` in `character/producer.go` (mirroring `appliedStatusEventProvider`, minus `FromId`).

4. **`kafka/message/character/kafka.go`** — the command/event constants and bodies from §3.

5. **`kafka/consumer/character/consumer.go`** — register `handleUpdateStatValue` alongside the four existing handlers; standard type-guard + processor call.

6. Update any `character` Processor mocks/tests for the interface addition.

## 5. atlas-channel changes

`services/atlas-channel/atlas.com/channel/`

1. **`kafka/message/buff/kafka.go`** — mirror the new command/event constants and bodies (channel keeps its own copy of the shapes, as it does for APPLY/CANCEL).

2. **`character/buff/producer.go` + `processor.go`** — `UpdateStatValueCommandProvider(...)` keyed by `producer.CreateKey(int(characterId))` like the existing providers, and a `Processor.UpdateStatValue(f field.Model, characterId uint32, sourceId int32, statType string, operation string, amount int32, cap int32) error` that emits it.

3. **New file `socket/handler/character_attack_combo.go`** — the orb logic, structured like the MP Eater block for testability (pure helpers + one side-effecting entry point):

   - `comboSkillIds(skills []skill.Model) (comboId skill3.Id, comboLevel byte, advId skill3.Id, advLevel byte, ok bool)` — resolves the adventurer vs Cygnus constant set from owned skills (`CrusaderComboAttackId`/`DawnWarriorStage3ComboAttackId`; `HeroAdvancedComboAttackId`/`DawnWarriorStage3AdvancedComboId`). `ok == false` when Combo Attack isn't owned at level > 0.
   - `isComboFinisher(id skill3.Id) bool` — the six finisher IDs from FR-2.
   - `comboGainAmount(advLearned bool, prop float64, roll float64) int32` — 1, or 2 on a successful Advanced Combo roll (pure; roll injected like `mpEaterShouldProc`).
   - `comboOrbTryUpdate(l, ctx, c character.Model, ai packetmodel.AttackInfo, f field.Model)` — the entry point:
     1. Resolve the combo line; return if not owned.
     2. If `isComboFinisher(ai.SkillId())`: emit `SET 1` for the line's Combo Attack sourceId — unconditionally (no hit requirement, no orb-count requirement, attack never rejected — FR-3), then return.
     3. Return if `ai.SkillId() == skill3.CrusaderShoutId` (1111008) or `len(ai.DamageInfo()) == 0`. Normal attacks (`SkillId() == 0`) qualify for gain.
     4. Governing effect: `GetEffect(advId, advLevel)` when `advLevel > 0`, else `GetEffect(comboId, comboLevel)`; on lookup error, log + return.
     5. Emit `INCREMENT` by `comboGainAmount(...)` with `cap = x + 1`. (Cosmic's "pre-roll value ≤ x" second-orb guard is exactly the clamp: `min(v + 2, x + 1)`.)
     - All emission errors are logged and swallowed — the attack pipeline never fails on orb bookkeeping (same policy as the projectile/MP Eater emits).

4. **`socket/handler/character_attack_common.go`** — replace the `// TODO apply combo orbs (add or consume)` line (`:500` on current `main`; was `:404` at draft — locate by the marker text) with a call to `comboOrbTryUpdate`, gated `ai.AttackType() == packetmodel.AttackTypeMelee` (close-range only, matching Cosmic's `CloseRangeDamageHandler`; the shared `processAttack` also serves ranged/magic/energy). Runs fire-and-forget after the attack broadcast, beside the projectile emit.

5. **`kafka/consumer/buff/consumer.go`** — register `handleStatusEventStatUpdated`: type-guard on `STAT_UPDATED`, world check, then the identical announce body as `handleStatusEventApplied` (build `buff.NewBuff(..., e.Body.CreatedAt, e.Body.ExpiresAt)`, `CharacterBuffGiveWriter` to the owner, `CharacterBuffGiveForeignWriter` to `ForOtherSessionsInMap`). Extract the shared announce into a helper reused by both handlers rather than duplicating it. Because the event carries the *original* `expiresAt` and the writer encodes duration relative to `now` (modern and legacy paths alike — see the version note in §1), the client receives the remaining duration and the buff never extends (FR-5 / acceptance criterion 10), on every supported version.

## 6. Data flow

```
melee attack packet
  → processAttack (damage, statuses, broadcast)
  → comboOrbTryUpdate                                   [atlas-channel]
      gain:    UPDATE_STAT_VALUE INCREMENT n cap    ─┐
      consume: UPDATE_STAT_VALUE SET 1              ─┤ key=characterId
                                                     ▼
  COMMAND_TOPIC_CHARACTER_BUFF ──────────────► handleUpdateStatValue   [atlas-buffs]
                                                registry.UpdateStatValue (clamp/no-op)
                                                changed? → STAT_UPDATED (orig createdAt/expiresAt)
                                                     ▼
  EVENT_TOPIC_CHARACTER_BUFF_STATUS ─────────► handleStatusEventStatUpdated  [atlas-channel]
                                                GIVE_BUFF → owner (remaining duration)
                                                GIVE_FOREIGN_BUFF → others in map
```

## 7. Error handling

| Failure | Behavior |
|---|---|
| Governing effect REST lookup fails | log Error, skip orb update; attack unaffected |
| Command emission fails | log Error, swallow; attack unaffected |
| Buff missing/expired at command time | Debug no-op in atlas-buffs, no event |
| Value already at cap / SET to same value | no mutation, no event, no packets |
| Registry Redis error | error logged by consumer handler (existing pattern) |

## 8. Testing

**atlas-channel** (patterns from `character_attack_common_test.go` / `character_attack_mp_eater_test.go`):
- Table tests for `comboSkillIds` (adventurer, Cygnus, neither, Combo at level 0), `isComboFinisher` (all six + negatives), `comboGainAmount` (no Advanced Combo, roll under/over `prop`, `prop ≥ 1`).
- Entry-point tests via injected emit closure: finisher emits SET 1 even with zero hits; Shout emits nothing; zero `DamageInfo` emits nothing; normal attack (skill 0) emits INCREMENT; governing-effect selection picks Advanced Combo when learned; cap value is `x + 1`.

**atlas-buffs** (patterns from `registry_test.go` / `processor_test.go`):
- `WithStatAmount`: replaces the right stat, preserves identity/expiry/other stats, `false` on missing type.
- `UpdateStatValue`: increments; clamps at cap; no-change at cap; SET resets; no-op on missing buff, expired buff, wrong sourceId, wrong stat type; `createdAt`/`expiresAt` unchanged after update.
- Processor: STAT_UPDATED emitted only on change; body carries original timestamps.

**Verification (per CLAUDE.md):** `go test -race ./...`, `go vet ./...`, `go build ./...` in atlas-channel and atlas-buffs; `docker buildx bake atlas-channel atlas-buffs`; `tools/redis-key-guard.sh`. In-game on v83: orb gain, double proc, finisher consume, foreign visibility, no buff-duration extension (PRD acceptance criteria).

## 9. Out of scope (reaffirmed)

Server-side damage scaling/validation, Aran combo systems and SHOW_COMBO, Energy Charge, finisher rejection on zero orbs, `isComboReset` (Aran-only), accumulate-mode stat updates, GM level substitution.

**Legacy version note:** the pre-Big-Bang columns (v12/v48/v61/v72/v79) are *in scope* — no code change absorbs them (self-protecting gate + version-independent COMBO stat). What remains a per-version *verification* item, not a code item: whether each legacy client visually renders orbs from the COMBO stat (only v83 was reverse-verified). Dawn Warrior does not exist pre-Cygnus (v48/v61); its skill IDs are a safe no-op there.
