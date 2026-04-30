# libs/atlas-tracing migration runbook

This is the per-service procedure used to swap each service from its private `tracing/tracing.go` to `libs/atlas-tracing`. atlas-channel was migrated as the canary in Task 5; this is the same procedure repeated for the remaining services.

## Prereqs

- `libs/atlas-tracing` exists, builds, and tests pass.
- atlas-channel migration is merged (canary verifies the pattern works end-to-end).
- `go.work` includes `./libs/atlas-tracing`.

## Per-service edits

For each service `<svc>` (path: `services/atlas-<svc>/atlas.com/<dir>/`):

### 1. `go.mod`

Add to the `require (` block (alongside other `github.com/Chronicle20/atlas/libs/...` entries):

```
github.com/Chronicle20/atlas/libs/atlas-tracing v0.0.0
```

Add a replace at the bottom:

```
replace github.com/Chronicle20/atlas/libs/atlas-tracing => ../../../../libs/atlas-tracing
```

(Path is `../../../../libs/atlas-tracing` for every service — they all live four levels deep under `services/atlas-*/atlas.com/<dir>/`.)

### 2. `main.go` import swap

Find:

```go
"<svc-module>/tracing"
```

Replace with:

```go
tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
```

Call sites for `tracing.InitTracer` and `tracing.Teardown` are unchanged.

### 3. Delete the old package

```
rm -rf services/atlas-<svc>/atlas.com/<dir>/tracing
```

### 4. `Dockerfile` — four line additions

Each Atlas service's Dockerfile follows the same shape. Four additions are needed in different blocks:

(a) `COPY libs/atlas-tracing/go.mod libs/atlas-tracing/go.sum libs/atlas-tracing/`
    after the equivalent line for `libs/atlas-tenant` in the "Copy library module definitions" block.

(b) `echo '    ./libs/atlas-tracing' >> go.work && \`
    inside the inline `go.work` generation block, alphabetically positioned before `./services/...`.

(c) `COPY libs/atlas-tracing libs/atlas-tracing`
    in the "Copy library source code" block.

(d) `-replace=github.com/Chronicle20/atlas/libs/atlas-tracing=/app/libs/atlas-tracing`
    in the `RUN go mod edit` block alongside the other `-replace=` arguments for shared libs (e.g. `atlas-tenant`, `atlas-socket`). This rewrites the relative replace path used in local development to the absolute `/app/...` path the container build needs.

> **Note on (d):** the original Task 5 plan only listed three Dockerfile edits and missed this one. Without (d), `go mod tidy` inside the container fails because the relative replace path (`../../../../libs/atlas-tracing`) does not resolve from the `/app` working directory. Verify against `services/atlas-channel/Dockerfile` (the canary) for the canonical pattern.

### 5. Per-service verification

```
cd services/atlas-<svc>/atlas.com/<dir>
go mod tidy
go build ./...
go test ./...
docker build -f services/atlas-<svc>/Dockerfile -t atlas-<svc>:task040 ../../../..
```

All four must succeed. If `docker build` fails, check the four Dockerfile insertions before anything else — almost every failure mode is a missed COPY line, missed `-replace`, or wrong indentation in the inline `go.work`.

## Service list

The remaining services (atlas-channel migrated in Task 5; atlas-account migrated in Task 15):

```
atlas-asset-expiration
atlas-ban
atlas-buddies
atlas-buffs
atlas-cashshop
atlas-chairs
atlas-chalkboards
atlas-character
atlas-character-factory
atlas-configurations
atlas-consumables
atlas-data
atlas-drop-information
atlas-drops
atlas-effective-stats
atlas-expressions
atlas-fame
atlas-families
atlas-gachapons
atlas-guilds
atlas-inventory
atlas-invites
atlas-keys
atlas-login
atlas-map-actions
atlas-maps
atlas-marriages
atlas-merchant
atlas-messages
atlas-messengers
atlas-monster-death
atlas-monsters
atlas-notes
atlas-npc-conversations
atlas-npc-shops
atlas-parties
atlas-party-quests
atlas-pets
atlas-portal-actions
atlas-portals
atlas-query-aggregator
atlas-quest
atlas-rates
atlas-reactor-actions
atlas-reactors
atlas-saga-orchestrator
atlas-skills
atlas-storage
atlas-tenants
atlas-transports
atlas-world
atlas-wz-extractor
```

> Note: a `services/atlas-character/atlas.com/character/tracing/` exists today even though atlas-character is being deprecated; migrate it anyway for parity unless the team decides otherwise during Task 16 scoping.

> Also note: not every Atlas service has its own `tracing/tracing.go`. Before migrating a service, run `find services/atlas-<svc> -name tracing.go -path '*/tracing/*'`. If the result is empty, that service did not call `tracing.InitTracer` to begin with — skip the source edit and the `tracing/` deletion, but still apply the Dockerfile edits so any future `tracing.InitTracer` call Just Works.
