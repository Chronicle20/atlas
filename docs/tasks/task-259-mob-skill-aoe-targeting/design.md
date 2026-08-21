# Mob Skill AoE Target Selection Fidelity — Design

Task: task-259-mob-skill-aoe-targeting
PRD: `docs/tasks/task-259-mob-skill-aoe-targeting/prd.md` (approved)
Status: Draft
Created: 2026-08-21

---

## 1. Summary

`ProcessorImpl.getDiseaseTargets` (`services/atlas-monsters/atlas.com/monsters/monster/processor.go:1281-1304`)
is replaced by a two-layer selector:

1. **A pure selection function** — `selectDiseaseTargets(mobX, mobY int16, sd mobskill.Model, skillId byte, candidates []positionedCharacter) []uint32` —
   holding all of the box test, the SEDUCE-only count cap, and the ordering rule. No I/O, no
   `math/rand`, a pure function of its arguments (FR-4.2).
2. **An I/O shell** — `ProcessorImpl.getDiseaseTargets(m Model, sd mobskill.Model, skillId byte) []uint32` —
   which decides single-target vs AoE, lists the field, resolves positions with bounded
   concurrency, and hands the ordered candidate list to the pure function.

The position data comes from a new read-only client package
`atlas-monsters/character/position`, wired into `ProcessorImpl` through a `positionFn` function
field, matching the `inFieldFn` / `hiddenFn` / `locationFn` seams already on the struct
(`processor.go:99-101`).

No schema change, no new endpoint, no change to any Kafka contract.

---

## 2. Current behavior and why it is wrong

```go
// processor.go:1281-1304 (today)
if !sd.HasBoundingBox() && sd.Count() <= 1 { /* controller */ }
ids, err := _map.NewProcessor(p.l, p.ctx).CharacterIdsInFieldProvider(m.Field())()
if sd.Count() > 0 && uint32(len(ids)) > sd.Count() {
    rand.Shuffle(...)
    ids = ids[:sd.Count()]
}
return ids
```

Four defects, one per PRD requirement group:

| Defect | Effect | Requirement |
|---|---|---|
| `lt`/`rb` used only as a boolean switch | every AoE mob skill hits the whole map | FR-1.x |
| `Count() <= 1` in the single-target guard | a boxless `count > 1` skill leaks into the field-wide branch | FR-2.1 |
| `count` cap applied to every AoE disease | non-seduce box diseases under-apply | FR-3.1 |
| `rand.Shuffle` before truncation | selection is non-deterministic and untestable | FR-4.x |

The same file already does the box arithmetic correctly for monster targets — `executeBuff`
(`processor.go:1168-1179`) and `executeHeal` (`processor.go:1199-1212`) both translate
`LtX/LtY/RbX/RbY` by the caster's `X()`/`Y()` and test `dx`/`dy` inclusively. This design reuses
that exact comparison form for characters so the two paths cannot drift.

---

## 3. Architecture

### 3.1 Data flow

```
executeDebuff / executeBanish / executeDispel   (skillId in hand)
        │
        ▼
p.getDiseaseTargets(m, sd, skillId)             — I/O shell
        │
        ├── no bounding box ──▶ [m.ControlCharacterId()] (or nil)      ← zero REST calls (FR-2.3)
        │
        └── bounding box
                ├── p.inFieldFn(m.Field())            → ordered []uint32   (atlas-maps)
                ├── resolvePositions(ids)             → ordered []positionedCharacter
                │       └── bounded fan-out over p.positionFn            (atlas-character)
                └── selectDiseaseTargets(...)         → []uint32          — pure
```

### 3.2 The pure selector

```go
// positionedCharacter pairs a character id with the world coordinates that
// were successfully resolved for it. Characters whose position could not be
// resolved never become a positionedCharacter (FR-1.4).
type positionedCharacter struct {
    id uint32
    x  int16
    y  int16
}

func selectDiseaseTargets(mobX, mobY int16, sd mobskill.Model, skillId byte, candidates []positionedCharacter) []uint32
```

Body, in order:

