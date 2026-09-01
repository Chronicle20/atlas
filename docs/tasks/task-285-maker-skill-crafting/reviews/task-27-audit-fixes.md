# Review — task-27 backend-audit fix rounds A + B

- **Range:** `3877d5047..8ada1d8ba` (`e25600ea5` round B, `8ada1d8ba` round A)
- **Briefs:** `.superpowers/sdd/plan/task-27-brief-fix2a.md`, `.superpowers/sdd/plan/task-27-brief-fix2b.md`
- **Reports:** `.superpowers/sdd/plan/task-27-report-fix2a.md`, `.superpowers/sdd/plan/task-27-report-fix2b.md`
- **Source audit:** `docs/tasks/task-285-maker-skill-crafting/reviews/audit.md` (7 blocking)

## Scope

Reviewed the diff of both commits in full (`git diff --stat 3877d5047..8ada1d8ba`, 41 files,
+879/-189), read every touched `.go` file's hunks, plus the library reference
(`libs/atlas-rest/server/register.go`) that round B's fix must match. Re-ran
`go build`/`go test`/`lint.sh`/`service-registration-guard.sh` myself over the merged HEAD
rather than trusting either round's self-report, per the task instructions. Did not run
`tools/verify.sh` (in flight in the controller session per instructions).

## 1. DOM-32 — `rest/handler.go` ParseEnvironment placement

**CLOSED.** Diff (`git diff 3877d5047..8ada1d8ba -- services/atlas-maker/atlas.com/maker/rest/handler.go`):

```
-  return server.ParseTenant(fl, sctx, func(tl ...) {
-    return handler(&HandlerDependency{l: tl, db: db, ctx: tctx}, ...)
+  return server.ParseEnvironment(fl, sctx, func(el ...) {
+    return server.ParseTenant(el, ectx, func(tl ...) {
+      return handler(&HandlerDependency{l: tl, db: db, ctx: tctx}, ...)
+    })
   })
```

Applied identically to both `RegisterHandler` (`rest/handler.go:45-63`, `ParseEnvironment` now at
line 51-52) and `RegisterInputHandler[M]` (`rest/handler.go:109-125`, `ParseEnvironment` now at
line 115-116). Compared side-by-side against `libs/atlas-rest/server/register.go:16-18` and
`:31-33` — the `RetrieveSpan → ParseEnvironment → ParseTenant` sequencing and the field-logger
threading (`fl`→`el`→`tl`) match the canonical implementation exactly.

`HandlerDependency` at `rest/handler.go:18-22` still carries `db *gorm.DB` and its `DB()` accessor
(`:28-30`); the local wrapper was not deleted or replaced with the library function, matching the
brief's explicit "do NOT delete them in favor of the library functions" instruction. `db` is still
threaded into every `&HandlerDependency{l: tl, db: db, ctx: tctx}` construction in both functions.

## 2. Merged-tree build/test/lint (both rounds together)

Round A's report notes it observed a transient `reagent/processor.go: undefined: Make` while round
B's build ran concurrently against round A's in-progress edit — a real hazard the report itself
flags and resolves by re-running before citing evidence. I re-ran all three gates myself from a
clean HEAD at `8ada1d8ba` (both commits merged), not trusting either self-report:

```
cd services/atlas-maker/atlas.com/maker && go build ./...        → exit 0, no output
cd services/atlas-maker/atlas.com/maker && go test ./... -count=1
```
All packages `ok` or `[no test files]`: `atlas-maker`, `character`, `compartment`, `craft`,
`crystalband`, `data/equipment`, `data/itemmake`, `kafka/consumer/saga`, `quest`, `reagent`,
`recipe`, `seed`, `skill`.

```
tools/lint.sh --go services/atlas-maker/atlas.com/maker → "0 issues." / "lint.sh: OK"
tools/service-registration-guard.sh → "service-registration-guard: clean"
```

**CLOSED** — merged result is clean under all four gates, run directly rather than accepted from
either round's report.

## 3. Round A symbol moves — no test regression, name collision preserved

`git diff 3877d5047..8ada1d8ba --stat -- '*_test.go'` returns **no output** — neither commit
touched a single `_test.go` file. That is the strongest possible evidence that no test was
deleted, skipped, or weakened to accommodate a move: the test files are byte-identical, and (per
gate 2 above) every one of them still passes against the refactored production code.

Per-move verification:

