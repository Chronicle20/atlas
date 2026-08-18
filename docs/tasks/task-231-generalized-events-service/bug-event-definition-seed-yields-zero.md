# bug: the Setup seed button runs but seeds zero event definitions

Task: task-231-generalized-events-service
Branch: task-231-generalized-events-service
Reported against: ephemeral environment `atlas-pr-1375`
Follow-on to: `bug-no-event-definition-seed-control.md` (fixed in `03e870091`)

## Reproduced

Yes, live against `atlas-pr-1375` running the fixed UI image
`ghcr.io/chronicle20/atlas-ui/atlas-ui:pr-1375-f4e62a9`.

## Observed

The control added in `03e870091` works end to end at the transport layer — it
is *not* an inert button:

- Ingress access log: `POST /api/events/definitions/seed HTTP/1.1" 202` at
  11:51:38, referer `/setup`.
- atlas-events log, same second:
  `"message":"Seed complete","group_name":"events","subdomains":{"definition.event":0},"catalog_revision":"b7d3218a6…+b7d3218a6…"`
- Status endpoint afterwards:
  `{"groupName":"events","subdomains":{"definition.event":{"count":0,…}},"tenantSeededAt":"2026-08-18T11:51:38.822075Z"}`

So the seed ran, claimed success, and created **zero** rows. The catalog is
present in the pod — `/var/run/seed-catalog/catalog/deploy/seed/shared/all/events/definitions`
holds `event-anniversary.json` and `event-crimson-balrog.json`, and
`SEED_CATALOG_ROOT=/var/run/seed-catalog/catalog/deploy/seed` is correct.

## Expected

Two definitions created, badge reads "2 definitions", and the Event Definitions
page lists Anniversary and Crimson Balrog (both disabled).

## Root cause

**The two event-definition seed files are malformed: they carry no `data.id`
and no `data.type`.**

`libs/atlas-seeder/seed.go:157-171` (`loadOne`) validates the JSON:API envelope
before handing the payload to the subdomain:

- `env.Data.Type != sd.Type()` → the files have `""`, `DefinitionSubdomain.Type()`
  is `"event-definition"` → **type mismatch**, the file is rejected.
- `ExtractEntityID(filename, ^event-(.+)\.json$)` vs `env.Data.ID` → `"anniversary"`
  vs `""` → id mismatch (unreached; the type check fails first).

Both files therefore land in `counts.Failed`, never `counts.Created`.

An audit of the whole catalog confirms these two files are the only offenders:
of 22,702 seed JSON files under `deploy/seed/`, 22,700 carry both `data.id` and
`data.type`; the two exceptions are
`deploy/seed/shared/all/events/definitions/*.json`. Every party-quest file is
well-formed (e.g. `party-quest-henesys_pq.json` → id `henesys_pq`, type
`party-quest-definition`).

### Why CI did not catch it

`tools/catalog-lint` checks exactly this invariant — `main.go:87-98` compares
`data.type` against the rule's type and `data.id` against the filename-derived
id. But `tools/catalog-lint/subdomains.go`'s `rules` table has **no entry for
`events/definitions`**, and `ruleFor` returning false makes the walker skip the
file as an "unrecognized subdomain — not an error per se" (`main.go:74-76`).
task-231 introduced a new seeded subdomain without registering it with the
linter, so the malformed envelope shipped unchallenged.

### Why the failure was invisible at runtime

`libs/atlas-seeder/handlers.go:102-108` (`summarize`) projects each subdomain to
`v.Created` only, dropping `Failed` and `Errors`. `postSeed` then logs
`"Seed complete"` at info level whenever `Seed()` returns a nil error — and
`Seed()` returns nil even when `classifyOutcome` (`seed.go:212-227`) computes
`"failure"`. A run in which every file was rejected is indistinguishable in the
logs from a run of an empty catalog.

### Separately: nothing seeds events on environment launch

`services/atlas-pr-bootstrap/scripts/bootstrap.sh:481-490` posts nine seed
endpoints in parallel at environment bootstrap. `/api/events/definitions/seed`
is absent, as is `/api/party-quests/definitions/seed`. This is the other half of
the original report ("on launch … there are no event definitions seeded") — even
with well-formed files, a fresh environment would still come up empty until an
operator clicked Setup.

## Fix

### 1. Repair the seed envelopes (the actual defect)

- `deploy/seed/shared/all/events/definitions/event-anniversary.json` — add
  `"id": "anniversary"` and `"type": "event-definition"` as siblings of
  `attributes` inside `data`.
- `deploy/seed/shared/all/events/definitions/event-crimson-balrog.json` — add
  `"id": "crimson-balrog"` and `"type": "event-definition"` likewise.

