# Priest Dispel — Party Debuff Cure — Design

Task: task-163-priest-dispel-party
PRD: docs/tasks/task-163-priest-dispel-party/prd.md (approved)
Status: Proposed
Created: 2026-07-10

---

## 1. Problem Recap

Priest Dispel (`2311001`) is dual-effect. The mob half (cancel mob buffs,
magic-reflect-aware) is already implemented in `applyToMobs`
(`services/atlas-channel/atlas.com/channel/skill/handler/common.go`). The party
half — cure CURSE, DARKNESS, POISON, SEAL, WEAKEN, SLOW on the caster and
bitmap-selected in-map party members — is missing. `atlas-buffs` already
consumes `CANCEL_BY_TYPES` and emits the `EXPIRED` status events that
`atlas-channel` already converts to client buff-cancel packets. Only two pieces
are missing, both in `atlas-channel`:

1. A `CANCEL_BY_TYPES` producer on the channel-side buff processor.
2. A per-skill Dispel handler wiring recipient selection + prop roll to that
   producer.

## 2. Approaches Considered

### Approach A — per-skill handler subpackage + shared buff-processor method (RECOMMENDED)

New `skill/handler/dispel` subpackage registering via the existing per-skill
registry (`channelhandler.Register` in `init()`, blank-imported from
`registrations/registrations.go`), plus a `CancelByTypes` method on the
existing `character/buff` Processor following `Apply`'s curried
field-bound-first shape.

- Matches the established precedent exactly (`heal`, `mysticdoor`): the
  per-skill dispatcher at `common.go:117` already runs after `applyToMobs`, so
  the two Dispel halves stay independent with zero orchestrator changes.
- `CancelByTypes` lands on the shared processor where task-156 (SuperGM
  Heal+Dispel) can consume it with its own (wider) stat set.
- Testable with the same local-seam idiom heal uses (package-level func vars,
  overridden and `t.Cleanup`-restored in tests).

### Approach B — inline the party cure in `common.go` next to `applyToMobs`

Add a `if skill2.Is(sid, skill2.PriestDispelId) { curePartyDebuffs(...) }`
branch in `UseSkill`.

