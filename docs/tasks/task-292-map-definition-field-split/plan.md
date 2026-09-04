# Map Definition / Field Separation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Separate the static Map *Definition* surface from the live *Field*
(runtime instance) surface — add `GET /api/fields` to `atlas-maps`, add
`GET /api/data/maps/{id}/objects` to `atlas-data`, route both through the
gateway, and build the `/fields` locator and `/fields/:w/:c/:m/:i` detail page
in `atlas-ui` alongside a re-labelled Map Definition page.

**Architecture:** Three independent axes. `atlas-data` gains a WZ-derived
map-object collection stored as a JSON:API relationship on the existing map
document. `atlas-maps` gains a tenant-scoped, additive read model over the
in-memory character registry, exposed as a new `field` package. `atlas-ui`
consumes both through separate service modules with disjoint React Query key
namespaces, so definition caching (10 min) and runtime caching (5 s) cannot
interfere. No service imports another service's internals; the only
cross-service agreement is the two literal `kind` strings `ENVIRONMENT` and
`OBSTACLE`.

**Tech Stack:** Go 1.27 (gorilla/mux, api2go/jsonapi, gorm, testify), nginx
gateway config, TypeScript / React 19 / Vite / TanStack React Query /
shadcn-ui / Vitest + @testing-library/react.

**Spec:** `docs/tasks/task-292-map-definition-field-split/design.md`
(PRD at `docs/tasks/task-292-map-definition-field-split/prd.md`)

## Global Constraints

- **`[278]` is NOT on `main`.** Verified: `services/atlas-maps/atlas.com/maps/map/environment/`
  does not exist on `origin/main`, and `field.ObjectKind` does not exist in
  `libs/atlas-constants` on `main`. Tasks 1–21 build on `main` today. **Task 22 is
  BLOCKED** and must not start until `task-278-map-environment-object-state`
  has merged to `main` and this branch has rebased onto it. Do not stub it.
- **`kind` vocabulary.** Exactly two literal strings, uppercase:
  `"ENVIRONMENT"` and `"OBSTACLE"`. `atlas-data` uses a local `string`, NOT
  `field.ObjectKind` (design D6 — that type is only on `[278]`).
- **Object id format.** `"{KIND}:{name}"`, e.g. `"ENVIRONMENT:gate"`,
  `"OBSTACLE:menhir0"`. Identical on both sides so the UI merge is an id join.
- **Field id format.** `"{worldId}:{channelId}:{mapId}:{instanceId}"`, e.g.
  `"0:1:910340000:00000000-0000-0000-0000-000000000000"`.
- **Liveness rule.** A field is listed if and only if its registry slice is
  non-empty. `RemoveCharacter` empties a slice without deleting the key
  (`services/atlas-maps/atlas.com/maps/map/character/registry.go:60-66`), so key
  existence is NOT liveness.
- **No polling** (FR-39). No new hook may set `refetchInterval`. A reviewer
  greps for it.
- **Definition/Runtime distinction is never colour-only** (FR-3). The badge
  always renders its word.
- **Multi-tenancy.** `GetFieldsWithCharacters` filters on `mk.Tenant == t`
  *inside* the registry read lock, before anything else.
- **Go tasks:** invoke the `backend-dev-guidelines` skill before writing code.
  **TS tasks:** invoke `frontend-dev-guidelines`.
- Never commit to `main`. All work lands on branch
  `task-292-map-definition-field-split`.

### Corrections to `design.md` established during planning — these override the design

| # | Design said | Reality (verified) |
|---|---|---|
| C1 | `getObjects` clones `getReactors` | **Wrong shape.** Reactors sit at one top-level node (`reader.go:341`). Objects sit per numeric layer: `<imgdir name="N"><imgdir name="obj">`. Clone the layer-iteration idiom of `getBackgroundTypes` (`reader.go:323-338`, `strconv.Atoi(bt.Name)`) with one extra descent. |
| C2 | ingest failure policy "follow the neighbour" | The two neighbours **differ**: `data/workers/mapw.go:39-41` warns and continues; `data/data/processor.go:117-121` logs Error and `return err`. Match each site's own severity. |
| C3 | `paginate.ParseParams` copied between services | atlas-maps uses `(paginate.MaxPageSize, paginate.MaxPageSize)`; atlas-data uses `(paginate.DefaultPageSize, paginate.MaxPageSize)`. Use each service's own convention. |
| C4 | gateway "near line 490" | atlas-maps instance blocks are `deploy/shared/routes.conf:490-503`. The `^/api/worlds(/.*)?$` catch-all is at **line 691 and proxies to `atlas-world`, not `atlas-maps`** — an unrouted `/environment` path would silently hit the wrong service. The explicit block is mandatory, not cosmetic. |
| C5 | sort-then-paginate at `map/resource.go:57-60` | Actual block is `services/atlas-maps/atlas.com/maps/map/resource.go:56-59`. |
| C6 | Characters tab columns "Position and State uncertain" | **Resolved.** `services/atlas-character/atlas.com/character/character/rest.go` emits `level`, `jobId`, `x`, `y`, `fh`, `stance` — but explicitly NOT `mapId` (comment at `rest.go:42-44`). Columns: Name, Character ID, Level, Job, Position (`x`,`y`). **No State column** — `stance` is an animation frame, not a state. |
| C7 | `useGridRefresh` "reuse, already in play" | It exists (`src/lib/hooks/useGridRefresh.ts:16-37`) but `MapDetailPage.tsx` does not use it. It is new to this feature area, not pre-wired. |
| C8 | `[278]` env type is `environment` | Type is **`environment-objects`**. |
| C9 | `610030300` shows six OBSTACLE objects | Seven rows total: six `OBSTACLE` (`menhir0`–`menhir5`) plus one `ENVIRONMENT` (`3pt`). Do not assert a row count of six. |

---

## Task 1: `atlas-maps` — tenant-scoped field occupancy read model

Module root for all `go` commands: `services/atlas-maps/atlas.com/maps`

### Files

- `services/atlas-maps/atlas.com/maps/map/character/registry.go` — add `FieldOccupancy` type and `GetFieldsWithCharacters`, after `GetMapsWithCharacters` (which ends at line 100). Do NOT modify `GetMapsWithCharacters` — its only caller, `services/atlas-maps/atlas.com/maps/tasks/respawn.go:39`, wants cross-tenant behaviour.
- `services/atlas-maps/atlas.com/maps/map/character/processor.go` — add the method to the `Processor` interface (lines 16-23) and to `ProcessorImpl` (next to `GetMapsWithCharacters` at lines 49-51).
- `services/atlas-maps/atlas.com/maps/map/character/mock/processor.go` — handwritten mock; add the `GetFieldsWithCharactersFunc` field (next to `GetMapsWithCharactersFunc`, line 17) and the method (next to lines 37-42).
- `services/atlas-maps/atlas.com/maps/map/character/registry_test.go` — **new file**.

Patterns to copy: `services/atlas-maps/atlas.com/maps/map/character/registry.go:74-87`
(`GetInMapAllInstances` — the only existing `mk.Tenant == t` filter under `RLock`).

Read-only reference: `libs/atlas-constants/field/model.go:21-45` — `Model` has
`Id()`, `WorldId()`, `ChannelId()`, `MapId()`, `Instance()`.

- [ ] **Step 1: Write the failing test**

`services/atlas-maps/atlas.com/maps/map/character/registry_test.go`, package
`character`. Because `getRegistry()` is a package-level `sync.Once` singleton,
tests must use distinct tenants rather than resetting it — build each tenant
with a fresh `uuid.New()` per test.

`TestGetFieldsWithCharacters` — table-driven, one subtest per case.
Setup per case: `ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)`;
build fields with `field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()`;
populate via `getRegistry().AddCharacter(MapKey{Tenant: ten, Field: f}, characterId)`
and drain via `getRegistry().RemoveCharacter(...)`.

| subtest | registry contents (all under tenant A unless noted) | expect |
|---|---|---|
| `empty registry` | nothing | `len(result) == 0`, non-nil slice |
| `single occupied field` | Add(w0,c1,m910340000,uuid.Nil, char 100) | one entry: `Field.WorldId()==0`, `ChannelId()==1`, `MapId()==910340000`, `Instance()==uuid.Nil`, `CharacterCount==1` |
| `drained field excluded` | Add(w0,c1,m100000000,uuid.Nil, char 100) then Remove(same key, 100) | that field absent from result |
| `drained plus live` | field X drained (add+remove char 100); field Y (w0,c2,m200000000) holds chars 200,201 | exactly one entry, it is field Y, `CharacterCount==2` |
| `tenant isolation` | tenant A holds (w0,c1,m300000000, char 1); tenant B holds (w0,c1,m400000000, char 2) | `GetFieldsWithCharacters(tenantA)` returns exactly one entry with `MapId()==300000000`; `GetFieldsWithCharacters(tenantB)` returns exactly one with `MapId()==400000000` |
| `two instances same map` | (w0,c1,m910340000, instance uuid.Nil, char 1) and (w0,c1,m910340000, instance I2, char 2) where `I2 := uuid.New()` | two entries, both `MapId()==910340000`, distinct `Instance()`, each `CharacterCount==1` |

Assertions use `testify/assert` + `require`, matching
`services/atlas-maps/atlas.com/maps/map/resource_paginate_test.go:15-16`.

- [ ] **Step 2: Run the test to verify it fails**

Run from `services/atlas-maps/atlas.com/maps`:
`go test ./map/character/ -run TestGetFieldsWithCharacters -v`
Expected: FAIL — `undefined: FieldOccupancy` / `r.GetFieldsWithCharacters undefined`.

- [ ] **Step 3: Implement**

Append to `registry.go` after line 100:

```go
// FieldOccupancy pairs a live field with the number of characters it held at
// the moment the registry snapshot was taken.
type FieldOccupancy struct {
	Field          field.Model
	CharacterCount uint32
}

// GetFieldsWithCharacters returns every field belonging to t that currently
// holds at least one character, with its occupancy sampled under the same read
// lock. Key existence is not liveness: RemoveCharacter empties a key's slice
// without deleting the key, so a drained field lingers as a zero-length entry
// and must be excluded here. Sampling the count under the same lock is what
// keeps a field from being listed as live and then reported with a count of
// zero within one response.
func (r *Registry) GetFieldsWithCharacters(t tenant.Model) []FieldOccupancy {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]FieldOccupancy, 0)
	for mk, mc := range r.characterRegister {
		if mk.Tenant != t {
			continue
		}
		if len(mc) == 0 {
			continue
		}
		result = append(result, FieldOccupancy{Field: mk.Field, CharacterCount: uint32(len(mc))})
	}
	return result
}
```

In `processor.go`, add to the `Processor` interface (after
`GetMapsWithCharacters() []MapKey`, line 19):

```go
	GetFieldsWithCharacters(t tenant.Model) []FieldOccupancy
```

and the impl next to lines 49-51:

```go
func (p *ProcessorImpl) GetFieldsWithCharacters(t tenant.Model) []FieldOccupancy {
	return getRegistry().GetFieldsWithCharacters(t)
}
```

Add the `tenant` import to `processor.go` if it is not already present.

In `mock/processor.go`, add the field next to line 17:

```go
	GetFieldsWithCharactersFunc func(t tenant.Model) []character.FieldOccupancy
```

and the method, following the shape of the existing
`GetMapsWithCharacters` mock method at lines 37-42 (return the zero value when
the func field is nil).

- [ ] **Step 4: Run the test to verify it passes**

`go test ./map/character/ -v` — Expected: PASS.
Then `go build ./...` and `go vet ./...` from the module root.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map/character/
git commit -m "feat(atlas-maps): tenant-scoped field occupancy read model"
```

---

## Task 2: `atlas-maps` — `GET /api/fields` resource

Module root: `services/atlas-maps/atlas.com/maps`

### Files

- `services/atlas-maps/atlas.com/maps/field/rest.go` — **new file**, `package field`
- `services/atlas-maps/atlas.com/maps/field/resource.go` — **new file**
- `services/atlas-maps/atlas.com/maps/field/resource_test.go` — **new file**
- `services/atlas-maps/atlas.com/maps/main.go` — add one `AddRouteInitializer` line in the chain at lines 147-155, after `_map.InitResource(GetServer())`

Patterns to copy:
- `services/atlas-maps/atlas.com/maps/map/rest.go:1-24` (RestModel / `GetID` / `GetName` / `SetID` shape)
- `services/atlas-maps/atlas.com/maps/map/resource.go:27-74` (`InitResource` closure, `paginate.ParseParams`, `server.WriteBadRequest`, sort-then-`paginate.Slice`, `server.MarshalPaginatedResponse`)
- `services/atlas-maps/atlas.com/maps/map/resource_paginate_test.go:26-77` (test server information stub, `setupMapRouter`, `mapRequestWithTenant`, tenant + field + registry setup)

Read-only references:
- `libs/atlas-rest/server/paginate/slice.go:9` — `func Slice[T any](items []T, page model.Page) model.Paged[T]`; it does **not** sort, the caller must.
- `libs/atlas-rest/server/paginate/params.go:27` — `func ParseParams(query url.Values, defaultSize, maxSize int) (model.Page, error)`.
- `libs/atlas-rest/server/error.go:81` — `func WriteBadRequest(l logrus.FieldLogger, w http.ResponseWriter, detail string)`.

Note the package name `field` collides with the imported
`libs/atlas-constants/field`. That is fine and has precedent
(`map/resource.go:1` declares `package _map` while importing
`_map "…/atlas-constants/map"`). Inside `package field`, import the constants
package as `cfield "github.com/Chronicle20/atlas/libs/atlas-constants/field"`.

- [ ] **Step 1: Write the failing test**

`services/atlas-maps/atlas.com/maps/field/resource_test.go`, package `field`.
Harness copied verbatim in shape from `map/resource_paginate_test.go:26-52`:
a `fieldTestServerInformation` stub returning base URL `http://localhost:8080`
and prefix `/api/`, a `setupFieldRouter()` calling `InitResource(&fieldTestServerInformation{})(r, l)`,
and a `fieldRequestWithTenant(method, url, tenantId)` setting `TENANT_ID`,
`REGION=GMS`, `MAJOR_VERSION=83`, `MINOR_VERSION=1`.

Registry population uses `character.NewProcessor(logrus.New(), ctx)` and its
`Enter(uuid.New(), f, characterId)` / `Exit(uuid.New(), f, characterId)`
methods, exactly as `map/resource_paginate_test.go:73-76` does. Each test
creates a fresh tenant via `tenant.Create(uuid.New(), "GMS", 83, 1)` so the
shared singleton registry cannot leak between tests.

`TestGetFieldsDrainedFieldExcluded` — the PRD's named acceptance test.
Enter char 100 into `(w0, c1, m100000000, uuid.Nil)`, then `Exit` it. Enter
char 200 into `(w0, c1, m200000000, uuid.Nil)`. `GET /api/fields`.
Expect `200`; exactly one item in `data`; its `id` is
`"0:1:200000000:00000000-0000-0000-0000-000000000000"`; no item has an id
beginning `"0:1:100000000:"`.

`TestGetFieldsTenantIsolation`. Tenant A: char 1 in `(w0,c1,m300000000,uuid.Nil)`.
Tenant B: char 2 in `(w0,c1,m400000000,uuid.Nil)`. Request as A → exactly one
item, `attributes.mapId == 300000000`. Request as B → exactly one item,
`attributes.mapId == 400000000`.

