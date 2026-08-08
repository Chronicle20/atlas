# Template Re-seed Trigger — Implementation Context

Companion to [`plan.md`](plan.md). Read this first if you are picking the task up cold.

- **PRD:** [`prd.md`](prd.md)
- **Design:** [`design.md`](design.md)
- **Worktree:** `.worktrees/task-201-template-reseed-trigger`, branch `task-201-template-reseed-trigger`
- **Services touched:** `atlas-configurations` (Go), `atlas-ui` (TypeScript). No other service.

---

## 1. The problem in one paragraph

`atlas-configurations` seeds configuration templates from JSON files baked into its image at `/seed-data/templates/`. The import is **create-if-absent**: `importTemplate` returns `"skipped"` when a row already exists on `(region, majorVersion, minorVersion)`, and never updates. That is the correct default — templates are editable through the Atlas UI, and a reconcile-on-boot would discard operator edits on every redeploy — but it means an edit to a checked-in seed file never reaches a database that already has that template row. Confirmed live on `atlas-main` (2026-08-07): the GMS 87.1, GMS 92.1, GMS 95.1 and JMS 185.1 rows are all missing `QuestActionHandle`, despite every one of those seed files carrying it since commit `69bffe88d` or `298ee6d73`. This task makes that drift **visible** and makes the repair a **button press**. It does not change boot behaviour, and it does not repair those four rows on its own.

---

## 2. Key files, with the line numbers that matter

### Go — `services/atlas-configurations/atlas.com/configurations/`

| File:line | Why it matters |
|---|---|
| `templates/rest.go:11-22` | `RestModel`. `Id` carries `json:"-"`. **Nothing with a JSON tag may be added here** — `Create` marshals it verbatim into `Entity.Data`. |
| `templates/processor.go:67-76` | `Make` — unmarshals `Entity.Data`, applies `socket.Normalize`, sets `Id`. The stored-side revision is computed over its output. |
| `templates/processor.go:86-122` | `Create` — `socket.Normalize` → `socketValidate` → `json.Marshal` → insert. Task 3 extracts the first three steps into `canonicalBytes`. |
| `templates/processor.go:124-158` | `UpdateById` — additionally runs the **preset validator**, which reassigns `input.Characters.Presets` before marshalling. Re-seed must NOT go through here (FR-3.4). |
| `templates/processor.go:64` | `AllProvider` already runs `model.ParallelMap()`, so the per-row SHA-256 parallelizes for free (NFR-2). |
| `templates/processor.go:167-169` | `socketValidate` — pure, dependency-free, never skippable. |
| `templates/administrator.go:12-29` | `update(ctx, id, region, major, minor, data)` transaction function. Re-seed reuses it with the **entity's** columns, not the file's. |
| `templates/provider.go:20-31` | `byIdEntityProvider` — returns `gorm.ErrRecordNotFound` for a missing id; that is what `ErrTemplateNotFound` wraps. |
| `templates/validation_error.go:14-27` | `validationFailureError` + `AsJSONAPIErrors()` — the `{"errors":[{"status","title","detail","meta"}]}` shape the new 404/409 writer mirrors. |
| `templates/socket/rest.go:30-43` | `socket.Normalize` — coerces the four socket collections from `nil` to `[]`. Both `Make` and `Create` call it, which is why `Revision` must too. |
| `templates/characters/preset/validator.go:37-41` | Assigns a UUID **only** when `presets[i].Id == ""`. This is why design §7's empirical finding holds. |
| `templates/resource.go:21-33` | `InitResource` — the subrouter the new `POST /{templateId}/reseed` registers on. |
| `seeder/seeder.go:39-44,138-182` | `ConfigMetadata`, `discoverFiles`, `extractMetadata` — all deleted in Task 7. |
| `seeder/seeder.go:197-251` | `importTemplate` — its create-if-absent behaviour is the thing the regression guard protects. |
| `main.go:69-79` | Where the seeder is constructed. `InitShippedCatalog` goes immediately before it. |
| `socket/corpus_test.go:14-22` | Already asserts all **11** seed templates pass `socket.Validate` — so `Create` will succeed on every one of them in the load-bearing drift test. |
| `templates/processor_test.go:17-58` | The sqlite in-memory harness: `testEntity`, `setupTestDB`, `testLogger`, `createTestRestModel`. Reuse these; do not redefine them. |
| `templates/resource_paginate_test.go:13-21` | `testServerInformation` — reuse for the new resource tests. |
| `libs/atlas-rest/server/error.go:48` | `WriteErrorResponse` maps everything to 500/503. There is no `WriteNotFound`/`WriteConflict`, which is why the handler writes 404/409 itself. |

### TypeScript — `services/atlas-ui/src/`

