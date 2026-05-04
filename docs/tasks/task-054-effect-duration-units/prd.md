# Effect Duration Units — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-03

---

## 1. Overview

`atlas-data` produces skill-effect durations in raw seconds (the value of
the wz `time` attribute) instead of milliseconds, due to an inverted
multiply-by-1000 at `services/atlas-data/atlas.com/data/skill/reader.go:164-169`:

```go
if e.Duration() > -1 {
    e.SetOverTime(true)               // sets the flag, never multiplies
} else {
    e.SetDuration(e.Duration() * 1000) // only fires when time was missing (-1)
    e.SetOverTime(overTime)
}
```

Two downstream consumers interpret the same `Duration` field with
different units, accidentally matching each other only because of the
upstream bug:

- `atlas-buffs/buff/model.go:112` treats it as **seconds**:
  `time.Now().Add(time.Duration(duration) * time.Second)` — currently
  correct on the wire because atlas-data delivers raw seconds.
- `atlas-monsters/kafka/consumer/monster/consumer.go` treats it as
  **milliseconds**:
  `time.Duration(c.Body.Duration) * time.Millisecond` — currently
  ~1000× too short because atlas-data delivers raw seconds.

The mismatch was surfaced by task-047 (Priest Doom), where a
60-second polymorph instead applies and expires within ~60ms (visible
as no in-game effect at all). Every monster-status skill (Stun,
Freeze, Poison, Seal, Showdown, etc.) inherits the same bug and will
fail the same way as those skills are implemented.

This task establishes **milliseconds as the single canonical unit** for
effect duration across atlas-data → atlas-channel → {atlas-buffs,
atlas-monsters}, fixes the inverted reader logic, updates atlas-buffs
to interpret the field as ms, and ships both changes in a single
coordinated PR.

## 2. Goals

Primary goals:

- atlas-data's `effect.RestModel.Duration` (and the channel-side
  `effect.Model.Duration()` it produces) is always in **milliseconds**,
  matching Cosmic v83's internal convention and Go's `time.Duration`
  arithmetic (`time.Duration(n) * time.Millisecond`).
- Every monster-status duration (Doom, Stun, Freeze, Poison, Seal,
  Showdown, etc.) lasts the wz-data-specified number of seconds in-game,
  not ~1000× shorter.
- Buff durations (Holy Symbol, Bless, Hyper Body, Iron Body, Meditation,
  the various MapleWarrior variants, etc.) continue to last the
  wz-data-specified seconds — atlas-buffs is updated to interpret the
  new ms value correctly, so the wall-clock outcome is preserved.
- Tests across atlas-data, atlas-buffs, and atlas-channel pin the
  ms-based contract so future drift surfaces immediately.
- The fix lands as one coordinated PR (atlas-data + atlas-buffs in
  lockstep) so there is no interim window where buffs misbehave.

Non-goals:

- Cooldown unit (`cooltime` XML attr at `reader.go:154`). Tracked as a
  TODO; will be its own task.
- Renaming `Duration` → `DurationMs`. Considered, declined; the contract
  is documented via comment and pinned by tests.
- Any change to `libs/atlas-packet`. Wire format for MonsterStatSet
  encodes `monsterStatExpiry` as the constant `-1`; unit-independent.
- Any new skills, mechanics, or game features.
- Any client-facing protocol changes.
- Persisted-state migration. atlas-buffs holds buff entries in an
  in-memory `Registry` keyed by character; durations are not persisted
  to a database. Validated below in §6.

## 3. User Stories

- As a Priest player, when I cast Doom on a mob, the snail morph lasts
  the full skill duration (e.g., 60 seconds at level 30) instead of
  flickering for 20 ms.
- As a future implementer of Stun / Freeze / Poison / Seal / Showdown,
  I get the wz-data-specified duration for free without each skill
  needing its own ms multiplication band-aid in atlas-data.
- As a player using a buff skill (Bless, Holy Symbol, Iron Body), the
  buff still lasts the same wall-clock time as before this task — the
  change is unit-internal.