1. Filter by box, preserving input order:
   `dx := int32(c.x) - int32(mobX)`, `dy := int32(c.y) - int32(mobY)`,
   keep iff `dx >= sd.LtX() && dx <= sd.RbX() && dy >= sd.LtY() && dy <= sd.RbY()` (FR-1.2).
   Identical arithmetic and identical inclusive form to `executeHeal`. No facing mirror (FR-1.3).
2. Cap **only** for seduce:
   `if uint16(skillId) == monster2.SkillTypeSeduce && sd.Count() > 0 && uint32(len(in)) > sd.Count() { in = in[:sd.Count()] }`
   (FR-3.1, FR-3.2, FR-3.3). `monster2` is the existing `libs/atlas-constants/monster` import
   already in the file; no literal `128`.
3. Return the ids in surviving order (FR-4.1).

Determinism follows structurally: the function reads nothing but its arguments, and the only
ordering operation is a filter that preserves input order. There is no sort and no shuffle.

### 3.3 The I/O shell

```go
func (p *ProcessorImpl) getDiseaseTargets(m Model, sd mobskill.Model, skillId byte) []uint32 {
    if !sd.HasBoundingBox() {                       // FR-2.1 — count is irrelevant here
        if m.ControlCharacterId() == 0 {            // FR-2.2
            return nil
        }
        return []uint32{m.ControlCharacterId()}
    }
    ids, err := p.inFieldFn(m.Field())              // FR-5.4 — list failure ⇒ empty, error log
    if err != nil {
        p.l.WithError(err).Errorf("Unable to get characters in field for monster [%d] disease targeting.", m.UniqueId())
        return nil
    }
    candidates := p.resolvePositions(m.UniqueId(), ids)
    targets := selectDiseaseTargets(m.X(), m.Y(), sd, skillId, candidates)
    p.l.Debugf("Monster [%d] skill [%d] AoE: [%d] in field, [%d] positioned, [%d] targeted.",
        m.UniqueId(), skillId, len(ids), len(candidates), len(targets))
    return targets
}
```

Note the switch from the inline `_map.NewProcessor(...)` call to the existing `p.inFieldFn` seam
— every other field listing in this file already goes through it (`processor.go:485, 512, 1596,
1836`); `getDiseaseTargets` is the outlier. Routing through the seam is what makes the AoE path
testable without an httptest server, and it removes the last direct `_map` construction in the
skill path.

**There is deliberately no field-wide fallback.** If positions cannot be resolved, the candidate
set shrinks; it never widens back to "everyone in the field" (PRD §8, explicitly forbidden).

### 3.4 Bounded position fan-out

```go
// positionLookupConcurrency bounds the number of in-flight atlas-character
// reads for a single AoE cast. A crowded field must not serialize N round
// trips into the mob's skill execution path (FR-5.3), and must not open N
// sockets at once either.
const positionLookupConcurrency = 8

func (p *ProcessorImpl) resolvePositions(uniqueId uint32, ids []uint32) []positionedCharacter
```

Implementation: a pre-sized `[]*positionedCharacter` indexed by the character's position in `ids`,
a `sync.WaitGroup`, and a buffered `chan struct{}` of size `positionLookupConcurrency` as the
semaphore. Each worker writes only its own index — no mutex, no shared slice append. After the
wait, compact the slice in index order, dropping nils.

Indexing by input position is what makes concurrency compatible with FR-4.1/4.2: goroutines finish
in arbitrary order but the output order is the field-listing order, always.

Per-character failure: log at warn with the character id and the mob unique id (FR-1.4), leave the
slot nil, continue. One unresolvable character never aborts the cast.

**Rejected: `errgroup`.** `golang.org/x/sync` is an *indirect* dependency of `atlas-monsters`
today (`go.mod:40`). `errgroup.WithContext` also cancels siblings on first error, which is the
opposite of FR-1.4's degrade-don't-abort rule; `errgroup` without a context plus swallowed errors
would be the same twelve lines with an extra direct dependency. Stdlib semaphore it is.

**Rejected: a position cache.** PRD §8 permits one "if the design justifies its staleness window."
It cannot be justified: positions change continuously during combat, mob skills fire on an
interval measured in seconds, and the bounded fan-out already collapses the whole cast into one
round-trip's worth of wall clock. A cache would trade the exact fidelity this task exists to
restore for latency the fan-out already removed.

