# Go 1.27 Migration — Pin Inventory

Machine-derived from `main` @ `855fef4d1` on 2026-08-25. Regenerate rather than
hand-edit. Target values: `GO_VERSION=1.27.0`, `ALPINE_VERSION=3.24`,
`GOLANGCI_LINT_VERSION=v2.13.1`.

This file is the authoritative worklist for prd.md FR-1 through FR-5 and the
checked set for the FR-6 guard.

## Summary

| Class | Sites | In scope |
|---|---|---|
| `go.mod` directives (non-fixture) | 103 | yes |
| `go.mod` directives (test fixtures) | 8 | **no** — FR-7 |
| Embedded fixture string in a `_test.go` | 1 | **no** — FR-7 |
| `go.work` | 1 | yes |
| Dockerfile ARGs | 4 | yes |
| `docker-bake.hcl` variables | 2 | yes |
| CI workflow / composite action pins | 6 | yes |
| Lint tool pin | 1 | yes |
| Dead Dockerfile (to delete) | 1 file, 2 pins | yes — FR-3.5 |

## A. Module `go` directives — 103 in scope

### A.1 Stragglers (not on the 1.25.5 majority) — 7 modules

| Module | Current |
|---|---|
| `libs/atlas-constants` | `1.24.4` |
| `libs/atlas-constants/gen` | `1.24.4` |
| `libs/atlas-retry` | `1.24.4` |
| `libs/atlas-script-core` | `1.24.4` |
| `tools/packet-audit` | `1.24` (minor-only form; normalize to patch-precise) |
| `tools/catalog-lint` | `1.25.0` |
| `services/atlas-data/atlas.com/data` | `1.26.0` (only module already ahead) |

### A.2 The 1.25.5 majority — 96 modules

`libs/` (19):

```
libs/atlas-database        libs/atlas-env             libs/atlas-kafka
libs/atlas-lock            libs/atlas-model           libs/atlas-object-id
libs/atlas-opcodes         libs/atlas-outbox          libs/atlas-packet
libs/atlas-redis           libs/atlas-rest            libs/atlas-routine
libs/atlas-saga            libs/atlas-seeder          libs/atlas-service
libs/atlas-socket          libs/atlas-tenant          libs/atlas-tracing
libs/atlas-wz
```

`services/` (67) — all `services/<name>/atlas.com/<module>` except
`services/atlas-kafka-precreate`, which is a flat module at the service root:

```
atlas-account/atlas.com/account                      atlas-asset-expiration/atlas.com/asset-expiration
atlas-ban/atlas.com/ban                              atlas-buddies/atlas.com/buddies
atlas-buffs/atlas.com/buffs                          atlas-cashshop/atlas.com/cashshop
atlas-chairs/atlas.com/chairs                        atlas-chalkboards/atlas.com/chalkboards
atlas-channel/atlas.com/channel                      atlas-character/atlas.com/character
atlas-character-factory/atlas.com/character-factory  atlas-configurations/atlas.com/configurations
atlas-consumables/atlas.com/consumables              atlas-doors/atlas.com/doors
atlas-dragons/atlas.com/dragons                      atlas-drop-information/atlas.com/dis
atlas-drops/atlas.com/drops                          atlas-effective-stats/atlas.com/effective-stats
atlas-events/atlas.com/events                        atlas-expressions/atlas.com/expressions
atlas-fame/atlas.com/fame                            atlas-families/atlas.com/family
atlas-guilds/atlas.com/guilds                        atlas-inventory/atlas.com/inventory
atlas-invites/atlas.com/invites                      atlas-kafka-precreate            <- flat module
atlas-keys/atlas.com/keys                            atlas-kites/atlas.com/kites
atlas-login/atlas.com/login                          atlas-map-actions/atlas.com/map-actions
atlas-maps/atlas.com/maps                            atlas-marriages/atlas.com/marriages
atlas-merchant/atlas.com/merchant                    atlas-messages/atlas.com/messages
atlas-messengers/atlas.com/messengers                atlas-mini-games/atlas.com/mini-games
atlas-monster-book/atlas.com/monster-book            atlas-monster-death/atlas.com/monster
atlas-monsters/atlas.com/monsters                    atlas-mounts/atlas.com/mounts
atlas-mts/atlas.com/mts                              atlas-notes/atlas.com/notes
atlas-npc-conversations/atlas.com/npc                atlas-npc-shops/atlas.com/npc
atlas-parcel/atlas.com/parcel                        atlas-parties/atlas.com/parties
atlas-party-quests/atlas.com/party-quests            atlas-pets/atlas.com/pets
atlas-portal-actions/atlas.com/portal                atlas-portals/atlas.com/portals
atlas-query-aggregator/atlas.com/query-aggregator    atlas-quest/atlas.com/quest
atlas-rankings/atlas.com/rankings                    atlas-rates/atlas.com/rates
atlas-reactor-actions/atlas.com/reactor              atlas-reactors/atlas.com/reactors
atlas-renders/atlas.com/renders                      atlas-reward-pools/atlas.com/reward-pools
atlas-rps/atlas.com/rps                              atlas-saga-orchestrator/atlas.com/saga-orchestrator
atlas-skills/atlas.com/skills                        atlas-storage/atlas.com/storage
atlas-summons/atlas.com/summons                      atlas-tenants/atlas.com/tenants
atlas-trades/atlas.com/trades                        atlas-transports/atlas.com/transports
atlas-world/atlas.com/world
```

