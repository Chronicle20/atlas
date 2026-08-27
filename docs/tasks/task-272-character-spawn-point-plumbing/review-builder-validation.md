# Review: DOM-01 builder-validation follow-on (61e5e4b94..e2b1723b8)

Reviewer: task-reviewer (Sonnet 5)
Scope: nine commits, one per service, making `character/builder.go` `Build()`
validating, plus the call sites each signature change touched. Reviewed
against `docs/tasks/task-272-character-spawn-point-plumbing/builder-validation.md`.

## Scope confirmed

Matches the stated range. `git diff --stat 61e5e4b94..e2b1723b8` touches only
`character/builder.go`, `character/model.go`, `character/processor.go`,
`character/rest.go`/`rest_test.go` in the nine named services, plus test
files that adapt to the new `Build()` signature, plus the task-272 docs
(`builder-validation.md`, `backend-audit/*`). No out-of-scope builder
(`account`, `world`, `saga`, `ring`, `pet` `Clone`, etc.) was touched. All
nine touched modules build clean (`go build ./...`), and `character` package
tests pass in the four services checked (`cashshop`, `npc-shops`,
`query-aggregator`, `messages`).

## PASS: invariant matches the table, all nine services

Verified `Build()` for each of the nine builders directly:

- `services/atlas-cashshop/atlas.com/cashshop/character/builder.go:121-124` — `id == 0` → `errors.New("id is required")`.
- `services/atlas-consumables/atlas.com/consumables/character/builder.go:125-128` — same.
- `services/atlas-login/atlas.com/login/character/builder.go:94-97` — same.
- `services/atlas-messages/atlas.com/messages/character/builder.go:117-120` — same.
- `services/atlas-npc-shops/atlas.com/npc/character/builder.go:124-127` — same.
- `services/atlas-pets/atlas.com/pets/character/builder.go:119-122` — same.
- `services/atlas-query-aggregator/atlas.com/query-aggregator/character/builder.go:134-137` — same.
- `services/atlas-dragons/atlas.com/dragons/character/builder.go:26-29` — same, on the `NewBuilder(id uint32)` shape.
- `services/atlas-character/atlas.com/character/character/builder.go:63-69` — `accountId == 0` → `errors.New("accountId is required")`; `name == ""` → `errors.New("name is required")`.

All nine use `errors.New` (no interpolation), matching the doc's error-text
rule.

## PASS: `atlas-character`'s `modelBuilder` correctly left alone

`services/atlas-character/atlas.com/character/character/builder.go:341`
(`func (c *modelBuilder) Build() Model`) is unchanged — still non-validating,
as the doc's "open item" section documents deliberately. Confirmed no other
commit in the range touches it.

## PASS: straightforward propagation sites

`services/atlas-dragons/atlas.com/dragons/character/rest.go:31` and
`services/atlas-login/atlas.com/login/character/rest.go:155` correctly
switched from `Build(), nil` to `Build()` inside functions that already
return `(Model, error)` — clean propagation, no swallow.

`services/atlas-login/atlas.com/login/character/processor.go` — `GetForWorld`
(line ~98), `decorateRankings` (line ~107), and `MergeRankings` (line ~131-153)
correctly thread the new builder error through as a genuine error return
(list-wide construction failure, not a per-character degrade), which is
consistent with the pre-existing "must never turn a successful fetch into a
failure" comment being about the *ranking lookup*, not the *builder call*.
`InventoryDecorator` (line 183) uses `model.ErrDecorator` with an explicit
error-handling callback that calls `degrade.Observe(p.l, ...)` — this is the
correctly-logged version of the decorator fallback pattern and should be the
reference example.

`services/atlas-consumables/atlas.com/consumables/character/model.go:289-306`
(`SetInventory`, `SetPets`) and
`services/atlas-pets/atlas.com/pets/character/model.go:239-260`
(`SetInventory`) fall back to the original `m` with a comment explaining why
the error is unreachable (`Clone(m)` carries a pre-validated `id`), matching
the doc's "no logger in scope → comment" branch of the controller ruling.

## BLOCKING: decorator fallback binds the error but never logs it, despite a logger being in scope

