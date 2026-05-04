# Effect Duration Units — Design

Version: v1
Status: Draft
Created: 2026-05-03
Companion to: `prd.md`

---

## 1. Architecture

### 1.1 Contract

After this task, `effect.Model.Duration()` (and its REST counterpart
`effect.RestModel.Duration`) is in **milliseconds**, end-to-end. `-1` is
the "no duration" sentinel; all other values are positive ms counts.
atlas-data is the single conversion point. Downstream consumers
(atlas-buffs, atlas-monsters, atlas-channel handlers) interpret the
field as ms with `time.Duration(n) * time.Millisecond` and perform no
unit conversion of their own.

### 1.2 Why producer-side, not consumer-side

Producer-side conversion in atlas-data means the wz "raw seconds"
representation is contained inside `getEffect()`; every consumer reads
an unambiguous ms value off the wire. Consumer-side conversion would
force every new consumer (and every Kafka re-emission of `Duration`)
to re-derive the unit from the schema, perpetuating the latent class
of bug task-047 surfaced.

### 1.3 Why no typed wrapper, no field rename

PRD §11 declined `Duration` → `DurationMs` and a `type DurationMs
int32` wrapper. This design honors that decision: the unit is pinned
by tests (§4) and a doc comment on the accessor (§5), not by the type
system. Tradeoff: silent contract change is detectable only via tests,
not the compiler. Mitigation: ms-pinning tests at every layer (§4) and
the audit confirmations in §3.

### 1.4 Blast radius

Three behavior changes:

| Site | Change |
|---|---|
| `atlas-data/skill/reader.go:164-169` | Invert if/else so `* 1000` runs on the populated branch (today it runs on the missing branch). |
| `atlas-buffs/buff/model.go:112` | `time.Second` → `time.Millisecond`. |
| `atlas-buffs/buff/model_test.go:47` | Mirror the unit change in the existing assertion. |

Audit-only (no production code change):

- atlas-channel — four callers of `e.Duration()`, all forward unchanged.
- atlas-monsters — Kafka consumer already on ms; persistence already on ms.
- All other reader-side `e.Duration()` math sites are unit-independent
  or scoped out (§3).

### 1.5 Coordination & rollout

Single PR. atlas-data and atlas-buffs deploy together. No bridge
field, no tolerant reader, no runtime canary. The PR-level tests
(§4) catch the regression at CI; the atomic deploy contains the
runtime risk window. If the deploy is interrupted between services,
every buff is broken in one direction (ms-as-seconds = ~1000× too
short, or seconds-as-ms = ~1000× too long) — that is the explicit
trust-the-operator failure mode the PRD accepts.

---

## 2. Component-Level Changes

### 2.1 atlas-data

**File:** `services/atlas-data/atlas.com/data/skill/reader.go`

Lines 164-169 today:

```go
if e.Duration() > -1 {
    e.SetOverTime(true)
} else {
    e.SetDuration(e.Duration() * 1000)
    e.SetOverTime(overTime)
}
```

After:

```go
// Why ms: the wz `time` attribute is in seconds; convert here so
// downstream consumers (atlas-buffs, atlas-monsters) interpret
// effect.Duration() uniformly as time.Millisecond. See task-054.
if e.Duration() > -1 {
    e.SetDuration(e.Duration() * 1000)
    e.SetOverTime(true)
} else {
    e.SetOverTime(overTime)
}
```

The `else` branch loses its erroneous `* 1000` (which was multiplying
the `-1` sentinel into `-1000` — a bug pinned by the new sentinel
test in §4). `OverTime` semantics are unchanged: it remains "this
effect carries a duration the runtime should track."

**File:** `services/atlas-data/atlas.com/data/skill/effect/model.go`

Add a doc comment on the accessor:

```go
// Duration returns the effect duration in milliseconds. -1 is the
// "no duration" sentinel (the wz `time` attribute was missing).
// Positive values are ms counts converted from raw wz seconds at
// read time. Consumers should use time.Duration(d) * time.Millisecond.
func (m Model) Duration() int32 {
    return m.duration
}
```

Mirror the same comment in the channel-side model
(`services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:78-80`).

### 2.2 atlas-buffs

**File:** `services/atlas-buffs/atlas.com/buffs/buff/model.go:112`

