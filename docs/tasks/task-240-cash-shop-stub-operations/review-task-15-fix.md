# Review: Task 15 fix round 1 (commit 863252f)

Scope: `d05dd7ff0..863252f` only. Original review at
`docs/tasks/task-240-cash-shop-stub-operations/review-task-15.md`; the fix commit's
job was to close its single blocking finding (`ParseItemId` hardcoded to `itemId`
while the route uses `packageId`, causing `GET /data/cashPackages/{packageId}` to
return 400 unconditionally, hiding the 404-on-miss path).

`git diff --stat d05dd7ff0..863252f`:

```
services/atlas-data/atlas.com/data/cashpackage/resource.go       |   2 +-
services/atlas-data/atlas.com/data/cashpackage/resource_test.go  | 194 +++++++++++++++++++++
services/atlas-data/atlas.com/data/data/processor.go             |   4 +-
services/atlas-data/atlas.com/data/rest/handler.go                |   4 +
4 files changed, 202 insertions(+), 2 deletions(-)
```

Exactly the four files the task described. No scope creep.

## 1. The route actually works now — PASS (verified by execution)

`services/atlas-data/atlas.com/data/cashpackage/resource.go:55` now wraps the
detail handler in `rest.ParsePackageId` instead of `rest.ParseItemId`.

Ran `TestCashPackageResourceIntegration` against the fix commit's code
(`go test ./cashpackage/... -run TestCashPackageResourceIntegration -v`):

```
--- PASS: TestCashPackageResourceIntegration (0.01s)
    --- PASS: TestCashPackageResourceIntegration/GetCashPackagesEndpoint (0.00s)
    --- PASS: TestCashPackageResourceIntegration/GetCashPackageById_Hit (0.00s)
    --- PASS: TestCashPackageResourceIntegration/GetCashPackageById_Miss (0.00s)
    --- PASS: TestCashPackageResourceIntegration/GetCashPackageById_BadId (0.00s)
```

`GetCashPackageById_Hit` asserts `200` + correct JSON:API body for a seeded
package, `GetCashPackageById_Miss` asserts `404` for an unseeded id, and
`GetCashPackageById_BadId` asserts `400` for a non-numeric path segment. All
three genuinely exercise the route through a real `httptest.Server` +
`mux.Router` built by `InitResource`, not a hand-called handler. The
404-on-miss path — the half the original defect fully hid — is asserted
directly at `resource_test.go:106`.

## 2. Independent RED reproduction — PASS

Reverted the fix in a throwaway edit (`sed -i 's/rest.ParsePackageId(d.Logger()/rest.ParseItemId(d.Logger()/' cashpackage/resource.go`)
and reran the same test:

```
--- FAIL: TestCashPackageResourceIntegration (0.02s)
    --- PASS: TestCashPackageResourceIntegration/GetCashPackagesEndpoint (0.00s)
    --- FAIL: TestCashPackageResourceIntegration/GetCashPackageById_Hit (0.00s)
        Error: Not equal: expected: 200, actual: 400
    --- FAIL: TestCashPackageResourceIntegration/GetCashPackageById_Miss (0.00s)
        Error: Not equal: expected: 404, actual: 400
    --- PASS: TestCashPackageResourceIntegration/GetCashPackageById_BadId (0.00s)
```

Both the Hit and Miss subtests fail red against the pre-fix wiring, exactly
as the original defect predicts (both would return the same wrong 400).
Restored the file with `git checkout -- .../cashpackage/resource.go`;
`git diff --stat` after restore is empty for the file. `go build ./... &&
go test ./...` for the whole `atlas-data` module (post-restore) is green.

## 3. `rest.ParsePackageId` correctness/idiom — PASS

`services/atlas-data/atlas.com/data/rest/handler.go:78-80`:

```go
func ParsePackageId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "packageId", next)
}
```

Byte-for-byte the same shape as every sibling helper in the file
(`ParseQuestId` at line 66, `ParseFaceId` at line 70, `ParseHairId` at line
74, etc.): same generic instantiation `server.ParseIntId[uint32]`, same
signature, same mux-var-string-literal pattern, no divergent error handling
or logging introduced. The other three helpers (`ParseEquipmentId`,
`ParseMapId`, ... through `ParseHairId`) are untouched in the diff — this is
a pure append at the end of the file, no reordering, no edits to existing
callers.

## 4. `data/processor.go` change — non-blocking finding, does not regress but does not achieve its stated purpose

`services/atlas-data/atlas.com/data/data/processor.go:176-180`:

```go
} else if name == WorkerCommodity {
	err = p.RegisterFileData(path, filepath.Join("Etc.wz", "Commodity.img.xml"), commodity.NewProcessor(p.l, p.ctx, p.db).RegisterCommodity)()
	if err == nil {
		err = p.RegisterFileData(path, filepath.Join("Etc.wz", "CashPackage.img.xml"), cashpackage.NewProcessor(p.l, p.ctx, p.db).RegisterCashPackage)()
	}
}
```

`RegisterFileData`'s implementation (`data/processor.go:302-307`):

```go
func (p *ProcessorImpl) RegisterFileData(rootDir string, wzFileName string, rf RegisterFunc) Worker {
	return func() error {
		rf(filepath.Join(rootDir, wzFileName))
		return nil
	}
}
```

`RegisterFileData` discards `rf`'s return value and *always* returns `nil`.
Consequently `err` after the first `RegisterFileData(...)()` call is always
`nil` regardless of whether `RegisterCommodity` internally failed, so the
added `if err == nil` guard is unconditionally true — the CashPackage
register is attempted on every invocation exactly as it was before this
commit. The gating described in the implementer's report ("only attempted
if the Commodity register succeeded") is not actually implemented; it is
dead code that has no observable effect.

This is **not a regression**: behavior for this legacy `StartWorker` path
(confirmed live — it is reachable from `kafka/consumer/data/consumer.go:45`
via the `COMMODITY` start-worker command) is identical before and after the
commit — both always attempt both registers in order, and any
missing-file/registration error from either is silently swallowed by
`RegisterFileData` either way, matching the tolerance-for-missing-file
requirement. `data/workers/commodity.go` (the newer Worker-interface path)
is untouched by this commit (`git diff d05dd7ff0..863252f --
.../data/workers/commodity.go` is empty) and still uses its own
log-and-continue (`Warnf`) pattern for a missing `CashPackage.img.xml`,
independent of this change.

Flagging this because the commit message and the implementer's report claim
a sequencing fix that the code does not actually deliver — worth a follow-up
if the intent (skip cash-package ingest when commodity ingest fails) is a
real requirement, but it is out of the blocking finding's scope and does not
break anything that worked before.

## 5. Scope — PASS

Only the four files listed above changed. No edits outside
`rest/handler.go`, `cashpackage/resource.go`, `cashpackage/resource_test.go`,
`data/processor.go`.

## Verification

- `go build ./...` (module `atlas-data`): exit 0.
- `go test ./...` (module `atlas-data`): all packages pass (post RED/GREEN
  round-trip and restore).

## Not evaluable

None — all five checklist items were verified by execution or direct
reading within the fix commit's diff.

## Verdict

APPROVED_WITH_FINDINGS. The blocking defect from the original review is
genuinely closed and independently reproduced as RED→GREEN. One non-blocking
finding on the `data/processor.go` change: the `if err == nil` gate is dead
code (`RegisterFileData` always returns `nil`), so it does not implement the
sequencing it claims to, though it introduces no regression.
