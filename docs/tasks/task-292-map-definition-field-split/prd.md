# Map Definition / Field Separation — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-02
---

## 1. Overview

`atlas-ui` today has exactly one map surface: `MapDetailPage` (`services/atlas-ui/src/pages/MapDetailPage.tsx`),
keyed by a bare `mapId` and fed entirely from `atlas-data` (`/api/data/maps`, `map-entities.service.ts`).
Everything it shows is static WZ configuration — portal definitions, monster spawn points, reactor
placements — but its tab labels (`Portals (3) | Monsters (22) | Reactors (0)`) read as though they
describe things that exist right now. There is no surface anywhere in the UI for the *runtime* side:
no way to see which live field instances exist, who is standing in one, what monsters are alive in it,
or what state its map objects are in. A repo-wide grep for `/api/worlds` across
`services/atlas-ui/src` returns exactly one hit — MTS listings — so the runtime half of the server is
currently invisible to operators.

This task introduces the missing concept and the vocabulary to keep it straight. A **Map Definition**
describes what a map is configured to contain. A **Field** is one live instantiation of that map at a
specific `world + channel + instance`. The existing page becomes explicitly definition-owned; a new
Field inspector becomes the runtime observation surface, reachable both from the definition page (a
Live Fields section) and from its own top-level locator for operators who know "world 0, channel 3"
but not which map they want.

The immediate operational driver is task-278 (branch `task-278-map-environment-object-state`, 29
commits ahead of `origin/main`, no PR). It added per-field environment/obstacle object state to
`atlas-maps` with `GET|POST|DELETE /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/environment`, but
its PRD lists frontend surfaces as an explicit non-goal and it never added a gateway route, so the
endpoints are unreachable from a browser. Exercising the feature currently requires a shell and a
hand-assembled instance UUID. The Field page's Map Objects tab is where that becomes a real testing
surface — but the Field concept it needs is worth building on its own terms, and the great majority
of this task does not depend on task-278 landing.

## 2. Goals

Primary goals:

- Establish Map Definition and Field as distinct, cross-linked resources in `atlas-ui`, with a
  consistent terminology rule (§4.1) that never uses the same label for a configured count and a
  runtime count.
- Give `atlas-maps` a field-instance enumeration endpoint, so "which fields are live" is answerable
  without knowing an instance UUID in advance.
- Give `atlas-data` a map-object resource derived from WZ `obj[].name`, so the addressable
  `SetObjectState` / `FieldObstacleOnOff` targets for a map are discoverable instead of guessed.
- Deliver a Fields locator (world → channel → map) and a Field detail inspector with Characters,
  Monsters, and Map Objects tabs.
- Expose task-278's environment object state — read and write — on the Field page's Map Objects tab.
- Route the new endpoints, and task-278's `/environment` endpoints, through the nginx gateway.

Non-goals:

- Runtime tabs for Drops, NPCs, Reactors, Summons, or Events. The Field page must accommodate them
  without restructuring, but this task ships three tabs.
- Interactive object placement or hit-testing on the map image. The Field overview decorates the
  existing map image with runtime pins; evolving toward true placement is an architectural
  requirement, not a deliverable.
- Any runtime write control other than environment object state. No monster spawning, no character
  teleport, no field teardown from this UI, even where the backend endpoint exists (e.g.
  `DELETE .../monsters`).
- Changes to task-278's Kafka commands, status events, saga actions, or packet emission. This task
  consumes its REST contract as designed.
- Persisting field state, field lifecycle management, or an instance-creation path. Fields are
  runtime-created; the UI observes them.
- A world/channel runtime dashboard. Fields may later hang off one; this task does not build it.

## 3. User Stories

- As an operator, I want to know whether a map is currently instantiated anywhere, so I can tell a
  configuration problem from a runtime one.
- As an operator, I want to find a live field by world and channel without knowing the map, so I can
  investigate "something is wrong on channel 3."
- As an operator, I want to see who is standing in a specific field instance, so I can correlate a
  player report with a place.
- As an operator, I want to see live monsters with their current HP and originating spawn, so I can
  tell "the spawn is broken" from "they are all dead."
- As a GM/tester, I want to set a map object's state from the UI and watch it apply to everyone in
  the field, so I can exercise the environment feature without a shell.
- As a GM/tester, I want to reset a field's objects to their defaults, so I can re-run a test from a
  clean state.
- As a content author, I want to see which named objects a map declares, so I know what names a
  `move_environment` script may legally target.