The controller ruling (`builder-validation.md:91-97`) requires: *"Log it. If a
`logrus.FieldLogger` is in scope, `l.WithError(err).Errorf(...)` before
returning the original `m`."* Four sites bind `err`, have `p.l
logrus.FieldLogger` in scope on the receiver, and silently `return m` with no
log call and no comment justifying its absence:

- `services/atlas-cashshop/atlas.com/cashshop/character/processor.go:53-56` — `InventoryDecorator`, `updated, err := m.SetInventory(i); if err != nil { return m }`. `p.l` is a struct field (declared line 23) and used elsewhere in the same file, but not here.
- `services/atlas-npc-shops/atlas.com/npc/character/processor.go:75-78` — `InventoryDecorator`, same pattern, `p.l` in scope (declared line 26).
- `services/atlas-query-aggregator/atlas.com/query-aggregator/character/processor.go:51-54` — `InventoryDecorator`, same pattern, `p.l` in scope (declared line 21).
- `services/atlas-query-aggregator/atlas.com/query-aggregator/character/processor.go:63-66` — `GuildDecorator`, same pattern, same `p.l`.

Each of these is exactly the shape the ruling covers (a `model.Decorator[Model]`-backing method rebuilding an already-valid `Model`), so the fallback-to-original behavior itself is correct. What's missing is the mandated `l.WithError(err).Errorf(...)` call before the fallback — these four now silently discard a genuine (if theoretically unreachable) `Build()` error where the login service's `InventoryDecorator` (via `degrade.Observe`) and the pets/consumables `model.go` methods (via the required comment) both did it correctly. This is a repo-wide pattern inconsistency introduced by three of the nine commits, not a hypothetical: it directly violates rule 2 of the "Controller ruling: decorator call sites" section.

## BLOCKING: one more fallback site with neither a log nor a comment, and no logger in scope

`services/atlas-messages/atlas.com/messages/character/model.go:237-243`:

```go
func (m Model) SetSkills(ms []skill.Model) Model {
	nm, err := Clone(m).SetSkills(ms).Build()
	if err != nil {
		return m
	}
	return nm
}
```

No `logrus.FieldLogger` is in scope here (it's a value-receiver `Model`
method), so per the ruling this needed the comment explaining why the
failure is unreachable — the same comment `consumables/character/model.go`
and `pets/character/model.go` both carry at their analogous `SetInventory`/
`SetPets` sites. It has neither. The error is bound (not `_`), so this isn't
a hard swallow, but it fails the documented condition for this branch of the
ruling.

## BLOCKING: no test in any of the nine services exercises the new validation

Requirement checked: "Tests cover the new validation." Swept every
`character/*_test.go` file touched or adjacent to the nine builders for an
assertion that `Build()` returns a non-nil error when the identity field is
zero (`id == 0`, or for `atlas-character`, `accountId == 0` / `name == ""`):

```
grep -rn "errors.New(\"id is required\")\|require.Error\|assert.Error" \
  .../cashshop/character/*_test.go .../consumables/character/*_test.go \
  .../npc/character/*_test.go .../pets/character/*_test.go \
  .../query-aggregator/character/*_test.go .../dragons/character/*_test.go \
  .../messages/character/*_test.go
# no matches
```

Every test-file change in the diff is a mechanical adaptation of an
*existing* test to the new two-return signature (`m, err := ...Build();
require.NoError(t, err)` or equivalent), never a new test asserting the
error path itself. `atlas-login/character/builder_test.go` (the only service
with a dedicated `builder_test.go` in this range) has five `Build()`-calling
tests, all constructing with a non-zero id and all asserting success — none
constructs with `id == 0` to assert the new failure. The precedent this task
cites, `atlas-buffs/character/builder.go`, has this coverage; none of the
nine new sites do. This is the same gap in all nine commits, not isolated to
one service.

## Not evaluable

- `tools/verify.sh` (flagless) has not been re-run against this range per
  the doc's own "Status at handoff" section, and re-running the full gate is
  outside this review's surface (a green build is a different gate per
  review-protocol). I confirmed `go build ./...` is clean for all nine
  touched modules and `go test ./character/...` passes for the four services
  whose decorator sites I flagged, but did not run the full monorepo
  verification pipeline.
- The `atlas-character` `modelBuilder` open item is explicitly deferred by
  the doc to a future ruling; not evaluated here as it is out of this
  range's fence.

## Verdict rationale

The invariant table, error text, `errors.New` usage, propagation at
already-erroring call sites, and the scope fence are all correct across all
nine services — the mechanical bulk of this task is solid. But three of the
four documented conditions for a passing decorator/fallback site are
violated at four call sites in three services, and the "tests cover the new
validation" requirement is unmet across all nine commits. These are
concrete, checklist-driven gaps against the document's own text, not
stylistic preferences.
