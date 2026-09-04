# task-289 — Implementation Context

Companion to `plan.md`. Everything here was read out of this worktree during
planning, not remembered.

---

## 1. Two design assumptions that turned out to be false

Both were verified against the code and both are handled by tasks in the plan.
They are recorded here because the design's ground-truth table asserts the
opposite, and a reader who trusts that table will be confused.

### 1.1 The two preset models are NOT key-identical (Task 1)

Design §1 states the two models are "key-identical minus `diagnostics`". They
are not.

| | `templates/characters/preset/rest.go` | `tenants/characters/preset/rest.go` |
|---|---|---|
| `ap` | absent | `AP uint16 \`json:"ap"\`` (line 46) |
| `sp` | absent | `SP string \`json:"sp"\`` (line 47) |

They were added on the tenant side by the Maple Life work. Nothing populates
them — the UI's shared `CharacterPresetAttributes`
(`services/atlas-ui/src/types/models/template.ts:50-66`) has neither, and no
shipped seed file carries either key (verified across all 11 files in
`services/atlas-configurations/seed-data/templates/`).

Consequence if left alone: every tenant preset serializes `"ap":0,"sp":""`
where the template's never does. `0` and `""` are values, not absences, so the
design's pruning rule cannot erase them. 10 of the 11 shipped templates carry
10–16 presets each, so `characters` drift would be permanently `true` for
essentially every tenant — NFR-5's exact failure, on day one.

Task 1 adds the two fields to the templates-side model. This is safe for
task-201: both sides of `templates.Revision` marshal the same
`templates.RestModel` struct (the shipped-catalog side parses the seed JSON
into it, the stored side unmarshals the row into it), so adding a field shifts
both hashes identically and `SeedDrift` equality is unchanged. No test in
`templates/` asserts a literal hash — `revision_test.go` compares revisions
only to each other.

### 1.2 There is no tenant `cashShop` or `npcs` page (Task 13)

Design §4.4 lists per-section reset buttons for all six sections. The tenant
routes in `services/atlas-ui/src/App.tsx:441-480` are:

```
/tenants/:id            /tenants/:id/handlers    /tenants/:id/worlds
/tenants/:id/writers    /tenants/:id/properties  /tenants/:id/character/templates
/tenants/:id/character/presets  /tenants/:id/character/maple-life
/tenants/:id/mts-config /tenants/:id/diagnostics
```

No `cash-shop`, no `npcs`. FR-6.4's per-section reset is therefore mounted on
the pages that exist:

| section | pages |
|---|---|
| `properties` | `tenants-properties-form.tsx` |
| `socket` | `TenantsHandlersPage`, `TenantsWritersPage` |
| `characters` | `TenantsCharacterTemplatesPage`, `TenantsCharacterPresetsPage` |
| `mapleLife` | `TenantsMapleLifePage` |
| `cashShop`, `npcs` | **no page** — API only, plus the header's whole-document reset |

Creating two new tenant editing pages is a materially different piece of work
from a drift/reset feature and is not in this plan. The API still accepts
`sections: ["cashShop"]` and `["npcs"]`; only the buttons are absent.

---

## 2. Key files and what they do

### `atlas-configurations` (module root `services/atlas-configurations/atlas.com/configurations`)