- As any user of the UI, I want configured counts and live counts to be labelled differently, so I
  never mistake 22 spawn points for 22 live monsters.

## 4. Functional Requirements

### 4.1 Terminology (applies to every requirement below)

- **FR-1.** Definition-side labels use configuration vocabulary: *Configured*, *Spawn*, *Definition*,
  *Target map*, *Template*. Runtime-side labels use observation vocabulary: *Live*, *Instance*,
  *Object ID*, *Current HP*, *Position*, *State*, *Owner*.
- **FR-2.** No label may appear on both surfaces with different meanings. Specifically, the existing
  Map Definition tab `Monsters (N)` is renamed `Monster Spawns (N)`; the Field tab keeps `Monsters (N)`
  meaning live instances. The `MapEntitySummary` heading for monsters becomes
  `Configured Monster Spawns`; the Field equivalent is `Live Monsters`.
- **FR-3.** Each page carries a single explicit kind indicator: the Map Definition page a
  `Definition` badge, the Field page a `Runtime` badge. These use distinct, consistently applied
  colour tokens and are also reflected in the Fields sidebar entry.

### 4.2 Map Definition page (`/maps/:id`)

- **FR-4.** The existing page keeps its route, its user-facing title (the map name), and all existing
  content: `MapHeader`, `MapImagePanel`, `MapEntitySummary`, `ConnectedMapsRow`, and the
  portal/monster/reactor tabs (`services/atlas-ui/src/pages/MapDetailPage.tsx:44-` and
  `src/components/features/maps/MapDetailTabs.tsx`). Map overlays and hover-highlight behaviour from
  task-009 are preserved unchanged.
- **FR-5.** Tab labels and summary headings are updated per FR-1/FR-2. No data source changes.
- **FR-6.** A new **Map Objects** tab lists the map's named WZ objects from the new `atlas-data`
  resource (§5.2), with columns: Kind, Object name, WZ source (`oS/l0`), Position. When the map
  declares none, the tab shows an empty state explaining that only named `obj` entries are
  addressable by object-state operations.
- **FR-7.** A new **Live Fields** section is placed after the map overview and before the definition
  tabs. It is a discovery/navigation aid, not an embedded inspector: it lists at most World, Channel,
  Instance, character count, and live-monster count, and each row navigates to the corresponding
  Field page. It must not list characters, monsters, or objects individually.
- **FR-8.** When the map has no live fields, the Live Fields section renders an explicit empty state —
  "No active fields. This map is not currently instantiated in any world or channel." — with a note
  that fields are runtime-created and cannot be created from the UI. The section is never hidden.
- **FR-9.** The Live Fields section spans every world and channel for that map. It is not filtered by
  any ambient world/channel selection.

### 4.3 Fields locator (`/fields`)

- **FR-10.** A new top-level `Fields` entry is added to the sidebar
  (`services/atlas-ui/src/components/app-sidebar-items.ts:46`), immediately after `Maps`. Fields is
  not a child route of Maps.
- **FR-11.** Navigating to `/fields` selects no field. The page is a locator: a World select, a
  Channel select, and a searchable Map filter, above a result table of matching live fields.
- **FR-12.** World and Channel options come from `atlas-world` (`GET /api/worlds`,
  `GET /api/worlds/{id}/channels`). Channel supports an "Any channel" option. The world defaults to
  the lowest-numbered world; channel defaults to "Any channel".
- **FR-13.** The Map filter is a text search over map name and map id — never a select listing every
  map. An empty Map filter means "all maps".
- **FR-14.** Result columns: Map (name + id, linking to the Field), Channel, Instance, Characters.
  Rows navigate to Field detail. Live-monster count is NOT fetched here (see FR-24).
- **FR-15.** When no fields match, the page shows an empty state that echoes the active filters by
  name (e.g. "World: Scania, Channel: 3") and offers an obvious control to clear them. The copy must
  not imply the map definition is missing.
- **FR-16.** `/maps/:id`'s Live Fields section offers a link into `/fields` pre-filtered to that map
  (`/fields?map=<mapId>`), and the locator honours that query parameter on load.

### 4.4 Field detail (`/fields/:worldId/:channelId/:mapId/:instanceId`)

- **FR-17.** Breadcrumb: `Fields / <World> / Channel <N> / <Map name> / Instance <instanceId>`,
  registered in `services/atlas-ui/src/lib/breadcrumbs/routes.ts` alongside the existing `MAP_*`
  entries and following `services/atlas-ui/docs/breadcrumb-navigation.md`.
