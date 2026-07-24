# Context — task-179 Mob Spawn Move-Action (Stance) Byte

Companion to `plan.md`. Key files, decisions, and dependencies an implementer needs before starting. Consumes `prd.md` and `design.md`.

## The problem in one paragraph

The clientbound spawn/control monster packets carry a one-byte "move action"
(stance). The v83 client treats `0`/`1` as a **sentinel** ("unresolved — compute
the idle action") and routes through `CMob::OnResolveMoveAction`, which
null-derefs (`m_pvc`) on bulk re-spawn → client access violation. A resolved byte
(`>= 2`) skips that path. Two defects: (1) after a mob moves, the raw
client-supplied move-action byte (which can be `0`/`1`) is persisted and
re-emitted verbatim on the next spawn/control → crash; (2) fresh flyers get a
hardcoded ground-idle `5` instead of the fly `12/13`, so they animate wrong.

## The fix, in three moves

1. **Shared helper** `libs/atlas-constants/monster.IdleMoveAction(isFly, fixedStance) byte`
   — the single source of truth for the encoding.
2. **Fresh-spawn origin** (`atlas-monsters` `Create`): replace hardcoded `5` with
   `IdleMoveAction(...)`.
3. **Emit-boundary guard** (`atlas-channel`, both `NewMonster` sites): rewrite only
   `0`/`1` to the fly-aware idle; pass `>= 2` through verbatim.

## Encoding (client-verified, v83)

```
actionIndex = isFly ? 6 : 2
facingBit   = (fixedStance != 0) ? (fixedStance & 1) : 0   // 0=right, 1=left
moveAction  = byte(actionIndex<<1) | facingBit
```

| isFly | fixedStance | byte |
|---|---|---|
| false | 0 | 4 |
| false | 4 | 4 |
| false | 5 | 5 |
| true  | 0 | 12 |
| true  | 4 | 12 |
| true  | 5 | 13 |

Every output satisfies `(byte & ^byte(1)) != 0` — crash invariant holds by
construction. `fixedStance` contributes **only** the facing bit; `isFly` always
drives the action index (a `noFlip` fly mob → `12/13`, never `4/5`).

Source: client `MapleStory_dump.exe` (v83) `CMob::GetFineAction @0x671999` →
`sub_671AFF` → `sub_664D42`: `v8 = (moveAbility != 3) ? 2 : 6; return (a2 & 1) | (2*v8)`.
`isFly` comes from fly-animation presence (no `moveAbility` scalar in WZ):
`atlas-data monster/reader.go:92-96` — `Flying = has "fly"`,
`Swimming = has "hover" || has "swim"`. Task 1 re-verifies this against IDA before
the constants are trusted.

## Key files and exact touch-points

| File | What / line |
|---|---|
| `libs/atlas-constants/monster/stance.go` (new) | `IdleMoveAction` + `FacingRight/FacingLeft` consts |
| `atlas-monsters .../monster/information/rest.go` | DTO already has `FixedStance uint32` (line 34) but `Extract` (line 89) drops it; add `Flying`/`Swimming`, map all three |
| `atlas-monsters .../monster/information/model.go` | add `flying/swimming/fixedStance` + `IsFly()`/`FixedStance()` |
| `atlas-monsters .../monster/information/builder.go` | `ModelBuilder` needs `SetFlying/SetSwimming/SetFixedStance` |
| `atlas-monsters .../monster/processor.go:192,198` | route lookup through `testInformationLookup` seam; `5` → `mobconst.IdleMoveAction(ma.IsFly(), ma.FixedStance())` |
| `atlas-channel .../monster/information/rest.go` | today only `Attacks`; add `Flying/Swimming/FixedStance` + map |
| `atlas-channel .../monster/information/model.go` | today only `monsterId`+`attacks`; add the three + getters |
| `atlas-channel .../monster/information/builder.go` | add the three setters |
| `atlas-channel .../socket/writer/monster_spawn.go:48-51` | `resolveSpawnStance` helper + call; log prints resolved value |
| `atlas-channel .../socket/writer/monster_control.go:50-51` | call `resolveSpawnStance` in the `controlType > Reset` branch |

## Decisions locked in design (do not re-litigate)

- **OQ-1 helper home = `libs/atlas-constants/monster`** (NOT `atlas-packet/model`).
  Both services already `require` `atlas-constants` directly (go.mod line 6);
  `atlas-monsters` does **not** import `atlas-packet` — homing it there would force
  a wrong-direction dependency. DOM-21 directs shared numeric constants here.
- **OQ-2 channel cache = reuse existing `monster/information` TTL cache.** No new
  cache. The new fields ride inside the already-cached `Model`; the guard's
  `GetById` is a cache hit after the first lookup per `(tenant, monsterId)`
  (`information/cache.go`, `processor.go:31-61`).
- **OQ-3 both fixes needed, cannot disagree.** `Create` fixes fresh flyers; the
  guard is the backstop for `0/1` a move-fragment introduces. Disjoint inputs
  (guard fires only on `0/1`, which `Create` never produces), shared helper,
  shared facing derivation.
- **OQ-4 noFlip flyer** is correct by construction (`fixedStance` = facing bit
  only); catalog existence is moot to correctness.

## Testability wrinkles (the two that matter)

1. **`atlas-monsters Create` is not unit-isolated today.** It calls
   `information.NewProcessor(p.l, p.ctx).GetById(...)` directly (processor.go:192),
   hitting the real network in tests (existing `TestSpawnPickerGuard*` at
   processor_test.go:710 acknowledges this and treats errors as expected). The
   plan routes `Create` through the existing package-level `testInformationLookup`
   seam (processor.go:71, already consulted at lines 803/1181/1481) so a
   deterministic Create-level stance test is possible. This is the established
   pattern, not a new mechanism.
2. **The channel guard needs a mock seam.** `resolveSpawnStance` constructs the
   processor via a package-level `var newInformationProcessor = information.NewProcessor`
   so the test substitutes `information/mock.ProcessorMock` (existing mock at
   `monster/information/mock/`). Both service mocks expose `GetByIdFunc`.

Both `information.ModelBuilder`s currently expose only a few setters
(`SetAttacks`, etc.) — the plan adds the three fly setters to each.

## Registry / model signatures (verified)

- `atlas-monsters` `Registry.CreateMonster(ctx, t, f, monsterId, x, y, fh, stance byte, team, hp, mp) Model` (registry.go:361).
- `atlas-monsters` `Model.Stance() byte` (model.go:143) — the Create test asserts on this.
- `atlas-channel` `monster.Model.Stance() byte` (model.go:80) — the guard input.
- `atlas-channel` `information.Processor` = `GetById(monsterId uint32) (Model, error)`; cache is read-through with positive/negative TTL, keyed `(tenant.Id(), monsterId)`.

## Scope fences (leave alone)

- Live movement-broadcast (relay-to-other-clients) path — unchanged (FR-5.1).
- `atlas-monsters .../processor.go:1511` zero-value `NewMonster(f, uniqueId, 0, 0, ...)`
  — diff/placeholder, not a spawn emit (FR-5.2).
- `libs/atlas-packet` — no change; `moveAction` is still `WriteByte` in
  `model/monster.go:521`. Only the byte's *value* changes.

## Changed modules to verify (CLAUDE.md gates)

`libs/atlas-constants`, `atlas-monsters`, `atlas-channel`:
`go test -race ./...`, `go vet ./...`, `go build ./...` each;
`docker buildx bake atlas-monsters` + `atlas-channel` from the worktree root;
`tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`
from the repo root. No `*_testhelpers.go` — use the Builder pattern.

## Grounding rule reminder

NFR-1: the encoding MUST be re-verified against the v83 client (Task 1), not
taken on faith. If IDA contradicts the truth table, Task 2's constants change and
the acceptance vectors update accordingly — the architecture is unaffected.
Quote actual decompiled lines before concluding (CLAUDE.md grounding rule).
