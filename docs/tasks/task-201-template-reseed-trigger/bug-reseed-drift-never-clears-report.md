# Fix report: re-seeded templates still report `seedDrift` on every row

Bug file: `docs/tasks/task-201-template-reseed-trigger/bug-reseed-drift-never-clears.md`
Branch: `fix/task-201-reseed-drift`

## What I implemented

Excluded `RestModel.Environment` from the content hash `Revision` computes,
exactly as `Id` already is excluded — per the bug file's root-cause analysis
and prescribed fix.

1. `services/atlas-configurations/atlas.com/configurations/templates/revision.go`
   - `Revision` now clears `rm.Environment = ""` alongside the existing
     `rm.Id = ""`, before `Socket` normalization and marshal.
   - Extended the doc comment: added a paragraph stating `Revision`'s
     contract explicitly ("hashes only client-authored content... any
     server-owned or read-time-derived field must be cleared or normalized
     here") per the bug file's "Not yet answered" note, so the next such
     field is caught at review. Extended the existing paragraph explaining
     `Id`/`Socket` to also explain *why* `Environment` is cleared: it is
     server-owned, stamped by `Make` from `Entity.Environment`, absent from
     every shipped seed file, and would otherwise make `SeedDrift`
     permanently true on any deployment with a non-empty
     `ATLAS_ENVIRONMENT` (atlas-main, pr-*).

2. `services/atlas-configurations/atlas.com/configurations/templates/revision_test.go`
   - Added `TestRevisionIgnoresEnvironment`: same `RestModel`, `Environment`
     set to `""`, `"main"`, and `"pr-123"` in turn, asserts all three
     produce the same hash as the base (`""`) case.

3. `services/atlas-configurations/atlas.com/configurations/templates/processor_test.go`
   - Added `TestReseedClearsDriftUnderNonEmptyEnvironment`: builds the
     processor under `envContext(t, "main")` (an existing helper from
     `overlay_test.go`, already used by
     `TestReseedByIdRejectsCrossEnvironmentWrite`), creates the GMS/83/1 row
     from the shipped catalog entry under that non-empty-environment caller
     (so `Create`/`Make` stamp `Entity.Environment = "main"` exactly as
     `atlas-main` does), and asserts `SeedDrift == false` and
     `StoredRevision == entry.Revision` both immediately after seeding
     **and** after `ReseedById`. This is stronger than the brief's suggested
     shape (manually stamping a DB column) because it drives the row
     through the real `Create`/`Make`/`ReseedById` code paths under a
     non-empty environment end to end — the exact condition every other
     `TestReseed*` test in the file misses by running under
     `context.Background()` (caller `Environment == ""`).

Not changed, per the bug file's explicit constraint: `RestModel` (no JSON
tag change), `canonicalBytes`, `Make`.

## Root cause (confirmed against source)

- `rest.go:24-30` — `RestModel.Environment` carries `json:"environment"`
  and is documented server-owned/read-only.
- `processor.go:137-140` — `Make` unconditionally overwrites
  `rm.Environment = e.Environment`.
- `revision.go` (before fix) — `Revision` cleared `Id`, normalized `Socket`,
  left `Environment` untouched, so it entered the SHA-256.
- No seed file under `seed-data/templates/` carries an `environment` key
  (confirmed: `entry.Model.Environment == ""` on the shipped side always).
- `deploy/k8s/overlays/main/kustomization.yaml` sets
  `ATLAS_ENVIRONMENT=main`; the base configmap ships `""`. Only non-base
  overlays exhibited the bug, and every existing unit test ran at the base
  value, which is why it shipped undetected.

## Testing

Module-local, from `services/atlas-configurations/atlas.com/configurations`:

```
$ go build ./...
(no output — success)

$ go test ./templates/... 
ok  	atlas-configurations/templates	0.498s
ok  	atlas-configurations/templates/socket	0.004s
(all other templates subpackages: no test files)

$ go test ./templates/ -run 'TestRevisionIgnoresEnvironment|TestReseedClearsDriftUnderNonEmptyEnvironment' -v
--- PASS: TestRevisionIgnoresEnvironment
--- PASS: TestReseedClearsDriftUnderNonEmptyEnvironment
PASS

$ go test ./...
ok for every package in the module (atlas-configurations, environmentcol,
environments, outbox, scope, seeder, services, services/service,
services/task, servicesuniq, socket, templates, templates/socket, tenants,
tenants/characters, tenants/characters/preset, tenants/maplelife,
tenants/socket) — no failures, no skips beyond packages with no test files.
```

Both new tests fail before the fix. The implementer reasoned this from the
code rather than running the pre-fix suite; task-reviewer subsequently
reverted the `rm.Environment = ""` line in `revision.go` and re-ran both
tests, observing the exact drift signature from the bug file's repro
(`shipped=34c8a2b18000 stored=6fd13d0f0b79`), then confirmed both pass
again with the line restored. That run — not inspection — is the evidence.

Both new tests exercise the identical
comparison this bug file's repro exercised (shipped-side hash built with
`Environment: ""` vs. stored-side hash built with `Environment` stamped
from a non-empty deployment value) and pass against the fixed code.

## Files changed

- `services/atlas-configurations/atlas.com/configurations/templates/revision.go`
- `services/atlas-configurations/atlas.com/configurations/templates/revision_test.go`
- `services/atlas-configurations/atlas.com/configurations/templates/processor_test.go`
- `docs/tasks/task-201-template-reseed-trigger/bug-reseed-drift-never-clears.md` (committed as-is; it was present but untracked in the worktree — the `Outcome` section is left for the controller to fill in after the gate verdict / live re-test, matching this repo's convention for bug-file lifecycle)

## Self-review

- Fix is minimal and matches the brief's prescribed scope exactly: only
  `revision.go`'s `Revision` function changed, no changes to `RestModel`,
  `canonicalBytes`, or `Make`.
- The doc-comment addition directly answers the bug file's "Not yet
  answered" question about making the contract explicit for future
  server-owned fields.
- The new processor_test.go test reuses the existing `envContext` helper
  and `seedTemplatesDir()`/catalog-lookup pattern already used by
  `TestReseedByIdRejectsCrossEnvironmentWrite` and `TestReseedRestoresShippedContent`
  rather than inventing a new test-setup style.
- No `*_testhelpers.go` files added; used existing `createTestRestModel`
  and `envContext` helpers already in the package's test files.
- Verified `Environment` defaults to `""` in `createTestRestModel`, so the
  new `revision_test.go` case's "base" comparison is meaningful.

## Issues or concerns

None. The fix is narrowly scoped, both new tests fail without it (by
inspection against the bug file's own repro numbers) and pass with it, and
the full module test suite is green.

## Outcome

Fix implemented and committed on `fix/task-201-reseed-drift`. Gate verdict
and live re-test on atlas-main are for the controller/reviewer to record in
the bug file's `Outcome` section, per repo convention.
