# Effect Duration Units — Implementation Context

> Companion to `prd.md`, `design.md`, `plan.md`. Captures every file touched
> or audited and the unit-interpretation status of each call site so the
> executor can resolve ambiguity without re-deriving the audit.

---

## Source of truth

- PRD: `docs/tasks/task-054-effect-duration-units/prd.md`
- Design: `docs/tasks/task-054-effect-duration-units/design.md`
- Plan: `docs/tasks/task-054-effect-duration-units/plan.md`

The unit contract this task establishes: `effect.Model.Duration()` and
its REST/Kafka counterparts are in **milliseconds** end-to-end. `-1` is
the "no duration" sentinel. atlas-data is the single conversion point.

---

## Files modified (production)

| File | Change |
|---|---|
| `services/atlas-data/atlas.com/data/skill/reader.go:164-169` | Invert if/else; `* 1000` runs on the `> -1` branch. Remove the spurious `* 1000` from the `else`. |
| `services/atlas-data/atlas.com/data/skill/reader.go:373` | Add inline `// TODO(post-task-054): SnowCharge stat amount` comment. |
| `services/atlas-data/atlas.com/data/skill/effect/model.go` | Add doc comment on `Duration()` accessor (no such accessor today on `Model` — task adds the accessor and comment together). Note: today the package only exposes `RestModel` + `ModelBuilder.Duration()`. The plan adds the comment to `ModelBuilder.Duration()` (line 165) since that is the in-package accessor. |
| `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:78-80` | Add doc comment on `Model.Duration()`. |
| `services/atlas-buffs/atlas.com/buffs/buff/model.go:112` | `time.Second` → `time.Millisecond`. |

## Files modified (tests)

| File | Change |
|---|---|
| `services/atlas-data/atlas.com/data/skill/reader_test.go:2822-2823` | `Duration != 30` → `Duration != 30000` (skill 1001 effect[0]). |
| `services/atlas-data/atlas.com/data/skill/reader_test.go:2835-2836` | `Duration != 30` → `Duration != 30000` (skill 1001 effect[1]). |
| `services/atlas-data/atlas.com/data/skill/reader_test.go:2848-2849` | `Duration != 30` → `Duration != 30000` (skill 1001 effect[2]). |
| `services/atlas-data/atlas.com/data/skill/reader_test.go:2868-2869` | `Duration != 4` → `Duration != 4000` (skill 1002 effect[0]). |
| `services/atlas-data/atlas.com/data/skill/reader_test.go:2881-2882` | `Duration != 8` → `Duration != 8000` (skill 1002 effect[1]). |
| `services/atlas-data/atlas.com/data/skill/reader_test.go:2894-2895` | `Duration != 12` → `Duration != 12000` (skill 1002 effect[2]). |
| `services/atlas-data/atlas.com/data/skill/reader_test.go` (append) | New: `TestReader_TimeAttributeEmittedAsMilliseconds`, `TestReader_TimeMissing_DurationStaysSentinel`, `TestReader_FreezeDoublesDuration`. |
| `services/atlas-buffs/atlas.com/buffs/buff/model_test.go:47` | `* time.Second` → `* time.Millisecond` in `expectedExpiry` calculation. |
| `services/atlas-buffs/atlas.com/buffs/buff/model_test.go` (append) | New: `TestBuff_DurationInMilliseconds` — pins ms contract with `duration=60000` → `expiresAt - createdAt ≈ 60s`. |

## Files modified (docs)

| File | Change |
|---|---|
| `docs/TODO.md` | Two new entries under a "Skill effects" section. |

---

## Files audited (no production change required)

### atlas-channel

All four `e.Duration()` callers forward unchanged; under the new contract every callee already expects ms.

| File:line | Behavior |
|---|---|
| `skill/handler/common.go:50` | `e.Duration() > 0` gate. Unit-independent. |
| `skill/handler/common.go:51` | `buff.NewProcessor(...).Apply(..., e.Duration(), ...)` → atlas-buffs (ms). |
| `skill/handler/common.go:101` | `mp.ApplyStatus(..., uint32(e.Duration()))` → atlas-monsters (ms). |
| `skill/handler/doom/doom.go:122` | Same as :101, Doom path. |
| `socket/handler/character_attack_common.go:127, :174` | Same as :101, magic-attack path. |