- Rejected: grows the orchestrator that the per-skill registry exists to
  shrink; every future cure-type skill (task-156, Hero's Will, cure items)
  would pile more special cases into `UseSkill`; testing requires driving the
  whole `UseSkill` pipeline instead of one handler.

### Approach C — push skill semantics into atlas-buffs (a `DISPEL` command)

Emit a skill-level command (`DISPEL`, carrying skillId/level) and let
atlas-buffs decide the stat set and prop roll.

- Rejected: atlas-buffs is deliberately a dumb buff registry with a generic
  `CANCEL_BY_TYPES` already in place; skill semantics (which stats, per-recipient
  prop) are channel-side knowledge (WZ effect data lives there). PRD explicitly
  scopes atlas-buffs to zero changes. Would also duplicate the prop-roll seam
  server-side without access to `effect.Model`.

**Decision: Approach A.** It is what the PRD's FRs describe; the alternatives
exist to record why not.

## 3. Component Design

All changes in `services/atlas-channel/atlas.com/channel/`. No atlas-buffs
changes (FR-9).

### 3.1 Kafka message mirror — `kafka/message/buff/kafka.go`

Add to the existing const block and types, mirroring
`services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`:

```go
CommandTypeCancelByTypes = "CANCEL_BY_TYPES"

type CancelByTypesCommandBody struct {
    Types []string `json:"types"`
}
```

Wire shape (consumed unchanged by atlas-buffs today):

```json
{ "worldId": ..., "channelId": ..., "mapId": ..., "instance": ...,
  "characterId": N, "type": "CANCEL_BY_TYPES",
  "body": { "types": ["CURSE","DARKNESS","POISON","SEAL","WEAKEN","SLOW"] } }
```

### 3.2 Producer — `character/buff/producer.go`

```go
func CancelByTypesCommandProvider(f field.Model, characterId uint32, types []string) model.Provider[[]kafka.Message]
```

Keyed `producer.CreateKey(int(characterId))`, envelope fields from `f`,
identical in structure to the existing `CancelCommandProvider`.

### 3.3 Processor — `character/buff/processor.go`

Interface gains:

```go
CancelByTypes(f field.Model, types []charcon.TemporaryStatType) model.Operator[uint32]
```

- Curried per FR-2: field + types bound first, returns the per-character
  emitter — the same shape as `Apply`, so the handler calls it once per cast
  and invokes the operator per recipient.
- Takes the typed `charcon.TemporaryStatType` slice (type safety at call
  sites); the impl converts to `[]string` once and delegates to
  `CancelByTypesCommandProvider` via `producer.ProviderImpl` on
  `buff2.EnvCommandTopic`. Typed→wire conversion in the producer layer is the
  existing precedent (`Apply` converts `[]statup.Model` → `[]StatChange`).
- Emits a `Debugf` line per call mirroring `Apply`/`Cancel`.
- No mock exists for this processor (`character/buff/` has none); nothing to
  update there.

### 3.4 Dispel handler — `skill/handler/dispel/` (new package)

`dispel.go`:

```go
func init() { channelhandler.Register(skill2.PriestDispelId, Apply) }

// dispellableStatTypes is the exact Dispel cure set (FR-5, Cosmic
// Character.dispelDebuffs parity). Package-level constant slice; ZOMBIFY /
// SEDUCE / CONFUSE intentionally excluded (purgeDebuffs semantics, task-156).
var dispellableStatTypes = []charcon.TemporaryStatType{
    charcon.TemporaryStatTypeCurse,
    charcon.TemporaryStatTypeDarkness,
    charcon.TemporaryStatTypePoison,
    charcon.TemporaryStatTypeSeal,
    charcon.TemporaryStatTypeWeaken,
    charcon.TemporaryStatTypeSlow,
}
```

`Apply` implements `channelhandler.Handler`. Flow:

1. **Recipients** (FR-4): the caster id, plus
   `SelectPartyMembersInMap(l, ctx, f, characterId, info.AffectedPartyMemberBitmap())`
   — map-wide, no rectangle. Only ids are needed; unlike heal, no caster
   character load, no position, no effective-stats fetch. The selector already
   filters offline / other-map / no-session / dead members and applies the
   MSB-first bitmap decode.
2. **Emitter**: `op := cancelByTypesFunc(l, ctx, f, dispellableStatTypes)`
   (one processor call per cast).
3. **Per recipient** (FR-6, FR-7): roll `propRollFunc(e.Prop())`; on fail,
   increment `propSkipped` and continue (never fails the cast). On pass, call
   `op(recipientId)`; on emit error, log at error level with recipient id and
   continue (heal's per-recipient error pattern). Count `curesEmitted`.
4. **Summary** (FR-8): one `Debug` line with structured fields
   `caster, skill_id, skill_level, bitmap, recipients_selected, cures_emitted,
   prop_skipped` — the `buildSummaryFields` precedent, local to the package.
5. Return `nil` always — all failures are logged, none abort.

**Seams** (heal-style package-level func vars, test-overridden with
`t.Cleanup` restore):

```go
var selectPartyMembersFunc = channelhandler.SelectPartyMembersInMap
var propRollFunc = func(prop float64) bool { ... } // mirrors common.go:45 semantics exactly
var cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, types []charcon.TemporaryStatType) model.Operator[uint32] {
    return buff.NewProcessor(l, ctx).CancelByTypes(f, types)
}
```

`propRollFunc` is **mirrored**, not shared: the parent package's seam is an
unexported var the subpackage can neither call nor override, and exporting a
mutable seam from `handler` just for this would leak test plumbing across the
package boundary. The FR-6 language ("share or mirror") anticipated this;
mirroring is the heal precedent (heal owns its variance roll locally). The
mirror keeps the exact semantics: `prop <= 0` → false, `prop >= 1` → true,
else `rand.Float64() <= prop`.

**Prop semantics note**: `effect.Model.Prop()` is already normalized to
0.0–1.0 (`data/skill/effect/model.go:134`), so no /100 scaling in the handler.
WZ-verified 2311001 props are 34–100 → 0.34–1.0; the `prop <= 0 → false`
defensive arm is unreachable for this skill but intentionally kept identical
to the mob path (a zero-prop effect cures nobody, consistent with
`applyToMobs`).

### 3.5 Registration — `skill/handler/registrations/registrations.go`

Add the blank import:

```go
_ "atlas-channel/skill/handler/dispel" // Priest Dispel party cure — task-163
```

### 3.6 Explicit non-changes

- `common.go` untouched: `applyToMobs` already handles the mob half (including
  the magic-class reflect skip); the per-skill dispatcher already runs after
  it. Dispel's WZ effect has no statups/duration, so the generic party-buff
  apply at `common.go:107-111` never fires for it.
- No skill-use announce packets added: the PRD scopes none, and the mob half's
  current cast presentation is unchanged.
- `services/atlas-buffs/` — zero diffs (acceptance-gated).

## 4. Data Flow

```
client UseSkill packet (2311001)
  └─ UseSkill (common.go)
       ├─ applyToMobs — mob-buff cancel, reflect-aware   [existing, unchanged]
       └─ registry dispatch → dispel.Apply               [new]
            ├─ SelectPartyMembersInMap(bitmap)  → member ids
            ├─ per recipient (caster + members):
            │    propRoll(e.Prop()) pass? → CANCEL_BY_TYPES → COMMAND_TOPIC_CHARACTER_BUFF
            └─ summary debug log
atlas-buffs consumer (existing)
  └─ CancelByStatTypes → cancels intersecting buffs → EXPIRED status events
atlas-channel buff status consumer (existing)
  └─ EXPIRED → client buff-cancel packet (recipient + foreign)
```

## 5. Error Handling

| Failure | Behavior |
|---|---|
| Party load fails / caster partyless | Selector returns nil → caster-only cure (existing selector contract) |
| Empty / invalid bitmap (0 or ≥128) | Selector returns nil → caster-only cure (PRD open question 2 accepted) |
| Prop roll fails for a recipient | That recipient skipped; counted in `prop_skipped`; cast continues |
| Kafka emit fails for a recipient | Error log with recipient id; remaining recipients still processed (FR-7) |
| Recipient has no matching debuffs | atlas-buffs `CancelByStatTypes` no-ops for them (existing behavior); harmless |

The handler never returns an error that would surface to the cast path; the
dispatcher at `common.go:117` logs handler errors anyway, but Dispel logs its
own and returns nil.

## 6. Testing Strategy

All in `atlas-channel`, Builder-pattern setup, no `*_testhelpers.go`.

- **`dispel` handler tests** (seam injection, table-driven):
  - caster + N bitmap-selected members, all prop-pass → N+1 operator calls
    with exactly the six types, in caster-first order.
  - deterministic prop pattern (e.g. pass/fail alternating) → only passing
    recipients emitted; `prop_skipped` count correct; cast never errors.
  - emit error on recipient k → recipients k+1.. still emitted (FR-7).
  - selector receives the exact `(f, casterId, bitmap)` from the packet info;
    empty selector result → caster-only.
  - prop boundary cases: 0 → no cures; 1.0 → all cures (no RNG).
- **Registration test**: `channelhandler.Lookup(skill2.PriestDispelId)`
  returns the dispel handler after blank import (mirrors `registry_test.go`
  precedent).
- **Producer/message test**: `CancelByTypesCommandProvider` output —
  key = characterId, JSON field names (`type: "CANCEL_BY_TYPES"`,
  `body.types` string array) match the atlas-buffs consumer contract (asserted
  by literal JSON field names; cross-module import of atlas-buffs types is not
  possible across service modules).
- **Regression**: existing `common_apply_to_mobs_test.go` untouched and green
  (mob half byte-for-byte unchanged).

Acceptance (live): seed a debuff on a party member via a direct buff `APPLY`
command with a debuff stat type, cast Dispel, observe `CANCEL_BY_TYPES` →
`EXPIRED` → client buff-cancel packet (FR-9 end-to-end verification; no
mob→character infliction path exists yet, per PRD open question 1).

## 7. Verification Gates

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
  `services/atlas-channel/atlas.com/channel`.
- `docker buildx bake atlas-channel` from the worktree root.
- `tools/redis-key-guard.sh` clean from repo root.
- `git diff --stat` shows zero changes under `services/atlas-buffs/`.

## 8. Coordination

task-156 (`gm-hide-heal-dispel`, in design) plans the same channel-side
`CancelByTypes` producer (its FR-8) with a wider disease set. task-163 builds
the shared processor method + producer; the stat set stays handler-local so
each skill owns its own cure list. Whichever task lands second rebases onto
the shared producer instead of re-adding it.
