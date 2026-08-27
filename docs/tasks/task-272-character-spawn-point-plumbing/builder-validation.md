# DOM-01 follow-on: validating `Build()` for the character builders

## Why this is in task-272

`backend-guidelines-reviewer` raised one blocking finding against the task-272
diff: `services/atlas-npc-shops/atlas.com/npc/character/builder.go` has a
non-validating `Build() Model`, pulled into scope because this task added three
setters to that file. DOM-01 requires `builder.go` to expose `NewBuilder()`,
fluent setters, and **a `Build()` that enforces invariants**.

The non-validating shape was not introduced by task-272 — it is the shape of
nine `character/builder.go` files across the repo. The ruling was to fix all
nine on this branch rather than defer.

## The decision

Every `character/builder.go` `Build()` becomes:

```go
func (b *Builder) Build() (Model, error)
```

and validates the **identity invariant only**. The precedent is
`services/atlas-buffs/atlas.com/buffs/character/builder.go:51`, the one
character builder in the repo that already validates:

```go
func (b *Builder) Build() (Model, error) {
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	...
}
```

**Do not invent additional invariants.** These are read-model builders
populated from a REST `Extract`; fields such as `level`, `hp`, `meso`, and
`spawnPoint` legitimately take their zero value. Requiring anything beyond
identity would reject valid characters.

Per-service invariant:

| Service | Builder shape | Invariant to enforce |
|---|---|---|
| atlas-cashshop | `NewBuilder()` | `id == 0` → error |
| atlas-consumables | `NewBuilder()` | `id == 0` → error |
| atlas-login | `NewBuilder()` | `id == 0` → error |
| atlas-messages | `NewBuilder()` | `id == 0` → error |
| atlas-npc-shops | `NewBuilder()` | `id == 0` → error |
| atlas-pets | `NewBuilder()` | `id == 0` → error |
| atlas-query-aggregator | `NewBuilder()` | `id == 0` → error |
| atlas-dragons | `NewBuilder(id uint32)` | `id == 0` → error |
| atlas-character | `NewBuilder(cfg, accountId, worldId, name, ...)` | `accountId == 0` → error; `name == ""` → error |

Error text: `errors.New("id is required")`, or for atlas-character
`errors.New("accountId is required")` / `errors.New("name is required")`.
Use `errors.New`, not `fmt.Errorf`, when there is nothing to interpolate.

## Call-site rule

Every `.Build()` on a character builder now returns two values.

1. **Propagate.** In a function that already returns an error, return the
   builder error unwrapped: `if err != nil { return Model{}, err }`.
2. **Never swallow.** `m, _ := b.Build()` is a FAIL. So is discarding the
   error into a logged-and-ignored branch in production code.
3. **Tests** use `require.NoError(t, err)` (or the file's existing assertion
   idiom) immediately after the `Build()`.
4. **If a call site genuinely cannot propagate** — a provider or decorator
   signature that returns a bare `Model` — do **not** paper over it. Stop and
   report the file:line in your report; the controller rules on it.

## Scope fence

- Touch only `character/builder.go` in the named service plus the call sites
  that stop compiling because of the signature change.
- Do **not** change any other builder in the service (`account`, `world`,
  `saga`, `ring`, `commodities`, `party`, …). Those are out of scope.
- Do **not** change model fields, REST models, `Extract`, `Transform`, or any
  wire encoding. The task-272 spawn-point work is already reviewed and green.

## Controller ruling: decorator call sites

Reported independently by the atlas-cashshop and atlas-pets implementers, so it
is ruled once here and applied uniformly.

`model.Decorator[Model]` is `func(Model) Model`. A decorator such as
`InventoryDecorator` rebuilds the model from an **already-valid** `Model` it was
handed, so it has no error return and no way to propagate.

**Ruling:** fallback-to-original is correct at these sites, with two conditions.

1. Never `_`. Bind the error and act on it.
2. Log it. If a `logrus.FieldLogger` is in scope, `l.WithError(err).Errorf(...)`
   before returning the original `m`. If no logger is in scope, the fallback
   carries a comment stating why the error is unreachable: the input `Model`
   already satisfies the identity invariant, so the rebuild cannot fail it.

This is a genuine signature constraint, not a swallowed error — the invariant
being re-checked was already established upstream. Do not widen
`model.Decorator` to return an error; that is a separate refactor across every
domain that uses it, and nothing on this branch needs it.

## Status at handoff

