# Review — Task 15: Cash package catalogue in `atlas-data`

Commit range: `08358225e..d05dd7ff0` (single commit `d05dd7ff0`)
Brief: `.superpowers/sdd/plan/task-15-brief.md`
Report: `.superpowers/sdd/plan/task-15-report.md`

## Scope confirmed

`git diff --stat 08358225e..d05dd7ff0` shows exactly the 9 files the brief
named: the six new `services/atlas-data/atlas.com/data/cashpackage/*` files
plus the three one-line wiring edits (`data/processor.go`,
`data/workers/commodity.go`, `main.go`). No extraneous diff. Matches the
report's own file list.

## Findings

### BLOCKING — `GET /data/cashPackages/{packageId}` always returns 400, never reaches the 404 path

`services/atlas-data/atlas.com/data/cashpackage/resource.go:24` registers the
route var as `packageId`:

```go
r.HandleFunc("/{packageId}", registerGet("get_cash_package", handleGetCashPackageRequest(db))).Methods(http.MethodGet)
```

But `resource.go:55` parses the id through `rest.ParseItemId`, which is a
hardcoded wrapper for a *different* var name:

```go
// rest/rest.go
func ParseItemId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "itemId", next)
}
```

`server.ParseIntId` (`libs/atlas-rest/server/id_parser.go:16-26`) reads
`mux.Vars(r)["itemId"]`. Since the route only ever sets `packageId`,
`mux.Vars(r)["itemId"]` is always the empty string, `strconv.Atoi("")`
always fails, and the handler writes `400 Bad Request` before `next` (the
package's own `handleGetCashPackageRequest` body, which contains the `404`
logic) is ever invoked. Every request to `GET /data/cashPackages/{packageId}`
— valid id, invalid id, or missing package — returns 400, never the 404 the
brief's route-shape requirement (item 5) calls for, and never a successful
200 for an existing package.

This is a straight copy of `commodity/resource.go`'s
`rest.ParseItemId(...)` call without noticing that commodity's own route var
genuinely is `{itemId}` while cashpackage's is `{packageId}` — the fix is a
dedicated `rest.ParsePackageId` (mirroring `rest.ParseQuestId` /
`rest.ParseFaceId` / `rest.ParseHairId`, i.e.
`server.ParseIntId[uint32](l, "packageId", next)`), or an inline
`server.ParseIntId[uint32](d.Logger(), "packageId", ...)` call in
`resource.go`.

**Reproduced independently** (not from the report, which never tests
`resource.go` since no `resource_test.go` exists for this package — unlike
`commodity/resource_test.go`/`cash/resource_test.go`, which do exist for
their siblings):

```go
// scratch test, added then deleted; not committed
h := rest.ParseItemId(l, func(id uint32) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }
})
req := httptest.NewRequest(http.MethodGet, "/data/cashPackages/9100000", nil)
req = mux.SetURLVars(req, map[string]string{"packageId": "9100000"})
rr := httptest.NewRecorder()
h(rr, req)
// rr.Code == 400, handler body never called
```

Ran via `go test ./cashpackage/... -run TestVarNameMismatch -v`:
`status=400 called=false`. Confirmed, then the scratch file was deleted
(`git status --porcelain` clean afterward).

This is the single-item GET endpoint the brief explicitly required in item
5 ("`404` on a miss"), and it is unreachable as landed.

## Mutation testing (per the standing review bar on this branch)

All three mutations applied to `cashpackage/reader.go`, confirmed RED, then
restored; `git diff --stat` on the module was empty after each restore and
is empty now (`git diff --stat -- services/atlas-data` returns nothing).

1. **Reproduced independently** — inverted the SN-lookup condition
   (`err == nil` → `err != nil`): panics with a nil-pointer dereference on
   `snNode.IntegerNodes` (`snNode` is `nil` when `ChildByName` *does* find
   `SN`, because the branch now only enters on the not-found path).
   `go test ./cashpackage/...` → `FAIL` (panic), matches the report's claim.
2. **Reproduced independently** — removed the
   `m.SerialNumbers = make([]uint32, 0)` initialization: the no-`SN` package
   (`9100002`) now yields `nil` instead of `[]uint32{}`.
   `go test ./cashpackage/...` →
   `reader_test.go:54: rms[2].SerialNumbers = nil, want non-nil (possibly empty) slice`,
   `FAIL`. Matches the report's claim.