`TestGetFieldsFilters` — table-driven. Setup (all one tenant, `uuid.Nil`
instance, one character each):

| field | world | channel | map |
|---|---|---|---|
| F1 | 0 | 1 | 100000000 |
| F2 | 0 | 2 | 100000000 |
| F3 | 1 | 1 | 200000000 |

| subtest | query | expected mapIds (as a set) |
|---|---|---|
| `no filter` | `/api/fields` | `{100000000, 100000000, 200000000}` — 3 items |
| `world only` | `?filter[worldId]=0` | 2 items, both `mapId == 100000000` |
| `channel only` | `?filter[channelId]=1` | 2 items: `(w0,c1,m100000000)` and `(w1,c1,m200000000)` |
| `map only` | `?filter[mapId]=200000000` | 1 item, id `"1:1:200000000:00000000-0000-0000-0000-000000000000"` |
| `all three` | `?filter[worldId]=0&filter[channelId]=2&filter[mapId]=100000000` | 1 item, id `"0:2:100000000:00000000-0000-0000-0000-000000000000"` |
| `unknown filter ignored` | `?filter[nope]=7` | 3 items |
| `no match` | `?filter[worldId]=9` | `200`, `data` is an empty array, NOT 404 |

`TestGetFieldsMalformedFilter` — table-driven over
`?filter[worldId]=abc`, `?filter[channelId]=abc`, `?filter[mapId]=abc`.
Each expects HTTP `400`.

`TestGetFieldsPaginationDeterminism`. Enter one character into each of six
fields, `(w0, c1, mapId)` for `mapId` in `{100000000, 200000000, 300000000,
400000000, 500000000, 600000000}`, entered in that shuffled order:
`400000000, 100000000, 600000000, 200000000, 500000000, 300000000`.
`GET /api/fields?page[number]=1&page[size]=3` → ids in exactly this order:
`0:1:100000000:…`, `0:1:200000000:…`, `0:1:300000000:…`.
`page[number]=2&page[size]=3` → `0:1:400000000:…`, `0:1:500000000:…`,
`0:1:600000000:…`. Assert the union of both pages is the full six-element set
with no overlap. Run the whole request pair **five times in a loop** and assert
identical output each time — Go map iteration order is randomised, so a single
pass can pass by luck.

`TestGetFieldsAttributes`. One field `(w0, c1, m910340000, uuid.Nil)` with
three characters (ids 1, 2, 3). Assert the single item's attributes are
exactly `worldId: 0`, `channelId: 1`, `mapId: 910340000`,
`instanceId: "00000000-0000-0000-0000-000000000000"`, `characterCount: 3`, and
its JSON:API `type` is `"fields"`.

- [ ] **Step 2: Run the test to verify it fails**

From `services/atlas-maps/atlas.com/maps`: `go test ./field/ -v`
Expected: FAIL to build — `undefined: InitResource`.

- [ ] **Step 3: Implement `field/rest.go`**

```go
package field

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is one live field instance. Id is
// "{worldId}:{channelId}:{mapId}:{instanceId}".
type RestModel struct {
	Id             string     `json:"-"`
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	MapId          _map.Id    `json:"mapId"`
	InstanceId     uuid.UUID  `json:"instanceId"`
	CharacterCount uint32     `json:"characterCount"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "fields"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}
```

- [ ] **Step 4: Implement `field/resource.go`**

Structure, following `map/resource.go:27-33` for registration:

```go
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/fields").Subrouter()
		r.HandleFunc("", rest.RegisterHandler(l)(si)(getFields, handleGetFields)).Methods(http.MethodGet)
	}
}
```

with `const getFields = "get_fields"`. The service prefix is `/api/`
(`GetServer().GetPrefix()`), so this is `/api/fields` end to end.

`handleGetFields(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc`
returns a plain `func(w, r)` (no `rest.Parse*` wrappers — there are no path
params) that:

1. `page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)`
   — matching this service's convention (C3). On error:
   `server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")`, return.
2. Parses filters into `*world.Id` / `*channel.Id` / `*_map.Id` via a helper
   `parseFilters(q url.Values) (*world.Id, *channel.Id, *_map.Id, error)` in the
   same file. Each uses `strconv.ParseUint(v, 10, 32)`; absent key → nil, no
   constraint; present-but-unparseable → error. On error:
   `server.WriteBadRequest(d.Logger(), w, "invalid filter value")`, return.
3. `t := tenant.MustFromContext(d.Context())`, then
   `occ := character.NewProcessor(d.Logger(), d.Context()).GetFieldsWithCharacters(t)`.
4. Filters `occ` in the handler (not in the registry — design D3): keep an entry
   when each non-nil filter matches `o.Field.WorldId()` / `o.Field.ChannelId()` /
   `o.Field.MapId()`.
5. Maps each surviving entry to a `RestModel` with
   `Id: fmt.Sprintf("%d:%d:%d:%s", o.Field.WorldId(), o.Field.ChannelId(), o.Field.MapId(), o.Field.Instance().String())`.
6. **Sorts before slicing** (design D2 — `paginate.Slice` does not sort):
   `sort.Slice(models, func(i, j int) bool { ... })` comparing in order
   `WorldId`, then `ChannelId`, then `MapId`, then `InstanceId.String()`.
7. `paged := paginate.Slice(models, page)` then
   `server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(paged.Items, paginate.EnvelopeFor(paged), r)`.

Log an enumeration failure with the tenant and the parsed filter values (PRD
observability NFR).

- [ ] **Step 5: Wire the route in `main.go`**

Insert into the `server.New(l)...` chain at `services/atlas-maps/atlas.com/maps/main.go:147-155`,
immediately after the `_map.InitResource(GetServer())` line:

```go
		AddRouteInitializer(mapField.InitResource(GetServer())).
```

with the import `mapField "atlas-maps/field"`.

- [ ] **Step 6: Run the tests to verify they pass**

From `services/atlas-maps/atlas.com/maps`:
`go test ./field/ -v` → PASS (all six test functions).
`go build ./...` and `go vet ./...` → clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/field/ services/atlas-maps/atlas.com/maps/main.go
git commit -m "feat(atlas-maps): GET /api/fields live field enumeration"
```

---

## Task 3: `atlas-maps` — document the field endpoint

### Files

- `services/atlas-maps/docs/rest.md` — 375 lines; add a section for `GET /api/fields`

Patterns to copy: the existing per-endpoint section format already in that
file (path, description, query parameters, response body, error codes).

- [ ] **Step 1: Read the surrounding format**

Run: `grep -n '^#\{2,3\} ' services/atlas-maps/docs/rest.md`
Use the section immediately describing the characters-in-map endpoint as the
structural template.

- [ ] **Step 2: Write the section**

Document, with exact values:

- `GET /api/fields`
- Purpose: enumerate live field instances for the requesting tenant.
- **Liveness rule**, stated explicitly: a field appears if and only if it
  currently holds at least one character. `RemoveCharacter` empties a key's
  slice without deleting the key, so a drained field is excluded.
- Tenant scoping: filtered on `MapKey.Tenant` inside the registry read lock;
  never returns another tenant's field.
- Query parameters: `filter[worldId]`, `filter[channelId]`, `filter[mapId]`
  (all optional, independently combinable, unrecognised `filter[...]` keys
  ignored, malformed values → `400`); `page[number]`, `page[size]`.
- Ordering: `(worldId, channelId, mapId, instanceId)` ascending, applied
  before pagination so pages partition the result set.
- Response: JSON:API collection, type `fields`, id
  `{worldId}:{channelId}:{mapId}:{instanceId}`, attributes `worldId`,
  `channelId`, `mapId`, `instanceId`, `characterCount`. Include the example
  document from `prd.md` §5.1.
- Errors: `400` invalid filter or page value. Empty result is `200` with
  `"data": []`, never `404`.

- [ ] **Step 3: Verify the doc has no absolute paths**

Run: `grep -n '/home/' services/atlas-maps/docs/rest.md`
Expected: no output (repo convention forbids literal home paths under `docs/`).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/docs/rest.md
git commit -m "docs(atlas-maps): document GET /api/fields"
```

---

## Task 4: `atlas-data` — obstacle definition registry

Module root: `services/atlas-data/atlas.com/data`

### Files

- `services/atlas-data/atlas.com/data/map/object_registry.go` — **new file**, `package _map`
- `services/atlas-data/atlas.com/data/map/object_registry_test.go` — **new file**

Patterns to copy: `services/atlas-data/atlas.com/data/map/string_registry.go:11-60`
— the whole file is the template (identifier struct, `sync.Once` accessor,
`Init*` walking XML and calling `.Add(t, …)`).

Read-only references:
- `services/atlas-data/atlas.com/data/document/registry.go:10-79` — `Registry[I string, M Identifier[I]]`; `M` must implement `GetID() string`. `Get` returns `(M, error)` with `errors.New("not found")` on a miss; `Clear(t)` returns `error`.
- `services/atlas-data/atlas.com/data/xml/model.go:12-21` — `Node.Name string`, `Node.ChildNodes []Node`, `ChildByName(name string) (*Node, error)`.
- `services/atlas-data/atlas.com/data/xml/model.go:124` — `GetIntegerWithDefault(name string, def int32) int32`. There is no `GetInteger` and no `GetInt`.

- [ ] **Step 1: Write the failing test**

`services/atlas-data/atlas.com/data/map/object_registry_test.go`, package `_map`.
The fixture is an inline backtick XML string, matching the `testXML` idiom at
`services/atlas-data/atlas.com/data/map/reader_test.go:19`. Because `InitObj`
reads from a directory, the test writes the fixture to `t.TempDir()` as
`effect.img.xml`.

Fixture content (`objTestXML`), representing `Map.wz/Obj/effect.img.xml`:

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="effect.img">
  <imgdir name="quest">
    <imgdir name="gate">
      <imgdir name="0"><int name="obstacle" value="1"/></imgdir>
      <imgdir name="1"></imgdir>
    </imgdir>
  </imgdir>
</imgdir>
```

`TestInitObjIndexesObstacles`:

| assertion | expected |
|---|---|
| `InitObj(ten, dir)` | returns `nil` |
| `GetMapObjectRegistry().Get(ten, "effect/quest/gate/0")` | no error; `.Obstacle() == true` |
| `GetMapObjectRegistry().Get(ten, "effect/quest/gate/1")` | returns an error (`"not found"`) — no `obstacle` child means no entry |
| `ResolveObjKind(ten, "effect", "quest", "gate", "0")` | `"OBSTACLE"` |
| `ResolveObjKind(ten, "effect", "quest", "gate", "1")` | `"ENVIRONMENT"` |
| `ResolveObjKind(ten, "nothing", "at", "all", "0")` | `"ENVIRONMENT"` |

`TestInitObjMissingDirectory`: `InitObj(ten, filepath.Join(t.TempDir(), "absent"))`
returns a non-nil error, and afterwards
`ResolveObjKind(ten, "effect", "quest", "gate", "0")` is `"ENVIRONMENT"` — an
uninitialised registry defaults every object to `ENVIRONMENT`, never panics.

`TestInitObjTenantIsolation`: init under tenant A only; then
`ResolveObjKind(tenantB, "effect", "quest", "gate", "0")` is `"ENVIRONMENT"`.

Each test builds its own tenant with `tenant.Create(uuid.New(), "GMS", 83, 1)`.

- [ ] **Step 2: Run the test to verify it fails**

From `services/atlas-data/atlas.com/data`:
`go test ./map/ -run 'TestInitObj' -v`
Expected: FAIL — `undefined: InitObj`.

- [ ] **Step 3: Implement `map/object_registry.go`**

```go
package _map

import (
	"atlas-data/document"
	"atlas-data/xml"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// The two object kinds. These literal strings are a cross-service contract:
// task-278's environment-object kind uses the same two values, and the UI
// merges the two collections on the composite key "{kind}:{name}".
const (
	ObjKindEnvironment = "ENVIRONMENT"
	ObjKindObstacle    = "OBSTACLE"
)

// MapObjectDefinition is one Map.wz/Obj definition node keyed by
// "{oS}/{l0}/{l1}/{l2}". Only nodes carrying obstacle=1 are indexed; absence
// from the registry means ENVIRONMENT.
type MapObjectDefinition struct {
	id       string
	obstacle bool
}

func (m MapObjectDefinition) GetID() string { return m.id }

func (m MapObjectDefinition) Obstacle() bool { return m.obstacle }

var (
	moRg   *document.Registry[string, MapObjectDefinition]
	moOnce sync.Once
)

func GetMapObjectRegistry() *document.Registry[string, MapObjectDefinition] {
	moOnce.Do(func() {
		moRg = document.NewRegistry[string, MapObjectDefinition]()
	})
	return moRg
}

// InitObj walks dir (= {root}/Map.wz/Obj) once and indexes every
// {l0}/{l1}/{l2} node carrying obstacle=1. Doing this per-map inside the
// reader would re-parse the same Obj images thousands of times across a
// 5261-map ingest.
func InitObj(t tenant.Model, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	indexed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".img.xml") {
			continue
		}
		oS := strings.TrimSuffix(e.Name(), ".img.xml")
		exml, err := xml.FromPathProvider(filepath.Join(dir, e.Name()))()
		if err != nil {
			return err
		}
		for _, l0 := range exml.ChildNodes {
			for _, l1 := range l0.ChildNodes {
				for _, l2 := range l1.ChildNodes {
					if l2.GetIntegerWithDefault("obstacle", 0) != 1 {
						continue
					}
					id := strings.Join([]string{oS, l0.Name, l1.Name, l2.Name}, "/")
					if _, err := GetMapObjectRegistry().Add(t, MapObjectDefinition{id: id, obstacle: true}); err != nil {
						return err
					}
					indexed++
				}
			}
		}
	}
	if indexed == 0 {
		// Visible in ingest logs, so a uniformly-ENVIRONMENT dataset is not
		// mistaken for correct data.
		return nil
	}
	return nil
}

// ResolveObjKind returns OBSTACLE when the referenced Obj definition carries
// obstacle=1, ENVIRONMENT otherwise. An uninitialised or empty registry
// therefore resolves everything to ENVIRONMENT, which is the correct default.
func ResolveObjKind(t tenant.Model, oS string, l0 string, l1 string, l2 string) string {
	d, err := GetMapObjectRegistry().Get(t, strings.Join([]string{oS, l0, l1, l2}, "/"))
	if err != nil || !d.Obstacle() {
		return ObjKindEnvironment
	}
	return ObjKindObstacle
}
```

`InitObj` takes no logger, so the "log the indexed count" requirement from
design §4.2 is satisfied by its two call sites in Task 5, which have loggers.
Change `InitObj`'s signature to return the count instead of swallowing it:

```go
func InitObj(t tenant.Model, dir string) (int, error)
```

returning `(indexed, nil)` on success and `(0, err)` on failure, and drop the
`if indexed == 0` block. Adjust the tests in Step 1 to destructure two values.

- [ ] **Step 4: Run the tests to verify they pass**

`go test ./map/ -run 'TestInitObj' -v` → PASS.
`go build ./...` and `go vet ./...` from `services/atlas-data/atlas.com/data`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/map/object_registry.go services/atlas-data/atlas.com/data/map/object_registry_test.go
git commit -m "feat(atlas-data): Map.wz/Obj obstacle definition registry"
```

