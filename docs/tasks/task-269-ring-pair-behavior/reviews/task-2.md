# Review — Task 2: shared ring codec, field blocks

Commit range reviewed: `c134a7fc9..3a2cd247a` (1 commit,
`3a2cd247a "feat(atlas-packet): shared ring field codec for spawn and
avatar-update"`).

Files in scope: `libs/atlas-packet/model/ring.go` (new, 129 lines),
`libs/atlas-packet/model/ring_test.go` (new, 189 lines). `git diff --stat`
confirms these are the only two files touched — matches the brief exactly,
no scope creep.

## Spec compliance

### Interface shape (brief "Interfaces" section)

- `PairRing{ OwnSN int64; PartnerSN int64; ItemId uint32 }` — matches
  (`ring.go:23-27`), field name `ItemId` per Task 1's verdict rather than
  the brief's `<third field per Task 1>` placeholder. Correct resolution.
- `MarriageRing{ MarriageCharacterId uint32; PartnerCharacterId uint32;
  ItemId uint32 }` — matches (`ring.go:39-43`). Field renamed from the
  brief's placeholder `First`/design.md's `MarriageId` to
  `MarriageCharacterId`, exactly per Task 1's verdict
  (`ring-field-derivation.md` "Verdict" section: "Marriage arm first field:
  it is `dwMarriageCharacterID` — the subject character's OWN character
  id"). Correctly documented in the doc comment at `ring.go:29-37`, citing
  the refutation of `design.md` §3.1.
- `RingSet{ Couple *PairRing; Friendship *PairRing; Marriage *MarriageRing }`
  — matches (`ring.go:47-51`), exported fields, no getters, matches the
  `model.Avatar`/`model.Pet` precedent as directed.
- `EncodeField(w *response.Writer, t tenant.Model)` / `DecodeField(rd
  *request.Reader, t tenant.Model)` — signatures match the brief exactly
  (`ring.go:66`, `ring.go:100`).

### Field widths vs Task 1's derivation

Cross-checked every field against `ring-field-derivation.md`'s per-field
width tables:

| field | derivation width | codec width |
|---|---|---|
| pair own SN | 8 (`DecodeBuffer`) | `int64` via `WriteInt64`/`ReadInt64` — 8 bytes |
| pair partner SN | 8 | `int64` — 8 bytes |
| pair `nItemID` | 4 (`Decode4`) | `uint32` via `WriteInt`/`ReadUint32` — 4 bytes |
| `dwMarriageCharacterID` | 4 | `uint32` — 4 bytes |
| `dwMarriagePairCharacterID` | 4 | `uint32` — 4 bytes |
| `nWeddingRingID` | 4 | `uint32` — 4 bytes |

All match. No width defects.

### Byte-fixture verification (the ambiguity I was asked to independently resolve)

I hand-verified every fixture byte sequence in `ring_test.go` against the
fixture values given in the brief, independent of the (admittedly broken)
`total len` column:

- `OwnSN = 0x1122334455667788` → LE bytes `88 77 66 55 44 33 22 11` —
  matches `ring_test.go:88` couple fixture.
- `PartnerSN = 0x99AABBCCDDEEFF00` → LE bytes `00 FF EE DD CC BB AA 99` —
  matches.
- `ItemId = 0x00001234` → LE `34 12 00 00` — matches.
- Friendship SN/ItemId fixtures verified the same way — match.
- `MarriageCharacterId = 0x000000AA` → LE `AA 00 00 00` — matches.
- JMS entry-count field `1` → LE `01 00 00 00` — matches
  `ring_test.go:100`.

Recomputing total lengths byte-by-byte (not trusting the brief's `total
len` column) gives 23/23/15/27/55, not the brief's stated 24/24/15/28/45.
The `all three` figure of 55 is exactly the sum of the three fully-elaborated
"only" blocks (21+21+13), which is internally consistent and is what the
row's own English description ("concatenation of the three populated
blocks, no separators") demands. I independently reach the same conclusion
the controller already ruled on: **the literal byte sequences are correct,
the `total len` column has arithmetic typos in the brief.** The
implementer's tests encode against the literal hex strings (`hex.DecodeString`
via `hexBytes`, `ring_test.go:44-52`), never the `total len` number, so no
invented byte or count was introduced. Confirmed by running the actual
suite (below) — all fixtures pass.

### Empty-path invariant (PRD FR-9)

- `EncodeField` on a zero-valued `RingSet{}`: three `WriteByte(0)` calls,
  no branch differences by region (`ring.go:78-96`, the `encodePair`
  closure returns immediately after `WriteByte(0)` for a nil pointer, and
  the marriage tail does the same). Confirmed byte-identical to the
  pre-task encoder's 3×`WriteByte(0)`.
- Asserted by test on **both** required variants: `"GMS empty"`
  (`ring_test.go:83`, `RingSet{}` → `"00 00 00"`) and `"JMS empty"`
  (`ring_test.go:98`, same). Also `"GMS v48 empty"` (`ring_test.go:96`).
  Satisfies the brief's explicit requirement to test this on both a GMS and
  a JMS variant.

### Version-gate idiom

`EncodeField`/`DecodeField` use a single `t.Region() == "JMS"` plain string
comparison (`ring.go:67`, `ring.go:101`) — no version gate at all inside the
GMS arm, exactly as the brief mandates ("there is no version gate inside the
GMS arm (design.md §2 OQ-1)"), with the rationale documented in the doc
comment at `ring.go:60-65` citing v48 `CUserPool::OnUserEnterField`
@0x6b277b and v95 `CUserRemote::OnAvatarModified` @0x954110, matching the
brief's instruction not to add one and explaining why to a future editor.
`t.Region() == "JMS"` is the same plain-string-comparison idiom already used
throughout `spawn.go` (e.g. `spawn.go:136` `t.Region() != "JMS"`) and
`avatar.go` (`t.Region() == "GMS" && t.MajorVersion() <= 28`). No third
idiom (`IsRegion`/`MajorAtLeast`) is introduced, which is correct here since
there is genuinely no version gate to express — using `IsRegion`/
`MajorAtLeast` would misleadingly imply a version dimension that doesn't
exist in this codec. No defect.

### `libs/atlas-constants` duplication check

Searched `libs/atlas-constants/` for any existing ring/pair/marriage/couple/
friendship domain type — none found (`grep -rli "ring"` and `grep -rli
"pair|marriage|couple|friendship"` turn up only unrelated skill/job/map
files). `PairRing`/`MarriageRing`/`RingSet` are correctly new wire-only
types local to `atlas-packet/model`, matching the `model.Avatar`/`model.Pet`
precedent of not routing through `atlas-constants` for packet-only shapes.
No duplicated constant.

### No invented values / no `dwPairCharacterId` name

`grep` for `dwPairCharacterId`/`dwFriendCharacterId`/`PairCharacterId` in
the two changed files finds exactly one hit: the doc comment at
`ring.go:21` explaining that the *export's* comment uses that name and is
wrong. The identifier itself is never used as a Go symbol. Correct per the
binding constraint.

## Correctness of the change itself

### `DecodeField`'s struct-literal-with-embedded-reads construction (the specific item I was asked to judge)

`ring.go:110-114` (pair) and `ring.go:124-128` (marriage) build the return
value via a keyed struct literal whose field values are `rd.Read*()` calls:

```go
return &PairRing{
    OwnSN:     rd.ReadInt64(),
    PartnerSN: rd.ReadInt64(),
    ItemId:    rd.ReadUint32(),
}
```

**Functional correctness**: safe. Go's spec ("Order of evaluation")
guarantees that when evaluating the operands of an expression — which
includes the element list of a composite literal — all function calls are
evaluated in lexical left-to-right source order, independent of the
destination keys. I verified this empirically rather than relying on the
spec text alone: I temporarily mutated the pair-decode block to assign
`PartnerSN` before `OwnSN` in source order (still using keys, so a reader
relying on "keys make order irrelevant" would not expect a change) and reran
`TestRingSetFieldRoundTrip` — it failed immediately across every non-empty,
non-marriage-only case (`GMS_v72/couple_only`,
`GMS_v72/friendship_only`, `GMS_v72/all_three`, etc.), confirming (a) the
current code decodes in the correct wire order and (b) a future accidental
reorder would be caught by CI, not silently shipped. File restored
immediately after (`git status --porcelain` clean, no residual diff).

**Readability / editor-safety**: this is the one place I'd push back
non-blockingly. A keyed Go struct literal normally signals "field order
doesn't matter" — that is the entire point of using keys instead of
positional literals. Here that signal is misleading: because the *values*
are side-effecting reads, the source order silently becomes load-bearing
for the wire format, and nothing in the code flags that inversion of the
normal convention. `EncodeField`'s mirror-image code avoids this trap
entirely by using sequential `w.Write*()` statements instead of a
struct-literal write (there is no `response.Writer` composite-literal
equivalent, so this asymmetry is somewhat structural, not a choice), which
makes the write order visually and mechanically unambiguous in a way the
decode side is not. A one-line comment on each of the two literals (e.g.
"field order below is the wire order — Go evaluates composite-literal
elements left-to-right, so do not reorder these keys") would remove the
ambiguity for a future editor who assumes keyed literals are
reorder-safe. Not blocking — the round-trip test already pins the correct
behavior and would fail loudly on a reorder — but worth doing given how
easy the mistake is to introduce silently in review (a diff that only
touches key *order*, not values, reads as a no-op).

### Encode/decode symmetry

`EncodeField`'s `encodePair` closure and `DecodeField`'s `decodePair`
closure are structurally mirror images (flag byte, optional JMS count,
`OwnSN`/`PartnerSN`/`ItemId` in the same order) and the marriage tail is
likewise mirrored. `TestRingSetFieldRoundTrip` (`ring_test.go:150-192`)
exercises this over every `test.Variants` entry (10 tenant configs,
including `JMS v185`) × 5 populated combinations (empty, couple-only,
friendship-only, marriage-only, all-three) = 50 sub-tests, asserting
field-by-field equality, nil-pointer round-trip for each arm independently,
and zero leftover reader bytes (`reader.Available() > 0` check,
`ring_test.go:172-174`). This is real coverage, not just the pinned-byte
table.

## Test honesty

- Ran `go build ./... && go vet ./... && go test ./model/ -run TestRingSet
  -v` in `libs/atlas-packet` — all green, matches the implementer's report.
- Confirmed `TestRingSetFieldRoundTrip` actually fails without correct
  field ordering (see mutation test above) — this is a real regression
  guard, not decorative coverage.
- Did not attempt to prove `TestRingSetEncodeField`'s pinned-byte cases are
  "vacuously passing" — each expected string is independently derived from
  the fixture values (verified above), not copied from the actual encoder
  output, so a wire-shape bug (wrong width, wrong byte order, extra/missing
  field) would be caught.

## Cross-service seam

Task 2 is a leaf library change — `EncodeField`/`DecodeField` are not yet
called from any of the four encoder sites (Tasks 3-6 are marked as not yet
landed in this range; `git diff --stat` confirms only `model/ring.go` and
`model/ring_test.go` changed). There is no consumer in this diff to trace
into. This is expected for Task 2 and not a defect — flagged here only so
the record is explicit that the seam-tracing check does not apply to this
unit.

## Not evaluable

- Whether the eventual call sites in Tasks 3-6 will actually preserve the
  FR-9 empty-path invariant end-to-end (e.g., that a "no ring equipped"
  character genuinely produces a nil `RingSet{}` before it reaches
  `EncodeField`) is outside this diff's surface — `RingSet` construction
  from character/inventory state does not exist yet.
- The JMS wire shape for `Friendship`-only and `Marriage`-only arms is
  round-trip tested (via `TestRingSetFieldRoundTrip` against `JMS v185`)
  but not pinned against a literal byte fixture — the brief's fixture table
  only specifies `JMS empty` and `JMS couple only` byte sequences, so this
  is in-scope-as-specified, not a gap introduced by the implementer.

## Verdict

Spec compliance: PASS — every brief requirement (interface shape, Task 1
field-name binding, field widths, empty-path invariant on both regions,
version-gate idiom, no invented values, no `libs/atlas-constants`
duplication) is met and independently verified against
`ring-field-derivation.md` and hand-computed byte fixtures, not just
"looks right."

Task quality: PASS with one non-blocking readability note (the
`DecodeField` struct-literal ordering risk, above) that a later editor
could stumble into, mitigated but not eliminated by the round-trip test.