```go
// before
expiresAt: time.Now().Add(time.Duration(duration) * time.Second),
// after
expiresAt: time.Now().Add(time.Duration(duration) * time.Millisecond),
```

This is the only production change in atlas-buffs. The expiration
ticker (`tasks/expiration.go:31`) already uses `time.Millisecond *
time.Duration(interval)` and is unaffected — it's the polling
cadence, not a duration interpretation.

### 2.3 atlas-channel — audit only

Four call sites forward `e.Duration()` directly to downstream APIs.
After this task all four callees expect ms; no change required:

| File:line | Behavior |
|---|---|
| `skill/handler/common.go:50` | `e.Duration() > 0` gate. Unit-independent. |
| `skill/handler/common.go:51` | `buff.NewProcessor(...).Apply(..., e.Duration(), ...)` → atlas-buffs Kafka producer. atlas-buffs interprets as ms. |
| `skill/handler/common.go:101` | `mp.ApplyStatus(..., uint32(e.Duration()))` → atlas-monsters Kafka producer. atlas-monsters interprets as ms. |
| `skill/handler/doom/doom.go:122` | Same as :101, in the per-skill Doom handler path. |
| `socket/handler/character_attack_common.go:127, :174` | Same as :101, in the magic-attack-with-status pipeline. |

### 2.4 atlas-monsters — audit only

`kafka/consumer/monster/consumer.go` constructs status effects with
`time.Duration(c.Body.Duration) * time.Millisecond` (lines 99, 116,
181, 198). Correct under the new contract; no change.

`monster/processor.go:869` (`executeStatBuff`) does
`time.Duration(sd.Duration()) * time.Second`, but `sd` is a
`mobskill.Model` — the **monster's own mob-skill data**, not the
character skill effect. Different `Duration()` source, different
unit story. Out of scope; flagged here so future readers don't
mistake it for a missed conversion.

`monster/registry.go:216` already deserializes the persisted snapshot
field `DurationMs` as ms. Persistence audit (PRD §6.2) closes:
status-effect snapshots are already ms-internal, no migration needed.

---

## 3. Edge Cases & PRD §9 Resolutions

### 3.1 PRD §9.1 — Persistence audit

**Resolved.** `monster/registry.go:216` uses a separate `DurationMs`
snapshot field already in ms. No migration script needed.

### 3.2 PRD §9.2 — Aran SnowCharge

**Deferred.** `reader.go:373` will not be modified in this task. After
the fix, `produceBuffStatAmount(..., e.Duration())` passes a 1000×
larger value than today, which is a regression on a code path that is
already semantically wrong (passing a duration as a charge stat
amount). Two outputs:

- Inline `// TODO(post-task-054): SnowCharge passes Duration as the
  WhiteKnightCharge stat amount, which is now 1000× larger after the
  ms conversion. The right fix is to pass the actual charge amount
  (likely e.X()), not Duration. Tracked in docs/TODO.md.` at
  `reader.go:373`.
- Entry in `docs/TODO.md` under a new "Skill effects" section:
  "SnowCharge stat amount uses Duration in ms after task-054; should
  use a charge-amount field. Pre-task-054 it was raw seconds — wrong
  but small. File: services/atlas-data/.../skill/reader.go:373."

### 3.3 PRD §9.3 — Other reader.go math sites on `e.Duration()`

**Enumeration complete.** The grep `e\.Duration\(\)` against
`reader.go` returns four sites:

| Line | Site | Status |
|---|---|---|
| 164 | `if e.Duration() > -1` | Unit-independent comparison. ✓ |
| 167 | `e.SetDuration(e.Duration() * 1000)` | The fix itself. |
| 346 | `e.SetDuration(e.Duration() * 2)` (FREEZE) | Unit-independent doubling; doubles ms instead of seconds. ✓ |
| 373 | `produceBuffStatAmount(..., e.Duration())` (SnowCharge) | Deferred per §3.2. |

No other math sites in atlas-data. PRD §9.3 closes.

### 3.4 FREEZE doubling

`reader.go:346` runs **after** the new `* 1000` conversion (the if/else
at 164-169 fires earlier in the function). FREEZE-class skills
(Cold Beam, Ice Strike, Blizzard, Element Composition, Frostprey,
Ice Splitter, Paralyze, Combo Tempest, Ice Breath) get `2 * time *
1000` ms — the doubled freeze duration carries through. Pinned by
the new FREEZE test in §4.