---

## 4. The character-position client

New package `services/atlas-monsters/atlas.com/monsters/character/position`, a sibling of the
existing `character/buff` and `character/hidden`. Files mirror
`services/atlas-maps/atlas.com/maps/character` one-for-one.

**`rest.go`** — minimal projection, `x`/`y` only (FR-5.2):

```go
type RestModel struct {
    Id uint32 `json:"-"`
    X  int16  `json:"x"`
    Y  int16  `json:"y"`
}
func (r RestModel) GetName() string { return "characters" }
// GetID / SetID / SetToOneReferenceID / SetToManyReferenceIDs per libs/atlas-rest/CLAUDE.md
```

`hp` is **not** projected — the dead-character filter is not adopted (§6, D5). Adding a field we
do not consume would violate FR-5.2.

**`requests.go`** — `requestById` against `requests.RootUrlFor(ctx, "CHARACTERS")`, with the same
`baseURLProvider` test seam atlas-maps uses so an httptest server can prove the projection parses
a real atlas-character payload.

**`processor.go`**:

```go
type Processor interface {
    // GetPosition returns the character's last known world coordinates.
    GetPosition(characterId uint32) (int16, int16, error)
}
```

**`mock/processor.go`** — `ProcessorMock` with a `GetPositionFunc` field and a
`var _ position.Processor = (*ProcessorMock)(nil)` assertion, matching `map/mock` and
`monster/mobskill/mock`.

Wiring in `NewProcessor` alongside the existing seams:

```go
p.positionFn = func(characterId uint32) (int16, int16, error) {
    return position.NewProcessor(p.l, p.ctx).GetPosition(characterId)
}
```

`atlas-character` already serves `x`/`y` on the `characters` resource
(`services/atlas-character/atlas.com/character/character/rest.go:49-50`) — no change there.
`CHARACTERS` service-root resolution is the same one `character/buff` and every other outbound
call in the service already relies on, so tenant scoping (PRD §8) comes for free through `p.ctx`.

### Alternatives considered

| Option | Verdict |
|---|---|
| **Per-character `GET /characters/{id}` (chosen)** | Zero cross-service surface change; the exact pattern `atlas-maps` already runs for its mist tick; bounded fan-out makes an N-character cast one round-trip deep. |
| Extend atlas-maps' in-field character list to carry `x`/`y` | Would collapse N+1 to 1 call, but atlas-maps is a *membership* authority, not a position authority — it would have to fan out to atlas-character itself, moving the same N calls one hop away while adding a public contract change and a second consumer to keep in sync. Rejected for this task; recorded below as the escalation path. |
| Bulk/filtered query on atlas-character (`?filter[ids]=…`) | Genuinely the right answer if N-call latency ever bites, but it is a new public endpoint on a service this task otherwise does not touch, and PRD §5 does not mandate one. Rejected as premature. |

**Escalation trigger, recorded so it is not re-derived later:** if a live cast on a full field
shows the position fan-out adding materially to skill-execution latency, the fix is the bulk
filter on atlas-character (option 3), and only `resolvePositions` changes — the pure selector and
every test around it are untouched by construction.

---

## 5. Caller changes

`getDiseaseTargets` needs the skill type for FR-3.1, and its three callers already hold it or know
it statically:

- `executeDebuff(m, sd, skillId, skillLevel)` — passes `skillId` through (already a parameter).
- `executeBanish(m, sd)` → `executeBanish(m, sd, skillId)`; called only from `executeDebuff`'s
  banish branch, which has `skillId` in hand.
- `executeDispel(m, sd)` → `executeDispel(m, sd, skillId)`; same.

Threading the real `skillId` rather than re-deriving the constant inside each function keeps the
selector's cap rule stated exactly once. Emission logic, topics, and payloads are untouched
(FR-6.1) — banish and dispel inherit box scoping and no cap purely because they share the
selector.