- **FR-18.** The header shows the map name as the primary title, with `World / Channel / Instance`
  rendered more prominently than the map id, plus a `View Map Definition` link to `/maps/:mapId`.
- **FR-19.** The overview reuses the same map image as the definition page, decorated with runtime
  pins (characters, live monsters, objects with non-default state). Pins are illustrative; exact
  hit-testing is out of scope, but the component must accept positioned runtime entities so it can
  evolve.
- **FR-20.** A **Live summary** panel shows character count, live monsters grouped by name, count of
  map objects with tracked state, and any other runtime counters already available. A separate
  **Map** panel gives static orientation only — map name, id, street, configured spawn count, NPC
  definition count, portal count — and a link to the Map Definition. The two panels are visually
  distinguished per FR-3.
- **FR-21.** Tabs: `Characters (N)`, `Monsters (N)`, `Map Objects (N)`. Tab state is local, with the
  active tab reflected in a `?tab=` query parameter so a tab is deep-linkable.
- **FR-22.** If the field no longer exists (torn down between navigation and load), the page shows a
  recoverable state explaining the field may have been torn down, with a link back to `/fields`. This
  is not an error toast.

### 4.5 Characters tab

- **FR-23.** Lists characters present in this exact field instance, sourced from
  `GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/characters` (`atlas-maps`, already routed at
  `deploy/shared/routes.conf:490`), which returns bare character ids.
- **FR-24.** Each id is enriched client-side via the existing `GET /api/characters/{id}` per row,
  through React Query so repeat views are cached. Rows render progressively; a slow or failed
  enrichment degrades to the character id, never blocking the table.
- **FR-25.** Columns: Name, Character ID, Level, Job, Position, State. Position and State are included
  only where the underlying payload provides them; a field the API does not supply is omitted from
  the table rather than rendered blank.
- **FR-26.** Character names link to the existing Character Detail page, giving bidirectional
  navigation (Character → current field, Field → character).
- **FR-27.** An empty tab reads "No characters are currently in this field." and is styled as a normal
  state, not an error.

### 4.6 Monsters tab

- **FR-28.** Lists live monster instances from the existing
  `GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/monsters`
  (`services/atlas-monsters/atlas.com/monsters/world/resource.go:36`, routed at
  `deploy/shared/routes.conf:505`). No backend change is required for this tab.
- **FR-29.** Columns: Object ID, Monster, HP (current/max), Position, Spawn, State — restricted to
  fields the endpoint actually returns; any column without a backing attribute is dropped, not faked.
- **FR-30.** The monster name links to the static Monster Definition page. Where the payload carries a
  spawn identifier, it links to the Map Definition's Monster Spawns tab.
- **FR-31.** Dead and alive monsters are visually distinguished.

### 4.7 Map Objects tab (consumes task-278)

- **FR-32.** Reads tracked state from `GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/environment`
  (`atlas-maps`, task-278). The response is a possibly-empty collection of
  `{kind, name, state}`; an empty collection is a normal state, not a 404.
- **FR-33.** The tab merges two sources: objects with tracked state (from the field), and objects the
  map *declares* but which have no tracked state (from the `atlas-data` map-object resource, §5.2).
  The latter are grouped under a visually separated "Defined on the map, no state tracked in this
  field" divider and display as being at their default state. Without this merge the tab is empty on
  a fresh field and unusable as a testing surface.
- **FR-34.** Each row offers a state input and a Set action issuing
  `POST .../environment` with `{kind, name, state}`. The `kind` is taken from the definition data,
  never guessed by the user.
- **FR-35.** A field-level **Reset all to default** action issues `DELETE .../environment`.
- **FR-36.** Both write actions require an explicit confirmation dialog before dispatch. The dialog
  names the field (world/channel/map/instance) and, for Set, the object and target state; for Reset,
  the number of tracked objects that will be cleared. Rationale: both writes broadcast to every
  character in the field.
- **FR-37.** After a successful write the tab refetches, and a toast confirms the action. A `400` from
  an invalid kind or malformed state surfaces as an inline error on the row, not a silent no-op.
- **FR-38.** The tab notes that object names are pass-through: the server does not validate a name
  against WZ data, and an unknown name is a client-side no-op (task-278 FR-3). This is why the name
  comes from a definition-derived list rather than free text.

### 4.8 Refresh semantics

- **FR-39.** Runtime queries use a short `staleTime` (a few seconds) and do NOT poll on an interval.
- **FR-40.** Field detail and the Fields locator each expose an explicit Refresh control and display a
  "last updated" timestamp, so the operator knows how stale what they are looking at is.