---

## 4. Test Pinning Strategy

The unit contract is pinned at three layers, each with a test the
reviewer can point at to defend the convention:

### 4.1 atlas-data (the source of truth)

`services/atlas-data/atlas.com/data/skill/reader_test.go` (or sibling):

- **`TestReader_TimeAttributeEmittedAsMilliseconds`** — XML fixture
  with `<int name="time" value="60"/>` → produced effect's
  `Duration == 60000`. Run for both monster-status and buff branches
  (Doom + Iron Body or similar) so the reviewer can see "ms applies
  on both paths."
- **`TestReader_TimeMissing_DurationStaysSentinel`** — XML with no
  `time` attribute → `Duration == -1`, `OverTime` reflects the
  caller-passed `overTime` arg. Pins the cleaned-up else branch.
- **`TestReader_FreezeDoublesDuration`** — Cold Beam-like fixture →
  `Duration == 2 * time * 1000`. Pins that FREEZE doubling operates
  on ms.

**Audit step:** any existing reader test that asserts a numeric
`Duration` is updated to ms. Likely candidates from grep:
`TestReader_PriestDoom_MapsDoomStatus`. Enumerated during planning.

### 4.2 atlas-buffs (the consumer)

`services/atlas-buffs/atlas.com/buffs/buff/model_test.go`:

- **Update existing assertion** at line 47:
  `expectedExpiry := b.CreatedAt().Add(time.Duration(duration) * time.Millisecond)`.
- **Strengthen / add** a test that calls
  `NewBuff(sourceId, level, duration=60000, changes)` and asserts
  `expiresAt - createdAt` is within ±50ms of 60s. Pins the ms
  contract at the consumer.

### 4.3 atlas-monsters (no production change, no new test)

The DOOM tests added in task-047 (`TestApplyStatusEffect_Doom_*`)
already pass `60000` ms and expect the status to apply with that
duration. They serve as the consumer-side pin.

### 4.4 atlas-channel (no production change, no new test)

Per-skill handler tests already use `60000` ms (per task-047). No
change expected; planning step verifies no regression.

### 4.5 What is *not* pinned

- The wire JSON value of `Duration` in atlas-data's HTTP responses.
  The reader-level test (atlas-data) plus the consumer-level test
  (atlas-buffs) provide unit coverage at both ends of the contract;
  an HTTP-integration assertion would be redundant given the
  reader's `RestModel.Duration` field is unconditionally derived from
  the model's `duration`.

---

## 5. Documentation

- `services/atlas-data/atlas.com/data/skill/effect/model.go` — doc
  comment on `Duration()` accessor (§2.1).
- `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:78-80` — mirror
  the doc comment (existing accessor).
- `services/atlas-data/atlas.com/data/skill/reader.go:164-169` —
  inline `// Why ms:` comment on the conversion (§2.1).
- `docs/TODO.md` — two new entries:
  - SnowCharge stat amount follow-up (§3.2).
  - Cooldown unit normalization follow-up (PRD §11; verbatim from
    PRD: "Skill effect cooldown unit normalization (post task-054)").

---

## 6. Plan Hand-off Notes

The plan phase will:

1. Enumerate existing reader tests that assert numeric `Duration`,
   adjust them to ms.
2. Order the changes so the CI signal is honest: write the new
   ms-pinning tests first (failing), then flip the reader logic, then
   verify all tests pass before touching atlas-buffs.
3. Run the build for atlas-data, atlas-buffs, atlas-channel,
   atlas-monsters at completion (per CLAUDE.md "Build & Verification").
4. Confirm via grep that no Kafka consumer of `appliedStatusEvent`
   reinterprets `Duration` as seconds (a 5-minute audit, surfaces in
   the planning step).

---

## 7. Out of Scope

Restated for clarity (these come from the PRD):

- Cooldown unit normalization (`cooltime` at `reader.go:154`).
- `Duration` → `DurationMs` field rename.
- `libs/atlas-packet` wire format changes.
- New skills, mechanics, features.
- Any persisted-state migration (audited above; not needed).
- Aran SnowCharge correctness (deferred per §3.2).
