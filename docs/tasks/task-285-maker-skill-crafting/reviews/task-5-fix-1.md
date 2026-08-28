# Task 5 Re-Review: Fix Round 1

**Commit:** `6ff166527` "fix(data): prove StartWorker actually reaches the ITEM_MAKE branch"

## Scope Confirmed

Single file changed: `services/atlas-data/atlas.com/data/data/processor_test.go` (128 insertions, 2 deletions).  
No production code was modified.

## Finding Assessment

**Blocking Issue from Original Review:**
TestStartWorkerDispatchesItemMake previously asserted only `StartWorker(WorkerItemMake, tmpDir) == nil`, which would pass identically whether or not the `WorkerItemMake` dispatch branch exists in `StartWorker` — because:
- `StartWorker` at line 118 initializes `var err error` to nil
- If the `WorkerItemMake` condition (processor.go:216–217) is absent, none of the if/else if statements match
- `err` remains nil
- Line 219 check `if err != nil` is false
- Function returns nil
- `RegisterFileData` (processor.go:306–310) discards the return value from `rf(...)` anyway

## RED/GREEN Verification

The fix is **sensitivity-verified** for the specific regression (branch deletion):

**GREEN Path (branch exists, test passes):**
1. Test calls `p.StartWorker(WorkerItemMake, tmp)` on line 232
2. Control reaches the `name == WorkerItemMake` case (processor.go:216)
3. Calls `p.RegisterFileData(path, "Etc.wz/ItemMake.img.xml", itemmake.RegisterItemMake)()`
4. RegisterFileData returns a Worker that calls `RegisterItemMake(tmp/Etc.wz/ItemMake.img.xml)`
5. RegisterItemMake reads the fixture XML and inserts a document into the database for item 0x04260000 (4260000 decimal)
6. Test queries `WHERE tenant_id = ? AND type = ? AND document_id = ?` with type="ITEM_MAKE" and document_id=4260000
7. Row exists, query succeeds, test **PASSES**

**RED Path (branch deleted, test fails):**
1. Test calls `p.StartWorker(WorkerItemMake, tmp)`
2. None of the if/else if conditions match (WorkerItemMake branch absent)
3. `err` remains nil (zero value)
4. Line 219 check is false, function returns nil
5. Test passes the first assertion: `if err = p.StartWorker(...); err != nil { ... }`
6. Test queries the database for document_id=4260000, type="ITEM_MAKE"
7. No row exists (RegisterItemMake was never called)
8. GORM `.First(&row)` returns `gorm.ErrRecordNotFound`
9. Test **FAILS** with message: `expected an ITEM_MAKE document for item 4260000 to have been persisted by the ITEM_MAKE worker branch, got: record not found`

The test now checks that the branch is not only named but actually **called and produces observable work** (a persisted database row). This is non-vacuous.

## Test Integrity

**Original assertion preserved:** Line 232–234 still asserts `StartWorker(...) == nil`, unchanged from the original test.

**New assertion added:** Lines 236–238 query the database and assert that exactly the document that only `RegisterItemMake` could have produced was persisted with the correct schema and ID.

**Setup enhancements:**
- **Database schema** (lines 52–62): `itemMakeDocumentEntity` mirrors production's `document.Entity` schema, including the composite unique index (`idx_documents_tenant_type_docid`) that the production code's ON CONFLICT upsert depends on
- **UUID function** (lines 35–48): `uuid_generate_v4` is registered in the SQLite driver via `ConnectHook`, matching the Postgres default that production's `document.DbStorage.Add` relies on (schema default `uuid_generate_v4()`). Without this function, INSERT silently fails (RegisterFileData discards the error), so nothing would be persisted regardless of whether the branch runs
- **Fixture XML** (lines 182–197): Valid minimal ItemMake.img.xml with item 0x04260000 (4260000 decimal), group 0, and recipe fields

**Isolated database:** Each test run gets its own in-memory SQLite database (line 77, DSN uses a unique UUID), preventing cross-test collision with other tests' migrations of the same "documents" table name.

**Tenant context:** Test stores `tenantId` in a variable (line 210) and uses it in the query (line 237), ensuring the assertion correctly binds tenant_id rather than accidentally checking a different tenant's data.

## No Weakening

- No pre-existing assertions were removed or weakened
- First assertion (`StartWorker == nil`) remains at line 232
- Test setup (tenant creation, logger, context) unchanged from original
- Only additions: real database, fixture file, schema migration, and the substantive database query assertion

## Verdict

The blocking finding is **ADDRESSED**. The test is now non-vacuous for the specific regression: it fails if the `WorkerItemMake` branch is deleted or if `RegisterItemMake` is not called, because the row that only that call produces will not exist in the database.

No production code changes, no weakening of pre-existing assertions, no new syntax errors.
