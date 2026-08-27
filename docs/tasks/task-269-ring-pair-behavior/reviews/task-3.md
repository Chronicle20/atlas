# Review: Task 3 (3a + 3b) — ring record codec (site A)

**Range:** `3a2cd247a..720bee275` (2 commits: `961fb763d` docs/3a, `720bee275` feat/3b)
**Scope:** `libs/atlas-packet/model/ring.go` (record-block additions),
`libs/atlas-packet/model/ring_test.go` (record-block tests),
`docs/tasks/task-269-ring-pair-behavior/ring-field-derivation.md` (3a appendix).
Read for contract only, not audited: `libs/atlas-packet/character/data.go:765-779`
(the gate this task copies verbatim), `libs/atlas-packet/model/padded_string.go`
(the existing helper this task deliberately did not reuse), `libs/atlas-packet/test/context.go`
(`test.Variants`, confirms `GMS v28` is exercised).

## Spec compliance — against `task-3b-brief.md`

1. **Record layouts match the pinned splits, verbatim.** `CoupleRecord{PairCharacterId
   uint32; PairCharacterName string; OwnSN int64; PairSN int64}` (ring.go ~163-171),
   `FriendRecord{CoupleRecord; FriendItemId uint32}` (~186-190), `MarriageRecord{MarriageNo,
   GroomId, BrideId uint32; Status uint16; GroomItemId, BrideItemId uint32; GroomName,
   BrideName string}` (~209-218) — field-for-field and order-for-order against the brief's
   tables. PASS.
2. **Widths verified by counting writes, not trusting the test:**
   - `CoupleRecord.encode`: `WriteInt`(4) + `writeRecordName`(13) + `WriteInt64`(8) +
     `WriteInt64`(8) = 33 = 0x21. PASS.
   - `FriendRecord.encode`: `CoupleRecord.encode` (33) + `WriteInt`(4) = 37 = 0x25. PASS.
   - `MarriageRecord.encode`: `WriteInt`×3(12) + `WriteShort`(2) + `WriteInt`×2(8) +
     `writeRecordName`×2(26) = 48 = 0x30. PASS.
3. **Gate copied verbatim, no third idiom.** `grep -n "MajorVersion() > 28"` in `ring.go`
   returns exactly two hits (`EncodeRecords` line ~269, `DecodeRecords` line ~290), both
   `(t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS"`, character-for-
   character identical to `character/data.go:767/775`. The two `isJMS := t.Region() ==
   "JMS"` occurrences in the file (lines 70, 101) predate this diff — they belong to Task
   2's `RingSet.EncodeField`/`DecodeField` and are a different, unrelated JMS-only branch
   (not a version gate); the diff there only adds the wire-order comment, confirmed by
   `git diff 3a2cd247a..720bee275 -- ring.go`. PASS, no third idiom.
4. **Empty-path invariant (PRD FR-9), both variants asserted.** `TestRingRecordsEncode`
   cases `"empty, modern"` (GMS v83) and `"empty, JMS"` (JMS v185) assert exactly
   `00 00 00 00 00 00` (6 bytes); `"empty, legacy"` (GMS v28) asserts exactly `00 00` (2
   bytes) — ring_test.go ~90-93. Independently re-ran: `go test ./model/ -run
   'TestRingRecords' -v` → all subtests PASS, including these three. PASS.
5. **Name truncation at 12 with wire width 13** (ruling 3). `writeRecordName` truncates to
   `recordNameWidth-1` = 12 bytes and always zero-pads to 13 (ring.go, `writeRecordName`).
   `"name truncation"` subtest (20-char name) asserts bytes `41 42 ... 4C 00` — 12 name
   bytes + one mandatory `00` — at extracted offset `2+4 : 2+4+13`, i.e. record bytes
   0x04..0x10 (the *corrected* offset per the brief, not the refuted "20..32"). `"name
   padding"` subtest (`"Ab"`) asserts `41 62` + eleven `00`. Both PASS on independent
   re-run. `GroomName`/`BrideName` share the same `writeRecordName` helper, so the same
   guarantee applies to them (not separately fixture-tested at byte level, but the
   roundtrip test exercises `fixtureMarriageRecord` with 3-char names through the same
   code path — see "Non-blocking" below for the one gap this leaves).
6. **Standalone record types, no reuse of `PairRing`/`MarriageRing`** (ruling 2). Confirmed:
   `CoupleRecord`/`FriendRecord`/`MarriageRecord` are new types with their own `encode`/
   `decodeXRecord` functions; no reference to `PairRing` or `MarriageRing` anywhere in the
   new code. Genuine cross-record duplication (the name-write/read logic) is factored into
   shared `writeRecordName`/`readRecordName` helpers rather than copy-pasted three times —
   this satisfies the "flag genuine duplication a shared helper should absorb" instruction;
   there is none left unaddressed.
7. **`GroomId`/`BrideId` naming, not `MarriageCharacterId`/`MarriagePairCharacterId`**
   (ruling 4). Confirmed in the `MarriageRecord` struct and doc comment, which explicitly
   states both namings are intended to coexist in the file. PASS.
