# Review: Task 2 — Anti-regression tests for compacted-topic config agreement

Commit range: `55283df28..c94f49c11` (single commit `c94f49c11`)
File touched: `services/atlas-kafka-precreate/internal/topics/topics_test.go`

## Scope confirmation

`git diff --stat 55283df28..c94f49c11` shows exactly one file changed
(131 insertions, 3 deletions): `services/atlas-kafka-precreate/internal/topics/topics_test.go`.
`git diff 55283df28..c94f49c11 -- .../topics.go` is empty — `topics.go` is
confirmed unmodified by this commit, not merely claimed unmodified by the
report. Scope matches the brief exactly.

## Requirement-by-requirement

1. **`TestEnsure_CompactConfigsMatchAcrossRequests`** (topics_test.go, added)
   - `wantPairs` uses literal strings (`"compact"`, `"600000"`, `"600000"`,
     `"0.01"`), not the package constants — confirmed by direct read of the
     diff. This satisfies the "cannot pass by reading the same constant"
     requirement.
   - Loops over `stub.createCalls[0].Topics` filtered to `c1`/`c2`, collapses
     `ConfigEntries` to a map, fails on duplicate `ConfigName`, and requires
     `reflect.DeepEqual(got, wantPairs)`.
   - Loops over `stub.alterCalls[0].Resources`, collapses `Configs` to a map
     keyed by `Name`, fails on duplicate, requires `reflect.DeepEqual`
     against the same `wantPairs`, and additionally requires
     `ConfigOperation == kafka.ConfigOperationSet` for every entry.
   - Both loops `t.Fatalf` if zero topics/resources matched, closing the
     vacuous-pass hole named in the brief.
   - PASS. Non-vacuous — see mutation proof below.

2. **`TestEnsure_PlainTopicsCarryNoConfig`** (added)
   - Setup: `Plain: {"p1","p2"}, Compact: {"c1"}`.
   - Asserts `p1`/`p2` appear in the create request with zero
     `ConfigEntries`, the alter request has exactly one resource
     (`c1`), and — via a standalone scan over `req.Resources` against
     `map[string]struct{}{"p1":{},"p2":{}}`, independent of any loop over
     compacted names — that no plain topic appears in the alter resources.
   - This genuinely covers the negative the old suite could not: production
     code (`Ensure` in `topics.go:211` / the `req.Resources` construction)
     only ever puts compacted topics into `IncrementalAlterConfigsRequest`,
     so in the *old* `TestEnsure_AlterConfigs` the removed `p1` check sat
     inside a loop over `sorted`, itself a copy of `req.Resources` — which by
     construction never contains a plain topic. The old check was therefore
     unreachable regardless of what topic mix the old test used. Confirmed
     by reading `topics.go`'s alter-resource construction (only compacted
     topics are appended) and the old test body before deletion.

3. **Dead-code deletion in `TestEnsure_AlterConfigs`**
   - The three-line `if res.ResourceName == "p1" { t.Errorf(...) }` block is
     removed from the diff (`@@ -336,9 +337,6 @@`). The remaining body of
     `TestEnsure_AlterConfigs` was verified by direct read
     (`internal/topics/topics_test.go:293-337`) to still assert: exactly one
     alter call, exactly two resources, `sort.Slice` by `ResourceName`,
     `ResourceType == kafka.ResourceTypeTopic`, and the four-config
     expectation (`cleanup.policy`, `max.compaction.lag.ms`, `segment.ms`,
     `min.cleanable.dirty.ratio` with literal values and
     `ConfigOperationSet`) in declaration order. Unchanged and intact.

4. **Declaration order / literal values** — verified against
   `internal/topics/topics.go:74-79` (`compactTopicConfigs`): order is
   `cleanup.policy`, `max.compaction.lag.ms`, `segment.ms`,
   `min.cleanable.dirty.ratio`; values match the plan's four literals
   exactly.

## Mutation proof — independently re-executed, not just report-trusted

Per the brief's Step 3 and the review brief's explicit instruction, I
independently reproduced the mutation:

```
sed -i '77d' internal/topics/topics.go   # deletes {name: "segment.ms", value: compactSegmentMs}
go test ./internal/topics/ -run TestEnsure_CompactConfigsMatchAcrossRequests -v
```

Result: `FAIL`, output byte-for-byte matching the report's quoted output —
both `c1`/`c2` create-side and `c1`/`c2` alter-side comparisons report the
map missing `segment.ms`. Restored `topics.go` from a pre-mutation copy;
`git diff --stat internal/topics/topics.go` was empty afterward, and
`go test ./internal/topics/ -run TestEnsure_CompactConfigsMatchAcrossRequests -v`
re-passed. `git status --porcelain` at the end of the review shows no
tracked-file modifications outside pre-existing untracked review
artifacts — the worktree was left clean.

This closes both review questions:
- The agreement test is non-vacuous: it can and does fail.
- It fails precisely on the defect class the task exists to guard (a config
  present in one projection's source and absent from what the request body
  carries) — and, since each side is compared to independent literal
  `wantPairs` rather than to each other, it would equally catch a
  *one-sided* projection bug (e.g. only `compactCreateEntries` diverging
  from `compactAlterConfigs`), which is a stronger guarantee than a bare
  create-vs-alter set comparison.
- The implementer's report quotes real, reproducible command output for
  this proof, not a bare claim.

## Build/test corroboration

`go build ./...`, `go vet ./...`, `go test ./...` all pass cleanly in the
module root (`services/atlas-kafka-precreate`), confirming
`internal/discover`, `internal/groups`, `internal/kafkaops`, and
`internal/topics` are unaffected.

## Findings

None blocking. No non-blocking findings either — the commit matches the
brief precisely, the mutation proof is real and independently reproduced,
and the dead-code claim is verified against production code rather than
taken on faith.

## Not evaluable

None — the full unit (one test file, one commit) was within reviewable
scope and every claim in the report was independently checked against the
diff, the source file it targets, and a re-run of the mutation test.