- **FR-41.** Definition queries keep their existing, longer cache behaviour; runtime and definition
  caches are keyed separately and a runtime refetch never invalidates definition data.

## 5. API Surface

### 5.1 `atlas-maps` — field enumeration (new)

```
GET /fields
GET /fields?filter[worldId]=0&filter[channelId]=1&filter[mapId]=910340000
```

Tenant-scoped via the standard `TENANT_ID` / `REGION` / `MAJOR_VERSION` / `MINOR_VERSION` headers.
Paginated using the service's existing pagination helper (see
`services/atlas-maps/atlas.com/maps/map/resource_paginate_test.go`).

Response — JSON:API collection of type `fields`, id `{worldId}:{channelId}:{mapId}:{instanceId}`:

```json
{
  "data": [{
    "type": "fields",
    "id": "0:1:910340000:6f1c8a2e-....",
    "attributes": {
      "worldId": 0,
      "channelId": 1,
      "mapId": 910340000,
      "instanceId": "6f1c8a2e-....",
      "characterCount": 4
    }
  }]
}
```

- **Liveness rule.** A field appears if and only if it currently holds at least one character.
  `services/atlas-maps/atlas.com/maps/map/character/registry.go:15` is
  `map[MapKey][]uint32`, and `RemoveCharacter` (line 57) empties a key's slice without deleting the
  key — so key existence alone is not liveness. Enumeration filters to non-empty slices.
- Filters are optional and combinable. Unknown filter keys are ignored; malformed filter values are
  `400`.
- Errors: `400` invalid filter value; `500` registry read failure.

### 5.2 `atlas-data` — map objects (new)

```
GET /api/data/maps/{mapId}/objects
```

Response — JSON:API collection of type `map-objects`, id `{kind}:{name}` (matching task-278's
`environment` RestModel id convention so the two correlate):

```json
{
  "data": [{
    "type": "map-objects",
    "id": "ENVIRONMENT:gate",
    "attributes": {
      "kind": "ENVIRONMENT",
      "name": "gate",
      "objectSource": "effect",
      "l0": "quest", "l1": "gate", "l2": "0",
      "x": 640, "y": 120, "z": 0, "layer": 3
    }
  }]
}
```

- Only `obj` entries carrying a non-empty `name` are exposed — these are the only objects addressable
  by `SetObjectState` / `FieldObstacleOnOff`.
- `kind` is `OBSTACLE` when the referenced object definition under `Map.wz/Obj/{oS}.img` →
  `{l0}/{l1}/{l2}` carries `obstacle=1`, otherwise `ENVIRONMENT`.
- Sorted by kind, then name.
- `200` with an empty array for a map with no named objects; `404` only if the map itself is unknown.

### 5.3 Existing endpoints consumed (no change)

| Endpoint | Service | Used by |
|---|---|---|
| `GET /api/worlds`, `GET /api/worlds/{id}/channels` | `atlas-world` (`world/resource.go:29`, `channel/resource.go:25`) | Locator selects |
| `GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/characters` | `atlas-maps` (`map/resource.go:31`) | Characters tab |
| `GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/monsters` | `atlas-monsters` (`world/resource.go:36`) | Monsters tab, Live Fields monster count |
| `GET /api/characters/{id}` | `atlas-character` | Characters tab enrichment |
| `GET|POST|DELETE /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/environment` | `atlas-maps` (task-278) | Map Objects tab |
| `GET /api/data/maps/{mapId}`, `/portals`, `/monsters`, `/npcs`, `/reactors` | `atlas-data` | Map Definition page |

### 5.4 Gateway (`deploy/shared/routes.conf`)

Two new location blocks, following the existing per-resource pattern at lines 490-520:

```
location ~ ^/api/worlds/[^/]+/channels/[^/]+/maps/[^/]+/instances/[^/]+/environment(/.*)?$ { ... atlas-maps:8080 }
location ~ ^/api/fields(/.*)?$ { ... atlas-maps:8080 }
```

The `/environment` block is required regardless of this task's UI work — task-278 added the handlers
but no route, so those endpoints are currently unreachable through the gateway.

## 6. Data Model

No database schema changes and no migrations.

- **`atlas-maps`** — the field enumeration reads the existing in-memory character registry. It
  introduces no new persistent state and no new registry. If a read-side aggregation type is needed
  it is derived, not stored.