- As a developer reading `effect.Duration()`, the unit is unambiguous
  (documented as ms) and matches the convention used elsewhere in the
  codebase.

## 4. Functional Requirements

### 4.1 atlas-data reader: emit Duration in ms

- `getEffect()` in
  `services/atlas-data/atlas.com/data/skill/reader.go` reads the wz
  `time` attribute (default `-1` if missing) and stores it via
  `SetDuration(...)`. Today the value is left as-is (raw seconds) when
  `time` is set, and incorrectly multiplied by `-1000` when missing.
- After this task: when `time` is set (`> -1`), `SetDuration` is
  immediately followed by `SetDuration(e.Duration() * 1000)` to convert
  to ms; `SetOverTime(true)` continues to fire. When `time` is missing
  (`== -1`), `Duration` stays at `-1` (sentinel), `SetOverTime(overTime)`
  fires; the spurious `* 1000` is removed.
- The `OverTime` flag's meaning is unchanged: "this effect carries a
  duration the runtime should track."
- The FREEZE special-case `e.SetDuration(e.Duration() * 2)` at
  `reader.go:346` continues to work (unit-independent doubling).
- The Aran SnowCharge mapping `produceBuffStatAmount(..., e.Duration())`
  at `reader.go:373` passes the raw value as the buff stat amount.
  Audit needed (§6); if the consumer treats this as seconds, scale or
  document accordingly.

### 4.2 atlas-buffs: interpret Duration as ms

- `services/atlas-buffs/atlas.com/buffs/buff/model.go:112` currently
  computes the buff `expiresAt` as
  `time.Now().Add(time.Duration(duration) * time.Second)`. After this
  task: `time.Now().Add(time.Duration(duration) * time.Millisecond)`.
- This is the only code site in atlas-buffs that interprets the
  duration unit; downstream timer scheduling
  (`tasks/expiration.go:31` already uses `time.Millisecond`) is unaffected.

### 4.3 atlas-channel: forward unchanged

- `services/atlas-channel/atlas.com/channel/skill/handler/common.go`
  passes `e.Duration()` directly to `buff.NewProcessor(...).Apply(...)`
  and to `mp.ApplyStatus(...)`. After this task both consumers expect
  ms, so the existing pass-through is correct.
- Same for `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`
  (the `processDamageInfoEntry` helper).
- Same for the Doom per-skill handler at
  `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go:122`.
- No production code change in atlas-channel; the audit confirms each
  call site is unit-correct under the new contract.

### 4.4 atlas-monsters: verify, no production change

- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`
  already constructs status effects with
  `time.Duration(c.Body.Duration)*time.Millisecond`. Correct under the
  new contract; no change.
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go`'s
  `ApplyStatusEffect` accepts the `time.Duration` and schedules expiry
  via the existing status-task subsystem; no unit assumption at this
  layer.
- The DOOM-specific tests added in task-047
  (`TestApplyStatusEffect_Doom_*`) pass `60000` as ms and expect the
  status to apply with that duration. No change.

### 4.5 Tests

#### atlas-data
- New: `TestReader_TimeAttributeEmittedAsMilliseconds` — given an XML
  fixture with `<int name="time" value="60"/>`, the produced effect's
  `Duration` is `60000`, not `60`. Cover both monster-status branches
  (Doom) and buff branches (Iron Body or similar).
- Update: existing reader tests that assert specific `Duration` values
  (audit needed; likely `TestReader_PriestDoom_MapsDoomStatus` or any
  fixture that asserts a numeric Duration) to use ms.
- New: `TestReader_TimeMissing_DurationStaysSentinel` — given XML with
  no `time` attribute, `Duration == -1` and `OverTime` reflects the
  caller-passed `overTime` arg. Pins the bug-free behavior of the now-
  cleaned-up else branch.
- New: `TestReader_FreezeDoublesDuration` — given a Cold Beam-like
  fixture, the produced effect's `Duration` is `2 *  time * 1000`.
  Pins that the FREEZE doubling continues to operate on ms.