---

## Task 5: `atlas-data` — wire `InitObj` into both ingest entry points

Module root: `services/atlas-data/atlas.com/data`

### Files

- `services/atlas-data/atlas.com/data/data/workers/mapw.go` — 39-41 is the neighbouring `InitString` call; the Obj init goes after `serializeArchive` (line 32-35) and before `registerAllInDirectory` (line 51)
- `services/atlas-data/atlas.com/data/data/processor.go` — `StartWorker`, `WorkerMap` branch, lines 113-129

**This is the correction C2 task.** The two sites have *different* existing
failure policies and each new call must match its own site, not the other's.

- [ ] **Step 1: Confirm the two existing policies before editing**

Run: `sed -n '36,53p' services/atlas-data/atlas.com/data/data/workers/mapw.go`
Expected to show `l.WithError(err).Warnf("map.InitString failed")` — warn, continue.

Run: `sed -n '113,130p' services/atlas-data/atlas.com/data/data/processor.go`
Expected to show `p.l.WithError(err).Errorf("Failed to initialize map string registry.")` followed by `return err` — hard error.

- [ ] **Step 2: Edit `mapw.go` (warn-and-continue)**

Insert after line 47 (the close of the `String.wz` `else` block) and before the
`mapDir` line at 50. The Obj directory lives inside `Map.wz`, which
`serializeArchive` has already unpacked to `root`, so this is NOT inside the
`String.wz` availability block:

```go
	// Index Map.wz/Obj once so the reader can resolve each named object's kind
	// without re-parsing the same Obj images per map.
	if n, err := _map.InitObj(t, filepath.Join(root, "Map.wz", "Obj")); err != nil {
		l.WithError(err).Warnf("map.InitObj failed; all map objects will resolve to ENVIRONMENT")
	} else {
		l.Infof("Indexed [%d] obstacle object definitions.", n)
	}
	defer func() { _ = _map.GetMapObjectRegistry().Clear(t) }()
```

- [ ] **Step 3: Edit `data/processor.go` (hard error)**

Insert in the `WorkerMap` branch, after the `npc.InitString` block (which ends
at line 121) and before the `RegisterAllData` call at line 122:

```go
		var objCount int
		if objCount, err = _map.InitObj(t, filepath.Join(path, "Map.wz", "Obj")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize map object registry.")
			return err
		}
		p.l.Infof("Indexed [%d] obstacle object definitions.", objCount)
```

and add the clear next to the existing
`_ = _map.GetMapStringRegistry().Clear(t)` at line 123:

```go
		_ = _map.GetMapObjectRegistry().Clear(t)
```

- [ ] **Step 4: Verify both sites compile and the policies differ as intended**

Run: `cd services/atlas-data/atlas.com/data && go build ./... && go vet ./...`
Expected: clean.

Run: `grep -n 'InitObj' services/atlas-data/atlas.com/data/data/workers/mapw.go services/atlas-data/atlas.com/data/data/processor.go`
Expected: exactly two call sites, one per file.

Run: `grep -n 'GetMapObjectRegistry().Clear' services/atlas-data/atlas.com/data/data/workers/mapw.go services/atlas-data/atlas.com/data/data/processor.go`
Expected: exactly two teardowns, one per file.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/data/workers/mapw.go services/atlas-data/atlas.com/data/data/processor.go
git commit -m "feat(atlas-data): initialize the map object registry in both ingest paths"
```

---

## Task 6: `atlas-data` — map object REST model and WZ reader

Module root: `services/atlas-data/atlas.com/data`

### Files

- `services/atlas-data/atlas.com/data/map/object/rest.go` — **new file**, `package object`
- `services/atlas-data/atlas.com/data/map/reader.go` — add `getObjects`; call it in `Read` next to `m.Reactors = getReactors(exml)` (line 104)
- `services/atlas-data/atlas.com/data/map/reader_test.go` — extend the `testXML` fixture and add tests

Patterns to copy:
- `services/atlas-data/atlas.com/data/map/reactor/rest.go:1-30` — the whole file is the `map/object/rest.go` template, except the id is a string, so `GetID` returns `r.Id` directly and `SetID` assigns it directly (no `strconv`).
- `services/atlas-data/atlas.com/data/map/reader.go:323-338` (`getBackgroundTypes`) — the numeric-layer iteration idiom, including `strconv.Atoi(bt.Name)` with `continue` on error. **This, not `getReactors`, is the shape to follow** (correction C1).

Read-only reference: `services/atlas-data/atlas.com/data/map/reader.go:41` —
`t := tenant.MustFromContext(ctx)` is already in scope inside `Read`, so
`getObjects` can take `t` as its first parameter exactly as `getLife(t, exml)`
does at line 105.

- [ ] **Step 1: Write the failing test**

Extend `services/atlas-data/atlas.com/data/map/reader_test.go`. The existing
`testXML` constant already contains empty `<imgdir name="obj"></imgdir>` nodes
under layers `0` and `1`. Add a second fixture constant `objTestXML` — a copy
of `testXML` with layer `1`'s `obj` node populated:

```xml
      <imgdir name="obj">
        <imgdir name="0"><string name="oS" value="effect"/><string name="l0" value="quest"/><string name="l1" value="gate"/><string name="l2" value="1"/><int name="x" value="640"/><int name="y" value="120"/><int name="z" value="0"/><int name="f" value="0"/><int name="zM" value="0"/><string name="name" value="gate"/></imgdir>
        <imgdir name="1"><string name="oS" value="effect"/><string name="l0" value="quest"/><string name="l1" value="gate"/><string name="l2" value="0"/><int name="x" value="10"/><int name="y" value="20"/><int name="z" value="3"/><int name="f" value="0"/><int name="zM" value="0"/></imgdir>
        <imgdir name="2"><string name="oS" value="effect"/><string name="l0" value="quest"/><string name="l1" value="gate"/><string name="l2" value="0"/><int name="x" value="30"/><int name="y" value="40"/><int name="z" value="1"/><int name="f" value="0"/><int name="zM" value="0"/><string name="name" value="rock"/></imgdir>
      </imgdir>
```

Entry `1` has no `name` and must be skipped. Entry `2`'s `{l0}/{l1}/{l2}` is
`quest/gate/0`, which the Task 4 fixture indexes as `obstacle=1`.

`TestGetObjectsOnlyNamedEntries` — parse `objTestXML` via the same
`xml.FromStringProvider`/`np` mechanism the existing reader tests use (read
`reader_test.go`'s existing `Read(...)` invocation to reuse it verbatim). With
the object registry NOT initialised for this tenant:

| index | id | kind | name | objectSource | l0 | l1 | l2 | x | y | z | layer |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | `ENVIRONMENT:gate` | `ENVIRONMENT` | `gate` | `effect` | `quest` | `gate` | `1` | 640 | 120 | 0 | 1 |
| 1 | `ENVIRONMENT:rock` | `ENVIRONMENT` | `rock` | `effect` | `quest` | `gate` | `0` | 30 | 40 | 1 | 1 |

Assert `len(m.Objects) == 2` — the unnamed entry is absent. Assert the order is
`(kind, name)` ascending, so `gate` precedes `rock`.

`TestGetObjectsResolvesObstacle` — same fixture, but first call
`InitObj(ten, dir)` with the Task 4 `effect.img.xml` fixture written to a
`t.TempDir()`. Now:

| index | id | kind | name |
|---|---|---|---|
| 0 | `OBSTACLE:rock` | `OBSTACLE` | `rock` |
| 1 | `ENVIRONMENT:gate` | `ENVIRONMENT` | `gate` |

(`rock` resolves to `OBSTACLE` because its `l2` is `0`; `gate`'s `l2` is `1`,
unindexed, so `ENVIRONMENT`. Sorting by `(kind, name)` puts `ENVIRONMENT`
before `OBSTACLE` alphabetically — assert the exact slice order the
implementation produces, `ENVIRONMENT:gate` then `OBSTACLE:rock`, and fix the
table above to that order.)

`TestGetObjectsDuplicateIdKeepsFirst` — a third fixture with two named entries
both `name="gate"` and both resolving to `ENVIRONMENT`, at different
coordinates (`x=1` and `x=2`). Expect exactly one object, `Id ==
"ENVIRONMENT:gate"`, `X == 1` — first wins (design D7).

`TestGetObjectsEmptyLayers` — the unmodified `testXML` (all `obj` nodes empty)
yields `len(m.Objects) == 0` and a non-nil slice.

- [ ] **Step 2: Run the test to verify it fails**

From `services/atlas-data/atlas.com/data`:
`go test ./map/ -run 'TestGetObjects' -v`
Expected: FAIL — `m.Objects undefined`.

- [ ] **Step 3: Implement `map/object/rest.go`**

```go
package object

