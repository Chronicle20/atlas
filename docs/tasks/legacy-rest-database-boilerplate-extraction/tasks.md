# REST/Database Boilerplate Extraction — Tasks

Last Updated: 2026-02-19

## Phase 1: Extract request.go into atlas-rest (Effort: S) — COMPLETE

- [x] **1.1** Add decorated request functions to `libs/atlas-rest/requests/decorated.go`
  - `GetRequest[A](url) Request[A]` — wraps MakeGetRequest with Span+Tenant headers
  - `PostRequest[A](url, body) Request[A]` — wraps MakePostRequest with Span+Tenant headers
  - `PatchRequest[A](url, body) Request[A]` — wraps MakePatchRequest with Span+Tenant headers
  - `DeleteRequest(url) EmptyBodyRequest` — wraps MakeDeleteRequest with Span+Tenant headers
  - `PutRequest[A](url, body) Request[A]` — wraps MakePutRequest with Span+Tenant headers
- [ ] **1.2** Add unit tests for decorated request functions
- [x] **1.3** Pilot: Migrate 5 simple services to use decorated requests, delete local request.go
- [x] **1.4** Migrate remaining 42 services, delete all local request.go files + 3 outlier inline callers

## Phase 2: Extract handler types into atlas-rest (Effort: M) — COMPLETE

- [x] **2.1** Add to `libs/atlas-rest/server/context.go`:
  - `HandlerContext` struct with `ServerInformation()` accessor
  - `GetHandler` type: `func(d *HandlerDependency, c *HandlerContext) http.HandlerFunc`
  - `InputHandler[M]` type: `func(d *HandlerDependency, c *HandlerContext, model M) http.HandlerFunc`
  - `ParseInput[M]()` function
- [x] **2.2** Add `HandlerDependency` struct (no-DB variant) with `Logger()` and `Context()` methods
  - Also added `NewHandlerDependency()` and `NewHandlerContext()` constructors
- [x] **2.3** Verify atlas-rest go.mod has NO gorm dependency
- [ ] **2.4** Add unit tests for handler types and ParseInput

## Phase 3: Extract RegisterHandler into atlas-rest (Effort: M) — COMPLETE (non-DB variants)

- [ ] **3.1** Design DB-aware handler variant (DEFERRED to Phase 6)
  - Option C recommended: `libs/atlas-rest/dbserver/` sub-package with gorm dependency
  - `dbserver.HandlerDependency` extends with `DB()` method
  - `dbserver.RegisterHandler(l)(db)(si)(name, handler)` — with tenant
  - `dbserver.RegisterInputHandler(l)(db)(si)(name, handler)` — with tenant
- [x] **3.2** Implement `server.RegisterHandler(l)(si)(name, handler)` — no DB, with tenant
- [x] **3.3** Implement `server.RegisterSimpleHandler(l)(si)(name, handler)` — no DB, no tenant
- [x] **3.4** Implement `server.RegisterInputHandler(l)(si)(name, handler)` — no DB, with tenant
- [x] **3.5** Implement `server.RegisterSimpleInputHandler(l)(si)(name, handler)` — no DB, no tenant
- [ ] **3.6** Implement `dbserver.RegisterHandler(l)(db)(si)(name, handler)` — with DB, with tenant (DEFERRED)
- [ ] **3.7** Implement `dbserver.RegisterInputHandler(l)(db)(si)(name, handler)` — with DB, with tenant (DEFERRED)
- [ ] **3.8** Add unit tests for all RegisterHandler variants

## Phase 4: Extract generic ID parsers (Effort: M) — COMPLETE

- [x] **4.1** Implement in `libs/atlas-rest/server/id_parser.go`:
  - `ParseIntId[T IntegerId](l, varName, next func(T) http.HandlerFunc) http.HandlerFunc`
  - `ParseUUIDId(l, varName, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc`
  - `ParseStringId(l, varName, next func(string) http.HandlerFunc) http.HandlerFunc`
  - Type constraint: `IntegerId interface { ~uint32 | ~int32 | ~int8 | ~uint8 | ~uint16 }`
