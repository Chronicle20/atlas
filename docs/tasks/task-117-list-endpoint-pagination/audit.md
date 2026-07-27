# Plan Audit — task-117-list-endpoint-pagination (plan-adherence)

**Plan Path:** docs/tasks/task-117-list-endpoint-pagination/plan.md
**Audit Date:** 2026-07-03
**Branch:** task-117-list-endpoint-pagination (HEAD da38d2da89)
**Base Branch:** main (38d4d0ba2), 131 commits all scoped `(task-117)`

## Executive Summary

All 29 plan tasks are faithfully implemented. The three PRD §10 acceptance greps
pass in substance: **grep 1 (`MarshalResponse[[]`) is EMPTY** — no list endpoint
marshals an unpaginated slice anywhere in the fleet. Library layer, per-service
conversions, consumer drains, convention doc, PS-5 resolution, and UI paging all
landed. `go build` is clean for the three lib modules and representative services;
all library pagination tests pass. No new `// TODO`/501/stub was introduced. Verdict:
**all 29 landed, READY_TO_MERGE.**

## Acceptance-grep results

- **Grep 1** `grep -rn "MarshalResponse\[\[\]" services/*/atlas.com` → **EMPTY (PASS)**. Every collection GET now paginates.
- **Grep 2** `func (p *ProcessorImpl) GetAll(` → 13 hits, **all safe** (see below). Not literally empty, but none back an unpaginated REST list.
- **Grep 3** `requests.SliceProvider` → 30 hits, **all filtered/by-id** (`requestByMemberId`, `requestByName`, `requestMembers(partyId)`, `requestByCompartmentId`, `requestByAccountId`, `requestInMapByName`, `requestNPCsInMapByObjectId`, `requestEquipmentSlotDestination(id)`, `requestPartyByMemberId`). None target a converted bare collection. PASS.

### Grep-2 classification (why 13 hits are all safe)

| Method | Classification | Evidence |
|---|---|---|
| gachapons/global:36, gachapon:31 | **False positive — already paged** | `GetAll(page model.Page) model.Provider[model.Paged[Model]]` |
| channel/data/quest:41 | REST consumer, **drains** | `DrainProvider[...](allQuestsUrl(), 250, ...)` |
| login/world:44 | REST consumer, **drains** | delegates `AllProvider()` → `DrainProvider(worldsUrl(),250,...)` |
| party-quests/tenant:37 | REST consumer, **drains** | `AllProvider()` → `DrainProvider(allTenantsUrl(),250,...)` |
| transports/tenant:44 | REST consumer, **drains** | `AllProvider()` → `DrainProvider(allTenantsUrl(),250,...)` |
| drop-information monster/reactor/continent drop + continent (4) | Same-service internal tenant-scoped read; REST siblings (`GetForMonster/GetForItem`) are paged | monster/drop/processor.go:36 feeds continent aggregation; provider `getAll()` local DB |
| saga:215, party-quests/instance:193, transports/channel:41 | Internal registry dumps | non-REST, local `[]Model`/`[]channel.Model` |

The plan's grep-2 note explicitly permits "REST-backed drains … renamed or justified
in the doc"; `endpoint-inventory.md` §Disposition and `.superpowers/sdd/task-29-report.md`
record the justification. One cosmetic deviation from Task 8's rename convention:
`channel/data/quest.GetAll`, `login/world.GetAll`, `party-quests/tenant.GetAll`,
`transports/tenant.GetAll` were left named `GetAll` rather than renamed
(atlas-channel *account* was renamed to `GetAllAccounts` per Task 8). Functionally
correct (all drain page-by-page); no truncation risk.

## Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | model.Page/Paged/MapPaged | DONE | libs/atlas-model/model/paged.go; tests PASS |
| 2 | database.PagedQuery | DONE | libs/atlas-database/paged.go; tests PASS |
| 3 | paginate.ParseParams/Slice/EnvelopeFor + server.WriteBadRequest | DONE | params.go, slice.go, envelope.go:100 (EnvelopeFor), server/error.go; tests PASS |
| 4 | requests.PagedGetRequest/PagedProvider/DrainProvider | DONE | libs/atlas-rest/requests/paged.go; tests PASS |
| 5 | atlas-data item-string search on ParseParams | DONE | string_resource.go:73; `parsePagingParams` deleted (grep empty) |
| 6 | atlas-data doc store AllPaged/AllPagedProvider/DrainAllProvider | DONE | db_storage.go:58, storage.go:79 & :110 |
| 7 | atlas-login account drain | DONE | account/processor.go:61-62 DrainProvider |
| 8 | atlas-channel account drain + GetAll→GetAllAccounts | DONE | account/processor.go:19,42-43,50 |
| 9 | atlas-account paginate GET /accounts | DONE | provider.go:34 PagedQuery, processor.go:107 AllProvider(page,...), resource.go:133 |
| 10 | atlas-character /characters (+siblings, sessions) | DONE | resource.go:64,105,145; session/history/resource.go |
| 11 | atlas-guilds GET /guilds + filter[name] | DONE | provider.go:22 escapeLike, :29 getByNameLike; resource_test.go filter cases |
| 12 | atlas-ban /bans, /history, /history/accounts/{id} | DONE | grep1 clean for atlas-ban; conversions present |
| 13 | atlas-notes GET /notes | DONE | grep1 clean for atlas-notes |
| 14 | atlas-merchant list routes + consumer check | DONE | grep1 clean for atlas-merchant |
| 15 | atlas-ui fetchPaged/fetchAll utility | DONE | services/api/pagination.ts:48,64; __tests__/pagination.test.ts |
| 16 | atlas-ui characters/accounts/bans paged views | DONE | characters.service.ts getPage; service tests present |
| 17 | atlas-ui guilds server-side filter[name] + paging | DONE | guilds.service.ts:57-58 filter[name] via fetchPaged; :77 fetchAll |
| 18 | atlas-data core doc-store list routes | DONE | grep1 clean for atlas-data; MarshalPaginatedResponse throughout |
| 19 | atlas-data remaining routes + delete Storage.GetAll/AllProvider | DONE | grep1 clean; DrainAllProvider present |
| 20 | map/reactor/portal-actions | DONE | grep1 clean for those services |
| 21 | npc-conversations/gachapons/drop-information/party-quests defs | DONE | grep1 clean; gachapons GetAll(page) paged; drop-info siblings paged |
| 22 | atlas-ui data browser views | DONE | gachapons/templates/reactors service+page getPage usage |
| 23 | inventory/storage/buddies/skills/keys | DONE | inventory asset provider.go:19 PagedQuery, resource.go:39,54 |
| 24 | pets/cashshop/quest/monster-book | DONE | grep1 clean for those services |
| 25 | marriages/families/invites/buffs/npc-shops | DONE | grep1 clean for those services |
| 26 | atlas-maps + in-field registries | DONE | maps/visit/resource.go + resource_paginate_test.go |
| 27 | parties/messengers/saga/pq-instances/portals | DONE | paginate.Slice in party/messenger/saga resource.go |
| 28 | LOW sweep world/tenants/configurations/transports | DONE | grep1 clean; tenant drains present |
| 29 | convention doc, PS-5, acceptance sweep | DONE | docs/rest-pagination.md; architectural-improvements.md:226 "RESOLVED (task-117)"; endpoint-inventory Disposition; backend-dev-guidelines SKILL.md + patterns-rest-jsonapi.md point to rest-pagination.md |

