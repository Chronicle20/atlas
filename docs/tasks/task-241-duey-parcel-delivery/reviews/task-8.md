# Task 8 review — PARCEL bodyless result arms

Commit reviewed: `f1e7d0229` (range `29ec83a32..f1e7d0229`), branch
`task-241-duey-parcel-delivery`. Read-only review; no edits made to any
codec, registry, template, evidence, or generated file.

## Priority 1 — Ruling 5 compliance

- `git show f1e7d0229 -- docs/packets/dispatchers/parcel.yaml` produces only
  the commit header (11 lines total, all metadata) — **zero diff content**.
  The authority file is untouched by this commit. Confirmed directly, not
  taken from the report.
- Read the current `parcel.yaml` (lines 68–87) directly: all 14 non-
  `SUCCESSFULLY_SENT` keys among the fifteen (`SEND_ENABLE_ACTIONS` through
  `UNKNOWN_ERROR_2`) have **no `jms_v185` field at all** in their `modes:`
  map — not a defaulted/copied v83 value, genuinely absent. Matches the
  report's table exactly (gms modes 9–22, 28 decimal / 0x09–0x16, 0x1C).
- `SUCCESSFULLY_SENT`'s `jms_v185: 19` predates this commit: `git log -p` on
  the file shows it was introduced by an earlier commit (task-6's derivation,
  hash `91a6e82aa`/ancestor), and the file diff for `f1e7d0229` is empty, so
  it could not have been touched here.
- Grepped the actual Go source myself (not trusting the report's self-audit):
  `grep -rn 'mode:\s*0x' parcel.go` and `grep -rn 'func(_ byte)' parcel_body.go`
  both return **zero matches**. No jms_v185 (or any) mode byte is
  hard-coded or extrapolated anywhere in the new code.

## Priority 2 — mode resolution / no-hard-coding

- Read the full diff of `parcel_body.go`: all fifteen new body functions are
  `atlas_packet.WithResolvedCode("operations", <fixed const>, func(mode byte)
  packet.Encoder { return New...(mode) })` — the same idiom as the
  pre-existing `ParcelOpenBody`, which the diff leaves untouched (only a
  `gofmt` const-block realignment touches its neighboring lines).
- Spot-checked three arms end-to-end (key const → yaml entry → encoded byte,
  cross-referencing the diff and the live `parcel.yaml`):
  - `SEND_ENABLE_ACTIONS`: const `"SEND_ENABLE_ACTIONS"` → yaml
    `gms_v83: 9` (0x09) → test encodes `{0x09}`. Match.
  - `SUCCESSFULLY_SENT`: const `"SUCCESSFULLY_SENT"` → yaml `gms_v83: 18`
    (0x12) → test encodes `{0x12}`. Match.
  - `UNKNOWN_ERROR_2`: const `"UNKNOWN_ERROR_2"` → yaml `gms_v83: 28`
    (0x1C) → test encodes `{0x1C}`. Match.
  - All fifteen `TestParcelResultArms` subtests pass (ran directly:
    `go test ./parcel/... -run TestParcelResultArms -v`), which is
    consistent across all fifteen, not just the three spot-checked.
- `tools/packet-audit/cmd/run.go` diff is a single hunk of pure appended
  `case` blocks inserted after the pre-existing `OPEN_QUICK` case and before
  the pre-existing `NOTE_ACTION` comment block — no existing `case` line is
  touched, reordered, or removed (confirmed via `git diff 797baebef
  f1e7d0229 -- tools/packet-audit/cmd/run.go`).

## Priority 3 — correctness and consistency

- Every one of the fifteen new structs: exactly one `w.WriteByte(m.mode)`
  in `Encode`, exactly one `m.mode = r.ReadByte()` in `Decode`, nothing
  else — verified by counting (`grep -c`) 15 struct decls, 15
  `WriteByte(m.mode)` calls, 15 matching `Decode` methods, all one-byte
  round-trips.
- `parcel_result_test.go`: the table genuinely asserts
  `bytes.Equal(got, want)` against the literal per-arm mode byte (not just
  calling the constructor and discarding the result). The negative case
  (`unresolved key falls back`) asserts the returned byte is exactly `99`
  and cites `resolve.go`'s documented sentinel — checked `resolve.go`
  directly: `ResolveCode` does return 99 on every lookup-miss path, and
  `resolve_test.go` already pins the same constant elsewhere. The fallback
  behaviour is the real documented miss path, not a masked zero-byte bug.
- Task 7's `Open` struct and `parcel_test.go` are byte-for-byte unchanged in
  this diff (`git diff 797baebef f1e7d0229 -- .../parcel_test.go` is empty).
  The shared `parcel.Parcel` struct package is untouched (no diff hunk
  touches it).
- **One undisclosed but non-breaking addition**: this commit adds a
  `func (m *OpenQuick) Decode(...)` method to Task 7's `OpenQuick` struct
  (lines 30–34 of the file diff), which Task 7 did not include. It is a pure
  appended addition — `OpenQuick.Encode` and every other Task 7 line is
  untouched — follows the exact same pattern as the reference
  `note/clientbound/operation.go` `SendSuccess.Decode`, compiles, and is not
  exercised by any test (neither `parcel_test.go` nor
  `parcel_result_test.go` calls it). It is not mentioned anywhere in the
  task-8 report. Harmless and consistent with the established pattern, but
  an undisclosed scope-adjacent edit to a prior task's arm — flagged as
  non-blocking.
- `go run ./tools/packet-audit dispatcher-lint` — ran it myself: output
  `dispatcher-lint: clean` (exit 0).
- `cd libs/atlas-packet && go build ./... && go vet ./...` — ran it myself:
  clean, no findings.
- `go run ./tools/packet-audit matrix --check` — ran it myself: exits 1
  (STATUS.md/status.json stale). This is the pre-established, already-
  assigned finding named in the dispatch instructions; not re-reported,
  not investigated further, not attributed to this commit's content.
- `tools/packet-audit/dispatcher-lint-baseline.yaml` has no `parcel` entries
  before or after — the family was never baselined and remains un-baselined
  (consistent with "family stays clean", not silently exempted).

## What I did NOT check

- Did not run the full repo-wide `tools/verify.sh` (out of scope for a
  read-only single-task review, and a repo-wide gate is running
  concurrently against this tree per the dispatch instructions).
- Did not re-verify Task 6's IDA-derived `parcel.yaml` mode values against
  the binary myself (explicitly pre-verified per the dispatch brief; only
  cross-checked that the file's *current* content matches the report's
  table and was not touched by this commit).
- Did not run `-race`; matches the brief's stated exclusion for this task.
- Did not evaluate `fname-doc --check` or `operations --check` myself (the
  report's quoted output for both is consistent with expected pre-existing
  DueyAction-writer-absent notes and the Ruling-5/Task-28 deferral, and
  re-running either is redundant with the dispatcher-lint/matrix/build
  checks I did run independently).
- Did not investigate the stale-matrix finding (`matrix --check` exit 1) —
  explicitly out of scope, already assigned per the dispatch instructions.
