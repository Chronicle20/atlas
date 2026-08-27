# Review: bug-reseed-drift-never-clears fix

Commit reviewed: `3c1bd1e93` (range `92cb7e4dd..HEAD`)
Brief: `docs/tasks/task-201-template-reseed-trigger/bug-reseed-drift-never-clears.md`

## Scope

`git diff --stat 92cb7e4dd..HEAD`:

```
.../bug-reseed-drift-never-clears-report.md        | 142 +++++++++++++++++++++
.../bug-reseed-drift-never-clears.md               | 117 +++++++++++++++++
.../configurations/templates/processor_test.go     |  52 ++++++++
atlas.com/configurations/templates/revision.go     |  14 +-
.../configurations/templates/revision_test.go      |  24 ++++
5 files changed, 348 insertions(+), 1 deletion(-)
```

Matches the brief's `## Fix` inventory exactly: `revision.go`,
`revision_test.go`, `processor_test.go`, plus the bug/report docs. No
unrelated files touched. `RestModel`, `canonicalBytes`, and `Make` were left
unchanged, as the brief required.

## Findings

### PASS — fix targets the correct field, matches prescribed shape

`services/atlas-configurations/atlas.com/configurations/templates/revision.go:29-33`
adds `rm.Environment = ""` alongside the existing `rm.Id = ""`, before
`Socket` normalization and marshal — exactly the brief's prescription
("clear `rm.Environment` alongside `rm.Id` in `Revision`"). The doc comment
was extended both to explain *why* (`revision.go:19-32`) and to state
`Revision`'s general contract ("hashes only client-authored content... any
field that is server-owned or read-time-derived... must be cleared or
normalized here"), which directly answers the brief's "Not yet answered"
item.

### PASS — no other server-owned/read-time-derived field can enter the hash

Checked `RestModel` (`rest.go:12-30`) and `Make` (`processor.go:129-141`):
the only two fields the codebase's own comments document as server-owned are
`Id` (`json:"-"`, already cleared) and `Environment` (now cleared). `Make`
overwrites only `rm.Id` and `rm.Environment` from the `Entity` after
unmarshal — no other field is stamped from anything but the stored JSON
blob. Grepped the nested sub-model packages (`characters`, `socket`,
`cashshop`, `maplelife`, `npcs`, `worlds`) for "server-owned"/"read-only"/
"overwrite" comments — none found, so no nested field is read-time-derived
either. `canonicalBytes` (write path) applies only `Socket` normalization
and validation, consistent with `Revision`'s own normalization. The fix's
scope is complete against the current codebase.

### PASS — regression tests would actually have caught the original bug

Reverted just the `rm.Environment = ""` line locally and reran the new
tests:

```
$ go test ./templates/... -run 'TestRevisionIgnoresEnvironment|TestReseedClearsDriftUnderNonEmptyEnvironment' -v
--- FAIL: TestReseedClearsDriftUnderNonEmptyEnvironment
    processor_test.go:798: SeedDrift = true immediately after seeding from
    the shipped file (stored "6fd13d0f0b79...", shipped "34c8a2b180...")
--- FAIL: TestRevisionIgnoresEnvironment
    revision_test.go:99: Revision(environment="main") = "523c299e...",
    want "8cc784ee..." (same as base)
```

Both fail pre-fix with the exact drift signature described in the bug file's
repro (`shipped=34c8a2b18000 stored=6fd13d0f0b79 equal=false`), and both pass
after restoring the fix. This directly addresses the review's central
concern: `TestReseedClearsDriftUnderNonEmptyEnvironment` builds its processor
via `envContext(t, "main")` (`processor_test.go:787`, an existing helper from
`overlay_test.go:22` that installs a non-`""` caller environment) rather than
`context.Background()`, so it is not subject to the same blind spot that hid
the bug from every prior `TestReseed*` test. `TestRevisionIgnoresEnvironment`
independently checks the unit boundary (`Revision` itself) across `""`,
`"main"`, `"pr-123"`.

Restored the file after the experiment (`git checkout --
.../templates/revision.go`); working tree confirmed clean afterward.

### PASS — full package/module suite green with the fix in place

```
$ go build ./...        (from services/atlas-configurations/atlas.com/configurations)
(success, no output)
$ go test ./templates/...
ok  	atlas-configurations/templates	(cached)
ok  	atlas-configurations/templates/socket
(remaining subpackages: no test files)
```

### Non-blocking — report's "verified" language is imprecise, but the underlying claim is true

`bug-reseed-drift-never-clears-report.md`'s Testing section states "Both new
tests failed before the fix (verified by inspection...)" and explicitly says
the pre-fix suite was *not* run as a separate step, relying instead on
reasoning from the bug file's repro numbers. Per this repo's evidence
convention ("never claim verified from a flagged or partial run"; "quote
actual tool output before concluding"), an inspection-based claim of
pre-fix-test-failure is weaker than what the report's phrasing implies. This
review closes that gap by actually running the pre-fix tests (see above) and
confirming the claim is correct — so there is no live discrepancy, but the
report itself should not be read as first-hand verification evidence for
that specific claim.

## Not evaluable

- Live re-test on `atlas-main` (the bug's `## Observed` section, and the bug
  file's `## Outcome` "to be filled in" placeholder) is out of this review's
  surface — it requires a cluster and is explicitly deferred to the
  controller per both the bug file and the fix report.

## Summary

The fix is the minimal, correctly-scoped change the brief prescribed:
`Environment` is now cleared in `Revision` alongside `Id`, with a doc comment
that generalizes the contract for future server-owned fields. No other field
on `RestModel` or its nested sub-models is server-owned or read-time-derived,
so the exclusion list is complete as of this commit. The new regression
tests are honest — confirmed by direct experiment that both fail against the
pre-fix code with the bug's exact hash-mismatch signature, and specifically
that `TestReseedClearsDriftUnderNonEmptyEnvironment` avoids the
`context.Background()` / `Environment == ""` blind spot that let the
original bug ship.
