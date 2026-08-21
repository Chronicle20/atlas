# Review — fix round 1 (backend guidelines audit, 4 blocking findings)

Range reviewed: `1104053da..2413cbf29` (3 commits)
Brief: `.superpowers/sdd/plan/fix-round-1-brief.md`
Report: `.superpowers/sdd/plan/fix-round-1-report.md`

## Scope confirmed

`git diff --stat 1104053da..2413cbf29` shows exactly the 7 files the brief
named: 2 new + 1 shrunk file under `atlas-channel/maps/location` (Fix 1), 2
`newTestDB` edits (Fix 2), and 2 new `testmain_test.go` files (Fix 3). Three
commits, one per fix, all conventional-commit scoped correctly
(`refactor(atlas-channel)`, `test(atlas-maps)` x2). No file outside the
brief's list was touched. This matches the work described — no scope
mismatch.

## Fix 1 — split `maps/location/requests.go` into `rest.go`/`model.go`/`requests.go`

**PASS — verified as a pure move, not just plausible.**

- `git diff 1104053da..2413cbf29 -- .../maps/location/` shows `rest.go` and
  `model.go` as pure additions whose content is byte-identical to the
  deletions in `requests.go`'s diff (same struct tags, same doc comments,
  same method bodies) — no hunk shows an add+delete pair on the same
  content, i.e. nothing was rewritten in transit.
- Independently re-verified by extracting every `^(func|type|const|var)`
  declaration from the pre-image `requests.go` (`git show 1104053da:...`)
  and diffing that set against the concatenation of the three post-image
  files: `diff` returns empty (exit 0). Every declared symbol exists exactly
  once post-split, with an identical signature.
- `go build ./...` and `go test ./maps/... -v` in atlas-channel: all 9
  `location` package tests pass unmodified (`atlas-channel/maps/location
  0.022s`), and `gofmt -l` reports no formatting drift on the 3 files.
- **`GetField`/`SetBaseURLForTest` placement in `requests.go`** (disclosed
  deviation from the brief's prose, which enumerated only `ErrNotFound`,
  `baseURLProvider`, `requestByCharacterId`, `Get`): correct call. Neither
  function is `RestModel`-shaped (→ not `rest.go`) nor a `Model`
  accessor/constructor (→ not `model.go`). `GetField` calls
  `requestByCharacterId` directly and returns `field.Model`, so it is
  request plumbing by the brief's own criterion; `SetBaseURLForTest` mutates
  `baseURLProvider`, which lives in `requests.go`. Keeping both there is the
  only placement consistent with the brief's stated three-way split.

## Fix 2 — register tenant callbacks in two GORM `newTestDB` helpers

**PASS, with the ordering risk resolved by evidence, not by the implementer's own tests-are-green claim.**

- Both edits (`character/location/processor_test.go:112`,
  `kafka/consumer/cashshop/consumer_test.go:34`) place
  `database.RegisterTenantCallbacks(l, db)` **after** `Migration(db)` /
  `location.Migration(db)`, diverging from both reference files (which place
  it before). This is exactly the risk the task flagged: a green suite is
  not evidence the registration is doing anything, since a no-op
  registration would also leave tests green.
- Read `libs/atlas-database/tenant_scope.go`: `RegisterTenantCallbacks`
  registers `gorm.DB` hooks on `Callback().Query()/Create()/Update()/Delete()`
  (record-level operations). `character/location/entity.go`'s `Migration`
  is `db.AutoMigrate(&entity{})` — schema DDL only, no record
  Create/Query/Update/Delete goes through the callback chain during
  `AutoMigrate`. GORM's callback registry is attached to the `*gorm.DB`
  config and is shared across sessions/clones derived from that same `db`,
  so registering after a schema-only migration and before any actual test
  query is functionally equivalent to registering before.
- This was confirmed empirically, not just reasoned about: ran
  `go test ./character/location/... -run TestSetIsTenantScoped -v`
  (`processor_test.go:178`, an existing test that creates a row under
  tenant A and asserts tenant B cannot see it). Output shows the actual SQL
  executed: `SELECT * FROM character_locations WHERE (tenant_id = "..." AND
  character_id = 7) AND character_locations.tenant_id = "..." ... ` followed
  by `record not found` — the tenant filter clause is present and doing
  real work, and the test passes. This is direct evidence the callback
  registered after `Migration` is genuinely active, not evidence-by-inertia.
- Full `atlas-maps` suite (`go test ./... -count=1`, isolated `GOCACHE`):
  all packages pass, no regressions from adding the filter to two
  previously-unscoped bootstraps (which would have surfaced as a newly
  failing assertion per the brief's stop condition — it did not).

## Fix 3 — noop Kafka producer in consumer test packages

**PASS — production diff confirmed empty, timings re-measured and consistent.**

- `git diff --name-only 1104053da..2413cbf29 -- services/atlas-maps/atlas.com/maps | grep -v '_test.go$'` returns nothing — zero non-test `.go` files changed anywhere in `atlas-maps`. `kafka/consumer/cashshop/consumer.go`, `kafka/consumer/character/consumer.go`, and `main.go` are untouched, confirmed by an explicit empty `git diff` against each.
- Both new `testmain_test.go` files are diffed byte-for-byte against
  `services/atlas-monster-book/.../kafka/consumer/character/testmain_test.go`
  (the named reference) — identical except the `package` line, which is the
  expected variation.
- `grep -c "func TestMain"` confirms exactly one `TestMain` per package (no
  duplicate/competing `TestMain` from another file).
- Re-measured independently: `go test ./kafka/consumer/cashshop/...
  ./kafka/consumer/character/... -count=1` with an isolated `GOCACHE` →
  `ok atlas-maps/kafka/consumer/cashshop 0.427s`,
  `ok atlas-maps/kafka/consumer/character 0.897s`. Same order of magnitude
  as the implementer's reported 0.586s/0.707s (small run-to-run variance is
  expected), both comfortably under the brief's "well under a second or
  two" bar. The reported ~114.5s/~40s before-numbers were not
  re-measured (would require reverting the fix, which is out of scope for a
  non-mutating review), but they are consistent with the brief's
  independently-diagnosed root cause (10x retry with exponential backoff at
  ~42s/emit against an unset `BOOTSTRAP_SERVERS`).

## Build / test verification (module-local, isolated GOCACHE)

- `atlas-channel/atlas.com/channel`: `go build ./...` clean;
  `go test ./...` — all packages pass.
- `atlas-maps/atlas.com/maps`: `go build ./...` clean; `go test ./...` — all
  packages pass, including the two Fix 3 target packages and the two Fix 2
  target packages.

## Not evaluable

- The implementer's reported "before" timings (114.535s / 39.992s) were not
  independently reproduced, since doing so would require reverting the
  `testmain_test.go` files (a source mutation), which is explicitly
  forbidden for this review. The root-cause mechanism (unconfigured
  `BOOTSTRAP_SERVERS` → retry backoff) was pre-diagnosed per the brief and
  not re-litigated here.

## Verdict

All three fixes hold up under independent verification, not just re-reading
the report. Fix 1 is a genuine pure move (symbol-set diff is empty). Fix 2's
riskiest claim — that callback order doesn't matter — is now backed by a
passing tenant-isolation test whose executed SQL was inspected directly,
not inferred from "tests are green." Fix 3's production-file diff is
confirmed empty and its new files match the reference verbatim.

No blocking findings.