The ids are fixed by `DefinitionSubdomain.EntityIDPattern()`
(`^event-(.+)\.json$`) applied to the filenames; the type is fixed by
`DefinitionSubdomain.Type()`. Do not touch any `CATALOG_REVISION` file — CI
stamps those (`.github/workflows/main-publish.yml:284`).

### 2. Close the linter gap so this class cannot recur

- `tools/catalog-lint/subdomains.go` — add to `rules`:
  `{path: "events/definitions", typ: "event-definition", pattern: regexp.MustCompile(`^event-(.+)\.json$`)}`.
  Place it beside the existing `party-quests/definitions` entry.
  After this, `go run ./tools/catalog-lint deploy/seed` must exit 0 — run it and
  show the output.

### 3. Seed events and party-quests at environment bootstrap

- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` — add
  `/api/events/definitions/seed` and `/api/party-quests/definitions/seed` to the
  endpoint list at lines 481-490. Both routes already exist
  (`deploy/shared/routes.conf:21` and `:250`) and both services are deployed.

### 4. Surface seed failures instead of swallowing them

- `libs/atlas-seeder/handlers.go` — in `postSeed`, log the run's outcome. Include
  per-subdomain `Failed` counts and `Errors` alongside `Created` (extend
  `summarize`, or add the failure detail as separate log fields), and log at
  error level when `classifyOutcome(res.Subdomains) != "success"` rather than
  unconditionally at info with the message "Seed complete". Keep the existing
  success-path message and fields intact so the nine existing consumers' log
  shape is unchanged on a clean run. Add a unit test covering the
  all-files-rejected case.

### 5. Party-quests Setup row (explicitly requested; same shape as `03e870091`)

`atlas-party-quests` registers seeder group `party-quests` at URL prefix
`/party-quests/definitions` (`services/atlas-party-quests/atlas.com/party-quests/definition/groups.go`)
with a single subdomain whose `Name()` is `definition.partyquest`
(`definition/subdomain.go:21`). It has no Setup row, same as event definitions
did. Mirror the event-definitions wiring exactly:

- `services/atlas-ui/src/services/api/seed.service.ts` — add
  `PartyQuestDefinitionsSeedStatus { definitionCount: number; updatedAt: string | null }`,
  `seedPartyQuestDefinitions(): Promise<void>` posting
  `/api/party-quests/definitions/seed` (202/void shape), and
  `getPartyQuestDefinitionsSeedStatus(tenant)` reading
  `/api/party-quests/definitions/seed/status` with
  `subdomainCount(s, "definition.partyquest")`.
- `services/atlas-ui/src/lib/hooks/api/useSeed.ts` — status key, mutation hook,
  and status query hook (`staleTime: 0`, `refetchInterval: 5000`).
- `services/atlas-ui/src/pages/SetupPage.tsx` — a `label: "Party Quest Definitions"`
  row with a suitable lucide icon.
- Extend `__tests__/useSeed.test.tsx` and `__tests__/SetupPage.test.tsx`.

## Not yet answered

- The bootstrap script also omits the three atlas-tenants configuration seeds
  (`/api/tenants/configurations/{routes,vessels,instance-routes}/seed`), which
  an operator currently has to click by hand. Pre-existing and outside this
  bug's blast radius — recorded so it is not rediscovered, not fixed here.

## Resolution

Fixed, all five parts:

1. `deploy/seed/shared/all/events/definitions/event-anniversary.json` and
   `event-crimson-balrog.json` now carry `data.id` / `data.type`
   (`anniversary` / `event-definition` and `crimson-balrog` /
   `event-definition`).
2. `tools/catalog-lint/subdomains.go` gained an `events/definitions` rule;
   `go run ./tools/catalog-lint deploy/seed` exits 0.
3. `services/atlas-pr-bootstrap/scripts/bootstrap.sh` now posts
   `/api/events/definitions/seed` and `/api/party-quests/definitions/seed`
   alongside the other nine bootstrap seeds.
4. `libs/atlas-seeder/handlers.go`'s `postSeed` now logs "Seed complete" at
   error level with `outcome`, `failed`, and `errors` fields whenever
   `classifyOutcome != "success"`, while the clean-run info-level log shape
   is unchanged. Covered by
   `TestRegisterRoutes_AllFilesRejectedLogsError` in `handlers_test.go`.
5. `atlas-ui` gained a "Party Quest Definitions" Setup row, mirroring the
   event-definitions wiring in `seed.service.ts`, `useSeed.ts`, and
   `SetupPage.tsx`, with matching test coverage.