3. **New mutation, not in the report** — changed
   `m.SerialNumbers = append(m.SerialNumbers, uint32(val))` to
   `append(m.SerialNumbers, uint32(val)+1)`, corrupting every parsed serial
   number by 1: `go test ./cashpackage/...` →
   `reader_test.go:61: rms[0].SerialNumbers[0] = 10001, want 10000`, `FAIL`.
   This proves `TestReadCashPackages` asserts concrete `SerialNumbers`
   values in order (not just length/shape) — the test is not a can't-fail
   test for the values it claims to pin.

## Checklist verification

1. **Empty slice, never nil.** `reader.go:26` (`m.SerialNumbers = make([]uint32, 0)`)
   unconditionally initializes the slice before the `SN` lookup, so a
   package with no `SN` child keeps the empty, non-nil slice. Verified in
   source (not only the test) and via mutation 2 above. PASS.

2. **Missing-file tolerance on both ingest paths.**
   - `data/processor.go:298-303` (`RegisterFileData`) discards `rf(...)`'s
     return value and always returns `nil` — read directly, confirms the
     brief's and report's claim. The second `RegisterFileData` call added at
     `data/processor.go:177` for `CashPackage.img.xml` inherits this
     tolerance for free. (Note: the second call overwrites `err` from the
     first `RegisterFileData` call at line 176, but since both calls always
     return `nil`, this has no functional effect — it is the same
     `err = ...; err = ...` shape already used elsewhere in this `if/else if`
     chain, e.g. `WorkerCharacterCreation`.) PASS.
   - `data/workers/commodity.go:41-44` — the second `RegisterCashPackage`
     call's error is logged via `l.WithError(err).Warnf(...)` and NOT
     returned, matching the surrounding idiom (compare the `MobSkill`
     worker's `mobskill.InitString` handling in `data/processor.go:200-202`,
     same `WithError(...).Warnf(...)`-then-continue shape). `Run` still
     returns `nil` after, so a tenant lacking `CashPackage.img.xml` does not
     fail the Commodity worker run. PASS.

3. **Identifier correctness.** Checked every `xml.Node` accessor and
   package identifier used in `reader.go` against `xml/model.go`:
   - `Node.ChildByName(name string) (*Node, error)` — `xml/model.go:20-27` —
     matches the `snNode, err := cxml.ChildByName("SN"); err == nil` usage.
   - `Node.IntegerNodes []IntegerNode` — `xml/model.go:14` — matches
     `for _, sn := range snNode.IntegerNodes`.
   - `IntegerNode{Name, Value string}` — `xml/model.go:164-167` — matches
     `sn.Value` used in `strconv.ParseUint`.
   - `Node.ChildNodes []Node` — `xml/model.go:12` — matches
     `for _, cxml := range exml.ChildNodes` and `cxml.Name`.
   - `xml.FromPathProvider` / `xml.FromByteArrayProvider` —
     `xml/reader.go:28`, `:51` — match `processor.go`'s and
     `reader_test.go`'s usage respectively.
   All PASS — no invented identifiers in `reader.go`. The one identifier
   problem on this task is not in `reader.go` but in `resource.go` (finding
   above): `rest.ParseItemId` exists and is correctly named, but its
   hardcoded `itemId` var name does not match this package's route.

4. **Scope.** Exactly the nine files the brief named; no extraneous diff.
   PASS.

5. **Route shape.** `GET /data/cashPackages` (paginated, via
   `paginate.ParseParams`/`AllPagedProvider`, `resource.go:23,33-40`) is
   correctly wired and matches commodity's collection-route shape. `GET
   /data/cashPackages/{packageId}` is wired via `AddRouteInitializer` in
   `main.go:194` next to commodity's, and its `404`-on-miss logic at
   `resource.go:59-62` is real code — but it is dead code, because the
   `rest.ParseItemId` var-name mismatch (finding above) means the handler
   body is never reached. FAIL (blocking).

## Not evaluable

None — the review surface (the `cashpackage` package plus the three wiring
edits and the `xml`/`rest`/`document` contracts the diff calls into) was
fully within reach; no part of the checklist was left unverified.

## Verdict rationale

One blocking, concrete, reproduced defect: the single-item GET route is
unreachable under its correct handler logic due to a hardcoded var-name
mismatch inherited from copying `commodity/resource.go`'s
`rest.ParseItemId` call without adjusting for cashpackage's own route
variable name. Everything else — ingest tolerance on both paths, the
`reader.go` accessor identifiers, the empty-vs-nil-slice handling, and scope
— checks out under direct verification and independent mutation testing.