// RestModel is one named WZ object declared on a map. Id is "{KIND}:{name}",
// deliberately the same composite key task-278's environment-object resource
// uses, so the UI merges the two collections by id rather than by heuristic.
type RestModel struct {
	Id           string `json:"-"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	ObjectSource string `json:"objectSource"`
	L0           string `json:"l0"`
	L1           string `json:"l1"`
	L2           string `json:"l2"`
	X            int16  `json:"x"`
	Y            int16  `json:"y"`
	Z            int32  `json:"z"`
	Layer        uint32 `json:"layer"`
}

func (r RestModel) GetName() string {
	return "map-objects"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}
```

- [ ] **Step 4: Implement `getObjects` in `map/reader.go`**

Add next to `getBackgroundTypes` (after line 338):

```go
// getObjects collects the map's named WZ objects. Unlike reactors, which sit
// at a single top-level node, objects live per numeric layer imgdir under an
// "obj" child, so this iterates the layers the way getBackgroundTypes does and
// descends one level further. Only entries carrying a non-empty "name" are
// exposed: those are the only objects addressable by SetObjectState /
// FieldObstacleOnOff.
func getObjects(t tenant.Model, exml xml.Node) []object.RestModel {
	results := make([]object.RestModel, 0)
	seen := make(map[string]bool)
	for _, layer := range exml.ChildNodes {
		layerNum, err := strconv.Atoi(layer.Name)
		if err != nil {
			continue
		}
		od, err := layer.ChildByName("obj")
		if err != nil {
			continue
		}
		for _, o := range od.ChildNodes {
			name := o.GetString("name", "")
			if name == "" {
				continue
			}
			oS := o.GetString("oS", "")
			l0 := o.GetString("l0", "")
			l1 := o.GetString("l1", "")
			l2 := o.GetString("l2", "")
			kind := ResolveObjKind(t, oS, l0, l1, l2)
			id := kind + ":" + name
			if seen[id] {
				// Duplicate {kind}:{name} within one map would produce
				// duplicate ids in the JSON:API included array. Keep the first.
				continue
			}
			seen[id] = true
			results = append(results, object.RestModel{
				Id:           id,
				Kind:         kind,
				Name:         name,
				ObjectSource: oS,
				L0:           l0,
				L1:           l1,
				L2:           l2,
				X:            int16(o.GetIntegerWithDefault("x", 0)),
				Y:            int16(o.GetIntegerWithDefault("y", 0)),
				Z:            o.GetIntegerWithDefault("z", 0),
				Layer:        uint32(layerNum),
			})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Kind != results[j].Kind {
			return results[i].Kind < results[j].Kind
		}
		return results[i].Name < results[j].Name
	})
	return results
}
```

Add `"sort"` and `object "atlas-data/map/object"` to the imports if absent.

In `Read`, after `m.Reactors = getReactors(exml)` (line 104):

```go
			m.Objects = getObjects(t, exml)
```

(The `RestModel.Objects` field itself is added in Task 7; to keep this task
independently buildable, add the field declaration
`Objects []object.RestModel \`json:"-"\`` to `map/rest.go` here, and add the
five relationship hooks in Task 7.)

- [ ] **Step 5: Run the tests to verify they pass**

`go test ./map/ -run 'TestGetObjects' -v` → PASS.
`go test ./map/ -v` → all existing reader tests still PASS.
`go build ./...` and `go vet ./...`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/map/object/ services/atlas-data/atlas.com/data/map/reader.go services/atlas-data/atlas.com/data/map/reader_test.go services/atlas-data/atlas.com/data/map/rest.go
git commit -m "feat(atlas-data): read named WZ map objects with obstacle kind"
```

---

## Task 7: `atlas-data` — persist map objects as a JSON:API relationship

Module root: `services/atlas-data/atlas.com/data`

`atlas-data` stores a map as its marshalled JSON:API *document*
(`document/db_storage.go:123-127` does `jsonapi.MarshalToStruct` then
`json.Marshal`). That is why `Portals`/`Reactors`/`NPCs`/`Monsters` are
`json:"-"` and still round-trip — they travel as relationships plus `included`.
Objects must do the same or they will silently vanish on storage.

### Files

- `services/atlas-data/atlas.com/data/map/rest.go` — four hooks (the `Objects` field itself landed in Task 6)
- `services/atlas-data/atlas.com/data/map/rest_test.go` — **new file** (round-trip test)

Patterns to copy — the `reactors` blocks, which are the exact template:
`services/atlas-data/atlas.com/data/map/rest.go:87` (`GetReferences`),
`:102-108` (`GetReferencedIDs`), `:131-133` (`GetReferencedStructs`),
`:160-171` (`SetToManyReferenceIDs`). Verify the current line numbers with
`grep -n 'reactors' services/atlas-data/atlas.com/data/map/rest.go` before editing.

- [ ] **Step 1: Write the failing test**

`services/atlas-data/atlas.com/data/map/rest_test.go`, package `_map`.

`TestMapObjectsSurviveJSONAPIRoundTrip`. Build a `RestModel` with `Id` set and
`Objects` populated with two entries:

| field | object A | object B |
|---|---|---|
| `Id` | `ENVIRONMENT:gate` | `OBSTACLE:menhir0` |
| `Kind` | `ENVIRONMENT` | `OBSTACLE` |
| `Name` | `gate` | `menhir0` |
| `ObjectSource` | `effect` | `trapGL` |
| `L0` | `quest` | `ckPQ` |
| `L1` | `gate` | `menhir` |
| `L2` | `1` | `0` |
| `X` | 640 | -30 |
| `Y` | 120 | 45 |
| `Z` | 0 | 7 |
| `Layer` | 3 | 2 |

Marshal with `jsonapi.MarshalToStruct(m, nil)` then `json.Marshal`, then
`jsonapi.Unmarshal` back into a fresh `RestModel`.

Assertions — this is the load-bearing part, since relationship ids alone would
round-trip while attributes silently would not:
- `len(out.Objects) == 2`
- the objects sorted by `Id` equal, **field by field**, the two rows above —
  assert `Kind`, `Name`, `ObjectSource`, `L0`, `L1`, `L2`, `X`, `Y`, `Z`, and
  `Layer` individually, not just `Id`.
- the marshalled document's `included` array contains two entries of
  `"type": "map-objects"` with ids `ENVIRONMENT:gate` and `OBSTACLE:menhir0`.

Copy the exact marshal/unmarshal helper calls from
`services/atlas-data/atlas.com/data/document/db_storage.go:120-130` so the test
exercises the same path storage does.

- [ ] **Step 2: Run the test to verify it fails**

`go test ./map/ -run TestMapObjectsSurviveJSONAPIRoundTrip -v`
Expected: FAIL — `len(out.Objects)` is 0, because no relationship hook emits them.

- [ ] **Step 3: Implement the four hooks**

In `GetReferences`, next to the reactors line:

```go
	rfs = append(rfs, jsonapi.Reference{Type: "map-objects", Name: "objects"})
```

In `GetReferencedIDs`, next to the reactors loop:

```go
	for _, x := range r.Objects {
		rfs = append(rfs, jsonapi.ReferenceID{
			ID:   x.Id,
			Type: "map-objects",
			Name: "objects",
		})
	}
```

In `GetReferencedStructs`, next to the reactors loop:

```go
	for _, x := range r.Objects {
		rfs = append(rfs, x)
	}
```

In `SetToManyReferenceIDs`, next to the reactors block:

```go
	if name == "objects" {
		res := make([]object.RestModel, 0)
		for _, x := range IDs {
			rm := object.RestModel{}
			err := rm.SetID(x)
			if err != nil {
				return err
			}
			res = append(res, rm)
		}
		r.Objects = res
	}
```

- [ ] **Step 4: Run the test to verify it passes**

`go test ./map/ -run TestMapObjectsSurviveJSONAPIRoundTrip -v` → PASS.
`go test ./map/ -v` → all PASS.
`go build ./...` and `go vet ./...`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/map/rest.go services/atlas-data/atlas.com/data/map/rest_test.go
git commit -m "feat(atlas-data): store map objects as a JSON:API relationship"
```

---

## Task 8: `atlas-data` — `GET /api/data/maps/{mapId}/objects`

Module root: `services/atlas-data/atlas.com/data`

### Files

- `services/atlas-data/atlas.com/data/map/processor.go` — add `objectProvider` + `GetObjects`, next to `GetReactors` (lines 300-302)
- `services/atlas-data/atlas.com/data/map/resource.go` — register the route (next to the `/{mapId}/reactors` line, line 38) and add `handleGetMapObjectsRequest` (next to `handleGetMapReactorsRequest`, lines 234-259)
- `services/atlas-data/atlas.com/data/map/resource_test.go` — **new file**

Patterns to copy: `services/atlas-data/atlas.com/data/map/processor.go:295-302`
(`reactorProvider` + `GetReactors`) and
`services/atlas-data/atlas.com/data/map/resource.go:234-259`
(`handleGetMapReactorsRequest`, verbatim except the type parameter and
processor call).

Note `paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)`
— this service's convention, different from `atlas-maps` (correction C3).

- [ ] **Step 1: Write the failing test**

`services/atlas-data/atlas.com/data/map/resource_test.go`, package `_map`.
The handler needs a `*gorm.DB`; find how the existing atlas-data tests obtain
one by running
`grep -rn 'gorm.Open\|sqlite\|testDB' services/atlas-data/atlas.com/data --include='*_test.go' | head`
and reuse whatever harness that turns up. If no such harness exists, seed the
map document through `NewStorage(l, db).…` against an in-memory SQLite handle
created the same way `document/db_storage.go` expects.

`TestGetMapObjectsEmpty` — store a map with `Objects` empty, then
`GET /api/data/maps/{id}/objects`. Expect `200`, `data` is `[]`, not `404`.

`TestGetMapObjectsReturnsRows` — store a map whose `Objects` are the two rows
from Task 7 Step 1. `GET /api/data/maps/{id}/objects` → `200`, two items,
`type == "map-objects"`, ids `ENVIRONMENT:gate` and `OBSTACLE:menhir0`, and
the first item's attributes are exactly
`{kind: "ENVIRONMENT", name: "gate", objectSource: "effect", l0: "quest",
l1: "gate", l2: "1", x: 640, y: 120, z: 0, layer: 3}`.

`TestGetMapObjectsUnknownMap` — `GET /api/data/maps/999999999/objects` with no
such map stored → `404`.

- [ ] **Step 2: Run the test to verify it fails**

`go test ./map/ -run 'TestGetMapObjects' -v`
Expected: FAIL — `404` on every case, because the route is unregistered.

- [ ] **Step 3: Implement the processor methods**

```go
func (p *ProcessorImpl) objectProvider(s *Storage, mapId _map.Id) model.Provider[[]object.RestModel] {
	m, err := s.ByIdProvider(p.ctx)(strconv.Itoa(int(mapId)))()
	if err != nil {
		return model.ErrorProvider[[]object.RestModel](err)
	}
	return model.FixedProvider(m.Objects)
}

func (p *ProcessorImpl) GetObjects(s *Storage, mapId _map.Id) ([]object.RestModel, error) {
	return p.objectProvider(s, mapId)()
}
```

Add `GetObjects(s *Storage, mapId _map.Id) ([]object.RestModel, error)` to the
`Processor` interface in the same file (find it with
`grep -n 'GetReactors' services/atlas-data/atlas.com/data/map/processor.go`
and add the new method beside every occurrence).

- [ ] **Step 4: Implement the handler and route**

Route, next to `resource.go:38`:

```go
			r.HandleFunc("/{mapId}/objects", registerGet("get_map_objects", handleGetMapObjectsRequest(db))).Methods(http.MethodGet)
```

Handler, cloned from `handleGetMapReactorsRequest`:

```go
func handleGetMapObjectsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, err.Error())
					return
				}

				s := NewStorage(d.Logger(), db)
				res, err := NewProcessor(d.Logger(), d.Context(), db).GetObjects(s, mapId)
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate map %d.", mapId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				paged := paginate.Slice(res, page)
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]object.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}
```

- [ ] **Step 5: Run the tests to verify they pass**

`go test ./map/ -v` → all PASS.
`go build ./...` and `go vet ./...`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/map/processor.go services/atlas-data/atlas.com/data/map/resource.go services/atlas-data/atlas.com/data/map/resource_test.go
git commit -m "feat(atlas-data): GET /api/data/maps/{mapId}/objects"
```

---

## Task 9: `atlas-data` — document map objects

### Files

- `services/atlas-data/docs/rest.md` — 1388 lines; add the `/objects` endpoint next to the `/reactors` section
- `services/atlas-data/docs/domain.md` — 211 lines; add the map-object entity

- [ ] **Step 1: Locate the reactors sections to sit beside**

Run: `grep -n 'reactors' services/atlas-data/docs/rest.md | head -20`
Run: `grep -n -i 'reactor' services/atlas-data/docs/domain.md`

- [ ] **Step 2: Write the `rest.md` section**

Document `GET /api/data/maps/{mapId}/objects`:
- JSON:API collection, type `map-objects`, id `{KIND}:{name}`.
- Attributes: `kind` (`ENVIRONMENT` | `OBSTACLE`), `name`, `objectSource`
  (the WZ `oS`), `l0`, `l1`, `l2`, `x`, `y`, `z`, `layer`.
- Only `obj` entries with a non-empty `name` are exposed — those are the only
  objects addressable by `SetObjectState` / `FieldObstacleOnOff`.
- Sorted by `kind`, then `name`.
- `200` with an empty array for a map with no named objects; `404` only if the
  map itself is unknown.
- Pagination via `page[number]` / `page[size]`.
- Include the example document from `prd.md` §5.2.

- [ ] **Step 3: Write the `domain.md` section**

Document the map-object entity:
- Derived from `Map.wz/Map/Map{n}/{id}.img.xml`, per numeric layer imgdir, under
  the layer's `obj` child.
- `kind` is `OBSTACLE` when the referenced definition at
  `Map.wz/Obj/{oS}.img.xml` → `{l0}/{l1}/{l2}` carries `obstacle=1`; otherwise
  `ENVIRONMENT`. That index is built once per ingest by `InitObj`.
- Stored on the map document as a JSON:API relationship named `objects`, the
  same way portals, reactors, NPCs, and monsters are.
- **Migration note (required by design §8):** adding `Objects` to the stored
  map document means **existing ingested tenants show no map objects until the
  MAP worker re-runs.** No migration is possible or needed — the data is
  re-derivable from the WZ archive.

- [ ] **Step 4: Verify no absolute paths**

Run: `grep -n '/home/' services/atlas-data/docs/rest.md services/atlas-data/docs/domain.md`
Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/docs/rest.md services/atlas-data/docs/domain.md
git commit -m "docs(atlas-data): document map objects and the obstacle kind rule"
```

---

## Task 10: Gateway routes

### Files

- `deploy/shared/routes.conf` — insert two location blocks in the atlas-maps instance-route group at lines 490-503

This is correction C4. The generic `^/api/worlds(/.*)?$` catch-all at
**line 691 proxies to `atlas-world`, not `atlas-maps`** — without an explicit
block, `/environment` requests would silently reach the wrong service and 404
misleadingly. The new blocks must appear *before* line 691; nginx matches
regex locations in file order.

- [ ] **Step 1: Confirm the insertion point and the catch-all target**

Run: `sed -n '488,510p' deploy/shared/routes.conf`
Expected: the `characters`, `weather`, and `jukebox` blocks pointing at
`atlas-maps:8080`, followed by a `monsters` block pointing at
`atlas-monsters:8080`.

Run: `sed -n '689,695p' deploy/shared/routes.conf`
Expected: `location ~ ^/api/worlds(/.*)?$ {` … `set $u "atlas-world:8080";`

- [ ] **Step 2: Insert the two blocks**

Immediately after the `jukebox` block (which ends at line 503), matching the
surrounding style exactly:

```
location ~ ^/api/worlds/[^/]+/channels/[^/]+/maps/[^/]+/instances/[^/]+/environment(/.*)?$ {
  set $u "atlas-maps:8080";
  proxy_pass http://$u$request_uri;
}

location ~ ^/api/fields(/.*)?$ {
  set $u "atlas-maps:8080";
  proxy_pass http://$u$request_uri;
}
```

The `/environment` block is inert until `[278]` lands (nginx proxies to
`atlas-maps`, which returns its own 404) and the PRD requires it regardless —
it is not a stub, it is routing config for an endpoint another branch already
implements.

Copy the exact directive set (including any `proxy_set_header` lines) from the
adjacent `characters` block rather than the abbreviated form above, if that
block carries more directives.

- [ ] **Step 3: Verify ordering and syntax**

Run: `grep -n 'api/fields\|instances/\[^/\]+/environment\|api/worlds(/' deploy/shared/routes.conf`
Expected: the `/environment` and `/fields` line numbers are both **less than**
the `^/api/worlds(/.*)?$` line number.

Run: `grep -c 'location ~ \^/api/fields' deploy/shared/routes.conf`
Expected: `1`.

- [ ] **Step 4: Commit**

```bash
git add deploy/shared/routes.conf
git commit -m "feat(gateway): route /api/fields and field environment to atlas-maps"
```

---

## Task 11: `atlas-ui` — Definition/Runtime terminology and kind badge

### Files

- `services/atlas-ui/src/components/features/maps/SurfaceKindBadge.tsx` — **new file**
- `services/atlas-ui/src/components/features/maps/MapDetailTabs.tsx` — 321 lines; rename the monster tab trigger at lines 90-92
- `services/atlas-ui/src/components/features/maps/MapEntitySummary.tsx` — 235 lines; rename the monster heading
- `services/atlas-ui/src/components/features/maps/MapHeader.tsx` — 52 lines; render `<SurfaceKindBadge kind="definition" />` in the badge row at lines 40-49
- `services/atlas-ui/src/components/features/maps/__tests__/SurfaceKindBadge.test.tsx` — **new file**

Patterns to copy:
`services/atlas-ui/src/components/features/maps/MapHeader.tsx:41` — the
`<Badge variant="secondary">` usage; the badge primitive is
`@/components/ui/badge`.
Test pattern: `services/atlas-ui/src/components/common/__tests__/EmptyState.test.tsx`
(a small `render`/`screen` component test).

Test command for every UI task, from `services/atlas-ui`:
`npm test -- <path>` (the `test` script is `vitest run`).

- [ ] **Step 1: Write the failing test**

`services/atlas-ui/src/components/features/maps/__tests__/SurfaceKindBadge.test.tsx`.

`describe("SurfaceKindBadge")`:

| test name | render | expect |
|---|---|---|
| `renders the word Definition` | `<SurfaceKindBadge kind="definition" />` | `screen.getByText("Definition")` is in the document |
| `renders the word Runtime` | `<SurfaceKindBadge kind="runtime" />` | `screen.getByText("Runtime")` is in the document |
| `never relies on colour alone` | both variants | each rendered element's `textContent` is non-empty — the requirement (FR-3) is that the word is always present, so a colour-only badge fails this |

- [ ] **Step 2: Run the test to verify it fails**

Run from `services/atlas-ui`:
`npm test -- src/components/features/maps/__tests__/SurfaceKindBadge.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `SurfaceKindBadge.tsx`**

```tsx
import { Badge } from "@/components/ui/badge";

interface SurfaceKindBadgeProps {
  kind: "definition" | "runtime";
}

/**
 * Marks a surface as static configuration ("Definition") or live instance
 * state ("Runtime"). FR-3: the distinction must never be carried by colour
 * alone, so the badge always renders its word; colour is a secondary cue.
 */
export function SurfaceKindBadge({ kind }: SurfaceKindBadgeProps) {
  const label = kind === "definition" ? "Definition" : "Runtime";
  return (
    <Badge variant={kind === "definition" ? "outline" : "default"}>
      {label}
    </Badge>
  );
}
```

- [ ] **Step 4: Apply the terminology renames**

In `MapDetailTabs.tsx`, the monster `TabsTrigger` (currently
`Monsters {monsters && \`(${monsters.length})\`}`) becomes
`Monster Spawns {monsters && \`(${monsters.length})\`}`. Verify the exact
current text first with
`grep -n 'Monsters' services/atlas-ui/src/components/features/maps/MapDetailTabs.tsx`.

In `MapEntitySummary.tsx`, the monster section heading becomes
`Configured Monster Spawns`. Find it with
`grep -n -i 'monster' services/atlas-ui/src/components/features/maps/MapEntitySummary.tsx`.

In `MapHeader.tsx`, add `<SurfaceKindBadge kind="definition" />` as the first
child of the badge row `<div className="flex items-center gap-2 flex-wrap">`
(line 40), before the `streetName` badge.

No data source changes (FR-5) — do not touch any hook or service in this task.

- [ ] **Step 5: Run the tests to verify they pass**

Run from `services/atlas-ui`:
`npm test -- src/components/features/maps/`
Expected: PASS.
Then `npx tsc --noEmit` → clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/maps/
git commit -m "feat(atlas-ui): definition/runtime kind badge and spawn terminology"
```

---

## Task 12: `atlas-ui` — map objects service, hook, and Definition tab

### Files

- `services/atlas-ui/src/services/api/map-entities.service.ts` — 89 lines; add `MapObjectData` and `getObjects`
- `services/atlas-ui/src/lib/hooks/api/useMapEntities.ts` — add `objects` to `mapEntityKeys` and a `useMapObjects` hook
- `services/atlas-ui/src/components/features/maps/MapObjectsTable.tsx` — **new file**
- `services/atlas-ui/src/components/features/maps/MapDetailTabs.tsx` — add the `Map Objects (N)` tab and content
- `services/atlas-ui/src/pages/MapDetailPage.tsx` — 89 lines; call `useMapObjects(id)` and pass through
- `services/atlas-ui/src/components/features/maps/__tests__/MapObjectsTable.test.tsx` — **new file**

Patterns to copy:
- `services/atlas-ui/src/services/api/map-entities.service.ts:34-45` (the `MapReactorData` interface) and `:80-82` (the `getReactors` method).
- `services/atlas-ui/src/lib/hooks/api/useMapEntities.ts:42-53` (`useMapReactors` — definition cache profile, `staleTime`/`gcTime` both `10 * 60 * 1000`, `enabled: !!mapId && !!activeTenant`).
- `services/atlas-ui/src/components/features/maps/MapDetailTabs.tsx:85-96` (the `TabsList` / `TabsTrigger` block) and the reactors `TabsContent` block below it.

- [ ] **Step 1: Write the failing test**

`services/atlas-ui/src/components/features/maps/__tests__/MapObjectsTable.test.tsx`.

Fixture rows (`MapObjectData[]`):

```ts
const objects = [
  { id: "ENVIRONMENT:gate", type: "map-objects", attributes: { kind: "ENVIRONMENT", name: "gate", objectSource: "effect", l0: "quest", l1: "gate", l2: "1", x: 640, y: 120, z: 0, layer: 3 } },
  { id: "OBSTACLE:menhir0", type: "map-objects", attributes: { kind: "OBSTACLE", name: "menhir0", objectSource: "trapGL", l0: "ckPQ", l1: "menhir", l2: "0", x: -30, y: 45, z: 7, layer: 2 } },
];
```

| test name | render | expect |
|---|---|---|
| `renders one row per object` | `<MapObjectsTable objects={objects} />` | `screen.getByText("gate")` and `screen.getByText("menhir0")` both present |
| `shows the kind` | same | `screen.getByText("ENVIRONMENT")` and `screen.getByText("OBSTACLE")` present |
| `shows the WZ source` | same | `screen.getByText("effect")` and `screen.getByText("trapGL")` present |
| `shows the position` | same | text matching `640` and `120` is present for the gate row |
| `empty state` | `<MapObjectsTable objects={[]} />` | a message is rendered and no table rows; assert on the empty-state copy, e.g. `screen.getByText(/no named objects/i)` |
| `loading` | `<MapObjectsTable objects={undefined} />` | a skeleton/loading affordance renders, no crash |

- [ ] **Step 2: Run the test to verify it fails**

`npm test -- src/components/features/maps/__tests__/MapObjectsTable.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Extend the service**

Append to `map-entities.service.ts`, beside `MapReactorData`:

```ts
export interface MapObjectData {
  id: string;
  type: string;
  attributes: {
    kind: string;
    name: string;
    objectSource: string;
    l0: string;
    l1: string;
    l2: string;
    x: number;
    y: number;
    z: number;
    layer: number;
  };
}
```

and inside `MapEntitiesService`, beside `getReactors`:

```ts
  async getObjects(mapId: string): Promise<MapObjectData[]> {
    return api.getList<MapObjectData>(`/api/data/maps/${mapId}/objects`);
  }
```

- [ ] **Step 4: Extend the hooks**

Add to `mapEntityKeys`:

```ts
  objects: (mapId: string) => ["maps", mapId, "objects"] as const,
```

and the hook, copying `useMapReactors` exactly — **the definition cache
profile** (`staleTime: 10 * 60 * 1000`, `gcTime: 10 * 60 * 1000`), no
`refetchInterval`:

```ts
export function useMapObjects(
  mapId: string,
): UseQueryResult<MapObjectData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: mapEntityKeys.objects(mapId),
    queryFn: () => mapEntitiesService.getObjects(mapId),
    enabled: !!mapId && !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
```

- [ ] **Step 5: Implement `MapObjectsTable.tsx` and wire the tab**

`MapObjectsTable({ objects, error }: { objects?: MapObjectData[]; error?: Error })`
renders columns **Kind, Name, WZ Source (`objectSource`), Position (`x`, `y`)**
— and additionally `l0/l1/l2` and `layer` if the surrounding table style
accommodates them. Use the same table primitive the reactors tab already uses;
read the reactors `TabsContent` block in `MapDetailTabs.tsx` and match it.

In `MapDetailTabs.tsx`: add the trigger after the reactors trigger —
`<TabsTrigger value="objects">Map Objects {objects && \`(${objects.length})\`}</TabsTrigger>`
— and a matching `TabsContent value="objects"` wrapping `<MapObjectsTable …/>`
in the same `Card`/`CardContent` shell the neighbouring tabs use. Extend the
component's props with `objects?: MapObjectData[]` and `objectsError?: Error`.

In `MapDetailPage.tsx`: add
`const { data: objects, error: objectsError } = useMapObjects(id);` beside the
other entity hooks (lines 23-26) and pass `objects` / `objectsError` into
`<MapDetailTabs>` (lines 77-85). Do **not** touch the
`HoverHighlightProvider` / `MapImagePanel` / `MapEntitySummary` /
`ConnectedMapsRow` wiring (FR-4).

- [ ] **Step 6: Run the tests to verify they pass**

`npm test -- src/components/features/maps/` → PASS.
`npx tsc --noEmit` → clean.
`npm run lint` → clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/services/api/map-entities.service.ts services/atlas-ui/src/lib/hooks/api/useMapEntities.ts services/atlas-ui/src/components/features/maps/ services/atlas-ui/src/pages/MapDetailPage.tsx
git commit -m "feat(atlas-ui): Map Objects tab on the map definition page"
```

---

## Task 13: `atlas-ui` — fields and worlds services and hooks

### Files

- `services/atlas-ui/src/services/api/fields.service.ts` — **new file**
- `services/atlas-ui/src/services/api/worlds.service.ts` — **new file**
- `services/atlas-ui/src/lib/hooks/api/useFields.ts` — **new file**
- `services/atlas-ui/src/lib/hooks/api/useWorlds.ts` — **new file**
- `services/atlas-ui/src/lib/hooks/api/__tests__/useFields.test.tsx` — **new file**

Patterns to copy:
- `services/atlas-ui/src/services/api/map-entities.service.ts` lines 65-87 (the service class + exported singleton shape).
- `services/atlas-ui/src/lib/hooks/api/useMapEntities.ts` lines 11-29 (key factory + `useQuery` shape).
- `services/atlas-ui/src/lib/hooks/api/__tests__/useMaps.test.tsx` — the existing hook-test harness (QueryClientProvider wrapper, `vi.mock` of the service module). Read it before writing the test.

Read-only references:
- `services/atlas-ui/src/lib/api/client.ts:357` — `getList<T>(url, options?): Promise<T[]>`.
- `services/atlas-ui/src/context/tenant-context.tsx:68` — `queryClient.clear()` fires on tenant change, which is why tenant is deliberately **not** in the query key (design D9). Do not add it.

**Trailing-slash gotcha:** the worlds *collection* is registered as
`r.HandleFunc("/", …)` under the `/worlds` subrouter
(`services/atlas-world/atlas.com/world/world/resource.go:29`), so the URL is
`/api/worlds/` **with a trailing slash**. Channels is registered as `""` under
`/worlds/{worldId}/channels`, so `/api/worlds/{id}/channels` without one.
Confirm both with
`grep -n 'HandleFunc' services/atlas-world/atlas.com/world/world/resource.go services/atlas-world/atlas.com/world/channel/resource.go`
before writing the URLs.

- [ ] **Step 1: Write the failing test**

`services/atlas-ui/src/lib/hooks/api/__tests__/useFields.test.tsx`, mocking
`@/services/api/fields.service`.

| test name | setup | expect |
|---|---|---|
| `useFields calls the endpoint with no filters` | `useFields({})` | `fieldsService.getFields` called with `{}` |
| `useFields passes each filter through` | `useFields({ worldId: 0, channelId: 2, mapId: 910340000 })` | `getFields` called with exactly `{ worldId: 0, channelId: 2, mapId: 910340000 }` |
| `useFieldsForMap filters by mapId only` | `useFieldsForMap("910340000")` | `getFields` called with `{ mapId: 910340000 }` — no world, no channel (FR-9: spans every world and channel) |
| `runtime cache profile` | inspect the query options | `staleTime === 5000` and `gcTime === 60000` |
| `no polling` | inspect the query options | `refetchInterval` is `undefined` (FR-39) |
| `key namespace is disjoint from definition` | `fieldKeys.list({})` | the key's first element is `"fields"`, never `"maps"` (FR-41) |

For the cache-profile and polling assertions, export the option objects (or the
key factory plus a small `fieldQueryOptions` helper) so the test can assert on
them directly rather than reaching into React Query internals.

- [ ] **Step 2: Run the test to verify it fails**

`npm test -- src/lib/hooks/api/__tests__/useFields.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `fields.service.ts`**

```ts
import { api } from "@/lib/api/client";

export interface FieldData {
  id: string; // "{worldId}:{channelId}:{mapId}:{instanceId}"
  type: string;
  attributes: {
    worldId: number;
    channelId: number;
    mapId: number;
    instanceId: string;
    characterCount: number;
  };
}

export interface FieldFilters {
  worldId?: number;
  channelId?: number;
  mapId?: number;
}

class FieldsService {
  async getFields(filters: FieldFilters): Promise<FieldData[]> {
    const params = new URLSearchParams();
    if (filters.worldId !== undefined)
      params.set("filter[worldId]", String(filters.worldId));
    if (filters.channelId !== undefined)
      params.set("filter[channelId]", String(filters.channelId));
    if (filters.mapId !== undefined)
      params.set("filter[mapId]", String(filters.mapId));
    const qs = params.toString();
    return api.getList<FieldData>(`/api/fields${qs ? `?${qs}` : ""}`);
  }
}

export const fieldsService = new FieldsService();
```

- [ ] **Step 4: Implement `worlds.service.ts`**

A `WorldsService` with `getWorlds(): Promise<WorldData[]>` hitting
`/api/worlds/` (trailing slash) and
`getChannels(worldId: number): Promise<ChannelData[]>` hitting
`/api/worlds/${worldId}/channels`. Define `WorldData` / `ChannelData` from the
actual payloads — read them first with
`grep -n 'json:' services/atlas-world/atlas.com/world/world/rest.go services/atlas-world/atlas.com/world/channel/rest.go`
and type only the attributes those files emit. Do not invent fields.

- [ ] **Step 5: Implement the hooks**

`useFields.ts` exports `fieldKeys`, `useFields(filters)`, and
`useFieldsForMap(mapId)`. All use the **runtime cache profile** — design D10:

```ts
export const fieldKeys = {
  all: ["fields"] as const,
  list: (f: FieldFilters) => ["fields", "list", f] as const,
};

const RUNTIME_STALE_TIME = 5 * 1000;
const RUNTIME_GC_TIME = 60 * 1000;
```

Every hook sets `enabled: !!activeTenant`, `staleTime: RUNTIME_STALE_TIME`,
`gcTime: RUNTIME_GC_TIME`, and **no `refetchInterval`**.

`useWorlds.ts` exports `useWorlds()` and `useChannels(worldId)`. These are
topology, not per-second runtime state — use the definition profile
(`10 * 60 * 1000` for both), matching `useMapEntities.ts`.

- [ ] **Step 6: Run the tests to verify they pass**

`npm test -- src/lib/hooks/api/__tests__/useFields.test.tsx` → PASS.
`npx tsc --noEmit` and `npm run lint` → clean.

Run: `grep -rn 'refetchInterval' services/atlas-ui/src/lib/hooks/api/useFields.ts services/atlas-ui/src/lib/hooks/api/useWorlds.ts`
Expected: no output (FR-39).

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/services/api/fields.service.ts services/atlas-ui/src/services/api/worlds.service.ts services/atlas-ui/src/lib/hooks/api/useFields.ts services/atlas-ui/src/lib/hooks/api/useWorlds.ts services/atlas-ui/src/lib/hooks/api/__tests__/useFields.test.tsx
git commit -m "feat(atlas-ui): fields and worlds services with the runtime cache profile"
```

---

## Task 14: `atlas-ui` — Live Fields section on the Map Definition page

### Files

- `services/atlas-ui/src/services/api/live-monsters.service.ts` — **new file**
- `services/atlas-ui/src/lib/hooks/api/useFieldRuntime.ts` — **new file** (`useLiveMonsters` only in this task; the rest arrives in Tasks 18/19/22)
- `services/atlas-ui/src/components/features/maps/LiveFieldsSection.tsx` — **new file**
- `services/atlas-ui/src/pages/MapDetailPage.tsx` — insert `<LiveFieldsSection mapId={id} />` between `<ConnectedMapsRow />` (line 75) and `<MapDetailTabs />` (line 77)
- `services/atlas-ui/src/components/features/maps/__tests__/LiveFieldsSection.test.tsx` — **new file**

Read-only reference: `services/atlas-monsters/atlas.com/monsters/world/resource.go:36`
registers `GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/monsters`.

**D12 — fan-out cap.** `useFieldsForMap(mapId)` is one request spanning every
world and channel (FR-9). FR-7 additionally wants a live-monster count per row,
which `/api/fields` deliberately does not return. Fan out one `useLiveMonsters`
query per listed field, **capped at 12 rows**. Beyond the cap, render the
monster column as `—` with a "showing monster counts for the first 12 fields"
note, and lean on the `View all in Fields` link. A pending or failed monster
query renders `—` and never blocks its row.

- [ ] **Step 1: Write the failing test**

`services/atlas-ui/src/components/features/maps/__tests__/LiveFieldsSection.test.tsx`,
mocking `@/lib/hooks/api/useFields` and `@/lib/hooks/api/useFieldRuntime`.

Fixture: a helper `makeFields(n)` producing `n` `FieldData` rows with
`worldId: 0`, `channelId: i + 1`, `mapId: 910340000`,
`instanceId: "00000000-0000-0000-0000-00000000000" + i`, `characterCount: i + 1`.

| test name | setup | expect |
|---|---|---|
| `renders one row per live field` | 3 fields | 3 rows; each shows its world, channel, instance, and character count |
| `empty state is explicit and never hidden` | `[]` | the section heading still renders **and** an empty-state message renders; assert on copy like `/no live fields/i`. The section is never conditionally unmounted (FR-8) |
| `each row links to the field page` | 1 field `(0, 1, 910340000, uuid A)` | a link whose `href` is `/fields/0/1/910340000/<uuid A>` |
| `offers a pre-filtered link into the locator` | 3 fields | a link whose `href` is `/fields?map=910340000` (FR-16) |
| `fan-out is capped at 12` | 15 fields | `useLiveMonsters` is invoked for exactly 12 of them; rows 13-15 show `—` in the monster column, and a "first 12" note renders |
| `a failed monster query does not unmount its row` | 2 fields, monster query for row 1 returns an error | row 1 still renders its world/channel/count and shows `—` in the monster column |
| `spans all worlds and channels` | fields across `(w0,c1)`, `(w0,c2)`, `(w1,c1)` | `useFieldsForMap` called with the mapId only; all three rows render |

- [ ] **Step 2: Run the test to verify it fails**

`npm test -- src/components/features/maps/__tests__/LiveFieldsSection.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `live-monsters.service.ts`**

A `LiveMonstersService` with

```ts
  async getMonsters(worldId: number, channelId: number, mapId: number, instanceId: string): Promise<LiveMonsterData[]> {
    return api.getList<LiveMonsterData>(
      `/api/worlds/${worldId}/channels/${channelId}/maps/${mapId}/instances/${instanceId}/monsters`,
    );
  }
```

`LiveMonsterData.attributes` is typed from
`services/atlas-monsters/atlas.com/monsters/monster/rest.go` and carries
**exactly**: `worldId`, `channelId`, `mapId`, `instance`, `monsterId`,
`controlCharacterId`, `x`, `y`, `fh`, `stance`, `team`, `maxHp`, `hp`,
`maxMp`, `mp`, `damageEntries`, `experienceEntries`, `statusEffects`,
`controllerHasAggro`, and the optional `nextEligibleRepickAtMs`,
`spawnSourceType`, `spawnSourceId`. There is **no `state` and no `alive`
attribute** — do not add one.

- [ ] **Step 4: Implement `useFieldRuntime.ts` (`useLiveMonsters` only)**

```ts
export const fieldRuntimeKeys = {
  monsters: (w: number, c: number, m: number, i: string) =>
    ["fields", w, c, m, i, "monsters"] as const,
};
```

`useLiveMonsters(worldId, channelId, mapId, instanceId, enabled = true)` uses
the runtime profile (`staleTime: 5 * 1000`, `gcTime: 60 * 1000`), no
`refetchInterval`, and `enabled: enabled && !!activeTenant`. The `enabled` flag
is how the caller implements the 12-row cap without violating the rules of
hooks — every row calls the hook, rows past the cap pass `enabled: false`.

- [ ] **Step 5: Implement `LiveFieldsSection.tsx` and wire it in**

Props: `{ mapId: string }`. Calls `useFieldsForMap(mapId)`. Renders a headed
section (always, per FR-8) containing a table with columns **World, Channel,
Instance, Characters, Live Monsters**, each row linking to
`/fields/{worldId}/{channelId}/{mapId}/{instanceId}`, plus a
`View all in Fields` link to `/fields?map={mapId}`.

Extract the per-row monster count into a `LiveFieldRow` child component so each
row owns its own `useLiveMonsters` call; pass `enabled={index < 12}`.

In `MapDetailPage.tsx`, insert between lines 75 and 77:

```tsx
        <LiveFieldsSection mapId={id} />
```

- [ ] **Step 6: Run the tests to verify they pass**

`npm test -- src/components/features/maps/` → PASS.
`npx tsc --noEmit` and `npm run lint` → clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/services/api/live-monsters.service.ts services/atlas-ui/src/lib/hooks/api/useFieldRuntime.ts services/atlas-ui/src/components/features/maps/ services/atlas-ui/src/pages/MapDetailPage.tsx
git commit -m "feat(atlas-ui): Live Fields section on the map definition page"
```

---

## Task 15: `atlas-ui` — routing, sidebar entry, and breadcrumbs

### Files

- `services/atlas-ui/src/App.tsx` — add two lazy imports beside the map ones (lines 113-118) and two `<Route>` entries beside lines 370-371
- `services/atlas-ui/src/components/app-sidebar-items.ts` — 92 lines; insert `Fields` into the Operations group between `Maps` (line 46) and `Reactors` (line 47)
- `services/atlas-ui/src/lib/breadcrumbs/routes.ts` — 714 lines; add four `RouteConfig` entries and two `ROUTES` constants beside `MAP_DETAIL` (line 682)
- `services/atlas-ui/src/pages/FieldsPage.tsx` — **new file**, a minimal placeholder-free shell in this task (the real locator is Task 16)
- `services/atlas-ui/src/pages/FieldDetailPage.tsx` — **new file**, same
- `services/atlas-ui/src/lib/breadcrumbs/__tests__/fields-routes.test.ts` — **new file**

Patterns to copy:
- `services/atlas-ui/src/lib/breadcrumbs/routes.ts:378-386` — a `nonNavigable: true` grouping node (the `/templates/[id]/character` entry), including its explanatory comment.
- `services/atlas-ui/src/App.tsx:113-118` and `:370-371` — the lazy-import and `<Route>` idioms.
- `services/atlas-ui/src/components/__tests__/app-sidebar.test.tsx` — the existing sidebar sync test; check whether it needs updating for the new entry.

**D11 — `nonNavigable` intermediates.** FR-17 wants
`Fields / <World> / Channel <N> / <Map name> / Instance <id>`, but no page
exists at `/fields/:w`, `/fields/:w/:c`, or `/fields/:w/:c/:m`. Register those
three as `nonNavigable: true` grouping nodes plus the navigable leaf.

The world label needs a *name*, which route params do not carry, and
`BreadcrumbResolverContext` today exposes only `jobName`. Rather than widen it
for one label, the world intermediate resolves to `World <id>` from params;
the **page header** (FR-18) shows the resolved name from `useWorlds()`. This
keeps the route table a pure function of params.

**These two page components are NOT stubs.** `FieldsPage` in this task renders
its heading, its `SurfaceKindBadge kind="runtime"`, and an empty result area;
`FieldDetailPage` renders its header from params. Both are complete, working
components — Tasks 16 and 17 add filtering and tabs on top. Each must render
real content on first load; an empty render body, a deferral marker, or an
"unimplemented" message in place of content is not acceptable here.

- [ ] **Step 1: Write the failing test**

`services/atlas-ui/src/lib/breadcrumbs/__tests__/fields-routes.test.ts`. Read
the existing breadcrumb test harness first:
`ls services/atlas-ui/src/lib/breadcrumbs/__tests__/` and copy whichever file
there resolves a path into a breadcrumb trail.

| test name | input path | expect trail labels, in order |
|---|---|---|
| `locator` | `/fields` | `["Fields"]` |
| `full field detail trail` | `/fields/0/1/910340000/00000000-0000-0000-0000-000000000000` | `["Fields", "World 0", "Channel 1", "910340000", "Instance 00000000-0000-0000-0000-000000000000"]` |
| `intermediates are non-navigable` | same path | the entries for `World 0`, `Channel 1`, and the map segment each have `nonNavigable === true`; `Fields` and the leaf do not |
| `ROUTES constants` | — | `ROUTES.FIELDS === "/fields"` and `ROUTES.FIELD_DETAIL === "/fields/[worldId]/[channelId]/[mapId]/[instanceId]"` |

Adjust the expected map-segment label to whatever the resolver can produce from
params alone; if the harness supports a name resolver, use the map id as the
label and note it, since the definition name is not available from params.

- [ ] **Step 2: Run the test to verify it fails**

`npm test -- src/lib/breadcrumbs/__tests__/fields-routes.test.ts`
Expected: FAIL — no route config matches `/fields`.

- [ ] **Step 3: Add the route configs and constants**

In `routes.ts`, add to `ROUTES` beside line 682:

```ts
  FIELDS: "/fields",
  FIELD_DETAIL: "/fields/[worldId]/[channelId]/[mapId]/[instanceId]",
```

and four `RouteConfig` entries — `/fields` (navigable, label `Fields`),
`/fields/[worldId]` (`nonNavigable: true`, `labelResolver` → `World ${worldId}`),
`/fields/[worldId]/[channelId]` (`nonNavigable: true`, → `Channel ${channelId}`),
`/fields/[worldId]/[channelId]/[mapId]` (`nonNavigable: true`, → the map id),
and the leaf `/fields/[worldId]/[channelId]/[mapId]/[instanceId]`
(navigable, → `Instance ${instanceId}`), each with the correct `parent`.
Match the exact `RouteConfig` field names by reading `routes.ts:20-45` first.

- [ ] **Step 4: Add the sidebar entry**

In `app-sidebar-items.ts`, between line 46 (`Maps`) and line 47 (`Reactors`):

```ts
      { title: "Fields", url: "/fields" },
```

FR-10 requires it directly after `Maps`.

- [ ] **Step 5: Add the routes and the two page shells**

In `App.tsx`, beside lines 113-118:

```tsx
const FieldsPage = lazyWithReload(() =>
  import("@/pages/FieldsPage").then((m) => ({ default: m.FieldsPage })),
);
const FieldDetailPage = lazyWithReload(() =>
  import("@/pages/FieldDetailPage").then((m) => ({
    default: m.FieldDetailPage,
  })),
);
```

and beside lines 370-371:

```tsx
              <Route path="/fields" element={<FieldsPage />} />
              <Route
                path="/fields/:worldId/:channelId/:mapId/:instanceId"
                element={<FieldDetailPage />}
              />
```

`FieldsPage.tsx` exports `FieldsPage`, rendering a page heading `Fields`, a
`<SurfaceKindBadge kind="runtime" />`, and a results area driven by
`useFields({})` — a complete working list, just without the filter bar.
`FieldDetailPage.tsx` exports `FieldDetailPage`, reading its four params via
`useParams()` and rendering the map id as the title with a
`<SurfaceKindBadge kind="runtime" />` and a `View Map Definition` link to
`/maps/{mapId}`.

- [ ] **Step 6: Run the tests to verify they pass**

`npm test -- src/lib/breadcrumbs/ src/components/__tests__/app-sidebar.test.tsx`
→ PASS (update the sidebar sync test's expectations if it enumerates entries).
`npx tsc --noEmit` and `npm run lint` → clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/App.tsx services/atlas-ui/src/components/app-sidebar-items.ts services/atlas-ui/src/lib/breadcrumbs/ services/atlas-ui/src/pages/FieldsPage.tsx services/atlas-ui/src/pages/FieldDetailPage.tsx services/atlas-ui/src/components/__tests__/app-sidebar.test.tsx
git commit -m "feat(atlas-ui): /fields routes, sidebar entry, and breadcrumbs"
```

---

## Task 16: `atlas-ui` — Fields locator

### Files

- `services/atlas-ui/src/components/features/fields/FieldsFilterBar.tsx` — **new file**
- `services/atlas-ui/src/components/features/fields/FieldsResultTable.tsx` — **new file**
- `services/atlas-ui/src/pages/FieldsPage.tsx` — **new file** as of Task 15; here its body is replaced with the filter bar + result table
- `services/atlas-ui/src/pages/__tests__/FieldsPage.test.tsx` — **new file**

Patterns to copy: `services/atlas-ui/src/pages/TransportsPage.tsx` (157 lines)
and `services/atlas-ui/src/pages/__tests__/TransportsPage.test.tsx` (273 lines)
— the closest existing filtered-list page and its test.

Refresh + last-updated (FR-40) uses `useGridRefresh`
(`services/atlas-ui/src/lib/hooks/useGridRefresh.ts:16-37`), which returns
`{ isRefreshing, onRefresh, lastUpdatedAt }`. Note it is **not** currently used
anywhere in the maps feature (correction C7) — read its signature before wiring.

- [ ] **Step 1: Write the failing test**

`services/atlas-ui/src/pages/__tests__/FieldsPage.test.tsx`, mocking
`@/lib/hooks/api/useFields`, `@/lib/hooks/api/useWorlds`, and the map-name
lookup hook. Render inside a `MemoryRouter` with an `initialEntries` URL, as
`TransportsPage.test.tsx` does.

Fixture worlds: `[{ id: "0", attributes: { name: "Scania" } }, { id: "1", attributes: { name: "Bera" } }]`
(type the attributes to whatever `worlds.service.ts` actually declares).
Fixture channels for world 0: channels `1`, `2`, `3`.
Fixture fields: `(0,1,910340000)`, `(0,3,100000000)`.

| test name | setup | expect |
|---|---|---|
| `selects no field on load` | `/fields` | no field detail content; the filter bar and an (possibly empty) result area render (FR-11) |
| `defaults to the lowest-numbered world` | worlds 1 and 0, unordered | the world select shows `Scania` (world 0) (FR-12) |
| `channel defaults to Any channel` | — | the channel select shows `Any channel` (FR-12) |
| `world and channel options come from the API` | — | `Scania` and `Bera` are selectable; the channel select lists `1`, `2`, `3`; assert no hard-coded list is rendered when `useWorlds` returns `[]` |
| `map filter searches name and id` | type `9103` into the map input | only the `910340000` row remains (FR-13) |
| `map filter matches on name` | type `Henesys` where field `910340000` resolves to map name `Henesys` | that row remains |
| `map filter is a text input, not a select` | — | the map filter element is an `input`, and no option list of every map is rendered (FR-13) |
| `?map= pre-fills the filter` | `/fields?map=910340000` | the map input's value is `910340000` and only that row shows (FR-16) |
| `changing the filter writes back to the URL` | type `100000000` | the location search contains `map=100000000` (FR-16) |
| `empty state echoes the filters by name` | filters world 0 / channel 3, `useFields` returns `[]` | a message containing `Scania` and `3` renders, and it does **not** say the map is missing (FR-15) |
| `empty state offers clear filters` | same | a `Clear filters` control renders; clicking it resets world to the default, channel to `Any channel`, and clears the map input |
| `result columns` | 1 field | columns Map (name + id), Channel, Instance, Characters; the Map cell links to `/fields/0/1/910340000/<instanceId>` (FR-14) |
| `exposes refresh and last updated` | — | a Refresh control renders and a last-updated timestamp is displayed (FR-40) |
| `does not poll` | — | grep assertion is in Step 5; here assert no timer-driven refetch occurs after `vi.advanceTimersByTime(60000)` |

- [ ] **Step 2: Run the test to verify it fails**

`npm test -- src/pages/__tests__/FieldsPage.test.tsx`
Expected: FAIL — the filter bar does not exist.

- [ ] **Step 3: Implement `FieldsFilterBar.tsx`**

Props: the current `{ worldId, channelId, mapQuery }` and their setters, plus
the world and channel option lists. Renders a World select (options from
`useWorlds()`, defaulting to the lowest-numbered world), a Channel select
(options from `useChannels(worldId)` plus an `Any channel` option that is the
default), and a **debounced text input** for the map filter. Use the same
shadcn `Select` and `Input` primitives `TransportsPage.tsx` uses.

- [ ] **Step 4: Implement `FieldsResultTable.tsx` and the page**

`FieldsResultTable` renders columns **Map** (name + id, linking to the field),
**Channel**, **Instance**, **Characters**.

`FieldsPage` holds the filter state, syncs it to `?map=` via
`useSearchParams`, calls `useFields({ worldId, channelId })`, resolves each
distinct `mapId` in the result to a map name through the existing definition
map hook (find it with
`grep -n 'export function useMap' services/atlas-ui/src/lib/hooks/api/useMaps.ts`),
and applies the map text filter **client-side** against name and id — correct,
because the server filter is exact-`mapId` only and the requirement is a
*search*.

Empty state echoes the resolved filter names (`World: Scania, Channel: 3`) with
a `Clear filters` control. The copy must not say the map is missing.

Wire `useGridRefresh` for the Refresh control and the last-updated timestamp.

- [ ] **Step 5: Run the tests to verify they pass**

`npm test -- src/pages/__tests__/FieldsPage.test.tsx` → PASS.
`npx tsc --noEmit` and `npm run lint` → clean.

Run: `grep -rn 'refetchInterval' services/atlas-ui/src/pages/FieldsPage.tsx services/atlas-ui/src/components/features/fields/`
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/fields/ services/atlas-ui/src/pages/FieldsPage.tsx services/atlas-ui/src/pages/__tests__/FieldsPage.test.tsx
git commit -m "feat(atlas-ui): Fields locator with world/channel/map filters"
```

---

## Task 17: `atlas-ui` — Field detail shell, header, summary, and tabs

### Files

- `services/atlas-ui/src/components/features/fields/FieldHeader.tsx` — **new file**
- `services/atlas-ui/src/components/features/fields/FieldSummaryPanels.tsx` — **new file**
- `services/atlas-ui/src/components/features/fields/FieldTabs.tsx` — **new file**
- `services/atlas-ui/src/pages/FieldDetailPage.tsx` — **new file** as of Task 15; here its body is replaced
- `services/atlas-ui/src/pages/__tests__/FieldDetailPage.test.tsx` — **new file**

Patterns to copy:
- `services/atlas-ui/src/pages/MapDetailPage.tsx:46-88` — the page layout shell (`flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16`, header, `grid gap-4 md:grid-cols-[2fr_1fr]`).
- `services/atlas-ui/src/components/features/maps/MapDetailTabs.tsx:85-96` — the `Tabs`/`TabsList` shape; this page additionally syncs the active tab to `?tab=` (FR-21).
- `services/atlas-ui/src/pages/__tests__/TransportRouteDetailPage.test.tsx` (504 lines) — the closest detail-page test.

**FR-22, torn-down field.** There is no `GET /api/fields/{id}`. Because the
liveness rule *is* "holds at least one character", an empty characters response
is exactly the torn-down signal. Render a recoverable "this field may have been
torn down" state with a link back to `/fields` — not an error toast, not a 404
page. The honest consequence: a genuinely live field cannot be empty, so this
is unambiguous.

- [ ] **Step 1: Write the failing test**

`services/atlas-ui/src/pages/__tests__/FieldDetailPage.test.tsx`, rendering at
`/fields/0/1/910340000/00000000-0000-0000-0000-000000000000` in a
`MemoryRouter`, with the runtime hooks mocked.

| test name | setup | expect |
|---|---|---|
| `map name is the primary title` | map `910340000` resolves to `Henesys`; 2 characters present | the `h2` (or heading role) text is `Henesys` |
| `world/channel/instance outrank the map id` | same | `World 0`, `Channel 1`, and the instance id are rendered; the map id `910340000` is present but not the heading (FR-18) |
| `renders a Runtime badge` | same | `screen.getByText("Runtime")` |
| `View Map Definition link` | same | a link with `href` `/maps/910340000` (FR-18) |
| `live summary shows character count` | 3 characters | the summary shows `3` |
| `live summary groups monsters by name` | live monsters: two with `monsterId` 100100, one with 100101 | the summary shows the two groups with counts `2` and `1` (FR-20) |
| `live summary shows tracked object count` | environment query disabled/absent on `main` | the panel renders a `—` or `0` for tracked objects without erroring (the tracked source arrives in Task 22) |
| `tabs render with counts` | 3 characters, 3 monsters, 2 objects | `Characters (3)`, `Monsters (3)`, `Map Objects (2)` (FR-21) |
| `tab state syncs to the query string` | click the `Monsters` tab | the location search contains `tab=monsters` |
| `?tab= selects the tab on load` | initial entry `…?tab=objects` | the Map Objects panel is the visible one |
| `torn-down field` | characters query resolves to `[]` | a recoverable message matching `/may have been torn down/i` renders **and** a link to `/fields` is present; no error toast is dispatched (FR-22) |
| `exposes refresh and last updated` | — | a Refresh control and a last-updated timestamp render (FR-40) |

- [ ] **Step 2: Run the test to verify it fails**

`npm test -- src/pages/__tests__/FieldDetailPage.test.tsx`
Expected: FAIL — the header component does not exist.

- [ ] **Step 3: Implement `FieldHeader.tsx`**

Props `{ worldId, channelId, mapId, instanceId, mapName, worldName }`. Renders
the map name as the primary title, a `<SurfaceKindBadge kind="runtime" />`,
`World {worldName ?? worldId} / Channel {channelId} / Instance {instanceId}`
more prominently than the map id, and a `View Map Definition` link to
`/maps/{mapId}`. `worldName` comes from `useWorlds()` in the page (design D11 —
the breadcrumb cannot resolve it, the header can).

- [ ] **Step 4: Implement `FieldSummaryPanels.tsx` and `FieldTabs.tsx`**

`FieldSummaryPanels` shows the character count, live monsters grouped by
monster id/name with counts, and the count of tracked objects (FR-20). The
tracked-object source does not exist on `main`; render `—` for it and leave the
slot in place — Task 22 supplies the query. Do not render a fabricated number.

`FieldTabs` renders `Characters (N)`, `Monsters (N)`, `Map Objects (N)` and
syncs the active value to `?tab=` via `useSearchParams` (FR-21). The three
panel bodies arrive in Tasks 18, 19, and 20; in this task each panel renders
its component with the data it is given.

Reuse `MapImagePanel` for the overview (FR-19) by passing runtime entities
through its existing positioned-entity props — live monsters carry `x`/`y`.
Verify the prop names first with
`sed -n '1,40p' services/atlas-ui/src/components/features/maps/MapImagePanel.tsx`
and pass only props that component actually declares. No hit-testing, no
placement, no new prop shape.

- [ ] **Step 5: Implement the page**

`FieldDetailPage` reads its four params, resolves the map definition (name,
`mapArea`) via the existing definition hook, resolves the world name via
`useWorlds()`, and calls the runtime hooks. It composes
`FieldHeader` → overview grid (`MapImagePanel` + `FieldSummaryPanels`) →
`FieldTabs`, in the layout shape of `MapDetailPage.tsx:46-88`. Wire
`useGridRefresh` for FR-40.

- [ ] **Step 6: Run the tests to verify they pass**

`npm test -- src/pages/__tests__/FieldDetailPage.test.tsx` → PASS.
`npx tsc --noEmit` and `npm run lint` → clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/components/features/fields/ services/atlas-ui/src/pages/FieldDetailPage.tsx services/atlas-ui/src/pages/__tests__/FieldDetailPage.test.tsx
git commit -m "feat(atlas-ui): Field detail header, summary, and tabs"
```

---

## Task 18: `atlas-ui` — Field Characters tab

### Files

- `services/atlas-ui/src/lib/hooks/api/useFieldRuntime.ts` — add `useFieldCharacters`
- `services/atlas-ui/src/services/api/fields.service.ts` — add `getFieldCharacters`
- `services/atlas-ui/src/components/features/fields/FieldCharactersTab.tsx` — **new file**
- `services/atlas-ui/src/components/features/fields/__tests__/FieldCharactersTab.test.tsx` — **new file**

Backend facts already established — do **not** re-derive them, and do not edit
either Go file in this task:

- The atlas-maps characters endpoint returns a JSON:API collection of type
  `characters` whose items carry **only an id**, no attributes (verified in
  `atlas-maps`' `map` package `rest.go`, lines 5-15). Per-row enrichment is
  therefore mandatory.
- The atlas-character enrichment payload (verified in `atlas-character`'s
  `character` package `rest.go`, lines 16-53) **has** `name`, `level` (byte),
  `jobId`, `x`, `y`, `fh`, `stance`, and much else. It **does not** return
  `mapId` — the comment at lines 42-44 of that file states atlas-maps owns
  character location.

**C6 — the column list is settled.** Columns are **Name, Character ID, Level,
Job, Position (`x`, `y`)**. There is **no State column**: the payload has no
state field, and `stance` is an animation frame, not a state. FR-25 permits
exactly this ("included only if the payload carries them").

Find the existing per-character hook to reuse for enrichment:
`grep -n 'export function useCharacter' services/atlas-ui/src/lib/hooks/api/useCharacters.ts`

- [ ] **Step 1: Write the failing test**

`services/atlas-ui/src/components/features/fields/__tests__/FieldCharactersTab.test.tsx`,
mocking the field-characters hook and the per-character enrichment hook.

| test name | setup | expect |
|---|---|---|
| `renders one row per character id` | ids `["100", "200"]`, both enriched | 2 rows |
| `renders the settled columns` | id `100` → `{ name: "Bob", level: 42, jobId: 110, x: 640, y: 120 }` | the row shows `Bob`, `100`, `42`, the job label for `110`, and a position rendering `640` and `120` |
| `has no State column` | same | `screen.queryByText("State")` is `null` |
| `pending enrichment shows the raw id` | id `100` enrichment pending | the row renders and shows `100`; it does not unmount (FR-24) |
| `failed enrichment shows the raw id` | id `100` enrichment errors | the row still renders and shows `100`; no error boundary trips (FR-24) |
| `one failure does not block others` | id `100` errors, id `200` succeeds as `Alice` | both rows render; `Alice` is visible |
| `name links to Character Detail` | id `100` → `Bob` | a link whose `href` is `/characters/100` (FR-26) — confirm the real route first with `grep -n 'CHARACTER_DETAIL\|"/characters/' services/atlas-ui/src/lib/breadcrumbs/routes.ts` |
| `empty tab copy` | ids `[]` | text exactly `No characters are currently in this field.`, rendered as a normal empty state, not an error (FR-27) |

- [ ] **Step 2: Run the test to verify it fails**

`npm test -- src/components/features/fields/__tests__/FieldCharactersTab.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the service method and hook**

In `fields.service.ts`:

```ts
  async getFieldCharacters(worldId: number, channelId: number, mapId: number, instanceId: string): Promise<{ id: string; type: string }[]> {
    return api.getList<{ id: string; type: string }>(
      `/api/worlds/${worldId}/channels/${channelId}/maps/${mapId}/instances/${instanceId}/characters`,
    );
  }
```

In `useFieldRuntime.ts`, add
`characters: (w, c, m, i) => ["fields", w, c, m, i, "characters"] as const` to
`fieldRuntimeKeys` and a `useFieldCharacters(...)` hook on the runtime profile
(`staleTime: 5 * 1000`, `gcTime: 60 * 1000`, no `refetchInterval`).

- [ ] **Step 4: Implement `FieldCharactersTab.tsx`**

The tab maps ids to a `FieldCharacterRow` child component; each row calls the
per-character enrichment hook itself, so React Query caches and deduplicates
across rows (FR-24) and each row renders from its own query state: pending →
the id, error → the id, success → the full row. **No `Promise.all`, no
blocking.**

- [ ] **Step 5: Run the tests to verify they pass**

`npm test -- src/components/features/fields/` → PASS.
`npx tsc --noEmit` and `npm run lint` → clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/services/api/fields.service.ts services/atlas-ui/src/lib/hooks/api/useFieldRuntime.ts services/atlas-ui/src/components/features/fields/
git commit -m "feat(atlas-ui): Field Characters tab with per-row enrichment"
```

---

## Task 19: `atlas-ui` — Field Monsters tab

### Files

- `services/atlas-ui/src/components/features/fields/FieldMonstersTab.tsx` — **new file**
- `services/atlas-ui/src/components/features/fields/__tests__/FieldMonstersTab.test.tsx` — **new file**

The service and hook already exist from Task 14
(`live-monsters.service.ts`, `useLiveMonsters`).

**D13 — the column list is settled.** Columns: **Object ID** (`id`), **Monster**
(`monsterId`, linking to the Monster Definition page), **HP** (`hp`/`maxHp`),
**Position** (`x`, `y`), **Spawn** (`spawnSourceType` + `spawnSourceId`, both
`omitempty` so blank-tolerant). **There is no State column** — verified against
`services/atlas-monsters/atlas.com/monsters/monster/rest.go:14-38`, which has no
`state` and no `alive` attribute. FR-31 ("dead and alive visually
distinguished") is satisfied by `hp === 0`: a killed monster exists at zero HP
until destroyed. Style the row; do not invent a status string.

FR-30's spawn→definition link is best-effort: `spawnSourceId` is an opaque
string that this design cannot correlate to a specific definition spawn row.
Render it as text and link the **tab**, `/maps/{mapId}?tab=monster-spawns`, not
a specific row.

- [ ] **Step 1: Write the failing test**

`services/atlas-ui/src/components/features/fields/__tests__/FieldMonstersTab.test.tsx`.

Fixture monsters:

| id | monsterId | hp | maxHp | x | y | spawnSourceType | spawnSourceId |
|---|---|---|---|---|---|---|---|
| `9001` | 100100 | 250 | 250 | 640 | 120 | `MAP` | `spawn-a` |
| `9002` | 100100 | 0 | 250 | 700 | 120 | — (absent) | — (absent) |
| `9003` | 100101 | 40 | 500 | -30 | 45 | `EVENT` | `boss-1` |

| test name | expect |
|---|---|
| `renders one row per monster` | 3 rows, showing object ids `9001`, `9002`, `9003` |
| `shows current and max HP` | the `9003` row renders both `40` and `500` |
| `shows position` | the `9003` row renders `-30` and `45` |
| `dead monsters are visually distinguished` | the `9002` row (hp 0) carries a distinguishing class or `data-` attribute that the `9001` row does not; assert on that attribute, not on colour (FR-31, D13) |
| `has no State column` | `screen.queryByText("State")` is `null` |
| `monster links to the definition` | the `9001` row has a link whose `href` targets the monster definition for `100100`; confirm the real path first with `grep -n 'MONSTER_DETAIL\|"/monsters/' services/atlas-ui/src/lib/breadcrumbs/routes.ts` |
| `spawn is blank-tolerant` | the `9002` row renders without error and shows an empty/`—` spawn cell |
| `spawn links to the definition tab` | a link whose `href` is `/maps/910340000?tab=monster-spawns` (FR-30) |
| `empty tab` | with `[]`, a normal empty state renders, not an error |

- [ ] **Step 2: Run the test to verify it fails**

`npm test -- src/components/features/fields/__tests__/FieldMonstersTab.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `FieldMonstersTab.tsx`**

Props `{ monsters?: LiveMonsterData[]; error?: Error; mapId: number }`.
Render the five settled columns. Do **not** render
`controlCharacterId`, `damageEntries`, `experienceEntries`, `statusEffects`,
`team`, or `mp`/`maxMp` — out of scope for v1, noted here so a later task knows
they are already available on the payload.

- [ ] **Step 4: Run the tests to verify they pass**

`npm test -- src/components/features/fields/` → PASS.
`npx tsc --noEmit` and `npm run lint` → clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/fields/
git commit -m "feat(atlas-ui): Field Monsters tab with hp-based liveness"
```

---

## Task 20: `atlas-ui` — Field Map Objects tab, definition half

### Files

- `services/atlas-ui/src/components/features/fields/FieldObjectsTab.tsx` — **new file**
- `services/atlas-ui/src/components/features/fields/__tests__/FieldObjectsTab.test.tsx` — **new file**

This task builds the tab's **whole layout** and its **declared-but-untracked**
half, which depends only on `atlas-data`'s `GET /api/data/maps/{id}/objects`
(Task 8) and is fully buildable on `main`. The **tracked** half and the Set /
Reset writes depend on `[278]` and are Task 22.

The tab is written so Task 22 supplies a `tracked` prop and nothing else moves:
the component takes `tracked?: TrackedObject[]` and, in this task, its only
caller passes `undefined`. That is a real, working component — a field with no
tracked state renders exactly this — not a stub.

Merge rule (FR-33): the two sources join on `{kind}:{name}`, which is
deliberately the id of *both* resources, so it is an id join, not a heuristic.
Tracked objects render first with their current `state`; declared-but-untracked
objects (definition ids minus tracked ids) render under the divider
**"Defined on the map, no state tracked in this field"** at default state.
Both sources empty is a normal empty state, not a 404 (FR-32).

- [ ] **Step 1: Write the failing test**

`services/atlas-ui/src/components/features/fields/__tests__/FieldObjectsTab.test.tsx`.

Definition fixture (`MapObjectData[]`, as in Task 12):
`ENVIRONMENT:gate` (name `gate`) and `OBSTACLE:menhir0` (name `menhir0`).

Tracked fixture (`TrackedObject[]`, shape
`{ id: string; kind: string; name: string; state: number }`):
`{ id: "OBSTACLE:menhir0", kind: "OBSTACLE", name: "menhir0", state: 3 }`.

| test name | `defined` | `tracked` | expect |
|---|---|---|---|
| `untracked only` | both | `undefined` | 2 rows, all under the "no state tracked" divider; the divider text renders |
| `untracked only, empty tracked array` | both | `[]` | same as above |
| `tracked only` | `[]` | the tracked row | 1 row in the tracked group, showing state `3`; no divider group renders |
| `both` | both | the tracked row | `menhir0` appears **exactly once**, in the tracked group with state `3`; `gate` appears once, under the divider |
| `an object in both appears once` | both | the tracked row | `screen.getAllByText("menhir0")` has length 1 (FR-33) |
| `both empty is a normal empty state` | `[]` | `[]` | an empty-state message renders; no error, no 404 copy (FR-32) |
| `rows carry their kind` | both | `undefined` | `ENVIRONMENT` and `OBSTACLE` both rendered |
| `names note pass-through` | both | `undefined` | a note renders explaining that object names are pass-through — the server does not validate a name against the map (FR-38) |

- [ ] **Step 2: Run the test to verify it fails**

`npm test -- src/components/features/fields/__tests__/FieldObjectsTab.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `FieldObjectsTab.tsx`**

```ts
export interface TrackedObject {
  id: string; // "{KIND}:{name}"
  kind: string;
  name: string;
  state: number;
}

interface FieldObjectsTabProps {
  defined?: MapObjectData[];
  definedError?: Error;
  tracked?: TrackedObject[];
  trackedError?: Error;
}
```

Build a `Set` of tracked ids, then partition `defined` into the untracked
remainder. Render the tracked group first (columns Kind, Name, State), then the
divider and the untracked group (columns Kind, Name, at default state). Include
the FR-38 pass-through note.

Wire it into `FieldTabs.tsx` / `FieldDetailPage.tsx` so the `Map Objects` panel
renders `<FieldObjectsTab defined={objects} definedError={objectsError} />`,
with `objects` coming from `useMapObjects(String(mapId))` (Task 12's hook — the
definition cache, correctly reused here).

- [ ] **Step 4: Run the tests to verify they pass**

`npm test -- src/components/features/fields/ src/pages/__tests__/FieldDetailPage.test.tsx` → PASS.
`npx tsc --noEmit` and `npm run lint` → clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/fields/ services/atlas-ui/src/pages/FieldDetailPage.tsx
git commit -m "feat(atlas-ui): Field Map Objects tab, declared-object half"
```

---

## Task 21: `atlas-ui` docs and full verification gate

### Files

- `services/atlas-ui/docs/service-layer.md` — 140 lines; document the new service modules and the two cache profiles
- `tools/verify.sh` — read-only; run it

- [ ] **Step 1: Document the new service layer**

Add to `services/atlas-ui/docs/service-layer.md`:

- The new modules: `fields.service.ts`, `worlds.service.ts`,
  `live-monsters.service.ts`, and `getObjects` on `map-entities.service.ts`.
- The two cache profiles (design D10), as a table:

  | | `staleTime` | `gcTime` |
  |---|---|---|
  | Definition (map, portals, monsters, reactors, objects, worlds, channels) | `10 * 60 * 1000` | `10 * 60 * 1000` |
  | Runtime (fields, field characters, live monsters, field environment) | `5 * 1000` | `60 * 1000` |

- The disjoint key namespaces — `["maps", mapId, …]` vs `["fields", …]` — and
  why: a runtime refetch must not invalidate definition data (FR-41).
- **No `refetchInterval` anywhere** (FR-39); a reviewer verifies this by
  grepping the new hooks.
- The tenancy convention (design D9): the tenant is deliberately **not** in
  query keys, because `src/context/tenant-context.tsx:68` calls
  `queryClient.clear()` on every tenant change. New hooks keep the existing
  `enabled: !!activeTenant` guard.

- [ ] **Step 2: Verify no absolute paths and no stray polling**

Run: `grep -n '/home/' services/atlas-ui/docs/service-layer.md`
Expected: no output.

Run: `grep -rn 'refetchInterval' services/atlas-ui/src/lib/hooks/api/useFields.ts services/atlas-ui/src/lib/hooks/api/useWorlds.ts services/atlas-ui/src/lib/hooks/api/useFieldRuntime.ts services/atlas-ui/src/lib/hooks/api/useMapEntities.ts`
Expected: no output.

- [ ] **Step 3: Run the full gate**

Run from the worktree root: `tools/verify.sh`

**Flagless.** `--quick` and `--no-docker` also exit 0 but skip the bake and
`-race`; they do not satisfy the gate. Expected: exit 0.

If it fails, fix the failure and re-run. Do not proceed to a PR on a red gate.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-ui/docs/service-layer.md
git commit -m "docs(atlas-ui): document field service modules and cache profiles"
```

---

## Task 22 — BLOCKED ON `[278]`: Map Objects runtime half

> **DO NOT START THIS TASK** until `task-278-map-environment-object-state` has
> merged to `main` and this branch has rebased onto it. Verify with:
>
> ```
> git fetch origin && git ls-tree -r --name-only origin/main -- services/atlas-maps/atlas.com/maps/map/environment
> ```
>
> Expected before starting: at least `services/atlas-maps/atlas.com/maps/map/environment/rest.go`.
> **If that command prints nothing, stop and report BLOCKED.** Landing a Set
> button that POSTs to a route no service answers is exactly the stubbed
> handler CLAUDE.md forbids.
>
> **Rebase note:** `[278]` also modifies
> `services/atlas-maps/atlas.com/maps/map/character/registry.go` (it changes
> `RemoveCharacterFromAllMaps` to return `[]MapKey`). Task 1 added
> `GetFieldsWithCharacters` to that same file. The two edits do not overlap
> textually, but the rebase will touch this file — resolve by keeping both.

### Files

- `services/atlas-ui/src/services/api/field-environment.service.ts` — **new file**
- `services/atlas-ui/src/lib/hooks/api/useFieldRuntime.ts` — add `useFieldEnvironment`, `useSetEnvironmentObject`, `useResetFieldEnvironment`
- `services/atlas-ui/src/components/features/fields/SetObjectStateDialog.tsx` — **new file**
- `services/atlas-ui/src/components/features/fields/ResetFieldObjectsDialog.tsx` — **new file**
- `services/atlas-ui/src/components/features/fields/FieldObjectsTab.tsx` — pass real `tracked` data and wire the two actions
- `services/atlas-ui/src/components/features/fields/FieldSummaryPanels.tsx` — replace the `—` tracked-object count from Task 17 with the real one
- `services/atlas-ui/src/pages/FieldDetailPage.tsx` — call `useFieldEnvironment`
- `services/atlas-ui/src/components/features/fields/__tests__/SetObjectStateDialog.test.tsx` — **new file**
- `services/atlas-ui/src/components/features/fields/__tests__/ResetFieldObjectsDialog.test.tsx` — **new file**

**C8 — the JSON:API type is `environment-objects`, not `environment`.**
`[278]`'s `RestModel.GetName()` returns `"environment-objects"`. The PRD prose
names the attributes but not the type.

**Writes.** `[278]` registers the POST via
`rest.RegisterInputHandler[RestModel]`, which expects the **JSON:API
envelope**, not a bare object:

```json
{"data":{"type":"environment-objects","id":"{KIND}:{name}","attributes":{"kind":"...","name":"...","state":0}}}
```

`DELETE .../environment` performs the reset.

**D14 — no free-text object entry in v1.** `kind` on a Set is always taken from
the row, never from user input and never guessed. The gap closes itself: any
object with tracked state appears in the tracked source *carrying its own kind*,
so once written it is editable forever after. The only unreachable case is the
first write to an undeclared name, which is a script author's concern; `Reset
all to default` clears tracked state regardless of provenance, so nothing
becomes unrecoverable.

- [ ] **Step 1: Confirm `[278]` is on `main`**

Run: `git ls-tree -r --name-only origin/main -- services/atlas-maps/atlas.com/maps/map/environment`
Expected: non-empty. If empty, STOP and report BLOCKED.

Then read the real contract rather than trusting this plan's summary:
`cat services/atlas-maps/atlas.com/maps/map/environment/rest.go` and
`grep -n 'HandleFunc\|RegisterInputHandler' services/atlas-maps/atlas.com/maps/map/environment/resource.go`

- [ ] **Step 2: Write the failing tests**

`SetObjectStateDialog.test.tsx`:

| test name | setup | expect |
|---|---|---|
| `does not dispatch before confirmation` | render with row `{kind: "OBSTACLE", name: "menhir0"}`, target state `3` | the mutation mock has not been called |
| `names the field in the confirmation` | field `(0, 1, 910340000, uuid A)` | the dialog text contains `0`, `1`, `910340000`, and the instance id (FR-36) |
| `names the object and target state` | same | the dialog text contains `menhir0` and `3` (FR-36) |
| `states the blast radius` | same | copy matching `/every character in (the |this )field/i` renders (FR-36) |
| `payload carries the row's kind` | confirm | the mutation is called with `kind === "OBSTACLE"` — taken from the row, never from input (FR-34) |
| `payload is a JSON:API envelope` | confirm | the service is called with `{ data: { type: "environment-objects", id: "OBSTACLE:menhir0", attributes: { kind: "OBSTACLE", name: "menhir0", state: 3 } } }` |
| `success invalidates only the environment key` | confirm, mutation resolves | `queryClient.invalidateQueries` called with a key whose tail is `"environment"` and whose head is `"fields"`; **not** called with any `["maps", …]` key (FR-41) |
| `success toasts` | confirm, resolves | a toast is dispatched (FR-37) |
| `400 renders inline on the row, not as a toast` | mutation rejects with a 400 | an inline error renders within the row so the failing row is identifiable; **no** toast is dispatched (FR-37) |

`ResetFieldObjectsDialog.test.tsx`:

| test name | setup | expect |
|---|---|---|
| `does not dispatch before confirmation` | 3 tracked objects | the mutation mock has not been called |
| `names the field` | field `(0, 1, 910340000, uuid A)` | all four values appear in the dialog copy |
| `names the count of tracked objects` | 3 tracked | the dialog copy contains `3` (FR-36) |
| `states the blast radius` | — | copy matching `/every character in (the \|this )field/i` (FR-36) |
| `confirm issues DELETE` | confirm | the reset mutation is called once with the four field coordinates |
| `success toasts and refetches` | resolves | a toast is dispatched and the environment key is invalidated (FR-37) |

Extend `FieldObjectsTab.test.tsx` with a case asserting that a tracked object
whose `kind` differs from any definition row still renders in the tracked group
with its own kind (D14's "carries its own kind" property).

- [ ] **Step 3: Run the tests to verify they fail**

`npm test -- src/components/features/fields/__tests__/SetObjectStateDialog.test.tsx src/components/features/fields/__tests__/ResetFieldObjectsDialog.test.tsx`
Expected: FAIL — modules not found.

- [ ] **Step 4: Implement the service**

`field-environment.service.ts` with `getEnvironment`, `setObjectState`, and
`resetEnvironment`, all against
`/api/worlds/${w}/channels/${c}/maps/${m}/instances/${i}/environment`.
`setObjectState` posts the JSON:API envelope shown above (use `api.post`);
`resetEnvironment` uses `api.delete`. Type the response from
`[278]`'s `RestModel` — `{ kind: string; name: string; state: number }` with
id `{KIND}:{name}` and type `environment-objects`.

- [ ] **Step 5: Implement the hooks**

In `useFieldRuntime.ts`, add
`environment: (w, c, m, i) => ["fields", w, c, m, i, "environment"] as const`
to `fieldRuntimeKeys`; `useFieldEnvironment(...)` on the runtime profile; and
`useSetEnvironmentObject(...)` / `useResetFieldEnvironment(...)` as
`useMutation`s whose `onSuccess` invalidates **only** the environment key.

- [ ] **Step 6: Implement the dialogs and wire the tab**

Both dialogs use the existing shadcn `AlertDialog` primitive (find its usage
with `grep -rln 'AlertDialog' services/atlas-ui/src/components/features | head`).
Each names world/channel/map/instance; the Set dialog additionally names the
object and target state, the Reset dialog the count of tracked objects; both
state that the change broadcasts to every character in the field.

`FieldObjectsTab` now receives `tracked` from `useFieldEnvironment`, renders a
per-row state input and Set action, and a field-level `Reset all to default`.
`kind` on a Set comes from the row (FR-34). A `400` renders inline on the row
using the existing `ErrorDisplay`/field-error pattern, not a toast (FR-37).

`FieldSummaryPanels` replaces its Task-17 `—` with the real tracked count.

- [ ] **Step 7: Run the tests to verify they pass**

`npm test -- src/components/features/fields/ src/pages/__tests__/FieldDetailPage.test.tsx` → PASS.
`npx tsc --noEmit` and `npm run lint` → clean.

- [ ] **Step 8: Re-run the full gate**

Run from the worktree root: `tools/verify.sh` (flagless). Expected: exit 0.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-ui/src/services/api/field-environment.service.ts services/atlas-ui/src/lib/hooks/api/useFieldRuntime.ts services/atlas-ui/src/components/features/fields/ services/atlas-ui/src/pages/FieldDetailPage.tsx
git commit -m "feat(atlas-ui): tracked field object state with confirmed set and reset"
```

---

## Requirement coverage

| PRD | Task |
|---|---|
| FR-1, FR-2, FR-3, FR-5 | 11 |
| FR-4 | 11, 12 (existing map page wiring untouched) |
| FR-6 | 12 |
| FR-7, FR-8, FR-9 | 14 |
| FR-10 | 15 |
| FR-11 – FR-15 | 16 |
| FR-16 | 14 (link), 16 (`?map=` round-trip) |
| FR-17 | 15 |
| FR-18 – FR-22 | 17 |
| FR-23 – FR-27 | 18 |
| FR-28 – FR-31 | 19 |
| FR-32, FR-33 | 20 (declared half), 22 (tracked half) |
| FR-34 – FR-38 | 22 |
| FR-39, FR-40, FR-41 | 13, 16, 17, 21 |
| §5.1 `GET /fields` | 1, 2, 3 |
| §5.2 `GET /api/data/maps/{id}/objects` | 4, 5, 6, 7, 8, 9 |
| §5.4 gateway | 10 |
| §8 NFR multi-tenancy | 1, 2 (tenant-isolation tests are required, not optional) |
| §8 NFR request volume | 14 (D12 cap), 18 (React Query dedup) |
| §8 NFR no polling | 13, 16, 21 (grep assertions) |
| §8 NFR write safety | 22 |
| §8 NFR observability | 2 (enumeration failure logging) |
| §8 NFR testing | every task |
| §10 acceptance: flagless `verify.sh` exits 0 | 21, 22 |
