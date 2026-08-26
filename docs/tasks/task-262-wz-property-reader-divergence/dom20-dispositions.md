# DOM-20 dispositions — task-262 pre-PR audit

`backend-guidelines-reviewer` returned CHANGES_REQUIRED with 10 DOM-20
findings against `docs/tasks/task-262-wz-property-reader-divergence/audit.md`
(flat `func Test...` bodies instead of `tests := []struct{...}` + `t.Run`).

**Basis for a partial disposition.** DOM-20's source text —
`.claude/skills/backend-dev-guidelines/resources/testing-guide.md:18` —
reads "**Prefer** table-driven tests": a preference, not a MUST. Repo
precedent grades this exact shape as non-blocking:
`docs/tasks/task-085-packet-audit-coverage-matrix/audit-backend.md:39`
dispositions a 16-function-per-scenario file as "PASS w/ WARN … preference,
not MUST → non-blocking". This record applies the same standard: convert
where a table genuinely improves the test (N independent scenarios sharing
one contract, varying only in data), keep flat where the test is a single
scenario, a sequential/stateful narrative, or a round-trip whose value is
the ordering.

| Finding | File | Action | Reason |
|---|---|---|---|
| 1 | `libs/atlas-wz/wz/directory_error_test.go` | KEPT FLAT | Single scenario each: `TestSubDirectoryParseFailurePropagates` (corrupted nested directory → error) and `TestValidNestedArchiveStillOpens` (same archive, uncorrupted → success) assert different contracts (failure vs. success) over the same fixture-building helper, not the same assertions varying by data. A one-row table per test would add indirection with no shared row to loop over. |
| 2 | `libs/atlas-wz/wz/trace_test.go` | KEPT FLAT | Each of the seven tests (`TestSetTraceEmitsNodeEvents`, `TestSetTraceEmitsSubEventDeclaredActualEnd`, `TestTraceNilHookCostsNoExtraPosCalls`, `TestTraceNilByDefault`, `TestSetTraceOnSubFileDelegatesToParent`, `TestSkipZeroPerformsNoSeek`, `TestSetTraceEmitsUOLEvent`) asserts a distinct, unrelated facet of the trace mechanism (event shape, DeclaredEnd/ActualEnd fields, Pos()-call cost accounting, no-hook default behavior, sub-file delegation, Skip(0) no-op, UOL event emission) against different fixtures and different assertion shapes. None are "the same assertions over different inputs" — each is its own single-scenario contract. |
| 3 | `libs/atlas-wz/wz/wztest_canvas_test.go` | KEPT FLAT | Two single-scenario tests: `TestBuilderCanvasWithDimensionsAndChildren` (dimensions + 3 mixed-type children) and `TestBuilderCanvasBackCompat` (1x1, zero children, back-compat wrapper). Their assertion bodies differ in shape (child-type walking vs. a zero-children check), not just in input data — not a table candidate. |
| 4 | `libs/atlas-wz/wz/wztest_dedup_test.go` | KEPT FLAT | Two tests exercise opposite behaviors of `SetStringDedup` (on vs. off-by-default) and assert different, non-mirrored conditions on the observed tag set (dedup-on asserts positive presence of `0x01`/`0x1B`; dedup-off asserts their *absence* plus presence of `0x73`) — not the same assertion varying by data, so each is its own single scenario. |
| 5 | `libs/atlas-wz/wz/wztest_kinds_test.go` — `TestBuilderEmitsAllPropertyKinds` | CONVERTED | 10 distinct property kinds decoded from one fixture, each checked with the identical shape (type-assert + value compare) — the textbook table candidate the controller called out. Converted to `tests := []struct{ kind string; check func(t *testing.T, props []property.Property) }` with one row per kind and `t.Run(kind)`. `TestBuilderFloatZeroWithoutMarker` and `TestBuilderWzIntAndLongBoundaries` in the same file were left as-is: the former is a single scenario, the latter already loops a boundary table internally. |
| 6 | `libs/atlas-wz/wzdiff/allowlist_test.go` | CONVERTED (partial) | Of the 11 original tests, `TestAllowed` was already table-driven (pre-existing, not part of this conversion). Of the remaining 10 flat `LoadAllowlist` tests, all 10 share exactly one of two contracts — "content parses into the expected entries, optionally validated via `Allowed`" or "content is rejected with an error" — varying only in input TSV content. Converted into two tables: `TestLoadAllowlistParsesAndNormalizes` (4 rows: basic parse, trailing-slash normalization, whitespace normalization, repeated-trailing-slash normalization) and `TestLoadAllowlistRejectsInvalid` (7 rows: malformed line, blank image, blank path, blank onlyIn, invalid onlyIn, bare slash path, repeated slash path). Every original assertion (exact `AllowEntry` values, the `:2:` line-number substring, the `Allowed` round-trip checks) is preserved verbatim per row; explanatory comments carried into the case literals they document. |
| 7 | `libs/atlas-wz/wzdiff/run_test.go` | KEPT FLAT | Three tests (`TestRunReportsImageSetMismatch`, `TestRunReportsImageSetMismatchWithEqualCounts`, `TestRunCleanArchiveHasNoDeltas`) each build a differently-shaped archive/reference-dir pair and check different `Result` fields (`ImagesOurs`/`ImagesReference`/log content vs. `OnlyOurs`/`OnlyReference` vs. `Divergent`) — distinct contracts about `Run`'s behavior, not one contract varying by data. |
| 8 | `libs/atlas-wz/wzdiff/selfcheck_test.go` | KEPT FLAT | `TestSelfCheckCorruptedArchiveFails` is sequential/stateful: it builds a clean archive, traces it to locate a byte offset (`traceSubEvent`), then corrupts exactly that offset and re-checks — the second half depends on output computed by the first half of the same test, not an independent row. `TestSelfCheckCleanArchivePasses` is a single scenario. Neither fits the table rule. |
| 9 | `libs/atlas-wz/wzdiff/xmlload_test.go` | KEPT FLAT | Three tests (`TestLoadImageXML`, `TestLoadImageXML_ScalarElementKinds`, `TestLoadImageXML_EmptyElement`) each parse a differently-shaped XML fixture and walk a differently-shaped resulting tree — nested imgdir/canvas/vector vs. a flat scalar-kind list vs. a single empty leaf. `TestLoadImageXML_ScalarElementKinds` already loops an internal `want` table over its shared leaf-kind fixture; the three outer tests are not interchangeable rows of one contract. |
| 10 | `services/atlas-data/atlas.com/data/data/wztoxml/adapter_test.go` | KEPT FLAT | Three tests (`TestRoundTripImage`, `TestSerializeDirectoryCountsFailures`, `TestSerializeDirectorySuccessLogsInfo`) assert three distinct contracts of the serializer (in-memory round-trip through `atlas-data/xml`, per-image failure + WARN-level summary counting, all-success INFO-level summary) over differently-built fixtures and differently-shaped log assertions. None share a single contract varying only by data. |