- **`craft/eligibility.go` → `craft/processor.go`:** confirmed `Processor` interface
  (`processor.go:91`), `ProcessorImpl` struct, `NewProcessor` (`processor.go:135`), and
  `NewSnapshot` method (`processor.go:142`) all now live in `processor.go`; `eligibility.go` keeps
  only `Reason`, `Eligibility`, `eligible`/`ineligible`, and the eligibility evaluation logic
  (`eligibility.go:1-40` read in full — no processor scaffolding remains). `craft/processor.go`
  also still holds `Create`/`create`/`createOrUpgrade` (`processor.go:169,189,206`) — the historical
  "Create" hazard the brief called out is a **method** on `*ProcessorImpl`, not a package-level
  symbol, so merging it into the same file as the interface/constructor introduces no collision;
  confirmed by the clean `go build`.
- **`crystalband`/`reagent` `provider.go` → `entity.go`:** `crystalband/entity.go:40-47` now
  defines `func Make(e entity) (Model, error)` and `:51-59` defines `func (m Model) ToEntity()
  entity`, both moved out of `provider.go` (now readers-only: `getAllPagedProvider`,
  `getAllProvider`, `getByMinLevel`). `reagent/entity.go` has the equivalent pair with
  reagent-specific fields (`ReagentItemId`, `Stat`, `Value` vs. crystalband's `MinLevel`/
  `MaxLevel`/`CrystalItemId`/`Count`) — not a copy-paste duplicate, a genuine per-domain
  reimplementation. Both `administrator.go`s (`crystalband/administrator.go:11`,
  `reagent/administrator.go:11`) now call `m.ToEntity()` instead of a field-by-field literal; both
  `processor.go`s call `Make(e)` (`crystalband/processor.go:73`, `reagent/processor.go:63`).
- **`compartment/model.go` → `compartment/builder.go`:** confirmed via file listing that
  `compartment/builder.go` is new and `compartment/model.go` shrank (33 deletions in the diff
  stat); not independently walked line-by-line since it's a pure relocation with no test changes
  and the merged build/test already exercises it (11 call sites cited in the round A report,
  consistent with `craft/*_test.go` still passing unmodified).

**CLOSED** for all three moves — no test weakened, deleted, or skipped; the historical
name-collision hazard did not resurface because it was never a package-level collision.

## 4. Round B — compose entry and Bruno collection

**Compose (`deploy/compose/docker-compose.core.yml:328-340`):** `atlas-maker` service block added
alphabetically between `atlas-keys`/`atlas-map-actions`, using `<<: [*atlas-defaults,
*seed-catalog]`. Cross-checked every env key the block sets or inherits against what the Go source
actually reads:

| Key | Read at |
|---|---|
| `REST_PORT` | `main.go:74` (`os.Getenv("REST_PORT")`) — via `env_file: .env` (`x-atlas-infra`), not redeclared, matching every peer |
| `BOOTSTRAP_SERVERS` | `kafka/consumer/consumer.go:24` — via `.env`, not redeclared, matching every peer |
| `SEED_CATALOG_ROOT` | `seed/groups.go:24` (`seeder.NewFilesystemCatalogSource("SEED_CATALOG_ROOT", ...)`) — explicitly set to `/var/run/seed-catalog` in the block, matching `atlas-drop-information`'s shape |
| `DB_NAME` | consumed by the shared `database.Connect` config path (same convention every peer uses); set to `atlas-maker` |
| `COMMAND_TOPIC_SAGA`/`EVENT_TOPIC_SAGA_STATUS` | `kafka/message/saga/kafka.go:13-14` — **not set** in the block; confirmed no other saga-emitting compose peer (`atlas-map-actions`, `atlas-storage`) sets them explicitly either — consistent pre-existing gap, correctly left alone per the brief's "do not invent a key" instruction |

No invented key found. `python3 -c "import yaml..."` (round B's own check, re-verified by inspection
of the merged YAML) parses cleanly; `kubectl kustomize` is unaffected since no k8s manifest was
touched.

**Bruno (`services/atlas-maker/.bruno/`):** 16 files. Verified every route URL against the actual
`r.HandleFunc`/`router.HandleFunc` call sites:

- `recipes/*`, `crafts/Create Craft.bru` → `craft/resource.go:94-96`
  (`GET /api/characters/{characterId}/maker/recipes[/{itemId}]`,
  `POST /api/characters/{characterId}/maker/crafts`) — matches.
- `reagents/Get Reagent(s).bru` → `reagent/resource.go:40-41` (`GET /api/reagents[/{itemId}]`) —
  matches.