| File | Role in this task |
|---|---|
| `templates/revision.go:32-43` | task-201's whole-document struct hash. **Untouched** except one added comment (Task 3). Its tests are the PRD's named guard. |
| `templates/rest.go:46-66` | The `ViewRestModel` precedent, including the comment explaining why the write model stays untouched. Task 5 mirrors it. |
| `templates/processor.go:74-127` | `WithCatalog`'s safe-degradation comment, `makeView`, the three view providers. Task 5's shape. |
| `templates/processor.go:239-297` | `ReseedById` — entity-first lookup, sentinel wrapping, best-effort before-revision, `update` inside `ExecuteTransaction`, structured info log. Task 6's shape. |
| `templates/processor.go:152-172` | `canonicalBytes` and the comment on why re-seed must not go through `UpdateById` (the preset validator's mutation). The same trap, arriving from the other direction, is Task 6's §3.6 handling. |
| `templates/resource.go:37-42, 188-236` | `viewProcessor`, `writeJSONAPIError`, and the sentinel→status switch. Task 7's shape. |
| `templates/provider.go:46-59` | `byRegionVersionEntityProvider` uses `OverlaySingle` — the overlay/baseline environment fallback that makes `GetByRegionAndVersion` the correct baseline lookup (FR-1.1). Re-implementing the query in `tenants` would be a second definition of visibility. |
| `tenants/rest.go:13-33` | The write model. Gains nothing; `ViewRestModel` is appended to the file. |
| `tenants/processor.go:125-177` | `UpdateById` — the `ExecuteTransaction` body Task 6 reuses in intent: `update`, re-read the row, `enqueueTenantStatus` with the persisted environment. |
| `tenants/administrator.go:58-91` | `update` — history-before-write plus `scope.AuthorizeWrite`. Task 6 gets FR-4.6 and FR-4.7 by calling it, with no new code. |
| `tenants/provider.go:23-35` | `byIdEntityProvider` — `scope.Strict`, so a tenant in another environment is invisible. That is why Task 6's missing tenant is a 404, not a 403. |
| `tenants/resource.go:77-113` | The synthesized tenant context (`tenantlib.Create` + `WithContext`) the PATCH path builds so the atlas-data-backed preset rules run. The reset handler needs the same, built from the stored row. |
| `tenants/characters/preset/validator.go:35-47` | `Validate` assigns `uuid.New()` to any preset with an empty `Id`, **mutating the slice in place**. Task 6 hands it a shallow copy and keeps only the errors. A shallow copy suffices: `RestModel` elements are values and only `Id` is assigned. |
| `scope/scope.go:21-48` | `ErrCrossEnvironmentWrite` and `AuthorizeWrite`. A legacy caller (`""`) is always authorized. |
| `rest/handler.go:41-59` | `rest.WriteErrorResponse` already maps `ErrCrossEnvironmentWrite` to 403; Task 7's handler writes it explicitly so all five reset statuses come from one switch. |

### `atlas-ui` (cwd `services/atlas-ui`)

| File | Role |
|---|---|
| `src/components/features/templates/TemplateReseedButton.tsx` | The component Task 11 re-cuts. Its comments carry the conventions FR-6.6/6.7 restate: always-mounted `Tooltip` root, `Cancel` first in the footer, `createErrorFromUnknown` error path, dialog stays open on failure. |
| `src/lib/hooks/api/useTemplates.ts:351-373` | `useReseedTemplate` — invalidate on success only, three keys. Task 10 mirrors it with tenant keys. |
| `src/lib/hooks/api/useTenants.ts:29-44` | `tenantKeys`. The relevant members are `configDetail(id)` and `configLists()`. |
| `src/lib/hooks/api/socketKeys.ts` | `socketKeys.all` — its own module to avoid an import cycle. Invalidated because a socket reset changes the socket matrix and the handlers/writers grids, none of which clear on their own. |
| `src/services/api/tenants.service.ts:76-135` | Where `TenantConfigAttributes` actually lives. `src/types/models/tenant.ts` is a 9-line re-export barrel — edit the service file. |
| `src/services/api/tenants.service.ts:306-323` | `updateTenantConfiguration` spreads the whole cached `attributes` into every PATCH body. Task 9 strips the five computed keys there. |
| `src/services/api/templates.service.ts:365-376` | `reseed`. Copy its shape but **not** `skipTenantHeaders: true` — that is template-specific because templates are global; tenant calls carry the ordinary tenant headers. |
| `src/lib/utils/config-export.ts:74-83` | The delete block, and the comment explaining why an exported file must not carry a stale hash of itself. Task 9 extends it. |
| `src/services/api/onboarding.service.ts:106-118` | The lossy clone. `mapleLife` is absent; Task 10 adds it unconditionally. |
| `src/pages/TenantsPage.tsx:49` | Sources **registry** tenants from `useTenant()` (`@/context/tenant-context`) — `TenantBasic`, no configuration attributes. This is why Task 12 needs a second query. |
| `src/pages/templates-columns.tsx:58-82` | The `seedDrift` cell. Task 12 mirrors it, including the `!== true` guard and the `variant="secondary"` NFR-4 badge. |
| `src/components/features/tenants/TenantDetailLayout.tsx:21, 40-49` | Already calls `useTenantConfiguration(id)` for the Maple Life nav gate, so the drift data arrives with no extra request. The header seam is the `flex items-start justify-between` div beside `ConfigExportButton`. |
| `src/components/DetailActionBarContext.tsx` | `DetailActionBar` is a **single shared instance** rendered once by `TenantDetailLayout:58`, populated via `useRegisterDetailActionBar({ dirty, isSaving, onSave, onDiscard })` from three editor components. It takes no children and is not a per-section mount point — hence Task 13 puts the buttons in each page's own JSX. |

---

## 3. Decisions taken at plan time

**`drift` lives in its own package, importing neither domain package.** Design
OQ-2, unchanged. Task 4's cross-type test lives in an **external** test package
(`package drift_test`) so it can import both `templates` and `tenants` without
`drift` itself importing either. No cycle: `templates` imports nothing from
`tenants` (verified by grep), and `tenants/characters/rest.go:4` already imports
`atlas-configurations/templates/characters/template`.

**`prune` keeps array length.** An array element that prunes to nothing is
replaced with `{}` rather than dropped: the array's *length* is content, and
silently shortening a list would hide a real divergence. Only whole empty
arrays are pruned.

**`AllViewProvider` is hand-rolled, not `model.MapPaged`.** The two-phase
baseline resolution (design §3.4) cannot be expressed inside `ParallelMap`
without a mutex, and even with one two goroutines would still race into the
same query. Task 5 writes the provider as a plain closure. The counting stub in
`TestAllViewProviderResolvesEachBaselineOnce` is what makes FR-3.4 enforced
rather than described.

**Reset returns 422 for a validation failure while PATCH returns 400.** The
design follows the PRD here and the plan follows the design. The inconsistency
is deliberate: a 400 on PATCH means "your body is bad", while a validation
failure on reset means "the server's own baseline is unprocessable" — the
request was fine. This is a new endpoint, so nothing regresses.

**Handlers and Writers share one `socket` reset, and both character pages share
one `characters` reset.** Two pages over one section. The copy says "socket
handlers and writers" / "character templates and presets" on both, so an
operator resetting from the Writers page is not surprised to find handlers
reverted too.

---

## 4. Task sizing

Fourteen tasks. Two are deliberately larger than the ~6-file guideline; both
are noted here per the plan-task rule.

| Task | Files | Note |
|---|---|---|
| 1 | 2 | |
| 2 | 3 | |
| 3 | 5 | One is a comment-only edit to `templates/revision.go`. |
| 4 | 1 | |
| 5 | 4 | |
| 6 | 4 | |
| 7 | 2 | Large in *content* (a new handler plus three marshal-type switches) but narrow in surface, and all three switches are the same mechanical change. |
| 8 | 2 | Documentation only. |
| 9 | 4 | |
| 10 | 4 | |
| 11 | 2 | |
| 12 | 5 | |
| **13** | **7** | **Deliberately large.** Six of the seven edits are the same three-line mount of one component with different props — a mechanical change repeated, which batches fine. Splitting it would mean six dispatches for six near-identical hunks. |
| 14 | 0 | Gate only. |

No task crosses a service boundary. Tasks 1–8 are `atlas-configurations`;
Tasks 9–13 are `atlas-ui`; Task 14 is the repo gate.

---

## 5. Dependency order

```
1 ──► 4
2 ──► 3 ──► 4
      3 ──► 5 ──► 6 ──► 7 ──► 8
                        7 ──► 9 ──► 10 ──► 11 ──► 12
                                                  11 ──► 13
                                          8,12,13 ──► 14
```

Task 4 must run after Task 1, or its cross-type assertion fails for the reason
§1.1 describes — which would be a correct failure, but the fix belongs in the
models, not in `drift`.

Tasks 9–13 depend on Task 7 only for the wire contract; they can be written
against it without Tasks 1–8 having landed, but the frontend tests mock the
service layer, so nothing in the frontend actually executes backend code.

---

## 6. Verification

- Go, per task: `go build ./...` and `go test ./...` from
  `services/atlas-configurations/atlas.com/configurations`.
- Frontend, per task: `npm test -- <files>`, `npx tsc --noEmit`, and
  `npm run lint` from `services/atlas-ui`.
- Branch gate: **flagless** `tools/verify.sh` must exit 0 (Task 14).
  `--quick` / `--no-docker` do not count.
- The `templates` package's existing tests must pass **unmodified** throughout.
  If a template test needs its expectations edited, the change that provoked it
  is wrong.
