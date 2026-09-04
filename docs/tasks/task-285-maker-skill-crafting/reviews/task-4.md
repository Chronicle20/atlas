# Review: Task 4 — `atlas-data` `itemmake` processor, storage, REST resource

Range: `f6d91735e..cda477ff5` (one commit, `cda477ff5`)

## Scope check

`git diff --stat f6d91735e..cda477ff5` touches exactly:
- `itemmake/mock/processor.go` (new, 29 lines)
- `itemmake/processor.go` (new, 61 lines)
- `itemmake/resource.go` (new, 71 lines)
- `itemmake/resource_test.go` (new, 447 lines)
- `main.go` (+2 lines)

This matches the brief's expected file set exactly. Confirmed by `git diff f6d91735e..cda477ff5 -- itemmake/rest.go itemmake/registry.go itemmake/reader.go itemmake/reader_test.go data/processor.go data/processor_test.go README.md` returning empty — none of the out-of-scope files (Tasks 2/3/5) were touched. No scope creep.

## Findings

### 1. No new table / no migration (C-4) — PASS
`itemmake/processor.go:90-92` calls `document.NewStorage(l, db, GetModelRegistry(), "ITEM_MAKE")`, the same shared-`documents`-table pattern as `commodity/processor.go:35-37` with `"COMMODITY"`. No new entity, no migration file in the diff.

### 2. Idempotency is structural, not hand-rolled (FR-1.6) — PASS
`Register` (`itemmake/processor.go:101-112`) is a plain per-item loop calling `s.Add(p.ctx)(m)()` with no existence check, no delete-then-insert, no dedup map. Idempotency comes entirely from `clause.OnConflict{Columns: [tenant_id, type, document_id], DoUpdates: ...}` in `document/db_storage.go:120-157`, confirmed by reading. `TestRegisterIsIdempotent` (`itemmake/resource_test.go:604-646`) calls `processor.Register` twice against the same storage and asserts the `documents` row count for `type = 'ITEM_MAKE'` is unchanged, then re-fetches `1082002` via REST and checks two scalar attributes still round-trip. This is a real regression pin, not a tautology — if a future change replaced the OnConflict upsert with plain `Create`, the second `Register` call would either error (unique constraint) or double the row count, and the test would fail either way.

### 3. Collection route paginates — PASS
`handleGetItemMakesRequest` (`itemmake/resource.go:151-173`) uses `paginate.ParseParams` + `s.AllPagedProvider` + `server.MarshalPaginatedResponse[[]RestModel]`, identical shape to `commodity/resource.go:32-52`. No bare `GetAll`/`MarshalResponse[[]...]` present. `TestListItemMakesPaginates` (`itemmake/resource_test.go:561-602`) seeds 25 rows, requests `page[number]=2&page[size]=10`, and asserts `len(data)==10`, `meta.total==25`, and `links.next` is present — a real assertion of the pagination envelope, not just a 200.

### 4. Interface/mock parity — PASS
`Processor` (`itemmake/processor.go:69-72`) declares `Register` and `RegisterItemMake`. `mock/processor.go:30-49` implements both, nil-checked, with `var _ itemmake.Processor = (*ProcessorMock)(nil)` — matches `commodity/mock/processor.go` method-for-method, including the compile-time interface assertion.

### 5. `RegisterItemMake` satisfies `data.RegisterFunc` — PASS
`data/processor.go:226`: `RegisterFunc func(filePath string) error`. `itemmake/processor.go:114-116`: `func (p *ProcessorImpl) RegisterItemMake(path string) error` — signature matches (param name differs, irrelevant for a func type).

### 6. Routes match the brief's table — PASS
`itemmake/resource.go:144-146`: `GET /data/item-makes` (list) and `GET /data/item-makes/{itemId}` (single), registered via `router.PathPrefix("/data/item-makes").Subrouter()`, matching the commodity template's route-registration shape (`commodity/resource.go:18-31`).

