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

### Open item the next session must not lose

`services/atlas-character/atlas.com/character/character/builder.go` contains
**two** builders:

- `Builder` (`NewBuilder(cfg, accountId, worldId, name, ...)`, line 101) — now
  validating. It has **zero call sites** anywhere in the module, which is why
  the signature change compiled with no caller churn.
- `modelBuilder` (`NewEmptyBuilder()` line 174, `CloneModel()` line 178) — the
  builder actually used, by ~40 sites including `character/processor.go:282`.
  Its `Build()` is still non-validating and was deliberately left alone.

Leaving it is a judgment call, not an oversight: `NewEmptyBuilder()` is used to
hydrate partial models in tests that legitimately set no name — e.g.
`character/hp_mp_gain_test.go:55` sets only `jobId` and `skills` — so a
`name != ""` guard there would reject valid input. Applying the identity
invariant to `modelBuilder` would need a different invariant than the creation
path uses, and that is a design question, not a mechanical fix.

**If the re-run of `backend-guidelines-reviewer` raises DOM-01 against
`modelBuilder`, that is a real finding and needs a ruling — do not treat it as
already-answered by this document.**

### Also unchanged, deliberately

`m, _ := ...Build()` sites on non-character builders (`coupon`, `compartment`,
`drop`, `saga`, `marriage`, `condition`, pet `Clone`) are pre-existing and
outside this change's fence. They are not evidence that the call-site rule was
violated here.