- `crystal-bands/Get Crystal Band(s).bru` → `crystalband/resource.go:40-41`
  (`GET /api/crystal-bands[/{itemId}]`) — matches the service's own mux registration (base path
  `/api/` set in `main.go:39`, `crystalband.InitResource` mounted at `main.go:76`).
- `*/Seed *.bru`, `*/Get * Seed Status.bru` → `libs/atlas-seeder/handlers.go:33-42`
  (`POST {prefix}/seed`, `GET {prefix}/seed/status`), confirmed the seed-status response body
  fields (`groupName`, `subdomains.<name>.{count,updatedAt}`, `catalogRevision`,
  `tenantSeededRevision`, `tenantSeededAt`) exactly match `libs/atlas-seeder/result.go:20-32`'s
  `SubdomainStatus`/`Status` structs — no invented field.
- `crafts/Create Craft.bru`'s request body (`mode`, `worldId`, `channelId`, `targetItemId`) matches
  `craft/rest.go:80-91`'s `CraftRequestRestModel` field/tag list exactly.

No invented route or field found in the sampled/checked `.bru` files.

**One pre-existing gap surfaced, not introduced by either round, non-blocking for this review:**
`deploy/shared/routes.conf` has no ingress `location` block matching `/api/crystal-bands` (only
`/api/characters/.../maker`, `/api/maker`, `/api/reagents` — see `routes.conf:146,696,701`). The
`/api/maker` regex (`^/api/maker(/.*)?$`) does not match `/api/crystal-bands`. The route is real
and genuinely served by the atlas-maker binary's own mux router (confirmed above), so the Bruno
entry is not "invented" — but through the K3S ingress it is currently unreachable. This is a
SCAFFOLD-04 gap the source audit's own SCAFFOLD-04 check (`audit.md:146`) rated PASS without
citing crystal-bands, so it predates both fix rounds and was not one of the 7 assigned findings.
Neither brief asked either implementer to touch `routes.conf`. Flagging as an observation only —
not a defect in this unit of work.

## Summary of the 7 blocking findings

| # | Finding | Status |
|---|---|---|
| 1 | DOM-32 — `rest/handler.go` missing `ParseEnvironment` | CLOSED |
| 2 | FILE-01/FILE-06 — `craft/eligibility.go` processor scaffolding | CLOSED |
| 3 | DOM-02/DOM-03 — `crystalband`/`reagent` missing `ToEntity`/`Make` | CLOSED |
| 4 | DOM-05 — `craft/resource.go` inline transform loop | CLOSED |
| 5 | DOM-01 — 5 EXT packages + `compartment` missing/misplaced `builder.go` | CLOSED |
| 6 | SCAFFOLD-06 — no compose entry | CLOSED |
| 7 | SCAFFOLD-08 — no Bruno collection | CLOSED |

All 7 blocking findings from `audit.md` are closed in the post-fix tree, verified against the
merged commit range with fresh gate runs rather than accepted from either round's self-report.

## Not evaluable

- `character/builder.go`, `quest/builder.go`, `skill/builder.go`, `data/equipment/builder.go`,
  `data/itemmake/builder.go` (5 new files, DOM-01 finding #5) were confirmed to exist and to be
  wired into each package's `Extract` (via `git diff --stat`), but not read line-by-line against
  each `model.go`'s full field list — a 13-field builder (`data/itemmake`) is plausible to spot-
  check field-for-field but was not exhaustively diffed against `model.go` in this pass. Build/test
  passing over the whole module is corroborating but not a substitute for a field-by-field read.
- The full contents of the remaining 5 `.bru` request files not explicitly quoted above
  (`recipes/Get Recipe.bru`, `reagents/Get Reagent.bru`, `crystal-bands/Get Crystal Band.bru`, the
  two "Seed" POST bodies) were listed and URL-checked but their full `docs {}` response-body blocks
  were not individually diffed against every `RestModel` field — sampled `Create Craft` and the two
  seed-status bodies as representative.

## Verdict

verdict: APPROVED
artifact: docs/tasks/task-285-maker-skill-crafting/reviews/task-27-audit-fixes.md
scope_confirmed: reviewed both commits (e25600ea5, 8ada1d8ba) in full diff, cross-referenced against libs/atlas-rest/server/register.go, re-ran build/test/lint/service-registration-guard on the merged tree myself
blocking: 0
non_blocking: 1
  - deploy/shared/routes.conf — no ingress location for /api/crystal-bands (pre-existing SCAFFOLD-04 gap predating both fix rounds, not assigned to either brief; the Bruno entry documents a route the binary genuinely serves, just one unreachable through K3S ingress today)
not_evaluable: 2