`tools/` (10):

```
tools/atlasguards      tools/buffdurationguard  tools/cideps        tools/envguard
tools/goroutineguard   tools/outboxguard        tools/producerseamguard
tools/rediskeyguard    tools/scopeguard         tools/seed-splitters
```

### A.3 Workspace membership

`go.work` has 95 `use` entries. Eight in-scope modules sit **outside** the
workspace and are easy to miss with a workspace-only sweep:

```
tools/atlasguards   tools/buffdurationguard  tools/envguard    tools/goroutineguard
tools/outboxguard   tools/producerseamguard  tools/rediskeyguard  tools/scopeguard
```

(103 in-scope − 95 workspace members = 8. All are guard analyzer modules.)

## B. Excluded fixtures — do NOT bump (FR-7)

```
tools/cideps/testdata/simple/libs/lib-a/go.mod                        go 1.25.5
tools/cideps/testdata/simple/libs/lib-b/go.mod                        go 1.25.5
tools/cideps/testdata/simple/services/svc-a/atlas.com/svc-a/go.mod    go 1.25.5
tools/cideps/testdata/transitive/libs/lib-a/go.mod                    go 1.25.5
tools/cideps/testdata/transitive/libs/lib-b/go.mod                    go 1.25.5
tools/cideps/testdata/transitive/libs/lib-c/go.mod                    go 1.25.5
tools/cideps/testdata/transitive/services/svc-a/atlas.com/svc-a/go.mod  go 1.25.5
tools/cideps/testdata/transitive/services/svc-b/atlas.com/svc-b/go.mod  go 1.25.5
tools/cideps/graph_test.go:14                                         go 1.25.5  (embedded string)
```

## C. Non-module pin sites

