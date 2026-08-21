# Review: Task 12 fix round 2 — inner-portal registration

Range: `3df557498..HEAD` (`9306fa4cc`, `a175e81d7`).
Repairs: `docs/tasks/task-250-inner-portal-registration/review-task-12-fix1.md`
(1 blocking: stale `Tool:`/`toolSha` fingerprint; 1 non-blocking: round-trip
test covered only the explicit `"discriminator": ""` state).
Brief: `.superpowers/sdd/plan/task-12-fix2-brief.md`.
Report section reviewed: `.superpowers/sdd/plan/reports/task-12-report.md`
(final "Fix round 2" section, lines ~450-596).

## Scope

```
$ git diff --stat 3df557498..HEAD
docs/packets/audits/STATUS.md                     |  2 +-
docs/packets/audits/status.json                   |  2 +-
tools/packet-audit/internal/idasrc/export_test.go | 78 +++++++++++++++++++++++
3 files changed, 80 insertions(+), 2 deletions(-)
```

Exactly the three files the brief named. No codec, no evidence YAML, no
registry, no seed template, nothing under `services/`. Confirmed no stray
`tools/packet-audit/docs/` directory exists (`find` / `ls` both empty).

Per-commit split:
- `9306fa4cc` — all three files (regenerate + new test), matches brief Step 1+2.
- `a175e81d7` — `STATUS.md`/`status.json` only, the follow-up regenerate
  after the test-file commit itself changed the tool tree hash input.

## 1. `matrix --check` / `fname-doc --check` / `operations --check` at final HEAD

Ran directly at `HEAD = a175e81d719222529dfca6033ba4d1b839d15db2` (not against
working tree mid-round, not before the last commit):

```
$ go run ./tools/packet-audit matrix --check
note	n-a evidence consumed: CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79 (docs/packets/feature-na-evidence.yaml)
note	n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 (docs/packets/feature-na-evidence.yaml)
$ echo $? → 0

$ go run ./tools/packet-audit fname-doc --check
fname-doc check OK (269 structs without an audit report carry no fname)
$ echo $? → 0

$ go run ./tools/packet-audit operations --check
operations check OK (0 absent-writer note(s))
$ echo $? → 0
```

**PASS.** All three exit 0 at final HEAD, run by me after both commits.

Also ran module-local build/test as the brief's Step 4 required:

```
$ cd tools/packet-audit && go build ./... && go test ./...
ok  	.../cmd
ok  	.../internal/atlaspacket
ok  	.../internal/csv
ok  	.../internal/diff
ok  	.../internal/discover
ok  	.../internal/evidence
ok  	.../internal/idasrc
ok  	.../internal/marker
ok  	.../internal/matrix
ok  	.../internal/opregistry
ok  	.../internal/report
ok  	.../internal/seedcsv
ok  	.../internal/template
```
**PASS.**

## 2. Regenerate changed only the fingerprint, not any verdict

```
$ git diff 3df557498..HEAD -- docs/packets/audits/STATUS.md docs/packets/audits/status.json
docs/packets/audits/STATUS.md
  -Tool: `a645319789efe3bb1ee39a93b24b49cf8753c1141c924061749828c68c001874`
  +Tool: `88975d34ce729f0d8fd71678f038b21eb637a10a32a53873ebad817c00660d5a`
docs/packets/audits/status.json
  -  "toolSha": "a645319789efe3bb1ee39a93b24b49cf8753c1141c924061749828c68c001874",
  +  "toolSha": "88975d34ce729f0d8fd71678f038b21eb637a10a32a53873ebad817c00660d5a",
```

Only the `Tool:`/`toolSha` line changed in each file — no data field, no
verdict cell. The `USE_INNER_PORTAL` row (`STATUS.md:615`) still shows all
ten cells as ✅:

```
| USE_INNER_PORTAL | CUserLocal::TryRegisterTeleport |  | 0x050 | ✅ | 0x05D | ✅ | 0x064 | ✅ | 0x063 | ✅ | 0x065 | ✅ | 0x065 | ✅ | 0x068 | ✅ | 0x070 | ✅ | 0x071 | ✅ | 0x060 | ✅ |
```

**PASS.**

## 3. `TestSpliceExportPreservesOmittedDiscriminator` genuinely pins the omitted-key case