- **`atlas-data`** — a new `map/object` package alongside the existing `map/monster`, `map/npc`,
  `map/portal`, `map/reactor` packages (`services/atlas-data/atlas.com/data/map/reader.go:4-8`),
  following their storage and REST shape. `reader.go` gains parsing of per-layer `obj` entries,
  retaining only those with a `name`, and resolves `kind` by reading
  `{root}/Map.wz/Obj/{oS}.img.xml` — reachable from the same ingest root the map worker already
  walks (`data/workers/mapw.go:50`). Records are tenant-scoped exactly as the sibling map entities are.
- **`libs/atlas-wz`** — `mapimage.objEntry` (`libs/atlas-wz/mapimage/entries.go:29`) drops the `name`
  property today because it exists for rendering. It is only modified if the design phase finds
  `atlas-data`'s ingest should share it; the map-object resource does not otherwise depend on it.

**WZ ground truth** (swept across all 5262 `Map.wz/Map/Map*/**.img.xml` in the reference data):

- 34 maps declare at least one named object; 595 distinct names.
- 2004 of the ~2100 occurrences are the four beach maps `109080000`–`109080003` (`001`…`501`).
- The meaningful set is small: `gate` on `103000800`-`804`, `922010100`-`800`, `670010300`-`500`;
  `gate0/1/2` on `670010200`; `r1`-`r4` on `926100300`/`926110300`; `menhir0`-`5` and
  `2pt`/`4pt`/`a1`… on the Crimsonwood PQ maps `610030200`-`500`; `oliviaMirror` on
  `682010100`-`102`; `trap1`-`12` on `920010910`/`920`/`930`.
