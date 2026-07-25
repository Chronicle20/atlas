# Character Rankings Leaderboard — Design

Date: 2026-07-24
Task: task-143-character-rankings (follow-on to the base rankings feature)
Status: approved (design), pending implementation plan

## 1. Context & Problem

The base task-143 work built the **atlas-rankings** service: it periodically
computes per-world overall and per-job-category rankings for every non-GM
character in each tenant, persists them, and exposes them over REST **for the
login character-select decoration**. That REST surface is point-lookup only:

| Endpoint | Returns |
|---|---|
| `GET /rankings/characters?ids=1,2,3` | ranking rows for those character ids |
| `GET /rankings/characters/{characterId}` | one character's ranking row |

There is **no "list top-N by world" query**, and the ranking row/REST model
carries only `worldId, rank, rankMove, jobRank, jobRankMove, computedAt` +
characterId — **no name, level, or job**. The PRD listed atlas-ui leaderboard
pages as a deliberate non-goal, with the `(tenant_id, world_id)` index seeded
"for future leaderboard queries."

This design adds that future work: a **full-stack leaderboard** — a new backend
list endpoint plus an atlas-ui page that renders ranked characters, including a
character render image per row.

## 2. Decisions (locked with the requester)

1. **Full-stack** — add the missing backend list endpoint, then build the UI.
2. **Store name/level/job on the ranking row at compute time** — the recompute
   already reads every character, so persisting these makes the leaderboard
   endpoint self-contained (one query, no read-path fan-out to atlas-character)
   and gives an instant, reliable *text* leaderboard.
3. **Character image is a frontend-only, progressive concern** — rendered per
   visible row via the existing `CharacterRenderer`, failing open to a
   placeholder. No appearance/equipment data is stored on the ranking row.

## 3. Backend — atlas-rankings

### 3.1 Store display fields on the ranking row

Thread three display fields end-to-end (name is the only genuinely new read;
level/jobId are already read but currently dropped after ranking):

- **`character.RestModel` / `character.Model`** (`character/rest.go`,
  `character/model.go`): add `Name string` (`json:"name"` — atlas-character's
  character resource already returns it). `Extract` populates it.
- **`Input`** (`ranking/compute.go`): add `Name string`. `Level` and `JobId`
  already present.
- **`Ranked`** (`ranking/compute.go`): add `Name string`, `Level byte`,
  `JobId job.Id`. `Rank()` passes them through from the sorted `Input` — the
  ranking math (level DESC / exp DESC / characterId ASC; jobId/100 category) is
  unchanged.
- **`Entity`** (`ranking/entity.go`): add columns `Name string`, `Level byte`,
  `JobId` (stored as the underlying integer). `JobCategory` already exists.
  `AutoMigrate` adds the columns; `Make` maps them onto the domain `Model`.
- **`upsertBatch` `DoUpdates`** (`ranking/administrator.go`): add `name`,
  `level`, `job_id` to the assignment column list so refreshes overwrite stale
  display values.
- **`Recompute`** (`ranking/processor.go`): populate `Name/Level/JobId` into
  `Input` and thence `Entity`.

No schema change to `ranking_cycles`. No change to the conflict key
(`tenant_id, character_id`) or prune logic.

### 3.2 New leaderboard list endpoint

```
GET /rankings?filter[worldId]=0[&filter[jobCategory]=1]&page[number]=1&page[size]=25
```

- **`filter[worldId]` required** (leaderboards are per-world). Missing/invalid
  worldId → `400`.
- **`filter[jobCategory]` optional.** Absent → overall view ordered by
  `overall_rank ASC`. Present → restricted to that category, ordered by
  `job_rank ASC`.
- **JSON:API paginated** using the same api2go pagination pattern as the
  task-117 `GET /characters` / `GET /tenants` list endpoints (`page[number]` /
  `page[size]`, page metadata in the response). A sensible default and max page
  size are enforced server-side.
- Served by the existing `idx_rankings_tenant_world` index.
- Tenant-scoped via the existing GORM query callback (context-bearing db
  handle) — identical to the current read paths.
- Empty world (no rows) → empty page with `200`, never `404`.