#### atlas-buffs
- Update: any tests that pass numeric durations to `buff.NewBuff` or
  similar — their expected `expiresAt` math changes from `* time.Second`
  to `* time.Millisecond`. Audit and adjust.
- New (or strengthened): a test that constructs a buff with
  `duration=60000` and asserts `expiresAt - createdAt ≈ 60s`. Pins the
  ms contract.

#### atlas-channel
- Existing tests at the per-skill handler layer use durations like
  `60000` already (per task-047). No change expected; verify no
  regression.

#### atlas-monsters
- Existing `TestApplyStatusEffect_Doom_*` tests pass `60000` ms. No
  change expected.

### 4.6 Documentation

- Add a doc comment on `effect.Model.Duration()` (in
  `services/atlas-data/atlas.com/data/skill/effect/model.go` and
  the channel-side `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go`)
  stating the unit is **milliseconds** and noting the `-1` sentinel.
- Add a `// Why ms:` comment near the reader's `* 1000` conversion
  pointing at this task and Cosmic parity (`StatEffect.java` constructor).
- `docs/TODO.md`: add an entry for the cooldown unit follow-up
  (see §11).

## 5. API Surface

No JSON:API endpoint additions or modifications. The
`effect.RestModel.Duration` JSON field changes its **semantic unit**
from "raw wz seconds (when set) / -1000 (when missing)" to "milliseconds
(when set) / -1 (when missing)". This is a **silent contract change**
on a JSON field whose name and JSON type are unchanged.

Mitigations:
- Land atlas-data and atlas-buffs in the same PR; deploy together
  (single coordinated rollout).
- Tests across services (§4.5) pin the new contract.

## 6. Data Model

### 6.1 Field types
- `effect.RestModel.Duration int32` — unchanged.
- `effect.Model.duration int32` — unchanged.
- atlas-buffs' `buff.Model.duration int32` — unchanged.

### 6.2 Persistence audit
- atlas-buffs uses an in-memory `Registry` keyed by character id;
  buffs do not persist across service restarts (the codebase has no
  Redis/Postgres persistence layer for buffs).
- atlas-monsters status effects are held in the in-memory monster
  registry plus Redis-backed monster snapshots
  (`monster/registry.go`); status effects are reconstructed when
  the monster is loaded but the duration is recomputed from
  `expiresAt` (a wall-clock timestamp), not from the original
  `duration` value. Validate this assumption in the implementation
  task.
- Conclusion: no migration needed for in-flight values, but the
  implementation plan includes a verification step. If the audit
  surfaces a persisted `duration` in seconds anywhere, that store
  needs a one-off migration script.

### 6.3 Kafka payloads
- `monsterApplyStatusCommandBody.Duration uint32` — interpreted as ms
  by the consumer; producer (atlas-channel) forwards `effect.Duration()`
  directly. After the fix, both ends agree on ms. No schema change.
- Buff-side Kafka events (`appliedStatusEvent`, `expiredStatusEvent`)
  carry `Duration int32`. Producers in atlas-buffs (`character/producer.go`)
  pass the value through unchanged. Consumers should be audited for
  unit assumptions.

## 7. Service Impact

| Service | Change |
|---|---|
| atlas-data | Production: invert + clean up the `if/else` at `reader.go:164-169`. Tests: 3 new + audit existing fixtures. Doc: comment on `Duration()` accessor. |
| atlas-buffs | Production: `* time.Second` → `* time.Millisecond` at `buff/model.go:112`. Tests: audit + adjust + add ms-pinning test. |
| atlas-channel | None (production). Audit only — confirm `effect.Duration()` callers (`skill/handler/common.go`, `skill/handler/doom/doom.go`, `socket/handler/character_attack_common.go`) all forward unchanged and downstream interpretation matches ms. |
| atlas-monsters | None (production). Audit only — confirm `kafka/consumer/monster/consumer.go` continues to interpret as ms. |
| atlas-character | Audit only — verify the cooldown application path is **separate** from the duration path and remains unchanged (cooldown is a follow-up TODO). |
| atlas-buffs Kafka consumers (downstream of atlas-buffs) | Audit only — confirm no consumer of `appliedStatusEvent.Duration` reinterprets the unit. |

