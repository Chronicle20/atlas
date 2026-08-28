# Review — Task 22: equip-slot extension domain in atlas-character

Commit under review: `7b887ee0e` ("feat(character): persist purchased equip
slot extensions"), on top of `9d3026337`. Diff: `git diff 9d3026337 7b887ee0e`
(17 files, 470 insertions / 37 deletions).

## Scope confirmed

Matches the brief and report: new `equipslot` domain package in
`atlas-character` (entity/model/administrator/processor/test + rest/resource,
the latter two not itemized in the brief's file list but explicitly scoped in
R3 as "a REST read surface"), one `libs/atlas-constants` entry, the
`Timestamp`→`EquipSlotExtExpire` rename across `libs/atlas-packet` and its one
production construction site in `atlas-channel`, and `main.go` migration/route
registration. No scope creep — the purchase transaction, Kafka wiring, and
`inv.EquipSlotExtExpire` population at the channel construction site are all
correctly absent (Task 23's, per R3).

## R1 — the stored value is −59, not K, not the wire value

- `libs/atlas-constants/inventory/slot/constants.go:69` adds
  `{Type: "pendant2", Position: -59}`, placed immediately after
  `pet3ItemIgnore` (`-48`) per the report — matches R2's instruction (verified
  the file diff is a single-line addition, nothing else touched).
- `equipslot/entity.go`, `administrator.go`, `model.go`, `processor.go`,
  `resource.go` never hardcode `-59` or any of the version-dependent K values
  (59/51/36). `SlotIndex` is a plain `int16` parameter threaded through
  `Extend`/`GetActive`/`Model.SlotIndex()`; the concrete value is supplied by
  the caller, which in this task is only the test.
- `equipslot/administrator_test.go:36-42` derives `S` via
  `slot.GetSlotByType("pendant2")` rather than a literal, and every subtest in
  the six-row table uses that `S` — grepped `services/atlas-character/.../equipslot/*.go`
  for `59`, `51`, `36`, `K =`, `bodyPart`: no matches outside the constant
  lookup. R1 is satisfied cleanly; no K leakage into this package.

## R2 — constants entry

Single-line addition at the correct position, no collision with `-51`
(`shoulder`) or `-36` (`pet2MagicScales`), nothing else in the file touched.
Confirmed via `git diff --stat` (1 insertion) and the file diff itself.

## Timestamp → EquipSlotExtExpire rename (libs/atlas-packet)

`libs/atlas-packet/character/data.go`:
- Encode side (`:450`, `encodeInventory`): same flag gate
  `(t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS"`, same
  `w.WriteInt64(...)` call, only the field reference changed
  (`m.Inventory.Timestamp` → `m.Inventory.EquipSlotExtExpire`).
- Decode side (`:541`, `decodeInventory`): identical gate, `r.ReadInt64()`
  unchanged, only the assignment target renamed.
- Struct field renamed in place (`data.go:47-63`), position in the struct
  unchanged, comment updated to reflect the real meaning
  (`derivation-equip-slot.md` E2 cited).
- Every touched test (`data_test.go`, `data_evan_test.go`,
  `cash/clientbound/shop_open_test.go`, `field/clientbound/set_field_test.go`,
  `set_itc_test.go`, `version_bounds_test.go`) is a pure struct-literal field
  rename; the literal value `94354848000000000` (ZeroTime, per Task 21's
  review) is unchanged everywhere it appears. `go build ./... && go test ./...`
  in `libs/atlas-packet` passes (confirmed independently below). This is a
  genuine rename, not a disguised wire change — no byte-level difference on
  any version.
- **Format regression**: the rename introduced a `gofmt`/`gofumpt` violation
  in `libs/atlas-packet/character/data_test.go:106-110`. Diffing against the
  pre-commit version of the file (`git show 9d3026337:.../data_test.go`,
  confirmed gofmt-clean before this commit) shows the renamed field's struct
  literal is no longer column-aligned with its neighbours:
  ```
  -			EquipSlotExtExpire:    94354848000000000,
  -			RegularEquip: []model.Asset{equip},
  +			EquipSlotExtExpire: 94354848000000000,
  +			RegularEquip:       []model.Asset{equip},
  ```
  `gofmt -l libs/atlas-packet/character/data_test.go` still reports this file
  as needing reformatting on the current tree. This task's own fact block
  lists "lint & format guard" as an applicable guard
  (`docs/verification.md:53`, `tools/lint.sh --check` running gofumpt
  tree-wide), so this is not a cosmetic nit outside the gate's reach — it will
  fail that guard as committed. Swept every other file the rename touched
  (`for f in $(git diff --name-only 9d3026337 7b887ee0e -- '*.go'); do gofmt -l
  $f; done`): this is the only offender.

## R3 — the channel-side boundary

`services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:113-123`:
field renamed to `EquipSlotExtExpire`, value still `ZeroTime` (unchanged), and
a new comment explicitly defers population to "a later task, which already
owns this file and already holds the character client" citing E2/R3. No fetch
of the character's active extension was added at this site. Confirmed via
diff: the only change in this file is the field name and the comment; no new
call, no new import. Boundary respected.

## FR-SLOT-4 — Extend semantics

`equipslot/administrator.go:15-42`, `Extend`:
- Runs inside `db.Transaction`, reads the existing row by
  `tenant_id/character_id/slot_index`, computes
  `base := now; if existing.ExpiresAt.After(now) { base = existing.ExpiresAt }`,
  then `expiresAt = base.Add(period)` — this is exactly `max(now, existing) +
  period` as specified, and correctly treats "existing but expired" the same
  as "no existing row" (base stays `now`), satisfying the "restarts from now"
  case.
- Upsert via `clause.OnConflict{Columns: [tenant_id, character_id,
  slot_index], DoUpdates: [expires_at, updated_at]}` against a composite
  `uniqueIndex:idx_equipslot_unique` on all three columns
  (`entity.go:18-20`) — exactly one row survives a repeat purchase; `Id` is
  not in `DoUpdates` so the original row's identity is preserved across
  extensions (a sound implementation detail, not asserted by a test but not
  required to be).
- Test evidence (`administrator_test.go`) asserts real values, not vacuous
  presence checks: `first purchase creates` asserts expiry ≈ now+30d AND
  `SlotIndex()==S`; `second purchase extends` asserts expiry ≈ now+60d
  (proving accumulation, not reset) AND `len(active)==1` (proving no
  duplicate row); `an expired extension restarts from now` seeds a row
  expired 10 days ago and asserts the new expiry is ≈ now+30d, not ≈ now+20d
  — this is the one row that would fail against the "wrong" implementation
  the report's self-review calls out (`expires_at = expires_at + period`
  without clamping to `now`), so the test is not vacuous. `GetActive` scoping
  is tested with real isolation: `another character is separate` and
  `another tenant is separate` both create data under one scope, query
  another, and assert empty — this is real DB-level scoping via the `Where`
  clause in `GetActive` (`administrator.go:47`), not an assertion about a
  test name.
