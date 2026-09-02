# Map Definition / Field Separation — Design

Version: v1
Status: Draft
Created: 2026-09-02
Consumes: `prd.md` (v1, approved), `ux-flow.md`, `ux-prototype.html`

---

## 0. Scope of this document

This is the architecture and tradeoff record for task-292. It decides the
things `prd.md` left to design (its §9 Open Questions), pins the exact
extension points in existing code, and records what was rejected and why.

Everything asserted about existing code below was read in this worktree at
`main`, or on `origin/task-278-map-environment-object-state` where explicitly
marked **[278]**. Everything asserted about WZ data was measured against the
serialized GMS 83.1 dataset at
`tmp/083839c6-c47c-42a6-9585-76492795d123/GMS/83.1/Map.wz` (5261 map `.img.xml`
files, 5261 of the PRD's 5262 — the PRD's count is from a different snapshot).

---

## 1. Architecture at a glance

Three independent axes of work, deliberately kept separable:

```
  atlas-data                atlas-maps                  atlas-ui
  ──────────                ──────────                  ────────
  map objects (WZ)          GET /api/fields             /maps/:id      (Definition)
  GET /api/data/maps/       (character-registry         /fields        (Locator)
      {id}/objects           read model)                /fields/w/c/m/i (Field)
        │                          │                          ▲
        └──────── definition ──────┴───── runtime ────────────┘
                                   │
                  [278] GET|POST|DELETE .../environment
```

The **only** hard cross-service coupling introduced is a vocabulary agreement:
`atlas-data`'s map-object `kind` and `[278]`'s environment-object `kind` must
be the same two strings, `ENVIRONMENT` and `OBSTACLE`, because the Map Objects
tab merges the two collections on the composite key `kind:name`. Nothing else
is shared; nothing is imported across a service boundary.

### 1.1 Dependency graph and the one real blocker

| Work | Depends on |
|---|---|
| `atlas-data` map objects | nothing — buildable on `main` |
| `atlas-maps` `GET /api/fields` | nothing — buildable on `main` |
| Definition-page terminology, Map Objects tab, Live Fields | `atlas-data` map objects |
| Fields locator, Field detail, Characters tab, Monsters tab | `atlas-maps` `GET /api/fields` |
| Gateway `/api/fields` block | `atlas-maps` `GET /api/fields` |
| Gateway `/environment` block | nothing (route may be added ahead of the handler) |
| **Field detail → Map Objects tab (FR-32, FR-34–FR-38)** | **`[278]` merged to `main`** |

`[278]` (branch `task-278-map-environment-object-state`, 29 commits ahead of
`origin/main`, no PR) is a genuine external blocker for exactly one panel. Its
REST contract is stable and was read directly:

```go
// [278] services/atlas-maps/atlas.com/maps/map/environment/rest.go
type RestModel struct {
    Id    string `json:"-"` // "{KIND}:{name}"
    Kind  string `json:"kind"`
    Name  string `json:"name"`
    State uint32 `json:"state"`
}
func (m RestModel) GetName() string { return "environment-objects" }
```

Note the JSON:API **type is `environment-objects`, not `environment`** — the
PRD §4.7 prose says "the response is a collection of `{kind, name, state}`",
which is right about attributes but does not name the type. The UI service
module must expect `environment-objects`.

**D0 — sequencing.** Build task-292 on `main` and implement the Map Objects
tab's *definition* half (FR-33's untracked side) plus its whole layout; land the
*runtime* half only once `[278]` is on `main` and this branch has rebased onto
it. Rationale: CLAUDE.md forbids landing a stubbed handler or an unimplemented
status response, and a Set button that POSTs to a route no service answers is
exactly that. The gateway `/environment` location block is the one exception —
it is inert and harmless before `[278]` lands (nginx proxies to `atlas-maps`,
which returns its own 404), and the PRD requires it regardless.

**This is the one decision that needs the user, not design.** Either `[278]`
merges first (preferred — it is complete, tested, and only lacks a PR), or
task-292's plan must schedule the Map Objects runtime half as a trailing unit
gated on that merge. See §10.

Note also that `[278]` touches
`services/atlas-maps/atlas.com/maps/map/character/registry.go` (it changes
`RemoveCharacterFromAllMaps` to return `[]MapKey`). This design adds a *new*
method to the same file (§3.1). The two edits do not overlap textually, but the
rebase will touch this file.

---

## 2. Terminology and the kind indicator (FR-1 – FR-3)

No abstraction is warranted here; this is a rename plus one shared badge.

