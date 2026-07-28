# Backend Audit — task-169 packet-process-consistency (tools/packet-audit)

- **Scope:** `git diff origin/main...HEAD -- tools/packet-audit/` (base 560b4fcd)
- **Nature:** CLI/analysis tool, not a DDD microservice → DOM-*/JSON:API/Kafka checks N/A. Focus: matrix-grading integrity, guard honesty, determinism, correctness.
- **Overall:** PASS

## Build / Test / Vet (worktree)
- `go build -o /tmp/pa ./tools/packet-audit` → exit 0
- `go test -race ./tools/packet-audit/...` → exit 0 (all 13 packages ok)
- `go test -count=1` → exit 0 (non-cached)
- `go vet ./tools/packet-audit/...` → exit 0
- Live: `matrix --check` rc=0 · `gate-check --check` rc=0 (19/19 gates) · `doc-freshness --check` rc=0 (9 versions, 7 CI gates)

## 1. Matrix grading integrity — PASS
- (a) Sub-struct NA is gated EXCLUSIVELY on `in.Unimplemented[vk][pkt]`:
  gap-fill branch `build.go:186-190`, defensive guard `build.go:309-311`
  (`gradeSubStructCell`). No unaudited sub-struct can render NA.
- gradeCore's only `StateNA` (`grade.go:185`) is reachable **only** for
  `opregistry.Absent` applicability. Sub-structs always pass
  `applicability: opregistry.Present` (`build.go:317`) → cannot reach it.
- (b) `⬜/❌ → n-a` reclassification is bounded to dispositioned cells: the only
  StateNA producers for sub-structs are the two Unimplemented-gated branches;
  `Unimplemented` is populated solely from each version's `_unimplemented.json`
  via `ResolveUnimplemented` (`matrix.go:245-262`), which resolves only explicit
  `packet:` paths or `#`-suffixed fnames (bare base fnames deliberately skipped,
  `unimplemented.go:76-89`).
- (c) status.json State enum unchanged — `model.go` and `load.go` are byte-unmodified
  in the diff. Partial disambiguation is render-only: `render.go:38 cellSymbol`
  maps StatePartial→glyph for STATUS.md only; JSON still carries `state:"partial"`+Note.
- (d) `✅` verified path unchanged: `gradeCore` still requires
  `marker.Found && hasEvidence && evidence.Fresh` (`grade.go:203/207/241/245/257/261`).
  The diff only ADDS the `Unimplemented` field to the Inputs struct; no verified
  logic weakened. FR-4.1 intact.
- **Can a sub-struct/op reach NA or ✅ without justification? NO.**

## 2. Guard honesty — PASS (no green-only tests)
- gate-check (`gatecheck_test.go`): upper-unverified→non-zero (:75), missing-reason→
  non-zero (:114), unknown-packet→non-zero (:135). File reads + yaml.Unmarshal
  error-handled (`gatecheck.go:94,99,212,216`).
- doc-freshness (`doclint_test.go`): version_count drift→non-zero (:61),
  version_key drift→non-zero (:90), missing CI gate→non-zero (:122).
- gate-lint (`gatelint_test.go`): `collectGateLintHits` asserts EXACTLY the seeded
  footgun forms (`>83`, `<=87`, `83>=`) and that correct idioms (`>=`,`<`) are NOT
  flagged (:39,:79); `_test.go` skipped. Report-only unless `--check` (by design).
- export (`export_test.go`): differing overwrite refuses+leaves file byte-unchanged+
  writes `.new` (:101-114), `--force` overwrites (:128), identical is idempotent (:148),
  `--splice` updates one entry and leaves the sibling byte-identical (:184-189).
- `SpliceExport` (`idasrc/export.go:103-125`) round-trips untouched entries via
  the shared struct + `MarshalIndent`+`\n`; error-handled on missing fname/read/parse.
  No corruption path.
- **Any green-only guard test? NO.**

## 3. Determinism — PASS
- ToolSHA now hashes only sorted `.go` blob entries (`matrix.go hashGoTreeEntries`,
  `sort.Slice` by path) — docs edits no longer churn the matrix.
- `summary.go` iterates the row slice (ordered) and `sortGaps` sorts every emitted
  slice by (direction,opcode,name); no map-range in output.
- `SpliceExport`/export use `json.MarshalIndent` (Go sorts map keys) → stable.
- `matrix --check` rc=0 confirms committed status.json/STATUS.md are fresh.

## Non-blocking notes
- `gate-lint --check` reports 35 pre-existing raw-boundary hits under
  `libs/atlas-packet` (NOT introduced by this diff) and is intentionally
  report-only / not wired blocking. Not a finding against this change.

## Blocking findings
None.
