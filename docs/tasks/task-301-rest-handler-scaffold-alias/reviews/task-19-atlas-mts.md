# Review: Task 19 (atlas-mts) — rest handler scaffold alias + id-parser delegation

Range: `cc0c55dcd..0fd55ce60` (commits `a4bf5c5d0` conversion, `0fd55ce60` id-parser delegation)
Module: `services/atlas-mts/atlas.com/mts`
Precedent compared: `git show ee9e712c2` (atlas-npc-conversations, Task 18)

## Scope

`git diff --stat cc0c55dcd..0fd55ce60` touches exactly 7 files: `holding/resource.go`,
`listing/resource.go`, `rest/handler.go`, `testsupport/resource.go`,
`transaction/resource.go`, `wallet/resource.go`, `wish/resource.go`. Matches the brief's
file list exactly. No test files, `main.go`, or `libs/` touched (confirmed empty diff for
all three — see check 7 below).

## Checks

### 1. Zero `d.DB()` remaining / 18 pre-image sites converted / curry drop

- `grep -rn 'd\.DB()' --include='*.go' services/atlas-mts/atlas.com/mts` → 0 hits. PASS.
- Per-file pre-image counts (`grep -c 'd\.DB()'` on `cc0c55dcd` blobs) matched the brief
  exactly: holding 2, listing 5, testsupport 6, transaction 1, wish 4 = 18 total. PASS.
- `wallet/resource.go`: no `d.DB()` pre-image (confirmed), but its `InitResource` curry
  dropped `registerGet := rest.RegisterHandler(l)(db)(si)` → `rest.RegisterHandler(l)(si)`
  (`git diff cc0c55dcd..a4bf5c5d0 -- .../wallet/resource.go`, 1 line changed). PASS.
- Spot-checked `transaction/resource.go` (single site) and `listing/resource.go` (5 sites,
  including the `handleCancelListing`/`handleGetCharacterActiveListings` Shape A wrap)
  full diffs — Shape A applied per enclosing handler, curry (`db`) dropped from both
  `RegisterHandler`/`RegisterInputHandler` calls in `InitResource`, and passed at each
  registration call site (`handleX(db)`). Consistent with the Task 18 precedent. PASS.

### 2. `rest/handler.go` structural match to Task 18 precedent

Read the file in full post-transform. Type aliases (`HandlerDependency`, `HandlerContext`,
`GetHandler`, `InputHandler[M]`), `RegisterHandler` var, and `RegisterInputHandler` wrapper
are byte-identical in shape to `ee9e712c2`'s `conversations/rest/handler.go`. Import block
after both commits: `net/http`, `jsonapi`, `logrus`, `atlas-constants/world`,
`atlas-rest/server` — all live, no unused imports (`go build` confirms). No local
`ParseInput` present (`grep -rn ParseInput` returns nothing in the module). PASS.

### 3. Commit-split integrity

`git diff --stat a4bf5c5d0..0fd55ce60 -- holding/resource.go listing/resource.go
testsupport/resource.go transaction/resource.go wallet/resource.go wish/resource.go`
→ empty. The second commit touches only `rest/handler.go`. PASS.

### 4. The `db` collision rename in `testsupport/resource.go`

Pre-image (line 274, `cc0c55dcd`): `db := d.DB().WithContext(d.Context())` immediately
followed by `db.Transaction(func(tx *gorm.DB) error {...})`, with the closure body using
only `tx` (verified — no other reference to the local `db` inside the transaction
closure or the rest of `handleSeedListings`).

Post-image (`0fd55ce60:testsupport/resource.go:275,277`):
```
sdb := db.WithContext(d.Context())
txErr := sdb.Transaction(func(tx *gorm.DB) error {
```
The local was renamed to `sdb` at its only declaration and its only use
(`sdb.Transaction`); the outer closed-over `db` parameter and the inner `tx` closure
parameter are both untouched and unshadowed. No missed use found (grep for `sdb\|db\.Transaction\|db :=\|d\.DB()` across the whole file confirms exactly these two `sdb` occurrences and no stray unrenamed local). PASS.