`math/rand` **stays** imported: `processor.go:709` uses `rand.Intn` for the basic-attack damage
formula. Only `rand.Shuffle` leaves. (PRD acceptance criterion says "removed if it has no other
use" — it has another use.)

---

## 6. Decisions on the PRD's open design questions

| # | Question | Decision | Rationale |
|---|---|---|---|
| D1 | Bulk vs per-character lookup (FR-5.3, §9) | Per-character, bounded at 8 | §4 table; escalation path recorded |
| D2 | Concurrency bound | 8 | Deep enough that a 40-character field is 5 sequential waves, shallow enough not to burst atlas-character on every mob tick across a busy channel |
| D3 | Position cache | None | §3.4 |
| D4 | How the skill type reaches the selector (FR-6.2) | Explicit `skillId byte` parameter | §5 |
| D5 | Dead-character filtering (FR-5.5) | **No filter** | No positive evidence either way; the current selector does not filter and neither does the reference `getPlayersInRange`. FR-5.5 says preserve current behavior absent evidence. Consequence: `hp` is not projected in the RestModel |
| D6 | Facing mirror (FR-1.3) | No mirror | Reference parity; remains an open question in the PRD, unchanged by this design |
| D7 | Seduce ordering | Field-listing order | Reference parity, documented approximation — see §8 |

---

## 7. Testing strategy

All new tests live in `services/atlas-monsters/atlas.com/monsters/monster/`, use the project's
Builder pattern for `Model`/`mobskill.Model` setup, and add no `*_testhelpers.go` file.

**Layer 1 — the pure selector** (`selectDiseaseTargets`, table-driven, no I/O at all). This is
where every acceptance criterion about *who gets picked* is asserted, with exact expected id
slices:

- one character inside the box, one outside → only the inside id;
- non-SEDUCE disease, `count: 2`, four in-box → all four;
- SEDUCE, `count: 2`, four in-box → exactly the first two in candidate order, and the same two on
  a repeated call with identical input (FR-4.2 asserted directly);
- SEDUCE, `count: 0`, three in-box → all three;
- in-box candidates ≤ `count` → no truncation.

**Layer 2 — the I/O shell**, driven through the `inFieldFn` and `positionFn` seams (the pattern
`force_control_test.go:15-16` and `processor_test.go:63` already use):

- boxless skill with `count > 1` → exactly the controlling character, and `positionFn` asserted
  never called (FR-2.3);
- boxless skill, `ControlCharacterId() == 0` → empty targets and, via the `emit` seam, zero Kafka
  emissions (FR-2.2);
- `positionFn` returning an error for one of three in-box characters → the other two are targeted,
  ordering preserved;
- `inFieldFn` error → empty target set, no emission.

**Layer 3 — caller inheritance**, one test each for `executeBanish` and `executeDispel` asserting
an out-of-box character receives no emission on the respective topic, via the `emit` seam.

**Layer 4 — the REST projection**, an httptest-backed test in `character/position` (mirroring
`services/atlas-maps/atlas.com/maps/character/processor_test.go`) proving a real atlas-character
JSON:API payload unmarshals into `x`/`y`. This is the one thing the seam-based tests structurally
cannot catch.

Gate: flagless `tools/verify.sh` exits 0; code review before the PR.

---

## 8. Documentation

`docs/research/missing-features/monsters-and-bosses.md` §8 and its "Unverified / needs deeper
data" entry are updated to state (a) that box-scoped AoE targeting, SEDUCE-only capping, and
deterministic ordering now match the reference server, and (b) that GMS-canonical seduce
*ordering* and facing-direction mirroring remain unverified and still need an IDA/WZ pass. The
task changes what is implemented, not what is proven — the doc must not read as though the
evidence question closed.

---

## 9. Risks

| Risk | Mitigation |
|---|---|
| Position staleness bounds achievable fidelity regardless of implementation | Acknowledged in PRD §9; out of this task's control. Reading positions at cast time is strictly better than not reading them |
| A boss encounter tuned around the current whole-map behavior becomes easier | Intended — the whole-map behavior is the bug. No per-mob special cases are added (PRD §2 non-goal) |
| N-call fan-out latency on a full field | Bounded at 8 in flight; escalation path to a bulk filter recorded in §4, isolated to `resolvePositions` |
| `atlas-character` unavailable mid-cast | Degrades to a smaller target set with warn logs; never widens to field-wide, never crashes |