## 8. Non-Functional Requirements

- **Backwards compat**: silent contract change on the JSON `Duration`
  field. Mitigated by single coordinated PR.
- **Multi-tenancy**: unaffected; durations are tenant-agnostic data
  derived from wz files which are themselves tenant-scoped via the
  existing data load path.
- **Observability**: no new log lines required. The existing
  `Doom: caster=…` Debugf at `services/atlas-channel/atlas.com/channel/monster/processor.go:73`
  now logs the correct ms value, which is itself a useful regression
  signal.
- **Performance**: no impact. Pure value-conversion change in atlas-data;
  one constant change in atlas-buffs.
- **Security**: no impact.
- **Rollout**: deploy atlas-data and atlas-buffs together. If atlas-buffs
  ships first, every buff lasts ~1000× shorter (catastrophic). If
  atlas-data ships first, every buff lasts ~1000× longer (also
  catastrophic). Single coordinated PR + atomic deploy is required.

## 9. Open Questions

1. **Persistence audit**: confirmed no schema changes expected. If the
   atlas-monsters Redis snapshots store raw `duration` (seconds) for
   persisted-then-reloaded status effects, the snapshots would re-
   apply with a 1000× shorter timer post-fix. Implementation must
   verify by tracing `monster.StatusEffect` serialization.
2. **Aran SnowCharge `produceBuffStatAmount(..., e.Duration())`** at
   `reader.go:373`: passes the raw duration value as a buff stat
   amount. After the fix, this stat amount is 1000× larger. Need to
   confirm the buff-stat consumer expects seconds vs ms. If it expects
   seconds, divide by 1000 inside the SnowCharge branch as a local
   compensation.
3. **Other reader.go consumers of `e.Duration()` that perform math on
   the value** (multiplication, comparison): the audit step in the
   implementation plan must enumerate them and adjust.

## 10. Acceptance Criteria

In-game (manual verification):
1. Cast a buff (e.g., Bless / Holy Symbol / Iron Body / Meditation).
   Buff lasts the same wall-clock time as before this task. (Regression
   guard for atlas-buffs.)
2. Cast Priest Doom on a mob. The snail morph persists for the full
   skill duration (60 seconds at level 30), not flickering. Confirms
   atlas-monsters now receives a correct ms value end-to-end.
3. (When the next monster-status skill is implemented — Stun / Freeze /
   Poison / Seal — that skill's status persists for the wz-data
   duration without any per-skill workaround.)

Code (automated):
- `services/atlas-data/atlas.com/data` builds and tests pass.
- `services/atlas-buffs/atlas.com/buffs` builds and tests pass.
- `services/atlas-channel/atlas.com/channel` builds and tests pass.
- `services/atlas-monsters/atlas.com/monsters` builds and tests pass.
- The new reader tests pin: `time=60` → `Duration=60000`; `time` missing
  → `Duration=-1`; FREEZE branch doubles ms.
- The new buffs test pins: `duration=60000` → `expiresAt - createdAt ≈ 60s`.

## 11. Follow-ups (out of scope)

- **Cooldown unit normalization**: `cooltime` XML attr at `reader.go:154`
  is read directly into `Cooldown uint32` with no conversion. Cooldown
  flows through atlas-character via the skill subsystem; unit semantics
  there need a separate audit + fix. Filed in `docs/TODO.md` as
  "Skill effect cooldown unit normalization (post task-054)".
- **Field rename `Duration` → `DurationMs`**: declined for this task to
  contain blast radius. If the codebase later grows another duration
  field with a different unit (e.g., DOT tick interval, animation time),
  reconsider the rename.
