# Fix round 2 — post-merge backend-guidelines findings

## Facts

```
task=task-278-map-environment-object-state
worktree=<repo-root>/.worktrees/task-278-map-environment-object-state
branch=task-278-map-environment-object-state
head=5efddd01b
toolchain=go1.27.0
```

Modules you will touch: `services/atlas-maps/atlas.com/maps`,
`services/atlas-data/atlas.com/data`.

## Origin

`backend-guidelines-reviewer` returned `CHANGES_REQUIRED` over the commit range
`9b7817a13..5efddd01b` (the four post-merge commits of task-278). Full audit:
`docs/tasks/task-278-map-environment-object-state/audit-postmerge.md`.

Five blocking findings were raised. **Four are in scope for you.** The fifth was
ruled out by the user — see "Explicitly out of scope" below. Do not fix it.

## The four fixes

### 1. DOM-01 — `data/map/object/` has `model.go` but no `builder.go`

`services/atlas-maps/atlas.com/maps/data/map/object/model.go` declares an
immutable `Model{name, state}` with accessors, but the package has no
`builder.go`.

Add one, mirroring the nearest sibling `data/map/info/builder.go` exactly:
`type Builder struct{ m Model }`, `NewBuilder() *Builder`, fluent
`SetName`/`SetState` returning `*Builder`, and `Build() Model`.

Keep `Build()` returning a bare `Model` like `info` does — do NOT invent a
validating `Build() (Model, error)` unless you are enforcing a real invariant,
and if you do, `Extract` must propagate the error rather than swallow it.

Then have `Extract` in `rest.go` construct through the builder instead of the
struct literal it uses today.

### 2. DOM-04 — no `Transform` in `data/map/object/rest.go`

`services/atlas-maps/atlas.com/maps/data/map/object/rest.go` defines
`Extract(RestModel) (Model, error)` but not `Transform(Model) (RestModel, error)`.
DOM-04 triggers on any package with a `rest.go`.

Add the genuine inverse:

```go
func Transform(m Model) (RestModel, error) {
	return RestModel{Id: m.Name(), Name: m.Name(), State: m.State()}, nil
}
```

Note `RestModel` carries both `Id` (the JSON:API resource id, `json:"-"`) and
`Name`; the resource id IS the object name. Confirm that against the current
`rest.go` before writing it — do not copy the snippet blind.

### 3. EXT-02 — the new atlas-maps→atlas-data client package has zero tests

`services/atlas-maps/atlas.com/maps/data/map/object/` has no `_test.go` at all.
EXT-02 requires an `httptest`-backed test that serves a representative JSON:API
fixture and asserts a **populated domain struct** — a mock client does not
satisfy it.

Copy the shape of
`services/atlas-maps/atlas.com/maps/data/map/monster/processor_drain_test.go`:
`httptest.NewServer`, `t.Setenv("DATA_SERVICE_URL", srv.URL+"/")`,
`tenant.Create(...)` + `tenant.WithContext`, `test.NewNullLogger()`, then call
the processor and assert on the result.

Cover, at minimum:

- a fixture whose `data[]` carries objects with `{"name":...,"state":...}` and
  `"type":"objects"`, asserting `GetDefaultState` returns the declared state for
  a named object;
- `GetDefaultState` returning `object.ErrUnknownObject` for a name the fixture
  does not declare (use `errors.Is`);
- the drain: `inMapProvider` uses `requests.DrainProvider` with page size 250,
  so serve two pages and prove an object on page 2 is found. The `monster`
  test's `meta.page` fixture shape is the reference for what makes the drain
  keep paging.

### 4. DOM-20 — `reader_object_test.go` is not table-driven

`services/atlas-data/atlas.com/data/map/reader_object_test.go:29-70`
(`TestGetObjects`) exercises three parallel `l2`-parsing scenarios with
sequential `if` checks. Rewrite it to the required
`tests := []struct{...}` + `t.Run(tt.name, ...)` form.

Preserve every scenario and assertion that exists today — this is a shape
change, not a coverage change. In particular keep the missing/empty-`obj`-node
case and the non-numeric-`l2`-falls-back-to-0 case.

## Explicitly out of scope — do not fix

**DOM-04 against `services/atlas-data/atlas.com/data/map/object/rest.go`.**
That package has no `Model` type; `reader.go` builds `RestModel` directly from
the WZ node, and all four siblings (`map/monster`, `map/npc`, `map/portal`,
`map/reactor`) are likewise `rest.go`-only with no model and no `Transform`.
The user ruled to accept the sibling convention rather than make the newest of
five packages the only one shaped differently. Leave that file alone.

## Ground truth you must not re-derive

Do NOT re-read the WZ tree to check the polarity of
`Obj/effect.img/quest/gate/0`. Map 103000800's `gate` has visible state `0`,
observed in-client; an earlier pass read the tree and got the polarity
backwards. Any existing test asserting `gate -> 1` as the declared default
(`l2=1`) is correct — do not "fix" it.

## Verification

Module-local only:

```sh
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./... -count=1
cd services/atlas-data/atlas.com/data && go build ./... && go test ./... -count=1
```

Do NOT run `tools/verify.sh`, `tools/lint.sh`, `-race`, or docker.

For fix 3 and fix 4, show the test RED before it is GREEN where that is
meaningful (fix 3's drain assertion in particular — a single-fetch
implementation must fail it).

## Files

- `services/atlas-maps/atlas.com/maps/data/map/object/builder.go` — NEW; `Builder`, `NewBuilder`, fluent `SetName`/`SetState`, `Build`
- `services/atlas-maps/atlas.com/maps/data/map/object/rest.go` — add `Transform`; rewrite `Extract` to build via the builder
- `services/atlas-maps/atlas.com/maps/data/map/object/model.go` — read only, for the field/accessor names
- `services/atlas-maps/atlas.com/maps/data/map/object/processor.go` — read only, for `GetDefaultState` / `inMapProvider` / `ErrUnknownObject`
- `services/atlas-maps/atlas.com/maps/data/map/object/requests.go` — read only, for the `DATA` root url and `data/maps/%d/objects` path
- `services/atlas-maps/atlas.com/maps/data/map/object/processor_test.go` — NEW; the httptest fixture test for EXT-02
- `services/atlas-data/atlas.com/data/map/reader_object_test.go` — rewrite `TestGetObjects` table-driven

Patterns to copy:

- `services/atlas-maps/atlas.com/maps/data/map/info/builder.go` (whole file — builder shape)
- `services/atlas-maps/atlas.com/maps/data/map/monster/processor_drain_test.go` (whole file — httptest + tenant + drain fixture shape)
- `services/atlas-maps/atlas.com/maps/data/map/monster/rest.go` (`Extract` shape)
- `services/atlas-data/atlas.com/data/map/reader_test.go` (an existing table-driven test in the same package, for the local `t.Run` idiom)

## Commit

One commit for all four fixes, on `task-278-map-environment-object-state`.
No `git add -A` / `git add .` — add the named paths. No destructive git ops.
Verify you are still on the task branch after committing.
