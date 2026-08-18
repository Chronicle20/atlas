# bug: event definitions are never seeded, and the Setup UI offers no way to seed them

Task: task-231-generalized-events-service
Branch: task-231-generalized-events-service
Reported against: ephemeral environment `atlas-pr-1375`

## Reproduced

Yes, against the live ephemeral namespace `atlas-pr-1375` (pod
`atlas-events-878f8676c-j6kw5`, 2/2 Running — app container plus the git-sync
seed-catalog sidecar).

## Observed

- `Event Definitions` page renders its empty state: no definitions exist for
  the tenant.
- The Setup page (`services/atlas-ui/src/pages/SetupPage.tsx`) has twelve seed
  rows — Monster & Reactor Drops, Reward Pools, NPC/Quest/Item Conversations,
  NPC Shops, Portal/Reactor/Map Action Scripts, Transport Routes, Transport
  Vessels, Instance Transport Routes — and **no row for event definitions**.
  There is therefore no operator-reachable trigger for the seed.
- `services/atlas-ui/src/services/api/seed.service.ts` has no
  `seedEventDefinitions` / `getEventDefinitionsSeedStatus`;
  `services/atlas-ui/src/lib/hooks/api/useSeed.ts` has no corresponding
  mutation/status hook or status key.

## Expected

A deployment operator can seed event definitions for the selected tenant from
the Setup page, the same way every other seeded subdomain is seeded, and sees a
count badge reflecting what landed.

## Root cause

The backend seed path is complete and correct; only the trigger is missing.

- `services/atlas-events/atlas.com/events/event/definition/subdomain.go:96-114`
  registers the seeder group `events` at URL prefix `/events/definitions`, so
  with the service base path the endpoints are
  `POST /api/events/definitions/seed` and
  `GET /api/events/definitions/seed/status`. Group name `events`, single
  subdomain `Name() == "definition.event"`.
- `deploy/shared/routes.conf:250-253` proxies `^/api/events(/.*)?$` to
  `atlas-events:8080`, so the endpoints are reachable through the gateway (the
  already-working Event Definitions page uses `/api/events/definitions` on the
  same route).
- `deploy/k8s/base/atlas-events.yaml:7` carries `atlas.seed-catalog: "true"`,
  so the git-sync sidecar is patched in, and the PR overlay pins
  `GITSYNC_REF` to the PR SHA
  (`deploy/k8s/overlays/pr/patches/seed-catalog-ref.yaml`). Verified in the
  live pod: `/var/run/seed-catalog/catalog/deploy/seed/shared/all/events/definitions`
  contains `event-anniversary.json` and `event-crimson-balrog.json`.

So the catalog is present, the endpoint is live, and the route resolves — but
**nothing ever calls `POST /api/events/definitions/seed`**:

1. `services/atlas-events/atlas.com/events/main.go:64-133` performs no seed at
   startup. It registers `definition.InitSeedResource` and nothing more; there
   is no auto-seed on boot in this service (nor in any peer seeded service —
   seeding is deliberately operator-driven).
2. The Setup UI, which is the operator's only seeding surface, was never given
   an Event Definitions row.

A secondary defect follows from the same gap: the Event Definitions page's
empty-state copy asserts a behavior that does not exist —
`services/atlas-ui/src/pages/EventDefinitionsPage.tsx:129-130` says
"Event definitions are seeded by atlas-events at startup." Nothing seeds at
startup; the correct instruction is to seed from the Setup page.

The fix is UI-only: add the seed row, matching the existing operator-driven
model rather than introducing a boot-time auto-seed that no other seeded
service has.

## Fix

Follow the Instance Transport Routes / Transport Routes shape, **not** the
Map Action Scripts shape — `libs/atlas-seeder`'s `postSeed`
(`libs/atlas-seeder/handlers.go:47-79`) answers `202 Accepted` with no body and
seeds in a background goroutine, so the mutation resolves to `void` and the
Setup page's 5s status poll is what surfaces the result. `seedTransportRoutes`
et al. already encode exactly this; `seedMapActionScripts` returning
`SeedResult` is the wrong template here.

- `services/atlas-ui/src/services/api/seed.service.ts`
  - Add `EventDefinitionsSeedStatus { definitionCount: number; updatedAt: string | null }`
    beside the other `*SeedStatus` interfaces.
  - Add `seedEventDefinitions(): Promise<void>` →
    `api.post("/api/events/definitions/seed", {})`, placed with the other
    202/void seeds.
  - Add `getEventDefinitionsSeedStatus(tenant: Tenant)` →
    `fetchSeedStatus("/api/events/definitions/seed/status", tenant)`, returning
    `{ definitionCount: subdomainCount(s, "definition.event"), updatedAt: s.tenantSeededAt ?? s.updatedAt }`.
    The subdomain key is `definition.event` — see `subdomain.go:27`.
- `services/atlas-ui/src/lib/hooks/api/useSeed.ts`
  - Add `eventDefinitionsSeedStatusKey(tenantId)` alongside the other key
    helpers (lines ~44-71).
  - Add `useSeedEventDefinitions()` (mutation, invalidates that key on success)
    and `useEventDefinitionsSeedStatus()` (query, `staleTime: 0`,
    `refetchInterval: 5000`), mirroring `useSeedInstanceRoutes` /
    `useInstanceRoutesSeedStatus`.
  - Export the new `EventDefinitionsSeedStatus` type import at the top.
- `services/atlas-ui/src/pages/SetupPage.tsx`
  - Import the two new hooks, call them beside the existing ones (~lines 85,
    105), and append a row `label: "Event Definitions"` with a suitable
    lucide icon (e.g. `CalendarClock`) and a badge formatted as
    `${formatCount(d.definitionCount)} ${pluralize(d.definitionCount, "definition", "definitions")}`.
- `services/atlas-ui/src/pages/EventDefinitionsPage.tsx`
  - Correct the empty-state description: event definitions are seeded from the
    Setup page, not at service startup.
- Tests
  - `services/atlas-ui/src/lib/hooks/api/__tests__/useSeed.test.tsx` — extend
    for the new mutation/status hook following the existing per-subdomain cases.
  - `services/atlas-ui/src/pages/__tests__/SetupPage.test.tsx` — assert the new
    row renders and its seed control posts.
  - `services/atlas-ui/src/pages/__tests__/EventDefinitionsPage.test.tsx` —
    update any assertion pinned to the old empty-state copy.

No Go change is required. Do not add a startup auto-seed.

## Not yet answered

- `atlas-party-quests` also registers a seeder group and likewise has no Setup
  row. Same class of gap, different task's surface — out of scope here, noted
  so it is not rediscovered.

## Resolution

_(pending)_