New pieces:
- **Provider** `byWorldPagedEntityProvider(worldId, jobCategory *uint16, offset, limit)`
  in `ranking/provider.go` (plus a count provider for pagination metadata).
- **List RestModel** (in `ranking/rest.go` or a sibling) carrying
  `characterId, name, level, jobId, jobCategory, rank, rankMove, jobRank,
  jobRankMove, computedAt`. The existing single-character `RestModel` is left
  as-is to avoid disturbing the login-decoration contract.
- **Handler** `handleGetLeaderboard` registered in `ranking/resource.go` under
  the `/rankings` subrouter (the bare-collection route; the existing
  `/rankings/characters...` routes are untouched).

### 3.3 What is NOT changed

- The login rank-decoration path (`/rankings/characters...`) and its RestModel.
- Ranking computation semantics.
- The `ranking_cycles` table and cadence/leader-election logic.

## 4. Frontend — atlas-ui

### 4.1 Page

A tenant-scoped page modeled on the existing tenant config pages
(`TenantsMtsConfigPage` as the structural template):

- **World selector** — worlds come from the active tenant (same source the
  existing `TenantsWorldsPage` uses); no new worlds endpoint.
- **Optional job-category filter** (beginner/warrior/magician/bowman/thief/
  pirate, mapping to `jobId/100`).
- **Pagination controls** bound to the endpoint's `page[number]`/`page[size]`.
- **Table rows:** rank number (the active view's rank — `rank` in the overall
  view, `jobRank` in a job-category view), **character render image**, name,
  level, job label, and an up/stay/down **movement arrow** driven by the sign of
  the active view's move field (`rankMove` overall, `jobRankMove` in a
  job-category view). Mirrors the in-game char-select rank board.

### 4.2 Data layer

- `rankings.service.ts` — JSON:API client for `GET /rankings` (list) following
  the existing service conventions (typed attributes, tenant headers, no raw
  fetch); reuses the established JSON:API envelope helpers.
- `useRankings` — TanStack React Query hook keyed by
  `(tenant, worldId, jobCategory, page)`.

### 4.3 Character image (progressive, fail-open)

Each visible row renders through the existing
`OptimizedCharacterRenderer`, which builds its loadout from a character +
equipped inventory (`characterToLoadout(character, inventory)`). The page
fetches that appearance data **per visible row** via the existing
character/inventory hooks (lazy — only rendered rows; the renderer already
memoizes and lazy-loads), and **fails open to a placeholder/skeleton** if a
render cannot load. The text leaderboard (name/level/rank from the stored
fields) is always meaningful even when an image lags or errors — exactly the
degrade-gracefully posture the accounts `CharactersPanel` already uses.

No appearance/equipment data is persisted on the ranking row.

## 5. Error Handling

- **Endpoint:** missing/invalid `filter[worldId]` → `400`; empty world → empty
  `200` page; tenant isolation enforced by the existing GORM callback.
- **Compute:** name/level/job are pure passthrough; a character missing a name
  (should not happen) stores an empty string — the row still ranks.
- **UI:** service/hook errors surface as a page-level error state; per-row image
  failures fall back to a placeholder without failing the row or the page.

## 6. Testing

- **compute / processor:** extend existing `compute_test.go` and
  `processor_test.go` fixtures to assert name/level/jobId flow through
  Input → Ranked → Entity and survive upsert; ranking order assertions
  unchanged.
- **leaderboard endpoint:** provider + handler tests — ordering (overall vs
  job-category), pagination (page boundaries, default/max size), worldId
  required, empty-world 200, tenant scoping.
- **UI:** `rankings.service` and `useRankings` tests (mock transport); page test
  for world/category/pagination interactions; image fail-open path tested like
  `CharactersPanel`.
- **Full backend verification** per CLAUDE.md: `go build`, `go vet`,
  `go test -race`, and `docker buildx bake atlas-rankings` (schema/columns +
  new route), plus `tools/lint.sh --check`.

## 7. Out of Scope

- Real-time / push updates (leaderboard freshness stays hourly-by-recompute).
- Storing appearance/equipment on the ranking row.
- A public (unauthenticated) leaderboard.
- Any change to the login rank-decoration path.
- A backend config-editor UI for the recompute interval (separate, previously
  deferred option).