Nine commits landed, one per service, on top of the spawn-point work
(`61e5e4b94..HEAD`). Every implementer reported its module's
`go build ./... && go test ./...` clean. `tools/verify.sh` has NOT been re-run
since these nine commits — the last flagless PASS was at `61e5e4b94`, before
them. Re-running it is the next step, followed by re-dispatching
`backend-guidelines-reviewer` to confirm the DOM-01 finding is cleared.

## Fix round 2: `modelBuilder.Build()` now validates too

The backend-audit round-2 review (`backend-audit/audit-round2.md`) judged the
open item below a real DOM-01 gap. The controller ruled: fix it on this
branch. This section is that fix and closes the open item.

### Derivation: enumerating every legitimate construction site

`modelBuilder` (`NewEmptyBuilder()` / `CloneModel()`) hydrates PARTIAL
models across ~95 call sites: DB row hydration (`character/provider.go`),
REST `Extract` (`character/rest.go`), kafka CREATE_CHARACTER
(`kafka/consumer/character/consumer.go`), a skill-decorator rebuild
(`character/processor.go:SkillModelDecorator`), and roughly twenty test
files' fixtures.

The creation path's invariant (`Builder`: `accountId != 0` AND `name != ""`)
is too strong here — swept across every site:

- Every **production** site (`modelFromEntity`, `Extract`, the kafka create
  handler, `CloneModel`-based rebuilds) sets `accountId` and `name`
  together, non-zero/non-empty.
- But `character/hp_mp_gain_test.go`'s `buildCharacter` helper sets **only**
  `jobId` and `skills` — no `accountId`, no `name` — to build a fixture for
  `resolveHPMPGainParams`, which never reads either field. Not a
  deliberately-invalid-model test; a legitimate partial.
- `character/model_test.go`'s `TestBuildPreservesHpMpUsed` sets **only**
  `name` and `hpMpUsed`, with `accountId` left at zero.

So neither `accountId != 0` nor `name != ""` alone survives every site.
What every site — production and test — agrees on, without exception: **no
site ever sets `accountId` to a real value while leaving `name` empty.**
Every production construction sets both together; every test that omits one
omits both (or omits `accountId` while `name` is present, which the
implication also allows).

### The derived invariant

```go
if c.accountId != 0 && c.name == "" {
    return Model{}, errors.New("name is required when accountId is set")
}
```

This is not vacuous — it rejects a real defect (a model claiming ownership
of an account with no name) — but it is deliberately weaker than `Builder`'s
identity invariant: `accountId == 0` is always accepted regardless of
`name`, because partial-hydration fixtures legitimately need that.

No test in the ~95-site sweep constructs a model this invariant rejects; no
test was edited to conform to the invariant (per the brief's rule, that
would have been evidence the invariant was too strong). One new test,
`character/model_test.go:TestBuildErrorsWhenAccountIdSetWithoutName`, pins
the invariant negatively; `TestBuildSucceedsWithAccountIdAndName` and
`TestBuildSucceedsWithNeitherAccountIdNorName` pin the two accepted shapes.

### Call-site handling

- `character/provider.go:modelFromEntity` and `character/rest.go:Extract`
  already returned `(Model, error)` — just bound the builder's error instead
  of discarding it.
- `character/processor.go:SkillModelDecorator` is a
  `model.Decorator[Model]` (`func(Model) Model`) — the controller ruling
  documented above for decorator call sites applies unchanged: log via
  `p.l.WithError(err).Errorf(...)` and fall back to the original `m`. The
  error is unreachable in practice (the input `m` already carries a real
  `accountId`+`name` pair; `CloneModel` only adds skills), and the fallback
  comment says so.
- `kafka/consumer/character/consumer.go:handleCreateCharacter` is a
  `message.Handler` (no error return). On a `Build()` error it logs with
  full saga correlation fields and returns without calling `CreateAndEmit`
  — the same "log and stop" shape the file already used for
  `CreateAndEmit`'s own error.
- All ~20 test files: bound the error and `t.Fatalf` on it (test helpers
  that don't already take `*testing.T` were given one — `hp_mp_gain_test.go`
  `buildCharacter`, `pending_change/processor_eligibility_test.go`
  `buildCharacter`).

### Status

`modelBuilder.Build()` is now `(Model, error)`, every call site updated, and
`go build ./... && go test ./...` is clean in
`services/atlas-character/atlas.com/character`. Both `character/builder.go`
builders now validate. The open item is closed.

### Also unchanged, deliberately

`m, _ := ...Build()` sites on non-character builders (`coupon`, `compartment`,
`drop`, `saga`, `marriage`, `condition`, pet `Clone`) are pre-existing and
outside this change's fence. They are not evidence that the call-site rule was
violated here.