- Cross-referencing the 214 `obstacle=1` frames in `Map.wz/Obj/*` against every map `obj` entry:
  276 maps contain obstacle-flagged objects, but exactly **one** (`610030300`, `menhir0`-`5`) gives
  them names. `OBSTACLE` is therefore a real but rare kind, which the UI must support without
  optimising for.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-ui` | Bulk of the work: terminology pass on the Map Definition page; new Map Objects definition tab; Live Fields section; new `/fields` locator and `/fields/:w/:c/:m/:i` detail with three tabs; sidebar entry; breadcrumb registrations; new service-layer modules and React Query hooks for fields, map objects, worlds/channels, live monsters, and environment state |
| `atlas-maps` | New `GET /fields` collection resource with filters and pagination; non-empty liveness rule over the existing character registry; docs update in `services/atlas-maps/docs/rest.md` |
| `atlas-data` | New `map/object` package: WZ `obj[].name` ingest, `obstacle=1` kind resolution, storage, REST resource, docs |
| `deploy` | Two gateway location blocks in `deploy/shared/routes.conf` |
| `libs/atlas-wz` | Possibly none; `objEntry.name` only if the design phase shares the parser |
| `atlas-monsters`, `atlas-world`, `atlas-character` | None — consumed as-is |

Dependency note: the Map Objects tab's runtime half (FR-32, FR-34-FR-37) consumes task-278's REST
contract, which is on branch `task-278-map-environment-object-state` and not yet on `main`. Every
other requirement builds on `main`. Implementation sequences that panel last.

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every new endpoint is tenant-scoped through the standard headers; the
  `atlas-maps` enumeration filters registry keys by `MapKey.Tenant` before anything else, and must
  never leak a field belonging to another tenant. The UI sends tenant context through the existing
  client, and all React Query keys include the active tenant so switching tenants cannot serve
  cached cross-tenant data.
- **Request volume.** The Characters tab issues one enrichment request per character (FR-24); this is
  bounded by field occupancy and must be cached and deduplicated by React Query. The Fields locator
  deliberately omits monster counts (FR-14) so listing N fields costs one request, not N+1; the
  Live Fields section on a single map's definition page may fan out because its row count is small
  and bounded.
- **No polling** (FR-39). Any future move to polling or streaming must be a deliberate decision, not
  an accident of a default.
- **Write safety.** The only writes are the two environment operations, both confirmed (FR-36). They
  broadcast to every character in the field; the confirmation copy must make the blast radius
  explicit.
- **Observability.** New handlers follow existing REST-handler registration so they inherit the
  standard request logging and tracing; `atlas-maps` logs an enumeration failure with tenant and
  filter context.
- **Accessibility and consistency.** New surfaces use existing shadcn/ui primitives, table, tab, and
  empty-state patterns already used by `MapDetailTabs` and the transports UI; the Definition/Runtime
  distinction is carried by a label, not by colour alone.
- **Testing.** New `atlas-maps` and `atlas-data` handlers get resource-level tests including the
  empty and filtered cases; the liveness rule (a drained field's lingering empty registry key must
  not be listed) gets an explicit test. Frontend gets tests for the locator's filter/empty-state
  behaviour, the Live Fields empty state, and the Map Objects merge of tracked and untracked objects.

## 9. Open Questions

1. **Field age and lifecycle state.** §3 of the source requirements suggested `age` and `state`
   columns. With the non-empty liveness rule there is no `createdAt` and no lifecycle state to report
   — the registry stores only character ids. Both columns are dropped from v1. Adding them means
   giving `atlas-maps` real instance lifecycle tracking; worth revisiting if operators miss it.
2. **A field with tracked object state but zero characters is invisible.** A PQ field whose party
   wiped disappears from the locator while its environment state is still tracked (task-278 clears
   state on field-empty, so the window is short, but it exists). Acceptable for v1; noted because it
   is the one observable consequence of the liveness rule.
3. **Monster payload shape.** FR-29's columns are written against the expected
   `atlas-monsters` response; the design phase must confirm which of Object ID / HP / Position /
   Spawn / State the endpoint actually returns and drop any that it does not, rather than inventing
   them.
4. **`kind` override on write.** FR-34 takes `kind` from definition data. For a name the map does not
   declare (a script-driven object, or WZ drift between client versions), there is no definition row
   and therefore no kind. Whether to allow an advanced free-text + explicit-kind entry path is
   deferred to design.
5. **Where `/fields` sits if a world/channel runtime dashboard is later added.** This task puts it at
   top level next to Maps; that placement should survive, but a future dashboard may also link into it.

## 10. Acceptance Criteria

Definition surface:

- [ ] `/maps/:id` renders a `Definition` badge and no runtime state.
- [ ] The monster tab reads `Monster Spawns (N)`; the summary reads `Configured Monster Spawns`. No
      label collides with a Field-page label of different meaning.
- [ ] A Map Objects tab lists the map's named WZ objects with kind, name, WZ source, and position;
      `103000800` shows one `ENVIRONMENT` object named `gate`; `610030300` shows six `OBSTACLE`
      objects `menhir0`-`menhir5`.
- [ ] A Live Fields section appears after the overview and before the tabs, listing world, channel,
      instance, and character count, with each row navigating to the Field page.
- [ ] A map with no live fields shows the explicit empty state, never a hidden section.

Locator:

- [ ] `Fields` appears in the sidebar directly after `Maps` and routes to `/fields`.
- [ ] `/fields` selects no field and presents World, Channel, and searchable Map filters.
- [ ] World and channel options are populated from `atlas-world`, not hard-coded.
- [ ] Filtering to a world/channel with no live fields shows an empty state naming those filters and
      offering a clear-filters control.
- [ ] `/fields?map=<id>` loads pre-filtered to that map.

Field detail:

- [ ] `/fields/:w/:c/:m/:i` shows the map name as title, world/channel/instance more prominently than
      the map id, a `Runtime` badge, and a working `View Map Definition` link.
- [ ] Breadcrumb reads `Fields / <World> / Channel <N> / <Map> / Instance <id>`.
- [ ] Characters tab lists live characters with names resolved, linking to Character Detail; an empty
      field shows the normal-state empty message.
- [ ] Monsters tab lists live monster instances with current HP, distinguishing dead from alive, and
      links monster names to the Monster Definition.
- [ ] Map Objects tab shows tracked object state and, separately, declared-but-untracked objects.
- [ ] Set and Reset each require confirmation naming the field; on confirm they issue `POST` and
      `DELETE` to `.../environment` and the tab refetches.
- [ ] Runtime views expose a Refresh control and a last-updated timestamp, and do not poll.

Backend and infrastructure:

- [ ] `GET /fields` returns only fields holding at least one character; a test proves a drained
      field's lingering empty registry key is excluded.
- [ ] `GET /fields` honours `filter[worldId]`, `filter[channelId]`, `filter[mapId]` independently and
      combined, and is tenant-scoped.
- [ ] `GET /api/data/maps/{mapId}/objects` returns only named `obj` entries and resolves `OBSTACLE`
      via `obstacle=1` in the referenced `Map.wz/Obj` definition.
- [ ] `deploy/shared/routes.conf` routes both `/api/fields` and the task-278
      `.../instances/{i}/environment` paths to `atlas-maps`.
- [ ] `services/atlas-maps/docs/rest.md` and the `atlas-data` service docs describe the new endpoints.
- [ ] Flagless `tools/verify.sh` exits 0.
