# Review — Task 19: the `mapleLife` tenant-configuration block

Commit reviewed: `f43e442c443523e371a9ffddd104c9622120880a` (single commit, as
requested — not a range).

Brief: `.superpowers/sdd/plan/task-19-brief.md`
Implementer report: `.superpowers/sdd/plan/task-19-report.md`

## Scope

`git show --stat f43e442`:

```
services/atlas-configurations/atlas.com/configurations/templates/maplelife/rest.go     |  78 +++++
services/atlas-configurations/atlas.com/configurations/templates/rest.go               |   2 +
services/atlas-configurations/atlas.com/configurations/tenants/maplelife/rest.go       |  78 +++++
services/atlas-configurations/atlas.com/configurations/tenants/maplelife/rest_test.go  | 156 +++++++
services/atlas-configurations/atlas.com/configurations/tenants/rest.go                 |   2 +
services/atlas-configurations/atlas.com/configurations/tenants/rest_test.go            |  52 ++++
6 files changed, 368 insertions(+)
```

Matches the brief's file list exactly (two new `maplelife` packages, two
`RestModel` field additions, two test files — one new, one extended). No file
outside `services/atlas-configurations` is touched. `scope_confirmed`.

## Findings

### 1. JSON tag contract (priority 1) — PASS

`services/atlas-configurations/atlas.com/configurations/tenants/maplelife/rest.go:134-155`
(`ClassEntry`) and `:161-163` (`RestModel`) carry exactly the tags the brief
enumerates:

- Top level: `looks`, `classes` — `rest.go:161-163`.
- Within `ClassEntry`: `ordinal`, `gender`, `jobId`, `level`, `mapId`, `stats`,
  `ap`, `sp`, `spSkillId` (`omitempty`, per brief), `meso`, `equipment`,
  `inventory` — `rest.go:134-155`.

Verified against the raw struct definitions (`git show`), not just the
report's transcription. No drift.

`TestMapleLifeBlockRoundTrips/json_tags`
(`tenants/maplelife/rest_test.go:113-155`) marshals the full fixture and
asserts each of the 12 expected `ClassEntry` keys is present on the first
class entry, plus the two top-level keys. One gap: the assertion only checks
that each expected key is *present* — it does not assert the key set is
*exactly* those keys (no unexpected extras). Given every field on the struct
has an explicit tag and there's no embedding, this is theoretical risk today,
not a live gap, but it means the test's title ("json tags... keys are
exactly...") slightly overstates what it verifies. Non-blocking.

### 2. `tenants/maplelife` vs `templates/maplelife` — PASS (verified independently)

```
$ diff services/atlas-configurations/atlas.com/configurations/tenants/maplelife/rest.go \
       services/atlas-configurations/atlas.com/configurations/templates/maplelife/rest.go
(no output)
```

Byte-identical, confirmed via `diff`, not taken on the report's word. Both
declare `package maplelife` (`git show f43e442` on both paths — line 1 of
each hunk is `+package maplelife`).

The two `RestModel` field additions (`tenants/rest.go` and `templates/rest.go`)
are also identical in shape:

```
+	"atlas-configurations/tenants/maplelife"   (resp. templates/maplelife)
...
+	MapleLife    maplelife.RestModel  `json:"mapleLife"`
```

Same field name, same tag, only the import path differs (correctly, since
each mirror package tree is import-path-scoped) — matches the established
`tenants/characters/template` vs `templates/characters/template` convention
the brief points to.

### 3. Absent-block case — PASS, test genuinely exercises it

`tenants/maplelife/rest_test.go:100-112` (`absent block` subtest): unmarshals
`{}` into `RestModel` and asserts `len(Looks)==0`, `len(Classes)==0`, no
error.

Also covered at the tenant-document level:
`tenants/rest_test.go` `TestTenantRestModelCarriesMapleLife/document without
mapleLife key` unmarshals a document with no `mapleLife` key and asserts
`decoded.MapleLife.Looks`/`.Classes` are zero-length and `Region` still
decodes correctly, confirming the new field doesn't disturb sibling fields
when absent. Both subtests are real assertions against real
`encoding/json` behaviour, not mocks.