Read the test (`tools/packet-audit/internal/idasrc/export_test.go:528-604`).
It builds an existing export entry `Foo::KeptNoDiscriminator` whose dispatch
selector's JSON has no `"discriminator"` key at all, splices in an unrelated
entry via `SpliceExport`, unmarshals, then re-marshals the round-tripped
`Selector` through its own custom `MarshalJSON` and asserts on the *raw
bytes*:

```go
selBytes, err := json.Marshal(kept.Dispatch[0])
...
if strings.Contains(string(selBytes), `"discriminator"`) {
    t.Errorf("round trip synthesized a \"discriminator\" key for a selector that omitted it:\n%s", selBytes)
}
```

This is a key presence/absence check, not a value-only comparison — it would
catch a marshaller that synthesizes an explicit `"discriminator": ""` even
though `Discriminator == ""` is also what "omitted" unmarshals to, which is
exactly the bug class that caused the original data loss.

**Verified it actually fails against a broken marshaller.** I temporarily
edited `tools/packet-audit/internal/idasrc/extract.go:60` from
`if s.Discriminator != "" || s.discriminatorExplicit {` to `if true {`
(forcing every Selector, including ones that omitted the key, to marshal an
explicit `"discriminator"` field), ran only this test, observed RED, then
restored the file from a backup and re-ran the full `idasrc` package to
confirm it was back to GREEN and the tree was clean:

```
$ go test ./internal/idasrc/... -run TestSpliceExportPreservesOmittedDiscriminator -v
=== RUN   TestSpliceExportPreservesOmittedDiscriminator
    export_test.go:599: round trip synthesized a "discriminator" key for a selector that omitted it:
        {"discriminator":"","case":0}
--- FAIL: TestSpliceExportPreservesOmittedDiscriminator (0.00s)
FAIL

# (restored extract.go)
$ go test ./internal/idasrc/... -run TestSpliceExportPreserves -v
=== RUN   TestSpliceExportPreservesCuratedProvenanceKeys
--- PASS
=== RUN   TestSpliceExportPreservesOmittedDiscriminator
--- PASS
PASS
```

`git status --porcelain` confirmed no residual diff in `extract.go` after
restoring. **PASS.**

## 4. Scope hygiene

Confirmed above under "Scope" — three named files only, correct commit
split, no stray `tools/packet-audit/docs/` directory, `git status --porcelain`
at final HEAD shows only pre-existing untracked task-review docs (none from
this round, none of them mine to touch).

**PASS.**

## 5. The implementer's `toolTreeSHA()` diagnosis

Read `tools/packet-audit/cmd/matrix.go:487-496`:

```go
func toolTreeSHA() string {
    out, err := exec.Command("git", "ls-tree", "-r", "HEAD", "tools/packet-audit").Output()
    ...
}
```

This confirms the implementer's diagnosis is correct: the fingerprint is
computed from `git ls-tree -r HEAD`, i.e. the *committed* tree at the moment
`matrix` runs, not the working tree and not what's about to be committed.
Regenerating the matrix and committing a `.go`/test-file change in the same
commit is therefore inherently unsafe — at the moment `go run ./tools/packet-audit
matrix` executes, HEAD is still the *prior* commit, so the fingerprint gets
baked in against the pre-change tree; the moment the commit that bundles both
lands, HEAD moves and the just-committed fingerprint is stale again. This is
exactly what happened with `9306fa4cc` (test file + regen in one commit) and
is why `a175e81d7` (a pure post-commit regen, touching only STATUS.md/
status.json) was required. This is a real trap for any future packet-audit
task: **the matrix regeneration must always be its own final commit, made
after every other `tools/packet-audit/**/*.go` change has already landed**,
not merely run after the file was written to disk.

## Not evaluable

None — every directed check was run and verified against real command output
at final HEAD.

## Summary

All four directed checks pass with hand-verified evidence. The blocking
finding from fix round 1 (stale `Tool:`/`toolSha` fingerprint) is closed —
`matrix --check` exits 0 at the actual final HEAD, not a mid-round snapshot.
The non-blocking finding (missing omitted-discriminator coverage) is closed
by a test that genuinely distinguishes "key absent" from "key present and
empty," confirmed by reproducing RED against a deliberately broken
marshaller. Scope stayed within the three named files across both commits.
No new findings.