| File:line | Why it matters |
|---|---|
| `types/models/template.ts:73-98` | `TemplateAttributes` — the three new optional fields go here. |
| `services/api/templates.service.ts:354` | `delete` — the pattern the new `reseed` sits beside. |
| `services/api/templates.service.ts` (`sortTemplate`) | Compensates for the documented `"npcs": null` five of the seeds produce. Unchanged by this task. |
| `lib/api/client.ts:238-256` | `api.post` omits the body entirely when `data` is `undefined`. |
| `lib/hooks/api/useTemplates.ts:31-38` | `templateKeys.detail(id)` and `templateKeys.lists()` — the two keys the re-seed mutation invalidates. |
| `components/features/config/ConfigExportButton.tsx` | The convention the reset button copies: always-mounted `Tooltip` root with gated content, `createErrorFromUnknown` in the catch, `disabled={!query.data}`. |
| `components/features/templates/TemplateDetailLayout.tsx:37` | Where `ConfigExportButton` is mounted; the reset button becomes its sibling. |
| `components/features/services/DeleteServiceDialog.tsx` | The `AlertDialog` convention: `AlertDialogCancel` first, `e.preventDefault()` in the action's `onClick`, `buttonVariants({ variant: "destructive" })`. |
| `lib/utils/config-export.ts:63-89` | `toConfigExportPayload` — passes every attribute through by design, which is why the three computed keys must be explicitly deleted. |
| `pages/templates-columns.tsx` | `getColumns` — the badge column goes between `attributes.minorVersion` and `actions`. |

---

## 3. Decisions already made (do not re-litigate)