- `MapDetailTabs.tsx`: monster tab trigger label `Monsters (N)` → `Monster Spawns (N)`.
- `MapEntitySummary.tsx`: monster heading → `Configured Monster Spawns`.
- New `src/components/features/maps/SurfaceKindBadge.tsx` exporting a single
  component with a `kind: "definition" | "runtime"` prop, rendering the existing
  `Badge` primitive with the literal text `Definition` / `Runtime`. FR-3 says
  the distinction must not be carried by colour alone; the badge always renders
  its word, and colour is a secondary cue.
- Used in three places: `MapHeader` (definition), the Field detail header
  (runtime), and the Fields sidebar/locator heading (runtime).

**Rejected:** a `<KindContext>` provider that tints descendant components.
Over-engineered for two call sites and it makes the distinction implicit —
the opposite of FR-2's intent.

---

## 3. `atlas-maps` — field enumeration

### 3.1 The read model lives in the registry, not the caller

`character.Registry` today (`map/character/registry.go`) is
`map[MapKey][]uint32` with `MapKey{Tenant tenant.Model; Field field.Model}`.
It already has `GetMapsWithCharacters() []MapKey` (line 89) which filters
`len(mc) > 0` — the liveness rule the PRD wants. But it is **not usable as-is**:

1. It does not filter by tenant. Using it and filtering in the caller still
   walks other tenants' keys, and any future change to it silently widens the
   blast radius of a tenancy bug (NFR: "must never leak a field belonging to
   another tenant").
2. It returns keys only. Getting counts means a second pass of `GetInMap` per
   key, **outside** the lock — so a field can be listed as live and then
   reported with `characterCount: 0`, contradicting the liveness rule inside a
   single response.
3. Its only production caller is `tasks/respawn.go:39`, which wants
   cross-tenant behaviour. Changing its signature would drag the respawn task
   into this task's blast radius for no benefit.

**D1.** Add a new, additive registry method rather than changing the existing
one:

```go
// map/character/registry.go
type FieldOccupancy struct {
    Field          field.Model
    CharacterCount uint32
}

// GetFieldsWithCharacters returns every field belonging to t that currently
// holds at least one character, with its occupancy sampled under the same
// read lock. Key existence is not liveness: RemoveCharacter empties a key's
// slice without deleting the key, so a drained field lingers as a zero-length
// entry and must be excluded here.
func (r *Registry) GetFieldsWithCharacters(t tenant.Model) []FieldOccupancy
```

and the matching `Processor` interface method + mock. `GetMapsWithCharacters`
is left untouched.

### 3.2 Package placement

New package at `services/atlas-maps/atlas.com/maps/field/` declaring
`package field`, containing `resource.go`, `rest.go`, and their tests.

It cannot go in the existing `map` package: that package's `RestModel` is
already the `characters` type (`map/rest.go`), and a second `RestModel` with
`GetName() == "fields"` cannot coexist in one package.

The name `field` collides with the imported `libs/atlas-constants/field`, which
is fine and has direct precedent: `maps/map/resource.go` declares `package _map`
while importing `_map "…/atlas-constants/map"`. A package's own name is not in
its files' scope. Consumers alias the local package (`mapField "atlas-maps/field"`).

### 3.3 Resource contract

Registered in `main.go` alongside the other initializers. The service base path
is `/api/` (`GetServer().GetPrefix()`), so `router.PathPrefix("/fields")`
produces `/api/fields` end to end — the PRD's §5.1 `GET /fields` and §5.4
`/api/fields` are the same route seen from inside and outside the gateway.

```
GET /api/fields
GET /api/fields?filter[worldId]=0&filter[channelId]=1&filter[mapId]=910340000
                &page[number]=1&page[size]=50
```

```go
// field/rest.go
type RestModel struct {
    Id             string     `json:"-"` // "{worldId}:{channelId}:{mapId}:{instanceId}"
    WorldId        world.Id   `json:"worldId"`
    ChannelId      channel.Id `json:"channelId"`
    MapId          _map.Id    `json:"mapId"`
    InstanceId     uuid.UUID  `json:"instanceId"`
    CharacterCount uint32     `json:"characterCount"`
}
func (m RestModel) GetName() string { return "fields" }
```

**D2 — deterministic ordering before pagination.** Go map iteration order is
randomised. `paginate.Slice` over an unsorted slice would let the same field
appear on two pages and another appear on none. Sort by
`(worldId, channelId, mapId, instanceId-string)` before slicing — the same
sort-then-slice discipline `map/resource.go:57-60` already applies to character
ids.

**D3 — filter parsing.** Parse `filter[worldId]`, `filter[channelId]`,
`filter[mapId]` from `r.URL.Query()` into `*world.Id` / `*channel.Id` /
`*_map.Id`. Absent → nil → no constraint. Present but unparseable → `400` via
`server.WriteBadRequest`, matching the existing `page[...]` failure path.
Unrecognised `filter[...]` keys are ignored (PRD §5.1). Filtering happens in
the resource over the returned slice, not in the registry: the registry method
stays a simple tenant-scoped snapshot, and the filter combination logic is then
plainly unit-testable without touching a global singleton.

**Rejected:** pushing filters into `GetFieldsWithCharacters`. It would grow
three optional parameters onto a lock-holding method to save a linear scan over
a slice whose length is bounded by live fields per tenant.

### 3.4 Tests

Resource-level, following `map/resource_paginate_test.go`:

- drained-field exclusion — `AddCharacter` then `RemoveCharacter` on a key, then
  assert that key is absent from the response (this is the PRD's named
  acceptance test);
- tenant isolation — two tenants' fields registered, each request sees only its
  own;
- each filter alone and all three combined;
- empty result is `200` with `data: []`, never `404`;
- malformed `filter[worldId]=abc` is `400`;
- pagination determinism — two pages of a multi-field set partition the whole
  set with no overlap.

Registry-level: `GetFieldsWithCharacters` returns occupancy consistent with the
key set it returns.

---

## 4. `atlas-data` — map objects

### 4.1 What the WZ data actually says

Measured on the GMS 83.1 dataset (not recalled):

- Named `obj` entries appear as `<string name="name" value="gate"/>` inside a
  per-layer `obj/<n>` imgdir alongside `oS`, `l0`, `l1`, `l2`, `x`, `y`, `z`,
  `f`, `zM`.
- **34** maps declare at least one named object; **595** distinct names —
  matching the PRD exactly.
- 2004 of ~2100 occurrences are the four beach maps `109080000`–`109080003`
  (501 named objects each).
- `103000800`'s `gate` resolves to `oS=effect, l0=quest, l1=gate, l2=1`
  (the PRD's example JSON shows `l2: "0"`; the real value is `1`).
- `obstacle` is a direct `<int name="obstacle" value="1"/>` child of the
  `{l0}/{l1}/{l2}` imgdir in `Map.wz/Obj/{oS}.img.xml`. There are **225** such
  entries in this dataset (the PRD says 214, from a different snapshot).
- `610030300` yields `menhir0`–`menhir5` at `trapGL/ckPQ/menhir/*`, all
  `obstacle=1` → `OBSTACLE`, **plus** `3pt` at `masteriaGL3/ckPQ/portal/0`,
  which has no `obstacle` flag → `ENVIRONMENT`. The PRD acceptance criterion
  "`610030300` shows six `OBSTACLE` objects" is satisfied, but the tab shows
  **seven rows**, not six. Do not write a test that asserts a row count of six.

### 4.2 The obstacle index is a tenant registry, not a per-map file read

Resolving `kind` needs a *different* file from the map being parsed
(`Map.wz/Obj/{oS}.img.xml`). Opening and parsing that file inside
`getObjects()` would re-parse the same Obj images thousands of times across a
5261-map ingest.

**D4.** Mirror the existing string-registry pattern exactly. Add
`services/atlas-data/atlas.com/data/map/object_registry.go`:

```go
func GetMapObjectRegistry() *document.Registry[string, ObjKind]
func InitObj(t tenant.Model, dir string) error   // dir = {root}/Map.wz/Obj
```

`InitObj` walks `dir/*.img.xml` once, and for every `{l0}/{l1}/{l2}` node
carrying `obstacle=1`, records the key `"{oS}/{l0}/{l1}/{l2}"`. Lookup returns
`OBSTACLE` for a hit and `ENVIRONMENT` otherwise. This is structurally identical
to `map/string_registry.go`'s `InitString` / `GetMapStringRegistry`, including
tenant scoping and the `Clear(t)` teardown.

**Both** ingest entry points must call it, because both already call
`InitString` and only one is the worker:

- `data/workers/mapw.go` — after `serializeArchive`, before
  `registerAllInDirectory`, with `defer GetMapObjectRegistry().Clear(t)`;
- `data/processor.go` `StartWorker` (the `WorkerMap` branch, ~line 116) —
  same placement, same clear.

**Failure policy.** `InitString` failure is a warning in the worker and a hard
error in `StartWorker`; follow whichever the neighbouring `InitString` call in
that same function does, so the two paths stay consistent with themselves. When
the registry is empty (Obj directory missing), every named object resolves to
`ENVIRONMENT` — the correct default, since `ObjectKindEnvironment` is `[278]`'s
own default for a blank kind — and `InitObj` logs the count it indexed so a
zero is visible in ingest logs rather than silently producing a
uniformly-`ENVIRONMENT` dataset.

**Rejected:** deriving `kind` lazily at request time in the resource handler.
It would put a filesystem read behind a REST endpoint whose whole point is that
the WZ archive is already ingested, and the serialized archive is not retained
after ingest.

### 4.3 Storage shape — a relationship, not an attribute

`atlas-data` persists a map as its **JSON:API document**, not as a plain struct:
`document/db_storage.go:123-127` does `jsonapi.MarshalToStruct(m, …)` then
`json.Marshal`. That is why `Portals`/`Reactors`/`NPCs`/`Monsters` are
`json:"-"` and still round-trip — they travel as relationships plus `included`.

**D5.** Add map objects the same way, touching all five hooks in
`map/rest.go`, exactly as `reactors` does:

- `Objects []object.RestModel \`json:"-"\`` on `RestModel`;
- `GetReferences()` → `{Type: "map-objects", Name: "objects"}`;
- `GetReferencedIDs()` → id `"{KIND}:{name}"`;
- `GetReferencedStructs()`;
- `SetToManyReferenceIDs("objects", …)`.

New package `services/atlas-data/atlas.com/data/map/object/rest.go` with a
string-id `RestModel` (`GetName() == "map-objects"`), matching the
`map/reactor`, `map/npc`, `map/portal` packages, which are each a single
`rest.go`. The PRD calls this "a new `map/object` package … following their
storage and REST shape" — those siblings have no storage of their own, and
neither does this.

```go
type RestModel struct {
    Id           string `json:"-"` // "{KIND}:{name}"  — matches [278]'s env id
    Kind         string `json:"kind"`
    Name         string `json:"name"`
    ObjectSource string `json:"objectSource"` // oS
    L0           string `json:"l0"`
    L1           string `json:"l1"`
    L2           string `json:"l2"`
    X            int16  `json:"x"`
    Y            int16  `json:"y"`
    Z            int32  `json:"z"`
    Layer        uint32 `json:"layer"`
}
```

**D6 — `kind` is a plain string here, not `field.ObjectKind`.**
`field.ObjectKind` and `field.ParseObjectKind` exist only on `[278]`'s branch
(`libs/atlas-constants/field/constants.go`), not on `main`. Importing it would
make the whole `atlas-data` half of this task depend on `[278]`, for a
two-valued enum. `atlas-data` emits the same two literal strings; the values
are the contract. If a later task wants one shared type, the move is mechanical
and can happen after both branches are on `main`. (CLAUDE.md's "check
`libs/atlas-constants/` first" is satisfied: the type was checked for, and it
is not on `main`.)

**D7 — id collision.** `{KIND}:{name}` must be unique within a map for the
JSON:API `included` array to be well-formed. It is: the merge key in FR-33 is
the same pair, so two same-kind same-name objects would be indistinguishable to
the *server* too. Where a map declares the same name twice (none do in this
dataset, but nothing forbids it), keep the first and drop the rest, logging at
debug. Silently emitting duplicate ids would corrupt the stored document.

### 4.4 Reader

`getObjects(exml xml.Node) []object.RestModel` in `map/reader.go`, called from
`Read` next to `m.Reactors = getReactors(exml)`. It iterates numeric layer
imgdirs (`0`–`7`), descends into each layer's `obj` child, and keeps entries
with a non-empty `name`. Sorted by `(kind, name)` per PRD §5.2. The layer index
is recorded on the row (`layer`) because it is free and orients the operator.

Note `Read` follows `info/link` and returns the *linked* map's model with the
requesting map's id — objects therefore inherit from the link target, which is
the same behaviour portals and reactors already have. No special handling.

**D8 — `libs/atlas-wz/mapimage.objEntry` is not touched.** The PRD flagged
`entries.go:29` as a possible sharing point. `atlas-data` parses serialized
`.img.xml` through its own `xml.Node`; `mapimage` parses the binary `wz.File`
for rendering. They are different inputs and different call graphs. Sharing
would mean routing ingest through the render parser for one string field.
Confirmed: `libs/atlas-wz` gets no change in this task.

### 4.5 Resource and tests

```
GET /api/data/maps/{mapId}/objects
```

Registered in `map/resource.go` next to `/{mapId}/reactors`, handler cloned
from `handleGetMapReactorsRequest` (paginated, `404` when the map document is
absent, `200` + `[]` when the map has no named objects).

Tests:
- reader: a fixture map with named + unnamed `obj` entries yields only the named
  ones; a fixture whose Obj definition carries `obstacle=1` yields `OBSTACLE`;
  the same fixture without the registry initialised yields `ENVIRONMENT`;
- **storage round-trip**: marshal a map carrying objects through
  `jsonapi.MarshalToStruct` → `jsonapi.Unmarshal` and assert the object
  *attributes* survive, not just the ids. This is the one non-obvious bit of D5
  and the reactors path is the reference for it;
- resource: empty-objects map is `200`/`[]`; unknown map is `404`;
- duplicate `{kind}:{name}` in one map keeps one row (D7).

---

## 5. Gateway

Two blocks in `deploy/shared/routes.conf`, in the existing per-resource style,
placed with the other `atlas-maps` instance routes (near line 490):

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

Ordering note: `/api/fields` does not overlap any existing prefix. The
`/environment` block must appear before the generic `^/api/worlds(/.*)?$`
catch-all at line 691 (nginx regex locations match in file order), which the
placement above satisfies — the same reason `/characters`, `/weather`, and
`/monsters` all sit at 490-520.

---

## 6. `atlas-ui`

### 6.1 Service and hook layout

Follows the existing `service → hook → page` layering
(`services/atlas-ui/docs/service-layer.md`). Definition and runtime are
separate modules, not one module with a mode flag — that separation is what
keeps FR-41's cache-independence structural rather than conventional.

| New file | Purpose |
|---|---|
| `src/services/api/fields.service.ts` | `GET /api/fields` with filters |
| `src/services/api/worlds.service.ts` | `GET /api/worlds/`, `GET /api/worlds/{id}/channels` |
| `src/services/api/field-environment.service.ts` | `[278]` GET/POST/DELETE `.../environment` |
| `src/services/api/live-monsters.service.ts` | `GET .../instances/{i}/monsters` |
| `src/services/api/map-entities.service.ts` (extend) | `getObjects(mapId)` |
| `src/lib/hooks/api/useFields.ts` | `useFields(filters)`, `useFieldsForMap(mapId)` |
| `src/lib/hooks/api/useWorlds.ts` | `useWorlds()`, `useChannels(worldId)` |
| `src/lib/hooks/api/useFieldRuntime.ts` | `useFieldCharacters`, `useLiveMonsters`, `useFieldEnvironment`, `useSetEnvironmentObject`, `useResetFieldEnvironment` |
| `src/lib/hooks/api/useMapEntities.ts` (extend) | `useMapObjects(mapId)` |

**Gotcha:** the worlds *collection* is registered as
`r.HandleFunc("/", …)` under the `/worlds` subrouter
(`atlas-world/world/resource.go:29`), so the URL is `/api/worlds/` **with a
trailing slash**. Channels is `""` under `/worlds/{worldId}/channels`, so
`/api/worlds/{id}/channels` without one.

### 6.2 Cache keys and tenancy

**D9 — do not put the tenant in query keys.** The PRD NFR asks for it, but
`src/context/tenant-context.tsx:68` already calls `queryClient.clear()` on every
tenant *change* (and deliberately does not on a same-id rehydrate). Every
existing hook — `useMapEntities.ts` included — relies on that and omits the
tenant from its key. Adding the tenant to only the new keys would create two
conventions in one file and imply the others are unsafe. The NFR's *intent*
(no cross-tenant cache serving) is met by the existing mechanism; the design
records that as the answer rather than diverging. New hooks keep the existing
`enabled: !!activeTenant` guard.

**D10 — two cache profiles.**

| | `staleTime` | `gcTime` |
|---|---|---|
| Definition (map, portals, monsters, reactors, **objects**) | `10 * 60 * 1000` (existing) | `10 * 60 * 1000` |
| Runtime (fields, characters, live monsters, environment) | `5 * 1000` | `60 * 1000` |

Key namespaces are disjoint — `["maps", mapId, …]` vs
`["fields", …]` — so a runtime refetch cannot invalidate definition data
(FR-41). No `refetchInterval` anywhere (FR-39); a reviewer can verify that by
grepping the new hooks for `refetchInterval`.

Refresh + last-updated (FR-40) reuses `useGridRefresh`
(`src/lib/hooks/useGridRefresh.ts`), which already returns
`{ isRefreshing, onRefresh, lastUpdatedAt }` from React Query's own
`isFetching`/`dataUpdatedAt`. No new mechanism.

### 6.3 Routing and breadcrumbs

Routes in `App.tsx` (lazy, as its neighbours):

```
/fields                          → FieldsPage
/fields/:worldId/:channelId/:mapId/:instanceId → FieldDetailPage
```

Sidebar: `{ title: "Fields", url: "/fields" }` immediately after
`{ title: "Maps", url: "/maps" }` in the Operations group of
`app-sidebar-items.ts`.

**D11 — breadcrumbs use `nonNavigable` intermediates.** FR-17 wants
`Fields / <World> / Channel <N> / <Map name> / Instance <id>`, but there are no
pages at `/fields/:w`, `/fields/:w/:c`, or `/fields/:w/:c/:m`. `RouteConfig`
already supports exactly this (`nonNavigable?: boolean`,
`src/lib/breadcrumbs/routes.ts:31-37`, used for the Character grouping node).
Register four configs — three `nonNavigable` intermediates plus the leaf — each
with a `labelResolver` reading its params, and add the leaf to `ROUTES`
(`FIELDS`, `FIELD_DETAIL`) alongside `MAP_DETAIL`.

The world label needs a *name*, which params do not carry.
`BreadcrumbResolverContext` today exposes only `jobName`. Rather than widen it
for one label, the world intermediate resolves to `World <id>` from params, and
the **page header** (FR-18) shows the resolved world name from `useWorlds()`.
This keeps the route table a pure function of params, which is what makes it a
module-level array in the first place.

### 6.4 Component inventory

```
src/pages/FieldsPage.tsx                       (locator)
src/pages/FieldDetailPage.tsx
src/components/features/maps/SurfaceKindBadge.tsx
src/components/features/maps/MapObjectsTable.tsx      (definition tab, FR-6)
src/components/features/maps/LiveFieldsSection.tsx    (FR-7/8/9)
src/components/features/fields/FieldsFilterBar.tsx    (FR-11–13, 15)
src/components/features/fields/FieldsResultTable.tsx  (FR-14)
src/components/features/fields/FieldHeader.tsx        (FR-18)
src/components/features/fields/FieldSummaryPanels.tsx (FR-20)
src/components/features/fields/FieldTabs.tsx          (FR-21)
src/components/features/fields/FieldCharactersTab.tsx
src/components/features/fields/FieldMonstersTab.tsx
src/components/features/fields/FieldObjectsTab.tsx
src/components/features/fields/SetObjectStateDialog.tsx
src/components/features/fields/ResetFieldObjectsDialog.tsx
```

`MapDetailPage.tsx` gains `useMapObjects(id)`, `<SurfaceKindBadge kind="definition">`
via `MapHeader`, `<LiveFieldsSection mapId={id} />` between `ConnectedMapsRow`
and `MapDetailTabs`, and a new tab wired into `MapDetailTabs`. Its existing
`HoverHighlightProvider` / `MapImagePanel` / overlay wiring is untouched
(FR-4).

### 6.5 Live Fields section (FR-7 – FR-9)

`useFieldsForMap(mapId)` calls `GET /api/fields?filter[mapId]=<id>` — one
request, spanning all worlds and channels (FR-9), unaffected by any ambient
selection. FR-7 also wants a live-monster count per row, which `/api/fields`
deliberately does not return.

**D12.** Fan out one `useLiveMonsters` query per listed field, and **cap the
fan-out at 12 rows**; beyond that, render the monster column as `—` with a
"showing first 12" note and lean on the `View all in Fields` link (FR-16). The
PRD permits the fan-out here "because its row count is small and bounded" — it
is bounded by *live instances of one map*, which for a town map on a busy
20-channel server is not small. The cap makes the bound explicit instead of
assumed. The monster count is per-row optional: a failed or pending monster
query renders `—`, never blocking the row.

### 6.6 Fields locator (FR-10 – FR-16)

- World select from `useWorlds()`, defaulting to the lowest-numbered world;
  channel select from `useChannels(worldId)` with an `Any channel` option that
  is the default (FR-12).
- Map filter is a debounced text input matched client-side against the result
  set's map name and map id (FR-13). Map names come from the definition side:
  the locator resolves each distinct `mapId` in the result through the existing
  `useMap`-style definition cache. Filtering after fetch is correct because the
  server filter is exact-`mapId` only and the requirement is a *search*.
- `?map=<mapId>` on load pre-fills the filter and is written back on change
  (FR-16), so the URL is the filter state.
- Empty state echoes filters by resolved name — "World: Scania, Channel: 3" —
  with a Clear filters control (FR-15). Copy must not say the map is missing.

### 6.7 Field detail

- Layout, panels, and tabs per FR-18 – FR-21; `?tab=` in the query string
  (FR-21).
- **FR-22, torn-down field.** There is no `GET /api/fields/{id}`. Existence is
  determined by `GET .../instances/{i}/characters` returning an empty
  collection *and* `GET /api/fields?filter[mapId]=…` not listing this instance.
  Since the liveness rule *is* "has at least one character", an empty
  characters response is exactly the torn-down signal — render the recoverable
  "this field may have been torn down" state with a link back to `/fields`,
  not an error toast. Note the honest consequence: a real live field cannot be
  empty by definition, so this is unambiguous.
- **FR-19 pins.** `MapImagePanel` already accepts positioned definition
  entities (`portals`, `npcs`, `monsters`, `reactors`) and renders them through
  `MapImageOverlay`. The Field overview passes runtime entities through the
  *same* props shape rather than a new one; live monsters carry `x`/`y`, and
  characters carry a position only if the character payload provides one
  (§6.8). No hit-testing, no placement — the requirement is that the component
  accepts positioned runtime entities, which it already does.

### 6.8 Characters tab (FR-23 – FR-27)

`GET /api/worlds/{w}/channels/{c}/maps/{m}/instances/{i}/characters` returns a
JSON:API collection of type `characters` whose only content is the id
(`map/rest.go`: `RestModel{ Id string }`, no attributes). Per-row enrichment via
the existing character hook, one query per id, cached and deduplicated by React
Query (FR-24). Each row renders from its own query state: pending → the id,
error → the id, success → the full row. No `Promise.all`, no blocking.

FR-25 lists `Name, Character ID, Level, Job, Position, State`. **Columns are
included only if the character payload actually carries them** — the plan phase
must read `atlas-character`'s REST model and drop what is not there rather than
render blanks. Name/ID/Level/Job are certain; Position and State are not, and
this design does not assert them.

### 6.9 Monsters tab (FR-28 – FR-31) — Open Question 3 resolved

Read from `atlas-monsters/monster/rest.go` on `main`. The payload carries:

| PRD column | Backing attribute | Verdict |
|---|---|---|
| Object ID | `id` (`m.UniqueId()`) | keep |
| Monster | `monsterId` → Monster Definition link | keep |
| HP | `hp` / `maxHp` | keep |
| Position | `x`, `y` (also `fh`, `stance`) | keep |
| Spawn | `spawnSourceType` + `spawnSourceId` (both `omitempty`) | keep, blank-tolerant |
| State | — no `state`/`alive` attribute exists | **drop the column** |

**D13.** There is no `State` column. FR-31 ("dead and alive visually
distinguished") is satisfied by `hp == 0`: `monster.Model` has an `Alive()`
predicate (used at `processor.go:1871`) and a killed monster exists at zero HP
until it is destroyed, so `hp === 0` is the honest client-side rendering of
dead. Style the row, do not invent a status string.

Additional attributes the endpoint returns and this tab does **not** show:
`controlCharacterId`, `damageEntries`, `experienceEntries`, `statusEffects`,
`team`, `mp`/`maxMp`. Out of scope for v1; noted so a later task knows they are
free.

FR-30's "link the spawn to the Map Definition's Monster Spawns tab" is
best-effort: `spawnSourceId` is an opaque string, and correlating it to a row in
the definition's spawn list is not a mapping this design can assert. Render it
as text, and link the *tab* (`/maps/{mapId}?tab=monster-spawns`) rather than a
specific spawn row.

### 6.10 Map Objects tab (FR-32 – FR-38) — Open Question 4 resolved

Two sources merged on `{kind}:{name}` — which is deliberately the id of *both*
resources (`map-objects` from `atlas-data`, `environment-objects` from `[278]`),
so the merge is an id join, not a heuristic:

1. tracked (`GET .../environment`) → rendered first, with current `state`;
2. declared-but-untracked (`GET /api/data/maps/{id}/objects` minus the tracked
   ids) → under the divider "Defined on the map, no state tracked in this
   field", shown at default state.

Both queries resolving empty is a normal empty state, not a 404 (FR-32).

**D14 — no free-text object entry in v1.** OQ4 asked whether an advanced
`name` + explicit `kind` path is needed for a script-driven object the map does
not declare. It is not, because the gap closes itself: any object with tracked
state appears in source (1) *carrying its own kind*, so once written it is
editable from the UI forever after. The only unreachable case is the *first*
write to an undeclared name, which is a script author's concern, not an
operator's, and FR-38's whole rationale is that names come from a
definition-derived list. `Reset all to default` (FR-35) clears tracked state
regardless of provenance, so nothing becomes unrecoverable. If this bites,
adding the advanced path later is additive.

`kind` on a Set is always taken from the row (FR-34) — never from user input,
never guessed.

**Writes.** `POST .../environment` with a JSON:API document
`{data:{type:"environment-objects", id:"{KIND}:{name}", attributes:{kind,name,state}}}`
— `[278]` registers it via `rest.RegisterInputHandler[RestModel]`, which
expects the JSON:API envelope, not a bare object. `DELETE .../environment` for
reset. Both behind `AlertDialog` confirmations (FR-36) naming
world/channel/map/instance, and for Set the object and target state, for Reset
the count of tracked objects; both dialogs state that the change broadcasts to
every character in the field. On success: invalidate the field's environment
key only, and toast (FR-37). A `400` renders inline on the row
(FR-37) using the existing `ErrorDisplay`/field-error pattern, not a toast, so
the failing row is identifiable.

### 6.11 Frontend tests

Per the PRD's testing NFR, plus what the design added:

- locator filter behaviour and the filter-echoing empty state (FR-15);
- `?map=` pre-filter round-trip (FR-16);
- Live Fields empty state renders, and the section is never hidden (FR-8);
- Live Fields fan-out cap at 12 (D12);
- Map Objects merge: tracked-only, untracked-only, both, and an object present
  in both appearing exactly once in the tracked group (FR-33);
- Set/Reset dialogs block dispatch until confirmed, and the Set payload carries
  the row's `kind` (FR-34, FR-36);
- Characters tab degrades to the raw id on a failed enrichment without
  unmounting the row (FR-24);
- Monsters tab distinguishes `hp === 0` (D13);
- no new hook sets `refetchInterval` (FR-39).

---

## 7. Alternatives considered and rejected

**A. Model field lifecycle in `atlas-maps` (createdAt, state) instead of
deriving liveness from occupancy.** Would answer PRD OQ1 and OQ2 directly — a
field with tracked object state but no characters would be visible. Rejected:
it means introducing real instance lifecycle tracking into a service that today
stores only a character-id slice, which is a larger design than this whole task
and would need Kafka lifecycle events to stay correct. The PRD already accepts
the consequence.

**B. Serve `/fields` from a new aggregating service or from `atlas-world`.**
Rejected: the authoritative data is the `atlas-maps` in-memory registry. Any
other owner would need to replicate it or proxy to it.

**C. One `/maps/:id` page with a Definition/Runtime toggle instead of two
routes.** Rejected: it makes the field instance un-linkable and un-bookmarkable,
which defeats the operator story ("correlate a player report with a place"), and
it re-creates exactly the definition/runtime label ambiguity FR-2 exists to
remove.

**D. Compute `kind` in `atlas-ui` by fetching Obj definitions.** Rejected: the
Obj archive is not exposed over REST and the client has no business parsing WZ.

**E. Add `ObjectKind` to `libs/atlas-constants` in this task.** Rejected —
see D6; it collides with `[278]`'s identical addition.

---

## 8. Multi-tenancy, safety, observability

- `GET /api/fields` filters by `MapKey.Tenant` **inside** the registry method
  before anything else (D1). The tenant-isolation test is a required test, not
  an optional one.
- `atlas-data` map objects are stored in the tenant-scoped map document; no new
  tenancy surface.
- Both new handlers register through `rest.RegisterHandler` /
  `registerGet`, inheriting request logging and tracing. The field enumeration
  logs failures with tenant and the parsed filter values.
- The only writes are `[278]`'s two environment operations, both confirmed, both
  labelled with their blast radius (§6.10).
- Ingest: adding `Objects` to the stored map document means **existing ingested
  tenants show no map objects until the MAP worker re-runs.** No migration is
  possible or needed — the data is re-derivable — but the plan should say so and
  the docs should mention it.

---

## 9. Documentation updates

- `services/atlas-maps/docs/rest.md` — `GET /api/fields`, filters, the liveness
  rule, the id format.
- `services/atlas-data/docs/rest.md` — `GET /api/data/maps/{id}/objects`.
- `services/atlas-data/docs/domain.md` — the map-object entity and the
  `obstacle=1` kind resolution.
- `services/atlas-ui/docs/service-layer.md` — the new service modules and the
  definition-vs-runtime cache profiles (D10).

---

## 10. Open items for the plan phase

1. **`[278]` merge order (needs a human decision).** Either merge `[278]` to
   `main` before task-292 executes — preferred, it is complete and only lacks a
   PR — or plan the Map Objects runtime half (FR-32, FR-34–FR-38) as a trailing
   unit that cannot start until it lands. Everything else in this design builds
   on `main` today.
2. **Character payload columns.** Read `atlas-character`'s REST model in the
   plan phase and fix the Characters-tab column list (§6.8). Do not render a
   column the payload does not supply.
3. **`InitObj` failure policy** must match whichever policy the neighbouring
   `InitString` call uses in each of the two ingest entry points (§4.2).

---

## 11. Requirement coverage

| PRD | Where |
|---|---|
| FR-1 – FR-3 | §2 |
| FR-4 – FR-6 | §2, §4.5, §6.4 |
| FR-7 – FR-9 | §6.5 (D12) |
| FR-10 – FR-16 | §6.3, §6.6 |
| FR-17 – FR-22 | §6.3 (D11), §6.7 |
| FR-23 – FR-27 | §6.8 |
| FR-28 – FR-31 | §6.9 (D13) |
| FR-32 – FR-38 | §6.10 (D14), §1.1 (D0) |
| FR-39 – FR-41 | §6.2 (D10) |
| §5.1 | §3 (D1–D3) |
| §5.2 | §4 (D4–D8) |
| §5.4 | §5 |
| §6 data model | §3.1, §4.2, §4.3 |
| §8 NFRs | §8 |
| OQ1, OQ2 | §7 alternative A — accepted as-is |
| OQ3 | §6.9 (D13) — `State` column dropped |
| OQ4 | §6.10 (D14) — no free-text entry in v1 |
| OQ5 | §6.3 — top-level `/fields`, unchanged |