8. **Task-2 deferred-minor comment, both old and new sites** (the specific check this
   review was asked to make). `git diff` shows the "field order IS wire order; do not
   reorder" comment added to both pre-existing Task-2 literals (`&PairRing{...}` and
   `&MarriageRing{...}` in `DecodeField`) and present on all three new `DecodeRecords`
   literals (`decodeCoupleRecord`, `decodeFriendRecord`, `decodeMarriageRecord`). Confirmed
   4 total occurrences via grep on the post-diff file. PASS.
9. **No invented values; every field traces to a decompile line or IDB local-type entry.**
   3a's appendix (`ring-field-derivation.md` Appendix, Task 3a) grades every field
   code-pinned or type-only, with addresses, and explicitly refutes `design.md` §2's
   couple split. 3b's field names (`PairCharacterId`, `OwnSN`, `PairSN`, `GroomId`,
   `BrideId`, etc.) are a legibility rename of the IDA-derived names
   (`dwPairCharacterID`→`PairCharacterId`, `liSN`→`OwnSN`, `liPairSN`→`PairSN`,
   `dwGroomID`→`GroomId`, ...) that the 3b brief itself specifies verbatim in its layout
   table (task-3b-brief.md lines 22-48) — not an independent invention. PASS.
10. **Constants check.** `grep` for `GroomId|BrideId|PairCharacterId|CoupleRecord|
    FriendRecord|MarriageRecord` under `libs/atlas-constants/` returns nothing; no
    duplicate domain type/alias/constant introduced. PASS.

## Task quality

- Build/vet/format clean, independently re-run: `go build ./...`, `go vet ./model/...`,
  `gofmt -l model/ring.go model/ring_test.go` all clean.
- `go test ./model/ -run 'TestRingRecords' -v` — independently re-run, full PASS including
  every `test.Variants` × 5-combination cell of `TestRingRecordsRoundTrip` (confirmed `GMS
  v28` is in `test.Variants`, so the legacy gate is exercised, not just asserted by name).
- `TestRingRecordsRoundTrip` asserts `reader.Available() == 0` after decode — catches
  over/under-read bugs that a value-equality check alone would miss.
- Test honesty: the byte-fixture assertions (`TestRingRecordsEncode`) compare literal hex
  strings hand-derived field-by-field against the pinned layout, not against
  `DecodeRecords`'s own output — a mistranscription in `EncodeRecords` cannot pass by
  symmetry with a matching bug in `DecodeRecords`. This is a real pinning test, not
  circular coverage.
- `TestRingRecordsRoundTrip`'s comparison helpers use `==` (length + element) rather than
  `reflect.DeepEqual`, with a documented reason (nil vs. empty-slice on zero-count decode);
  reasonable and doesn't weaken the assertion since all three record types are plain
  comparable structs.
- 3a's derivation is unusually rigorous: exhaustive reader sweep (`xrefs_to` on every
  `ZList<T>::GetNext`), explicit code-pinned vs. type-only grading per field, and an
  explicit correction of a design.md misread (`sPairCharacterName @0x4fde40` turned out to
  be an unrelated stack local). No field's position or width is left unpinned.

## Non-blocking

- `GroomName`/`BrideName` truncation/padding are not independently byte-pinned the way
  `PairCharacterName` is — only exercised through `fixtureMarriageRecord`'s 3-char names
  (well under the 12-byte limit) in the encode-hex and roundtrip tests. They share
  `writeRecordName` with `PairCharacterName`, whose boundary behavior *is* pinned, so the
  residual risk is low (a regression in `writeRecordName` would break the couple-record
  test first), but a reviewer relying only on the marriage-record tests could not tell a
  12-vs-13 truncation regression in the marriage arm from a passing suite. Consider adding
  one boundary-name subtest for `MarriageRecord` in a follow-up if this codec sees further
  changes.
- Byte-level truncation in `writeRecordName` (`b[:recordNameWidth-1]`) truncates by raw
  bytes, not runes/codepoints — a name landing exactly on a multi-byte boundary could split
  a character. This matches the pre-existing behavior of `WritePaddedString` elsewhere in
  the package, so it is not a regression this task introduces; noting for completeness only.

## Not evaluable

- Whether `dwPairCharacterID`/`dwFriendItemID`/`dwMarriageNo`/`usStatus`/`nBrideItemID`/
  `sGroomName`/`sBrideName` semantics are correct cannot be checked from this diff or from
  live capture — 3a's own appendix marks these "unknown / unverified" because no client
  code reads them; the codec only needs position and width, which are pinned. This is 3a's
  own disclosed limitation, not a gap this review can close.
- Task 4 (the consumer wiring `EncodeRecords`/`DecodeRecords` into `CharacterData`) is out
  of scope for this review; whether the interface `RingRecords{Couple, Friend, Marriage
  []T}` is what Task 4 actually needs cannot be confirmed from this diff alone.

## Scope confirmation

The diff matches the described unit exactly: 3a is docs-only (the derivation appendix, no
`.go` touched), 3b touches only `ring.go` and `ring_test.go` as scoped. No drift from the
task-3b-brief file list.

## Rulings review

All four rulings given in the task brief are followed by the code as written, and I did not
find grounds to disagree with any of them — 3a's evidence for the refuted couple split (an
address-level misread of an unrelated stack local) and the 12-byte truncation ruling (a
disassembled `%s` argument with no length check) are both concretely sourced, not
judgment calls.