| # | Decision | Why |
|---|---|---|
| D1 | The catalog lives in package **`templates`**, not `seeder` — contradicting the PRD's Service Impact table. | `seeder` already imports `templates`. Putting the catalog in `seeder` closes an import cycle, because drift detection and re-seed both live in `templates`. |
| D2 | `LoadCatalog` is **pure**; `InitShippedCatalog`/`ShippedCatalog` are the `sync.Once` singleton; the processor takes a catalog via `WithCatalog`. | Applying FR-1.2's registry convention literally would make the source directory implicit and the processor untestable without touching `$SEED_DATA_PATH`. |
| D2b | The catalog load is **unconditional on `SEED_ENABLED`**. | That flag gates *importing*, not *knowing what ships*. An operator who disabled seeding still needs the badge and the button. |
| D3 | The three attributes ride on a **`ViewRestModel`** embedding `RestModel`, not on `RestModel` itself. | `Create` persists `json.Marshal(input)` verbatim. A tagged field on `RestModel` would be written into the stored document, read back by `Make`, and folded into the next revision — self-reference and permanent phantom drift. The view/write split makes that class of failure not exist rather than defended against. |
| D4 | Re-seed writes through `canonicalBytes` (Create's semantics), never `UpdateById`. | `UpdateById` runs the preset validator, which reassigns presets before marshalling. A row re-seeded through it would report drift the instant it was reset. |
| D5 | `ConfigMetadata` / `extractMetadata` / `discoverFiles` are **deleted**; the catalog is the single parse path. | The seeder parsed each file twice. FR-1.7 requires one parse path so the drift comparison and the boot import can never disagree about a file's contents. |
| D6 | 404/409 are written by the handler from sentinel errors, via a local `writeJSONAPIError`. | `server.WriteErrorResponse` maps everything to 500. The 400 path reuses the existing `validationFailureError` branch verbatim so validation failures render identically across create/update/reseed. |
| D7 | Drift is computed inside the processor, by mapping over the providers that already exist. | One query path, not two. `AllProvider`'s existing `ParallelMap` carries over. |
| — | Badge wording is **"Differs from image"**. | "Outdated" asserts the row is wrong, which is false when an operator edited it deliberately. "Behind seed" implies an ordering that does not exist. NFR-4 requires the flag not read as an error state. |
| — | The action lives in the detail **layout** header, beside `ConfigExportButton`. | Both act on the whole template document, not the sub-tab being viewed; the layout gives them every sub-tab with no per-page wiring. No new destructive-action grouping is introduced for one button. |
| — | `Revision` clears `Id` rather than trusting `json:"-"` — a deliberate strengthening of FR-2.2. | Costs one assignment and keeps the FR-2.2 test meaningful instead of tautological. |

---

## 4. The load-bearing invariant

Everything about false-positive-free drift reduces to one chain. Let `C = Marshal(Normalize(Unmarshal(fileBytes)))`.

1. **Shipped side.** `LoadCatalog` parses the file, `Revision` normalizes and marshals → `sha256(C)`.
2. **Persist side.** `Create` normalizes then marshals the same parsed model → `Entity.Data = C`, byte-identical.
3. **Stored side.** `Make` unmarshals `Data` and re-normalizes; `Revision` marshals → `sha256(Marshal(Unmarshal(C)))`.

Step 3 equals step 1 **iff `Marshal ∘ Unmarshal` is the identity on `C` for this struct tree**. It is, for four reasons, each of which is a real hazard someone could reintroduce:

- **Unmodelled keys** are dropped on parse — but *symmetrically*, since `C` was produced by the same parse.
- **`map[string]interface{}` options** (`socket/handler/rest.go:18`, `socket/writer/rest.go:10`): Go marshals map keys in sorted order, and `encoding/json` formats `float64` with `'f'` notation for `1e-6 ≤ |x| < 1e21` — which covers every opcode, id and count these tables carry, so `123` round-trips as `123`, not `1.23e+02`.
- **`nil` vs empty slices**: `socket.Normalize` coerces the four socket collections to `[]`. `NPCs`, `Worlds`, `Characters.Templates` and `Characters.Presets` have no `omitempty`, so an absent key becomes `nil` and marshals as `null` on **both** sides. Five of the eleven seeds omit `npcs` (gms 12/87/92/95, jms 185) — the documented `"npcs": null` `sortTemplate` already compensates for. Symmetric, therefore invisible to the revision.
- **`RestModel.Id`** is cleared by `Revision` before marshalling.

`TestShippedSeedsReportNoDrift` (Task 4) is what actually holds this — prose is not the guard. If it ever fails on `StoredRevision != ShippedRevision`, the round-trip identity broke; **fix the round-trip, do not weaken the test.**

---

## 5. Dependency order

```
Task 1  Revision
   └─ Task 2  Catalog ────────────────┐
Task 3  canonicalBytes ───────────┐   │
   └─ Task 4  ViewRestModel + drift ◄─┘   (needs 1, 2)
        └─ Task 5  ReseedById              (needs 2, 3)
             └─ Task 6  REST surface       (needs 4, 5)
Task 7  seeder + main.go                   (needs 2)
Task 8  Go verification gate               (needs 1-7)
Task 9  UI types + service
   └─ Task 10 useReseedTemplate
        └─ Task 11 TemplateReseedButton
Task 12 drift badge                        (needs 9)
Task 13 config-export strip                (needs 9)
Task 14 docs + full sweep                  (needs everything)
```

Tasks 9-13 depend only on Task 9's type change and can start in parallel with the Go work if you want; nothing in the UI needs a running server.

---

## 6. Traps

- **Do not add a JSON-tagged field to `RestModel`.** See D3. `TestComputedAttributesAreNotPersisted` catches it, but the failure mode (universal phantom drift) is subtle enough to be worth naming twice.
- **Do not route re-seed through `UpdateById`.** See D4. `TestReseedProducesSameBytesAsFreshCreate` catches it.
- **`go.mod` must not change.** Everything used is stdlib (`crypto/sha256`, `encoding/hex`, `encoding/json`, `sync`, `sort`, `os`, `path/filepath`). If it changes, `docker buildx bake atlas-configurations` becomes mandatory per CLAUDE.md item 4.
- **Seven `seeder_test.go` tests must be deleted, not "kept passing."** The design's §8 claim that existing seeder tests survive is wrong for `TestDiscoverFiles`, `TestDiscoverFilesSorting`, `TestDiscoverFilesOnlyJson`, `TestExtractMetadata`, `TestExtractMetadataGmsV84`, `TestGmsV84DistinctFromV83`, `TestSeedDataDiscoversBothV83AndV84` — they call methods D5 deletes. Task 2 carries their coverage forward against `LoadCatalog`, including the GMS 83.1-vs-84.1 distinctness assertion.
- **`exactOptionalPropertyTypes` is on in atlas-ui** (the service's own CLAUDE.md note claiming otherwise is stale). The three new attributes are `?:`; the badge checks `=== true`, and the button checks `Boolean(shippedRevision)`. `npm run build` type-checks tests, so a violation surfaces there and not in vitest.
- **`tools/lint.sh --check` false-fails without nvm on PATH**, and under cross-worktree golangci-lint lock contention. If it fails on the atlas-ui leg, run fix mode first and re-check.
- **The four template guards do not apply** — no file under `seed-data/templates/` is modified by this task. Verify with `git diff --name-only main... -- services/atlas-configurations/seed-data/`.
- **`InitShippedCatalog` is `sync.Once`-guarded**, so tests in the `templates` package that call it share one catalog for the whole run. The resource tests deliberately init from the real seed corpus; the processor tests use `LoadCatalog` directly against a fixture directory so they are not order-dependent.

---

## 7. Out of scope

Restating the PRD, unchanged: **tenant-configuration drift** (owned by `task-189-tenant-config-seed-provisioning`, and more damaging — an unmapped opcode is silently dropped by the channel dispatcher), **automatic reconcile**, **arbitrary JSON import** (deferred by `task-199-config-json-export`), **a diff preview**, **bulk re-seed**, **merge/additive semantics**, and **alerting** (NFR-4: the flag is advisory and image-relative, and must not be treated as an alertable condition).

---

## 8. Post-merge remediation (manual, must be in the PR description)

Shipping this task does **not** repair the drifted rows. After deploy:

1. The GMS 87.1, GMS 92.1, GMS 95.1 and JMS 185.1 templates in `atlas-main` should report `seedDrift: true` and render the "Differs from image" badge.
2. Re-seeding each one restores `QuestActionHandle`.

If a row does *not* report drift after deploy, the catalog did not load — check the `main.go` INFO/WARN line carrying the resolved directory and the entry count.
