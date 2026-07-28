# MP Recovery (Brawler 5101005) — Design

Task: task-151-brawler-mp-recovery
Status: Approved PRD → design
PRD: `docs/tasks/task-151-brawler-mp-recovery/prd.md`

## 1. Problem Restated

Casting MP Recovery today runs the generic `UseSkill` path (cooldown applies, animation
plays) but no per-skill handler exists, so the registry lookup misses and nothing changes
the caster's HP or MP. The effect is entirely server-authoritative (IDA-verified in the
PRD): the fix is a new per-skill handler in atlas-channel, nothing else.

Target behavior (Cosmic parity, integer arithmetic):

```
hpLost = MaxHP / x          // floor via integer division
mpGain = hpLost * y / 100   // floor via integer division
ChangeHP(f, caster, -hpLost)
ChangeMP(f, caster, +mpGain)
```

`x` and `y` come from the cast level's effect model (v83: x=10 at all levels; y=55→100).
No low-HP guard — HP may reach 0 and the existing death path fires.

## 2. Architecture

### 2.1 Placement — new `skill/handler/mprecovery` subpackage

Follows the established per-skill handler pattern exactly (Heal task-045, Mystic Door
task-093):

- `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/mprecovery.go`
  - `init()` calls `channelhandler.Register(skill2.BrawlerMPRecoveryId, Apply)`.
  - `Apply` matches the `handler.Handler` signature
    (`skill/handler/registry.go:18-24`).
- One blank-import line added to
  `skill/handler/registrations/registrations.go` (the package main.go already imports).

The handler runs from the existing dispatch point at the end of `UseSkill`
(`skill/handler/common.go:117`), after generic cost/cooldown/buff steps. The generic path
needs no change: 5101005 has no `hpCon`/`mpCon`/`duration` in WZ, so those branches are
already no-ops, and `cooltime` already applies via `e.Cooldown()`.

### 2.2 Handler internals — mysticdoor-style seams, not heal-style direct calls

Two precedents exist for structuring the handler:

- **Heal style**: call `character.NewProcessor(l, ctx)` methods directly inside `Apply`;
  tests exercise formula helpers separately and use internal tests for the rest.
- **Mysticdoor style**: package-level `var` seams for each side effect
  (`loadMap`, `loadCaster`, `emitSpawn`), overridden in tests.

**Chosen: mysticdoor style.** MP Recovery has exactly three side effects (load caster,
ChangeHP, ChangeMP) and no packet writes, so three seam vars make the whole handler
testable end-to-end without any mock processor plumbing, and the PRD's error-path
requirement (caster load fails → zero emits) is directly assertable. The PRD's
"`loadCasterFunc` precedent" refers to this same seam-override style (`common.go:31`).

```go
// mprecovery.go — shape (illustrative, not final code)

func init() {
	channelhandler.Register(skill2.BrawlerMPRecoveryId, Apply)
}

var loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (uint16, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, err
	}
	return c.MaxHp(), nil
}

var changeHP = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, amount int16) error {
	return character.NewProcessor(l, ctx).ChangeHP(f, characterId, amount)
}

var changeMP = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, amount int16) error {
	return character.NewProcessor(l, ctx).ChangeMP(f, characterId, amount)
}
```

`Apply` body:

1. `maxHp, err := loadCaster(...)` — on error, log at error level and **return the error**
   with no emits (FR-5; note this differs from Heal, which swallows the load error — see
   §5).
2. `hpLost, mpGain := Amounts(maxHp, e.X(), e.Y())` — pure formula function.
3. If `hpLost == 0` (defensive: `x == 0` or absurd data), log a warning and return nil —
   never divide by zero, never emit a zero delta.
4. `changeHP(-hpLost)` then `changeMP(+mpGain)`. A ChangeHP error is logged and aborts
   before ChangeMP (never MP gain without the HP cost having been *requested*, per FR-5).
   A ChangeMP error is logged and returned.
5. Debug log: caster id, level, hpLost, mpGain (NFR observability).

No AnnounceSkillUse/foreign broadcast in the handler: unlike Heal, the show-effect
broadcast for a plain stat-change cast already happens on the generic path, and the PRD
scopes the task to HP/MP only. (Verified: `UseSkill`'s dispatcher is the last step and
Heal added broadcasts because its recipients differ; MP Recovery adds none.)

### 2.3 Formula — pure function in its own file

