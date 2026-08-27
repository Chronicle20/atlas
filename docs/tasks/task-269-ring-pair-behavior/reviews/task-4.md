# Review: Task 4 — wire encoder site A (`libs/atlas-packet/character/data.go`)

Commit range: `720bee275..a5f8ac146` (1 commit, `a5f8ac146`).
Brief: `.superpowers/sdd/plan/task-4-brief.md`. Report: `.superpowers/sdd/plan/task-4-report.md`.

## Scope reviewed

`git diff --stat 720bee275..a5f8ac146`:
```
libs/atlas-packet/character/data.go      |  17 +++--
libs/atlas-packet/character/data_test.go | 103 +++++++++++++++++++++++++++++++
2 files changed, 110 insertions(+), 10 deletions(-)
```
Matches the brief's file list exactly. `set_field_test.go` is untouched (`git diff` for that path is empty) — confirmed read-only as required. No files outside the brief's list were touched.

## Spec compliance (brief requirement by requirement)

1. **`Rings model.RingRecords` field added to `CharacterData`.** `libs/atlas-packet/character/data.go:120` — exported, matches the `Stats`/`Inventory` convention. PASS.
2. **`encodeRings`/`decodeRings` delegate to `RingRecords.EncodeRecords`/`DecodeRecords`.** `data.go:770-776`:
   ```go
   func (m *CharacterData) encodeRings(w *response.Writer, t tenant.Model) {
       m.Rings.EncodeRecords(w, t)
   }
   func (m *CharacterData) decodeRings(r *request.Reader, t tenant.Model) {
       m.Rings.DecodeRecords(r, t)
   }
   ```
   No ring-shape logic remains at the call site. PASS.
3. **Call sites `data.go:167`/`data.go:239` unmoved.** Verified in the diff — `m.encodeRings(w, t)` / `m.decodeRings(r, t)` call lines are unchanged context, only the function bodies were rewritten. PASS.
4. **Gate ownership — exactly one copy, in `model.RingRecords`.** `libs/atlas-packet/model/ring.go:263-281` (`EncodeRecords`/`DecodeRecords`) carries `(t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS"` verbatim, and it is the *only* copy — grepped `data.go`'s `encodeRings`/`decodeRings` bodies and confirmed zero gate logic remains there. PASS.
5. **Version-gate idiom in `data.go` — `Region()==... && MajorVersion() > N`, not `MajorAtLeast`.** The pre-existing `MajorAtLeast` calls throughout `data.go` (29 matches) all predate this diff and are outside its hunks; the diff itself introduces no new gate of either idiom into `data.go` (the ring functions now contain zero gate logic, having delegated it entirely to `model/ring.go`). PASS — no idiom violation introduced by this task.
6. **Test idiom — round-trip, no golden-byte markers.** `TestCharacterDataRingsRoundTrip`'s two populated subtests use `pt.RoundTrip` (`data_test.go:519`, `:538`) and field-by-field equality, matching the existing `TestCharacterData*RoundTrip` pattern. No `packet-audit:verify` marker was introduced. PASS.
7. **Four required subtests present** (empty GMS v83, empty legacy GMS v28, populated GMS v83, populated JMS v185) — `data_test.go:471-556`. PASS.
8. **`go test ./character/... ./field/...` and `go build ./...` pass**, `gofmt`/`go vet` clean (reverified independently in this review, not just taken from the report). PASS.

## The FR-9 regression-guard test — the flagged concern, resolved

The two "empty" subtests assert `bytes.Equal(got, want)` where `want` is a **compile-time `[]byte{...}` literal baked into the source** (`data_test.go:487` for GMS v83, `data_test.go:503` for GMS v28) — it is *not* computed at runtime by calling the encoder inside the test body. This is not the tautology the report's phrasing risked implying: `got` is produced by the current code under test on every run; `want` is frozen at commit time. A future regression in `RingRecords.EncodeRecords`'s zero-value output would change `got` while `want` stays fixed, so the test would fail. **Not blocking.**

I independently verified the *ring segment specifically* within each frozen literal, rather than trusting the report's narrative about how the literal was derived (the report describes fixing a hand-transcription bug by regenerating from the post-edit encoder, which is a legitimate way to author a frozen literal but not, by itself, proof the ring bytes are correct). Using a throwaway, unstaged scratch test (removed before finishing; `git status` on the reviewed directory is clean) that encodes with an empty `Rings` and with a populated one and diffs the two byte streams to locate the ring segment by position:

- **GMS v83**: first divergence between empty/populated streams at byte offset 143, delta of exactly 33 bytes (= 4-byte id + 13-byte name + 8-byte OwnSN + 8-byte PairSN, one `CoupleRecord`). The empty stream's bytes `[143:149)` are `00 00 00 00 00 00` — six zero bytes, i.e. three `WriteShort(0)` (couple count, friend count, marriage count). This is exactly the pre-task shape and is present, at that same position, inside the committed `want` literal (the committed round-trip test already asserts `bytes.Equal` against this exact literal and passes).
- **GMS v28**: first divergence at byte offset 96, delta 33 bytes; empty stream's bytes `[96:98)` are `00 00` — two zero bytes, i.e. one `WriteShort(0)` (couple count only; the legacy gate excludes friend/marriage). Matches the pre-task legacy shape.

Both segments are provably all-zero and of the correct width independent of how the literal was authored — this is exactly the a-priori-knowable "empty path" value the task description asked me to check against. **Verdict: the committed test is a genuine frozen-literal regression guard, not a runtime tautology. Not blocking.**

## Correctness of the change itself

- Delegation is a pure pass-through; no new branching, no new error paths, no nil-slice handling introduced beyond what `RingRecords.EncodeRecords`/`DecodeRecords` (Task 3, out of this task's scope) already implements.
- `CharacterData.Rings` zero value is a `model.RingRecords{}` with three nil slices; `EncodeRecords` writes `len(nil slice) == 0` correctly for all three arms. Confirmed structurally in `model/ring.go:263-281` and confirmed by-bytes above.

## Cross-service seams

None — this is a codec-internal wiring change with no service boundary crossed. `set_field_test.go` (the outer packet's opaque-span test) is unaffected because the empty-path byte shape did not move, which is exactly what Task 4 was required to preserve for the benefit of that (out-of-scope-for-this-task) test.

## Repo conventions

- Struct field addition matches existing `CharacterData` field style (exported, doc'd where non-obvious).
- Comment on `encodeRings`/`decodeRings` documents the delegation and cites the FR-9 invariant — consistent with the file's comment density elsewhere.
- No invented values: the two FR-9 byte literals are captured/derived from the encoder itself (verified above), and the populated-subtest fixture values (`PairCharacterId: 4000`, `OwnSN: 111`, etc.) are arbitrary test data for round-trip assertions, not wire-format claims requiring a decompile citation — consistent with how the brief's table describes those cases ("round-trips field-for-field").

## Not evaluable

- `model.RingRecords.EncodeRecords`/`DecodeRecords` correctness against the IDA derivation (`ring-field-derivation.md`) is Task 3's surface, not this task's diff; I read it only to confirm gate placement, not to re-review its byte layout.
- Tasks 9/11 (population of the `Rings` field from domain data) do not exist yet in this range; nothing to evaluate there.

## Verdict

APPROVED — all brief requirements met, FR-9 empty-path invariant genuinely enforced by a frozen (not runtime-tautological) literal whose ring-segment bytes I independently confirmed correct, gate ownership is singular and correctly placed, no idiom violation introduced, tests/build/vet/gofmt all clean.