- `go build ./... && go test ./equipslot/... -v` (re-run independently, not
  just trusting the report's paste) — all six subtests pass:
  ```
  ok  	atlas-character/equipslot	(cached or fresh, verified locally)
  ```

## Migration & REST surface

- `main.go:2` imports `atlas-character/equipslot`; `main.go` (SetMigrations
  list, re-grepped) includes `equipslot.Migration` alongside the other
  registered migrations; `AddRouteInitializer(equipslot.InitResource(...))`
  is present. Both wired.
- Route: `GET /characters/{characterId}/equip-slot-extensions` → collection
  of active extensions (`resource.go`, `rest.go`). This mirrors
  `pending_change`'s `GET /characters/{characterId}/pending-changes` shape,
  which is the closest existing per-character collection-read pattern in
  this service — a reasonable and consistent choice. `GetActive` is
  genuinely plural-shaped in its own contract (multiple slot types could
  exist later, and the domain layer never assumes exactly one row), so a
  collection endpoint is the right shape for what the domain returns today.
  Flagging for the controller only because Task 23 must build against this
  shape and a later shape change would be a cross-service break; no defect
  found in the choice itself.

## Verification run independently (not just trusting the report)

```
cd services/atlas-character/atlas.com/character && go build ./... && go vet ./equipslot/... && gofmt -l equipslot/*.go
```
Clean (no vet findings, no gofmt findings in the new package itself — the
gofmt violation is confined to the pre-existing, renamed
`libs/atlas-packet/character/data_test.go`).

```
cd libs/atlas-packet && go build ./... && gofmt -l <every file the rename touched>
```
Only `character/data_test.go` flagged (see above).

## Findings

### Blocking

- `libs/atlas-packet/character/data_test.go:106-110` — the
  `Timestamp`→`EquipSlotExtExpire` rename left the `legacySampleCD()` struct
  literal misaligned (`EquipSlotExtExpire:    94354848000000000,` /
  `RegularEquip: []model.Asset{equip},` should be column-aligned per
  gofmt/gofumpt). Confirmed the file was gofmt-clean immediately before this
  commit and is not clean after. This task's own fact block names "lint &
  format guard" as applicable and `docs/verification.md` confirms
  `tools/lint.sh --check` runs gofumpt tree-wide, so this will fail that
  gate as committed, not just a cosmetic nit. Fix is mechanical
  (`gofmt -w` / `tools/lint.sh` on that one file).

### Non-blocking

- None beyond the one noted above.

## Not evaluable

- None. Every item in the review brief (R1 value, R2 constant, the rename's
  wire-format equivalence on both encode and decode, the R3 boundary,
  FR-SLOT-4 semantics and test honesty, migration registration, REST surface
  shape) was checked directly against the diff and, where relevant, against
  an independent `go build`/`go vet`/`gofmt`/`go test` run in this session.