The file's other two named sites, both plain `d.DB()` → closed-over `db` substitutions
inside functions parameterized `func handleExpireListing(db *gorm.DB) rest.GetHandler`
and `func handleRunSweep(db *gorm.DB) rest.GetHandler`:
- `listing.BackdateEndsAt(db.WithContext(d.Context()), listingId, ...)` (line 400) — PASS.
- `task.Sweep(d.Logger(), d.Context(), db)` (line 425) — PASS.
Also confirmed the third `d.DB()` use inside `handleExpireListing`
(`listing.NewProcessor(d.Logger(), d.Context(), d.DB())` in the pre-image) was converted
to `listing.NewProcessor(d.Logger(), d.Context(), db)` (line 390) — this wasn't called out
by name in the brief's per-site description but is one of the file's 6 counted `d.DB()`
sites and is correctly converted.

### 5. The five id-parser helpers

`git diff a4bf5c5d0..0fd55ce60 -- rest/handler.go` shows each of the five hand-rolled
bodies replaced with a one-line delegation:
- `ParseWorldId` → `server.ParseIntId[world.Id](l, "worldId", next)` — pre-image body is a
  bare `mux.Vars` + `strconv.ParseUint(s, 10, 8)` + `world.Id(byte(...))` lookup with no
  extra semantic check beyond the parse-error branch. Delegates cleanly. `world.Id` is
  `type Id byte` (`libs/atlas-constants/world/constants.go:3`), confirmed satisfying
  `server.IntegerId`'s `~uint8` term via the module building green. Narrowing
  (`ParseUint(s,10,8)` now correctly rejects out-of-range values the old
  `strconv.Atoi`+truncate silently accepted) is the settled, pre-approved one. PASS.
- `ParseCharacterId`, `ParseAccountId` → `server.ParseIntId[uint32](l, "<name>", next)` —
  both pre-images are bare `mux.Vars` + `strconv.ParseUint(s, 10, 32)` lookups, no extra
  check. Delegates cleanly. PASS.
- `ParseListingId`, `ParseHoldingId` → `server.ParseStringId(l, "<name>", next)` — both
  pre-images are `mux.Vars` + `ok || val == ""` bad-request check, no extra semantic
  validation (no UUID-format check). The `== ""` narrowing to absence-only is the second
  settled, pre-approved narrowing. PASS.

No helper in this file carried an extra semantic check beyond what `server.ParseIntId`/
`server.ParseStringId` already perform, so all five were correctly delegated (none needed
to stay a byte-identical keeper).

`strconv` and `mux` imports pruned from `rest/handler.go` post-delegation — confirmed via
`grep -n "mux\|strconv" rest/handler.go` (no hits) and `go build ./...` passing.

### 6. Deleted type/helper references

No types or exported helpers were deleted in this task (only converted `d.DB()` sites and
delegated parser bodies); the aliases keep the same names as before. `grep -rn ParseInput`
across the whole module (code and comments) returns nothing, confirming the omitted local
`ParseInput` helper has zero stray references. PASS.

### 7. Untouched files

`git diff --stat cc0c55dcd..0fd55ce60` lists only the 7 files named in the brief;
`main.go`, `libs/atlas-rest/`, and every `*_test.go` file are absent from the diff (empty
diff). PASS.

### 8. Build / test / format

Run from `services/atlas-mts/atlas.com/mts`:
- `go build ./...` → exit 0.
- `go test ./...` → all packages `ok` (holding, listing, testsupport, transaction, wallet,
  wish, bid, configuration, serial, task, kafka/consumer/*, root `atlas-mts`); no failures.
- `gofmt -l .` → no output (clean).

## Not evaluable

None. The full review surface (7 changed files, both commits, id-parser delegation,
build/test/format) was directly inspected and verified.

## Verdict

APPROVED. All 18 `d.DB()` sites converted with correct curry drop, the `wallet` curry-only
case handled, the `testsupport` collision rename is complete and correct with no missed
uses, all five id-parser helpers correctly delegate per the settled precedent, the
commit split is clean, and build/test/gofmt are all green.
