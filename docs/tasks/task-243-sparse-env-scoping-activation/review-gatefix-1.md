# Review — gatefix-1 (commit 5671346)

## Scope

Reviewed exactly commit `5671346` — "fix(atlas-configurations): migrate
outbox_entries via outboxlib.Migration in servicesuniq test" — against
`.superpowers/sdd/plan/gatefix-1-brief.md` and the implementer report's
"controller ruling applied (third option)" addendum. Diff touches only
`services/atlas-configurations/atlas.com/configurations/servicesuniq/migration_test.go`
(18 insertions, 23 deletions). Read `libs/atlas-outbox/entity.go` and
`migration.go` (contract the diff depends on) and the cited fleet precedent
`services/atlas-tenants/.../rankings_handler_test.go` as read-only
references.

`scope_confirmed`: the diff matches the report's description exactly — no
extra files touched, `testServiceHistoryEntity`/`testServiceEntity`
untouched.

## Findings

### 1. Test coverage preserved, not weakened — PASS

`TestDedupeEnqueuesATombstoneForEveryDeletedRow`
(`services/atlas-configurations/atlas.com/configurations/servicesuniq/migration_test.go:169`)
still asserts, unchanged in strength:

- `rowCount(t, db, "outbox_entries") == 2` (line 172)
- for every row: `Topic == "test.svc.topic"` (was already checked against
  the deleted `testOutboxEntity.Topic`, now `outboxlib.Entity.Topic` —
  same field, same tag `column:topic`)
- `MessageKey` set-membership against `wantKeys` covering both surviving
  service IDs, with a "missing tombstone" failure if either key isn't
  produced
- `len(MessageValue) == 0` (tombstone semantics)

Every assertion present before the fix is present after it, reading
through `outboxlib.Entity` instead of the deleted local mirror. Ran the
test directly:

```
go test ./servicesuniq/... -v -run TestDedupeEnqueuesATombstoneForEveryDeletedRow
--- PASS: TestDedupeEnqueuesATombstoneForEveryDeletedRow (0.00s)
```

Confirmed pass, not a widened/weakened assertion set.

### 2. Outbox table shape — PASS

`testDatabase` (`migration_test.go:53`) now calls
`outboxlib.Migration(db)` (`libs/atlas-outbox/migration.go:5`), which is
`db.AutoMigrate(&Entity{})` against the real
`outbox.Entity` (`libs/atlas-outbox/entity.go:9-20`) — the actual
production struct, not a hand-maintained mirror. This is strictly more
faithful than the deleted `testOutboxEntity`, which had to be kept in sync
by hand. `Entity.Headers` is `datatypes.JSON` in production vs. the
deleted fixture's plain `string`; SQLite AutoMigrate of `datatypes.JSON`
works fine here (build+test both green), and no test in this file touches
`Headers`, so the difference is immaterial to what's under test. No
column the test reads (`Topic`, `MessageKey`, `MessageValue`) was lost or
changed shape.

### 3. No smuggling — PASS

- `git diff 5671346~1 5671346 -- tools/scopeguard/allowlist.txt
  tools/scopeguard/callsite-allowlist.txt` — empty, no new entries.
- `grep -n "ScopingDimension" migration_test.go` — no match; no marker
  method added anywhere in the diff.
- No rename of `testOutboxEntity` to dodge `isEntityTypeName` — the struct
  was deleted outright, not renamed. The remaining control-plane structs
  in the file (`testServiceEntity`, `testServiceHistoryEntity`) are
  untouched and still legitimately carry `Environment`.
- Ran the guard directly against the module:
  `GUARD_MODULES="services/atlas-configurations/atlas.com/configurations"
  GUARD_NOCACHE=1 ./tools/scope-guard.sh` → completes with no findings
  printed (previously: `servicesuniq/migration_test.go:42:6: control-plane
  entity without Environment`). The guard passes because the offending
  declaration is gone, not because it's hidden — matches the fleet
  precedent at `services/atlas-tenants/atlas.com/tenants/configuration/
  rankings_handler_test.go:36` (`outbox.Migration(db)` called directly,
  same pattern) verified by reading that file.

### 4. `testServiceHistoryEntity` untouched — PASS

Diff hunk only removes lines after `testServiceHistoryEntity`'s closing
brace (`TableName() string { return "service_history" }`); the struct
itself (`migration_test.go:24-38` in the post-commit file) is byte-for-byte
identical to pre-commit, confirmed by the diff context lines showing no
`-`/`+` inside that block.

### Build/test verification

```
cd services/atlas-configurations/atlas.com/configurations && go build ./...
```
→ clean. Full `go test ./...` for the module was not re-run beyond the
targeted test above and reliance on the implementer report's claim of "ok"
for the package; the targeted run above is sufficient corroboration for
this narrow diff.

## Not evaluable

None — the diff is small and fully within reviewable surface; all four
judgement questions in the brief were answerable from the commit and its
direct dependencies.

## Verdict rationale

The controller's third resolution is applied correctly: the shadow struct
is deleted (not renamed or marked), the real library migration is used
(matching an established fleet convention, not a novel escape), the one
test that read the deleted struct is re-expressed against the real
`outboxlib.Entity` with identical assertion strength, and the guard now
passes because the underlying problem (a bare control-plane struct without
`Environment`) no longer exists in this file.