**Completion Rate:** 29/29 (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Build & Test Results

| Module | Build | Tests | Notes |
|---|---|---|---|
| libs/atlas-model | PASS | PASS | Paged tests ok |
| libs/atlas-database | PASS | PASS | PagedQuery tests ok |
| libs/atlas-rest | PASS | PASS | paginate + requests paged/drain tests ok |
| services/atlas-account | PASS | — | go build clean |
| services/atlas-data | PASS | — | go build clean |
| services/atlas-guilds | PASS | — | go build clean |
| atlas-ui | (reported) | (reported 851/851) | pagination.test.ts present at __tests__/ |

Full `docker buildx bake all-go-services` (79 images) was reported green by the
execution session; not re-run here. Representative `go build ./...` re-runs above
all exit 0.

## Notes / minor observations (non-blocking)

1. **plan.md checkboxes not ticked** — 0/96 step checkboxes marked `[x]` despite full
   execution. Documentation hygiene only; work is done (131 task-117 commits, grep 1 clean).
2. **Grep-2 naming inconsistency** — four cross-service drain consumers still named
   `GetAll` (channel/quest, login/world, pq/tenant, transports/tenant) rather than
   renamed like Task 8's `GetAllAccounts`. All drain correctly; justified in
   endpoint-inventory Disposition + task-29-report. No truncation risk.
3. **Pre-existing TODOs** in touched processor files (character, guilds, inventory,
   login, npc-shops, pets) are unrelated to pagination and confirmed **not added by
   this branch** (diff of added `// TODO` lines is empty).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None blocking. Optional polish: (a) tick plan.md checkboxes for record-keeping;
(b) rename the four remaining `GetAll` drain consumers to `GetAll<Resource>` to make
grep 2 literally empty.

---

# Backend Audit — task-117 pagination libraries (backend-guidelines-reviewer)

- **Audit Date:** 2026-07-03
- **Scope:** foundational pagination libs + two reference conversions (atlas-account, atlas-guilds)
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/SEC-*)
- **Build:** PASS — atlas-model, atlas-database, atlas-rest, atlas-account, atlas-guilds all `go build ./...` clean
- **Tests:** PASS — all packages green (`go test ./... -count=1`)
- **Overall:** PASS (no Critical/Important findings; 2 Minor gofmt nits)

## Correctness / Security verifications (file:line evidence)

| Area | Status | Evidence |
|------|--------|----------|
| Tenant scoping — COUNT + Find on same scoped db, no cross-tenant leak | PASS | `libs/atlas-database/paged.go:32-58` runs both against `db.Session(&gorm.Session{})`; tenant WHERE injected by the query callback for both (`tenant_scope.go:74-78`). Proven by `paged_test.go:34` `TestPagedQueryTenantScopeAgreement` (count==7 not 12, every row t1). |
| COUNT strips ORDER BY, caller order preserved, PK tie-break appended | PASS | `paged.go:33-38` (clause map copied then `delete(clauses,"ORDER BY")`), `paged.go:55` appends `OrderByColumn{pk.DBName}`. `paged_test.go:84` `TestPagedQueryCallerOrderPreservedAndCountUnaffected`. |
| Missing-PK guard (stable paging requires total order) | PASS | `paged.go:48-51` returns explicit error. |
| Invalid page rejected, not clamped | PASS | `paged.go:19-21`; `paginate/params.go:31,39`; legacy `?limit=` rejected `params.go:44-46`. |
| SQL-injection / LIKE-wildcard escaping in `filter[name]` | PASS | `guild/provider.go:22-27` escapes `\` then `%` then `_` (correct order), `provider.go:33` uses parameterized `LOWER(name) LIKE LOWER(?) ESCAPE '\'`. Tests `provider_test.go:100` (`0%_r`) + `:77` substring case-insensitive. No string concatenation of user input into SQL. |
| No swallowed errors in lib/provider layer | PASS | every provider propagates via `ErrorProvider`/returned err; handlers check Transform err (`account/resource.go:124-129`, `guild/resource.go:46-51`). |
| 400 JSON:API error shape | PASS | `libs/atlas-rest/server/error.go:12-38` emits `{"errors":[{status,title,detail}]}`, `WriteHeader(400)`. |
| DOM-11 lazy providers | PASS | `PagedQuery` returns `model.Provider` (nothing runs until invoked) `paged.go:17-18`; `MapPaged` lazy `model/paged.go:25-40`; `AllProvider` composes lazily `account/processor.go:107-112`, `guild/processor.go:100-103`. |
| DOM-21 shared-type reuse | PASS | `guild/provider.go:6` uses `atlas-constants/world.Id`; Page/Paged are new generic containers with no atlas-constants equivalent (not a redeclaration). |
| DOM-06/07 FieldLogger + d.Logger() | PASS | processors take `logrus.FieldLogger`; handlers pass `d.Logger()` (`account/resource.go:117`, `guild/resource.go:39`). |
| DOM-10 tenant callbacks in tests | PASS | `guild/processor_test.go:54` `RegisterTenantCallbacks`; `paged_test.go` uses `databasetest.NewInMemoryTenantDB`. |
| DOM-20 table-driven tests | PASS | `paginate/params_test.go:18 cases := []struct`. |
| Client integration test (httptest, real body decode) | PASS | `libs/atlas-rest/requests/paged_test.go:49-186` — httptest servers exercise decode, multi/single/empty page drain, no-envelope compat, warn>20, error mapping. |
| Consumer-first rollout safety (no envelope ⇒ full collection) | PASS | `requests/paged.go:119-121`; `TestDrainProviderNoEnvelopeCompat` `paged_test.go:146`. |
| No `// TODO`/stubs/501/absolute paths in pagination code | PASS | grep of all scoped foundational files: none. |