| File | Line | Current | Target |
|---|---|---|---|
| `go.work` | 1 | `go 1.26.0` | `go 1.27.0` |
| `Dockerfile` | 17 | `ARG GO_VERSION=1.26.0` | `1.27.0` |
| `Dockerfile` | 18 | `ARG ALPINE_VERSION=3.23` | `3.24` |
| `Dockerfile` | 148 | `FROM alpine:3.24` | unchanged (already 3.24) |
| `services/atlas-kafka-precreate/Dockerfile` | 9 | `ARG GO_VERSION=1.26.0` | `1.27.0` |
| `services/atlas-kafka-precreate/Dockerfile` | 10 | `ARG ALPINE_VERSION=3.23` | `3.24` |
| `docker-bake.hcl` | 26 | `GO_VERSION` default `"1.26.0"` | `"1.27.0"` |
| `docker-bake.hcl` | 30 | `ALPINE_VERSION` default `"3.23"` | `"3.24"` |
| `.github/workflows/pr-validation.yml` | 41 | `GO_VERSION: '1.26.0'` | `'1.27.0'` |
| `.github/workflows/main-publish.yml` | 33 | `GO_VERSION: '1.26.0'` | `'1.27.0'` |
| `.github/workflows/catalog-lint.yml` | 17 | `GO_VERSION: '1.25.5'` | `'1.27.0'` |
| `.github/workflows/packet-matrix.yml` | 13 | `GO_VERSION: '1.25.5'` | `'1.27.0'` |
| `.github/actions/go-test/action.yml` | 11 | `default: '1.25.5'` | `'1.27.0'` |
| `.github/actions/detect-changes/action.yml` | 280 | `go-version: '1.26.0'` | `'1.27.0'` |
| `tools/lint.versions` | — | `GOLANGCI_LINT_VERSION=v2.12.2` | `v2.13.1` |

### C.1 Indirections — no edit needed

These resolve from the values above and must NOT be hand-edited:

```
.github/workflows/catalog-lint.yml:29     go-version: ${{ env.GO_VERSION }}
.github/workflows/packet-matrix.yml:25    go-version: ${{ env.GO_VERSION }}
.github/workflows/pr-validation.yml:149   go-version: ${{ env.GO_VERSION }}
.github/workflows/pr-validation.yml:424   go-version: ${{ env.GO_VERSION }}
.github/workflows/pr-validation.yml:460   go-version: ${{ env.GO_VERSION }}
.github/workflows/pr-validation.yml:668   go-version: ${{ env.GO_VERSION }}
.github/workflows/pr-validation.yml:694   go-version: ${{ env.GO_VERSION }}
.github/actions/go-test/action.yml:40     go-version: ${{ inputs.go-version }}
Dockerfile:20                             FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION}
services/atlas-kafka-precreate/Dockerfile:12  FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION}
docker-bake.hcl:115-116                   args = { GO_VERSION, ALPINE_VERSION }
```

## D. Dead file to delete (FR-3.5)

`services/atlas-renders/Dockerfile` — two stale pins:

- line 1: `FROM golang:1.25.5-alpine3.21 AS build-env`
- line 17: `RUN echo 'go 1.25.5' > go.work && \` (synthesized workspace)

It is not built. `atlas-renders` is listed in `docker-bake.hcl:93` `go_services`,
so it is produced by the matrix `go-service` target (`docker-bake.hcl:106-120`),
which sets `dockerfile = "Dockerfile"` against `context = "."` — the repo-root
Dockerfile. `.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md:169`
already records it as "dead, not an exception to the rule."

Contrast `services/atlas-kafka-precreate/Dockerfile`, which **is** live: its bake
target (`docker-bake.hcl:143-148`) sets `context = "services/atlas-kafka-precreate"`,
so `dockerfile = "Dockerfile"` resolves to the service's own file.

## E. Upstream availability — verified 2026-08-25

- `go1.27.0` is `"stable": true` in `https://go.dev/dl/?mode=json&include=all`.
- Docker Hub `library/golang` publishes `1.27.0-alpine3.23` and `1.27.0-alpine3.24`
  (also `1.27.0`, `1.27.0-trixie`, `1.27.0-bookworm`).
- Docker Hub `library/alpine` newest minor is `3.24` (current patch `3.24.1`).
- golangci-lint newest release is `v2.13.1`. Its `go.mod` declares `go 1.26.0`;
  `v2.12.2`'s declares `go 1.25.0`. Upstream comment in both: *"The minimum Go
  version must always be latest-1."* Go 1.27 support is **inferred from this
  convention, not quoted from release notes** — the release-notes fetch was
  rate-limited. prd.md FR-5.2 resolves it by execution.