### 7. `main.go` wiring — PASS
Diff adds exactly one import (`"atlas-data/itemmake"`, alphabetically placed between `item` and `job`) and one `AddRouteInitializer(itemmake.InitResource(db)(GetServer())).` line, placed immediately after `cashpackage` and before `etc`, matching the brief's prescribed insertion point exactly (`main.go:684`).

### 8. `Group` (and all `RestModel` fields) survive the REST round-trip — PASS
`itemmake/rest.go` (unmodified, Task 2) declares `Group uint32 \`json:"group"\`` on `RestModel`; nothing in this diff drops it from serialization (no custom marshalling is introduced — `server.MarshalResponse[RestModel]` / `MarshalPaginatedResponse[[]RestModel]` serialize the struct as-is). `TestGetItemMakeById` (`itemmake/resource_test.go:443`) asserts `attributes["group"] == 1` for `1082002`, and `TestGetItemMakeByIdFromEachGroup` (`itemmake/resource_test.go:473-506`) asserts the correct group for all six group ids (`0,1,2,4,8,16`), directly covering the downstream `atlas-maker` group-`0` lookup contract.

### 9. Ordered list fields survive REST (FR-1.3, FR-1.4) — PASS
`TestGetItemMakeById` asserts `recipe` and `reqQuest` element-by-element in seeded order (`resource_test.go:453-470`). `TestGetItemMakeRandomRewardOrderSurvivesREST` asserts `randomReward` in exact seeded order (`resource_test.go:524-559`). Both are real JSON-decoded-and-indexed assertions, not membership checks.

### 10. 404 handling — PASS
`handleGetItemMakeRequest` (`itemmake/resource.go:175-193`) writes `http.StatusNotFound` on a storage miss. `TestGetItemMakeByIdNotFound` (`resource_test.go:508-522`) asserts this against an empty store.

### 11. Template fidelity — PASS
Diffed `itemmake/processor.go` against `commodity/processor.go` and `itemmake/mock/processor.go` against `commodity/mock/processor.go` side by side: identical shape, substitutions limited to `RestModel`/`Read`/`"ITEM_MAKE"`/`RegisterItemMake` as prescribed. The task-076 no-outer-transaction comment is preserved verbatim (`itemmake/processor.go:94-100`). `resource.go`'s two handlers mirror `commodity/resource.go`'s `handleGetCommodityItemsRequest`/`handleGetCommodityItemRequest` exactly (the commodity file has a third by-item route that itemmake correctly omits, since it isn't in the brief's route table).

### 12. `libs/atlas-constants` — N/A
No new domain type, alias, or numeric constant introduced; `RestModel` and `GetModelRegistry` are reused from Task 2 unmodified.

### 13. No stray `go` statements, no placeholders — PASS
No background goroutines introduced by this diff at all (none needed for a synchronous REST resource + processor); no TODO/stub found by inspection.

### 14. Test DB scaffolding — PASS, matches established pattern
`testDocumentEntity` in `resource_test.go:227-238` (a sqlite-compatible shadow of `document.Entity` sans Postgres `uuid_generate_v4()` default) is the same pattern already used in `commodity/resource_test.go:27-36` — not an invented shape.

## Non-blocking notes

- `itemmake/resource_test.go:412` (`TestRegisterIsIdempotent`) constructs `&ProcessorImpl{l: l, ctx: ctx, db: db}` directly rather than via `NewProcessor(l, ctx, db)`. This is an internal, same-package test needing concrete-type field access it wouldn't get through the `Processor` interface returned by `NewProcessor`, so it isn't a Builder-pattern violation (no domain model is being hand-built) — noting only because `NewProcessor` exists and is otherwise unused in the diff's own tests.

## Not evaluable

- None. The full diff, the template it copies, and every referenced storage/registry method signature were read and cross-checked within this review's surface.

## Verdicts

- **Spec-compliance verdict: APPROVED.** All FR-1.6 idempotency, C-4 no-migration, route table, and downstream `Group`-field requirements are met and pinned by tests.
- **Task-quality verdict: APPROVED.** Tests assert real HTTP/JSON:API behavior (pagination envelope contents, element-order round-trips, 404, idempotent row-count) rather than bare 200s; the idempotency test would fail if the OnConflict upsert regressed.