## Behaviour-preservation evidence

Baseline test names/counts were captured before any edit
(`go test ./wz/... ./wzdiff/... -count=1 -v`,
`go test ./data/wztoxml/... -count=1 -v`), then compared against the
post-conversion run:

- `libs/atlas-wz/wz` package: same 52 top-level test names present after the
  change; `TestBuilderEmitsAllPropertyKinds` additionally reports 10
  `t.Run` subtests (`null`, `short`, `int`, `long`, `float`, `double`,
  `string`, `vector`, `uol`, `convex`) that did not exist before — pure
  addition of per-scenario isolation, no assertion dropped.
- `libs/atlas-wz/wzdiff` package: the 10 flat `LoadAllowlist`/validation
  tests collapsed into 2 top-level tests carrying exactly 11 `t.Run`
  subtests total (4 + 7) — the same 11 scenarios that existed as 11
  separate `func Test...` bodies before, now isolated by `t.Run` instead of
  by function name. `TestAllowed`, `TestDiff`, `TestNodePath`,
  `TestDeltaString`, and every `wzdiff/run_test.go` /
  `wzdiff/selfcheck_test.go` / `wzdiff/xmlload_test.go` test are byte-for-
  byte unchanged.
- `services/atlas-data/.../wztoxml` package: unchanged; all 3 tests pass.

All runs: `GOCACHE=/tmp/gocache-262 go build ./... && GOCACHE=/tmp/gocache-262
go test ./... -count=1` (module-scoped as above) — 0 failures, 0 skips
beyond the pre-existing `TestPropertiesConcurrentParse` skip (unrelated to
this task).