## Minor findings (non-blocking)

- **BG-MINOR-1 (task-117-introduced):** `services/atlas-account/atlas.com/account/account/provider.go:37` has a trailing blank line — `gofmt -l` flags the file. Introduced by the pagination commit (cfc22c977a) that added `getAll`. Run `gofmt -w`.
- **BG-MINOR-2 (pre-existing, in a touched file):** `services/atlas-account/atlas.com/account/account/resource.go:26` `registerPicAttemptInput` is mis-indented (gofmt flags the file). Predates task-117 (commit a0459bfe6e "Add PIN/PIC tracking"), but the file was edited for pagination so the branch is not `gofmt`-clean. Trivial fix.
- **BG-NOTE (out of scope, pre-existing):** `go vet ./...` on atlas-rest reports `server/server.go:187 WaitGroup.Add called from inside new goroutine` — introduced by task-032, untouched by task-117.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- BG-MINOR-1, BG-MINOR-2: run `gofmt -w` on the two atlas-account files so the branch is gofmt-clean.

**Overall: PASS.** The shared pagination primitives are correct, tenant-safe, and injection-safe, with strong table-driven + httptest coverage. The COUNT/Find-on-same-scoped-db design is empirically verified against cross-tenant leakage and caller-order preservation. Only cosmetic gofmt nits remain.

---

# Frontend Audit — task-117 pagination (frontend-guidelines-reviewer)