- [ ] **4.2** Add unit tests with gorilla/mux test router

## Phase 2-4 Migration: Migrate non-DB services to shared types — COMPLETE

**32 services migrated** (type aliases + delegated ID parsers):
- [x] atlas-buddies (pilot, with tenant)
- [x] atlas-configurations (pilot, no tenant — uses RegisterSimpleHandler)
- [x] atlas-asset-expiration
- [x] atlas-buffs
- [x] atlas-cashshop
- [x] atlas-chairs
- [x] atlas-chalkboards
- [x] atlas-character-factory
- [x] atlas-consumables
- [x] atlas-data
- [x] atlas-drops
- [x] atlas-effective-stats
- [x] atlas-families
- [x] atlas-guilds
- [x] atlas-inventory
- [x] atlas-invites
- [x] atlas-keys
- [x] atlas-maps
- [x] atlas-marriages
- [x] atlas-messengers
- [x] atlas-monsters
- [x] atlas-parties
- [x] atlas-portals (custom ParseCharacterId — query param fallback)
- [x] atlas-query-aggregator
- [x] atlas-rates
- [x] atlas-reactors
- [x] atlas-saga-orchestrator
- [x] atlas-skills
- [x] atlas-storage (custom ParseWorldId — query params)
- [x] atlas-tenants (no tenant — uses RegisterSimpleHandler)
- [x] atlas-transports
- [x] atlas-world
- [x] atlas-wz-extractor

**Services without rest/handler.go** (skipped — no migration needed):
- atlas-channel
- atlas-expressions
- atlas-login
- atlas-messages

**15 DB-dependent services** (deferred — `*gorm.DB` in HandlerDependency):
- atlas-account
- atlas-ban
- atlas-character
- atlas-drop-information
- atlas-fame
- atlas-gachapons
- atlas-map-actions
- atlas-notes
- atlas-npc-conversations
- atlas-npc-shops
- atlas-party-quests
- atlas-pets
- atlas-portal-actions
- atlas-quest
- atlas-reactor-actions

## Phase 5: Migrate services to atlas-database (Effort: L) — COMPLETE

- [x] **5.1** Audit atlas-database for missing helpers
  - [x] Added `Query[E]` (19 services used it)
  - [x] Added `SliceQuery[E]` (19 services used it)
  - [x] Added `FoldModelProvider` (5 services used it)
  - [x] Added `Teardown` (1 service used it — atlas-ban)
- [x] **5.2** Pilot: Migrate 3 services to atlas-database
  - [x] atlas-fame — already using library, deleted dead database/ dir
  - [x] atlas-ban — replaced import, deleted database/ (with Teardown)
  - [x] atlas-keys — replaced import, deleted database/ (with Query/SliceQuery)
- [x] **5.3** Migrate remaining services (25 total, 3 batches)
  - All 27 database/ directories deleted
  - 125 files now import `database "github.com/Chronicle20/atlas-database"`
  - 0 remaining local database imports
- [x] **5.4** Alternate-pattern services migrated cleanly
  - atlas-map-actions — EntityProvider/SliceProvider functions were dead code, deleted
  - atlas-reactor-actions — EntityProvider/SliceProvider functions were dead code, deleted
  - atlas-portal-actions — standard pattern, migrated normally

## Phase 6: Full REST Migration for DB services (Effort: XL)

- [ ] **6.1** Design DB-aware handler variant in `libs/atlas-rest/dbserver/`
- [ ] **6.2** Migrate 15 DB-dependent services to shared handler types
- [ ] **6.3** Final cleanup: remove dead imports, unused functions
- [ ] **6.4** Full workspace build + test validation

## Verification Checklist (per service)

For each migrated service:
- [ ] `go test ./... -count=1` passes
- [ ] `go build` succeeds
- [ ] No local `rest/request.go` (Phase 1+)
- [ ] No local `database/connection.go` (Phase 5+, if applicable)
- [ ] No local `retry/retry.go` (Phase 5+, if applicable)
- [ ] No local `database/transaction.go` (Phase 5+, if applicable)
- [ ] Handler registration uses atlas-rest imports (Phase 2-4+)