Other `Body.Duration` consumers — these are unrelated payloads with their own unit conventions; out of scope:

| File:line | Why out of scope |
|---|---|
| `kafka/consumer/buff/consumer.go:67, :103` | Just stores the value into `character/buff.NewBuff`; expiresAt is taken from the Kafka payload directly (already computed by atlas-buffs). No reinterpretation. |
| `kafka/consumer/instance_transport/consumer.go:66` | `e.Body.DurationSeconds` (different field, unit in name). |
| `kafka/consumer/mist/consumer.go:86` | Mist Kafka payload, separate `Duration` field, unit-passthrough on the wire. |
| `kafka/consumer/system_message/consumer.go:252` | System message guide-hint duration; default 7000ms suggests already ms. |
| `kafka/consumer/monster/consumer.go:423` | Already uses `time.Millisecond`. |

### atlas-monsters

| File:line | Status |
|---|---|
| `kafka/consumer/monster/consumer.go:99, :116, :181, :198` | Already `time.Duration(c.Body.Duration) * time.Millisecond`. Correct under the new contract. |
| `monster/registry.go:92, :124, :216` | Persistence uses a separate `DurationMs` snapshot field, already in ms. PRD §6.2 closes. |
| `monster/processor.go:869` (`executeStatBuff`) | `time.Duration(sd.Duration()) * time.Second` where `sd` is `mobskill.Model`. Different `Duration()` source — monster's own mob skills, not character effect. Out of scope. |

### atlas-buffs Kafka producer

`character/producer.go:33, :73` — `Duration: duration` passthrough into `appliedStatusEvent` / `expiredStatusEvent`. After this task, ms goes onto the wire; downstream atlas-channel `kafka/consumer/buff/consumer.go` only stores the value, no unit reinterpretation. Safe.

### atlas-data

The grep for `e\.Duration\(\)` in `reader.go` returns four sites:

| Line | Site | Status |
|---|---|---|
| 164 | `if e.Duration() > -1` | Unit-independent comparison. |
| 167 | `e.SetDuration(e.Duration() * 1000)` | The fix itself. |
| 346 | `e.SetDuration(e.Duration() * 2)` (FREEZE) | Unit-independent doubling — doubles ms instead of seconds, correct. |
| 373 | `produceBuffStatAmount(..., e.Duration())` (SnowCharge) | Wrong before, more wrong after. Deferred per design §3.2; tagged with TODO. |

---

## Key conventions (for the executor)

- **Atlas project rules (`CLAUDE.md`):** TDD; frequent commits; build all affected services after changes; never break service boundaries.
- **TDD ordering:** every test step writes the failing assertion first, runs to confirm red, then the implementation step makes it green. The atlas-data reader tests are the most subtle — the existing `1001`/`1002` assertions become red the moment they're updated to expect ms (because production code still does `time` raw seconds), then green after the reader flip.
- **Coordinated rollout:** atlas-data and atlas-buffs ship in the same PR. There is no intermediate state where one is on ms and the other on seconds; the plan's task order keeps them in lockstep within a single branch.
- **Comments:** keep them short. The design specifies a `// Why ms:` line on the reader conversion and a doc comment on each `Duration()` accessor — both stay terse, no multi-paragraph blocks.

## Build verification

After production changes, the four affected services must build and test green:

```
go test ./... # in each of:
  services/atlas-data/atlas.com/data
  services/atlas-buffs/atlas.com/buffs
  services/atlas-channel/atlas.com/channel
  services/atlas-monsters/atlas.com/monsters
```

A single failing test in any of these blocks the PR.

## Acceptance summary (from PRD §10)

- atlas-data, atlas-buffs, atlas-channel, atlas-monsters all build & test green.
- New reader tests pin: `time=60` → `Duration=60000`; `time` missing → `Duration=-1`; FREEZE doubles ms.
- New buffs test pins: `duration=60000` → `expiresAt - createdAt ≈ 60s`.
- In-game (manual, not part of automated plan): Bless/Holy Symbol/Iron Body unchanged wall-clock; Priest Doom snail morph lasts the full skill duration.