- **Audit Scope:** atlas-ui TypeScript/React pagination changes on branch `task-117-list-endpoint-pagination` (35 non-test source files: `services/api/pagination.ts` + shared util, converted services, `use*` hooks, list pages, `Pager`).
- **Guidelines Source:** frontend-dev-guidelines skill (FE-* checklist)
- **Date:** 2026-07-03
- **Build:** PASS (`tsc -b` exit 0; run with linux node v24 — the repo's `npm run build` shells to Windows CMD under WSL and silently no-ops)
- **Tests:** 851 passed, 0 failed (104 files; `vitest run`)
- **Overall:** NEEDS-WORK (build + tests green; one provable stale-view invalidation bug + minor findings, none blocking)

## Build & Test Results

- `node node_modules/typescript/bin/tsc -b` → exit 0 (production build type-checks test files too; clean).
- `vitest run` → `Test Files 104 passed (104) / Tests 851 passed (851)`.
- NOTE: `npm run build` / `npm test` from the default WSL shell hit a Windows CMD.EXE UNC-path fallback and do NOT actually run tsc/vitest (they exit 0 without running). Verified by invoking the node_modules binaries directly with a Linux node on PATH.

## File Inventory (in-scope, non-test)

- Util: `services/api/pagination.ts` (Page util — `fetchPaged`/`fetchAll`, `PageMeta`/`PagedResult`).
- Services: `accounts`, `bans`, `characters`, `guilds`, `merchants`, `templates`, `gachapons`, `monsters`, `maps`, `npcs`, `items`, `reactors`, `drops`, `commodities`, `conversations`, `quest-conversations`, `quests`, `mob-skills`, `portal-scripts`, `reactor-scripts`, `index.ts`.
- Hooks: `useAccounts`, `useBans`, `useCharacters`, `useGuilds`, `useTemplates`, `useGachapons`.
- Pages: `AccountsPage`, `BansPage`, `CharactersPage`, `GachaponsPage`, `GuildsPage`, `MerchantsPage`, `TemplatesPage`.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | grep `: any`/`as any` across all 35 in-scope files → 0 matches |
| FE-02 | No manual class concat | PASS | grep `className={"…"+` / `className={\`` → 0 matches |
| FE-03 | No direct API client in components | PASS | pages import services/hooks only; `@/lib/api/client` imported only inside `services/api/*` (e.g. pagination.ts:12) |
| FE-04 | No inline Zod in components | PASS (pre-existing note) | `pages/TemplatesPage.tsx:23` `cloneTemplateFormSchema = z.object(...)` is inline, but it is PRE-EXISTING (0 z.object lines in the task-117 diff for that file). Not introduced by this task; flag for future cleanup |
| FE-05 | No spinners for content loading | PASS | Only `animate-spin` match is `MerchantsPage.tsx:195` — a `Loader2` inside a submit `<Button disabled={searchLoading}>` (lines 193-200). Content loading uses Skeletons (`CharacterPageSkeleton`, `AccountPageSkeleton`, `TemplatePageSkeleton`, `GuildPageSkeleton`, `BansPageSkeleton`, `PageLoader`) |
| FE-06 | No hardcoded colors | PASS | grep bg/text/border-(white/black/gray-N/...) → 0 matches; semantic tokens used (`text-muted-foreground`, `bg-background`, `bg-destructive`) |
| FE-07 | No state mutation | PASS | Service sorts (`sortGuilds`/`sortAccounts`/`sortBans`/`sortById`/`sortMaps`) mutate freshly-fetched local arrays, not React state; pages set state immutably. Minor: `guilds.service.ts:136-152 transformGuildResponse` shallow-copies then reassigns `attributes.members` on the shared `attributes` ref — cosmetic, on fetched data only |
| FE-08 | Named exports for components | PASS | All 7 pages `export function XPage()` (project convention; App.tsx imports by name). grep `export default` → 0 matches |
| FE-09 | Tenant guard in hooks | PASS | Explicit-tenant hooks guard `enabled: !!tenant?.id` (useAccountsPage useAccounts.ts:67, useBansPage useBans.ts:49, useCharactersPage useCharacters.ts:59, useGuildsPage useGuilds.ts:57); context hooks guard `enabled: !!activeTenant` (useGachaponsPage useGachapons.ts:42). Templates are tenant-agnostic (Pattern C) — no guard needed |
| FE-10 | Tenant ID in query keys | PASS | Explicit-tenant paged keys include tenant: `accountKeys.pagedList` (useAccounts.ts:23-24), `banKeys.pagedList` (useBans.ts:17-18), `characterKeys.pagedList` (useCharacters.ts:24-25), `guildKeys.pagedList` (useGuilds.ts:24-25), inline merchants key (MerchantsPage.tsx:76,87). `gachaponKeys.pagedList` (useGachapons.ts:13) omits tenant — CONSISTENT with the documented Pattern-B data-browser convention (`mapKeys.list` in the skill also omits it; isolation via `queryClient.clear()` on tenant switch). Templates key omits tenant correctly (tenant-agnostic) |
| FE-11 | Error handling | PASS (with note) | Paged pages surface errors via `query.error?.message` into `DataTableWrapper` (e.g. AccountsPage.tsx:32, GuildsPage.tsx:63). Pre-existing `console.error` in mutation `onError` callbacks (useTemplates/useAccounts/useCharacters, npcs.service) are unchanged by task-117 |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `PagedResult<T>.data: T[]`, `PageMeta` = `{total, page:{number,size,last}}` (pagination.ts:15-27) matches the Go envelope `meta.{total,page:{number,size,last}}`; models keep `{id, attributes}` |
| FE-13 | Service pattern | PASS | Object-literal / class services compose `api` + `fetchPaged`/`fetchAll` (documented direct-client pattern); no BaseService regressions |
| FE-14 | Query key `as const` | PASS | All key factories use `as const` (e.g. useBans.ts:12-22, useGuilds.ts:20-36) |
| FE-15 | Forms use RHF + zodResolver | PASS | TemplatesPage forms use `useForm({resolver: zodResolver(...)})` (TemplatesPage.tsx:87-102) — unchanged by task-117 |
| FE-16 | Schema + inferred type | PASS | `PagedResult`/`PageMeta` are plain TS interfaces (not Zod); `type X = z.infer` pattern intact in touched schema-consuming code |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests for changed components | PASS | New tests: `pagination.test.ts` (147 lines — page-param preservation, meta-null compat, multi-page drain, empty-page early stop, default size), `accounts/bans/characters/guilds/merchants.service.test.ts`, all 7 `*Page.test.tsx`, `useGuilds.test.tsx` |
| FE-18 | Mocks updated | PASS (N/A) | No atlas-ui `__mocks__/` dir; tests mock `@/lib/api/client` inline via `vi.mock` (pagination.test.ts:3-6). The `mock/processor.go` files in the diff are Go backend mocks |

## `fetchPaged`/`fetchAll` util review (focus area)

- Query-param preservation: PASS — `withPageParams` (pagination.ts:31-40) splits existing query, rebuilds via `URLSearchParams`, `.set()`s `page[number]`/`page[size]`. Verified by pagination.test.ts:36-47 (`filter[name]=x` preserved) and :129-146 (preserved across all drained pages).
- No-envelope compat: PASS — `doc.meta ?? null` (pagination.ts:55); `fetchAll` returns page-1 data unchanged when `meta === null` (pagination.ts:71-73). Verified pagination.test.ts:90-97.
- Loop termination: PASS — `fetchAll` iterates `2..meta.page.last` and `break`s on an empty page (pagination.ts:78-82). Bounded by server `last`; no infinite loop. Verified :99-114.
- Server-side filtering (anti-pattern): PASS — guild search now hits server `filter[name]` (`guilds.service.ts:56-60` `search`), replacing the old fetch-all-then-filter. Remaining `fetchAll` drains in guilds (`getByWorld`/`getWithSpace`/`getRankings`) are documented as semantic-all consumers with no server filter route (verified comment cites `atlas-guilds .../guild/resource.go`). Browse views use `getPage`; semantic-all consumers use `fetchAll`.
- `placeholderData: keepPreviousData` present on every paged browse hook (useAccounts.ts:68, useBans.ts:50, useCharacters.ts:60, useGuilds.ts:58, useGachapons.ts:43, useTemplates.ts:66, MerchantsPage.tsx:79). Pages guard `{meta && rows.length > 0 && <Pager .../>}`.

## Summary

### Blocking (must fix)
- None. Build and tests pass; no FE-* hard failure.

### Non-Blocking (should fix)
- **FE-INVAL-1 (Important):** Paged-list query keys are structured as `[...Keys.all, tenant?, page, size]` — SIBLINGS of, not children of, `Keys.lists()` (`[...all, 'list']`). Mutation hooks that invalidate only `Keys.lists()`/`Keys.list()` therefore do NOT invalidate the corresponding paged browse view. Provable bug: `useDeleteTemplate` invalidates `templateKeys.lists()` (useTemplates.ts:235) while `TemplatesPage` renders from `templateKeys.pagedList` (useTemplates.ts:24 / TemplatesPage.tsx:56) → deleting a template leaves the stale row on screen until a manual refresh. Same shape for `useUpdateTemplate`/`usePatchTemplate` (useTemplates.ts:166,198) and `useUpdateCharacter` (useCharacters.ts:134 invalidates `characterKeys.list`, not pagedList) and the account session-terminate hooks (useAccounts.ts:207,273 invalidate `accountKeys.lists()`). Bans and account-create dodge it by invalidating `Keys.all` (useBans.ts:85,93,101 / CreateAccountDialog.tsx:84). Fix: either nest `pagedList` under `lists()` (`[...Keys.lists(), 'page', ...]`) or have those mutations invalidate `Keys.all`.
- **FE-MINOR-1 (pre-existing):** Inline `z.object` schema `cloneTemplateFormSchema` at `pages/TemplatesPage.tsx:23` violates FE-04 (schemas belong in `lib/schemas/`). Not introduced by task-117; move it in a future cleanup.
- **FE-MINOR-2 (cosmetic):** `guilds.service.ts:136` `transformGuildResponse` reassigns `.attributes.members`/`.titles` on the shallow-cloned (shared) `attributes` object; assigns new sorted arrays so no array is mutated, but the original guild's `attributes` pointer is touched. On freshly-fetched data only — no React-state impact.

**Overall: NEEDS-WORK.** Pagination primitives are correct, tenant-safe, injection-safe (param preservation + termination proven by tests), and the list pages page server-side with `keepPreviousData` and URL-synced page state. The one substantive issue is the paged-key/invalidation mismatch (FE-INVAL-1), which leaves Templates (and, on mutation, Characters/Accounts) browse views stale after a write until manual refresh.