Traced the actual decode path used at runtime, not just in the test:
`tenants/processor.go:100-114` (`Make`) does a plain
`json.Unmarshal(e.Data, &rm)` with no special-casing for `mapleLife` — the
zero-value/absent-key behaviour the tests exercise is exactly the behaviour
`Make()` will hit for the 7 of 11 templates that carry no Maple Life block.
Confirms the brief's "no entity change... a new field on the struct is
sufficient" claim.

### 4. Pre-existing `atlas-configurations/socket` failure — CLAIM VERIFIED INDEPENDENTLY

Checked out `27d3b3d59` (the commit immediately preceding `f43e442`, i.e.
task-19's actual parent) in an isolated worktree and ran
`go test ./socket/...` there directly (not via `git stash`, to rule out any
stash-application side effect):

```
--- FAIL: TestValidate_AcceptsEverySeedTemplate (0.02s)
    corpus_test.go:64: corpus size = 3329 entries, want 3317 (...)
FAIL
FAIL	atlas-configurations/socket	0.020s
```

Identical failure mode (3329 vs want 3317) reproduces at the pre-task-19
commit. The report's claim is correct: this failure predates `f43e442` and
is unrelated to the `mapleLife` block. It is not caused by any task-246
commit up to and including this one — it was already failing at `27d3b3d59`,
which is itself a task-246 commit (`feat(atlas-character): persist unspent
ap and sp on creation`), several commits into this task's own branch. I did
not walk further back to find the exact commit that introduced the corpus
drift, since that's outside this unit's diff — but the finding that matters
for the reviewer's question ("does f43e442 cause it") is negative: it does
not. Flagging for Task 20 awareness only, not as a defect of this commit:
Task 20 seeds four templates and will land on top of an already-broken
template-corpus test, so its own verification will need to either fix or
explicitly carry forward this pre-existing failure rather than being
surprised by it.

## Other checks

- `go build ./...` and `go test ./tenants/... ./templates/...` — both green,
  re-run directly in this worktree (not trusting the report's transcript).
- `gofmt -l` on all six touched files — no output (clean).
- `go vet ./tenants/... ./templates/...` — no output (clean).
- Field types match the brief's Step 2 code block verbatim: `SP string`
  (kept as the ten-slot string encoding, consistent with earlier task
  decisions), `AP uint16`, byte/uint32 types elsewhere as specified. No
  deviation found.
- `TestTenantRestModelCarriesMapleLife` (`tenants/rest_test.go`) correctly
  asserts the sibling `Region` field is unaffected in both the
  present-key and absent-key cases — genuinely exercises non-interference,
  not just presence.

## Not evaluable

- Whether the ordinal→class mapping, SP skill IDs, or other *content*
  encoded in seed data are correct is out of scope for this commit (Task 19
  only adds the Go struct/JSON contract; no seed data is introduced here).
- Downstream consumption by Tasks 20-22 is necessarily unverified from this
  unit alone — the tag contract is checked exactly as specified, but whether
  Task 20's seed JSON actually round-trips through this contract is that
  task's own verification surface.
- The root cause / originating commit for the pre-existing `socket` corpus
  mismatch is not identified (only that it predates `f43e442`) — out of this
  unit's diff surface.

## Verdict rationale

No requirement from the brief was dropped or drifted. The two mirror files
are byte-identical (independently verified). The absent-block case is
genuinely tested at both the package level and the tenant-document level.
The pre-existing test-failure claim independently reproduces at the parent
commit. The only note is the "json tags... exactly" test asserting key
presence rather than key-set equality — cosmetic relative to the stated test
title, not a functional gap given the struct's tag coverage — hence
`APPROVED_WITH_FINDINGS` rather than a clean `APPROVED`, with zero blocking
findings.