`formula.go` (mirrors Heal's `formula.go` split):

```go
// Amounts returns (hpLost, mpGain) per Cosmic SpecialMoveHandler:
// hpLost = maxHp / x, mpGain = hpLost * y / 100, integer floor division.
func Amounts(maxHp uint16, x int16, y int16) (int16, int16)
```

- Internally compute in `int32` then narrow: `hpLost = int32(maxHp) / int32(x)`,
  `mpGain = hpLost * int32(y) / 100`. `maxHp ≤ 65535`, `x ≥ 1` ⇒ `hpLost ≤ 65535` — with
  real data (`x = 10`, MaxHP ≤ 30000) both values fit `int16` comfortably (PRD FR-2).
  Defensive clamp to `math.MaxInt16` on the narrow so pathological tenant data can't wrap
  negative.
- `x <= 0` returns `(0, 0)` — the caller treats that as "skip, warn".

### 2.4 Effect model getter

`data/skill/effect/model.go` gains:

```go
// Y returns the integer Y attribute (for MP Recovery it is the percent of
// the HP loss returned as MP).
func (m Model) Y() int16 {
	return m.y
}
```

The `y` field exists and is populated from REST already (`rest.go:115`); this is the only
model change. atlas-data needs nothing (reader already ingests x/y generically).

## 3. Alternatives Considered

**A. Handle inline in `UseSkill` (special-case branch in common.go).** Rejected: the
registry exists precisely to keep per-skill logic out of the generic path; Heal and
Mystic Door set the pattern and the PRD mandates it (FR-1).

**B. Heal-style direct processor calls + internal test file.** Workable, but every
handler test would need the full `character.Processor` surface faked or an internal test
reaching into unexported state. Seam vars give the same production code path with
three-line test overrides. Rejected in favor of mysticdoor style.

**C. Server-side low-HP clamp or cast rejection.** Explicitly rejected by owner decision
(PRD FR-3): Cosmic parity, unguarded. The handler contains no HP floor logic —
atlas-character's existing ChangeHP semantics own the 0-floor/death path.

**D. Compute mpGain from post-clamp actual HP delta.** Rejected: Cosmic computes the gain
from the intended loss (PRD FR-2), and the handler doesn't know the post-clamp value
anyway (ChangeHP is a fire-and-forget command emission).

## 4. Data Flow

```
client cast 5101005
  → socket/handler (existing) → UseSkill (existing: cooldown from e.Cooldown())
    → Lookup(5101005) → mprecovery.Apply
      → loadCaster: GET character (MaxHp)
      → Amounts(maxHp, e.X(), e.Y())
      → character.ChangeHP(f, caster, -hpLost)   → Kafka → atlas-character
      → character.ChangeMP(f, caster, +mpGain)   → Kafka → atlas-character
        → stat-changed events → existing client packet flow renders HP/MP
        → (hp hits 0 → existing death path in atlas-character)
```

No new REST endpoints, Kafka topics, packets, or migrations. Multi-tenancy rides the
`ctx` exactly like every other handler; `x`/`y` arrive on the tenant-version-resolved
`effect.Model` that `UseSkill` already receives.

## 5. Error Handling

| Failure | Behavior |
|---|---|
| Caster load fails | Log error, return error, **no** ChangeHP/ChangeMP emitted (FR-5). |
| `x <= 0` or `hpLost == 0` | Log warn (bad tenant data), return nil, no emits. |
| ChangeHP emit fails | Log error, return error, **skip ChangeMP** (no gain without the cost requested). |
| ChangeMP emit fails | Log error, return error (HP cost already requested — matches Cosmic, which applies loss first; acceptable partial per FR-5's asymmetry: the forbidden partial is gain-without-cost, not cost-without-gain). |

`UseSkill` already logs and swallows handler errors (`common.go:118-120`), so returning
errors here is observability-only — the cast is never "rolled back" (there is nothing to
roll back; cooldown already applied, matching Cosmic).

## 6. Testing

All in `skill/handler/mprecovery`, Builder pattern for models, no `*_testhelpers.go`:

- **`formula_test.go`** — table-driven `Amounts` tests pinned to WZ-verified v83 values:
  - L1 (x=10, y=55), L5 (x=10, y=75), L10 (x=10, y=100) at representative MaxHP values,
    verifying floor behavior (e.g. MaxHP 1234 → hpLost 123, L1 mpGain 67).
  - Edge: `x=0` → (0,0); large MaxHP near uint16 max → no overflow/negative.
- **`mprecovery_test.go`** — seam-override handler tests:
  - Happy path: loadCaster returns MaxHp → asserts ChangeHP called with `-hpLost` and
    ChangeMP with `+mpGain`, in that order.
  - Low-HP path is *not* a handler branch (FR-3) — covered implicitly by asserting the
    handler emits the full unclamped `-hpLost` regardless (no currentHP input exists).
  - Error path: loadCaster errors → zero ChangeHP/ChangeMP calls.
  - ChangeHP errors → ChangeMP not called.
- **Registration test** — `handler.Lookup(skill2.BrawlerMPRecoveryId)` resolves after
  blank import (mirrors existing registry tests).
- **`Y()` getter** — exercised by the handler tests via a built `effect.Model`
  (acceptance criterion).

Verification gates (PRD §10): `go test -race ./...`, `go vet ./...`, `go build ./...` in
atlas-channel; `tools/redis-key-guard.sh` from repo root; `docker buildx bake
atlas-channel` is **not** required (no `go.mod` change) but harmless if run.

## 7. File Change Inventory

| File | Change |
|---|---|
| `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/mprecovery.go` | new — handler + seams + init registration |
| `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/formula.go` | new — `Amounts` pure function |
| `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/formula_test.go` | new |
| `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/mprecovery_test.go` | new |
| `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` | +1 blank import |
| `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` | +`Y()` getter |

No other service, lib, config, or deploy change.
