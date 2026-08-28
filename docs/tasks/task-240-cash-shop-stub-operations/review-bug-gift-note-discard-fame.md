# Review — bug-gift-note-discard-fame (commit d5281504e, range 8ada25e79..d5281504e)

Reviewer: atlas-reviewer (Sonnet 5)
Scope: the diff of `8ada25e79..d5281504e` (25 files, +600/-79) across
`libs/atlas-saga`, `services/atlas-channel`, `services/atlas-saga-orchestrator`,
`services/atlas-notes`, plus the bug/report docs. `git diff --stat` matches the
implementer's "Files changed" list in the report — no scope drift.

## Verdict summary

APPROVED_WITH_FINDINGS. The four-module seam is threaded correctly end to end,
every hop's JSON tag matches, `buildFameAwardSaga`'s new skip is correctly
wired with a genuine mixed-batch test, and both build+targeted tests are
green in the modules I ran them in. One non-blocking gap in the mixed-batch
test's discriminating power, and one already-flagged-and-accepted known gap
(`updateNote`) which I re-verified rather than re-litigate.

## Requirement-by-requirement

### 1. JSON tag parity at every Kafka hop

- `libs/atlas-saga/payloads.go:1385` (approx) — `CreateNotePayload.GiftNote bool `json:"giftNote,omitempty"``.
- `services/atlas-saga-orchestrator/.../kafka/message/note/kafka.go:34-37` (approx, orchestrator's own outbound `CommandCreateBody`) — `GiftNote bool `json:"giftNote,omitempty"``.
- `services/atlas-notes/atlas.com/notes/kafka/message/note/kafka.go:37-43` (consumer-side `CommandCreateBody`) — `GiftNote bool `json:"giftNote,omitempty"``.

All three use the identical tag `giftNote`. Confirmed by direct diff read
(not grep alone) — PASS.

Note the orchestrator has *two* structs shaped like `CommandCreateBody`: its
own outbound copy (`saga-orchestrator/kafka/message/note/kafka.go`, written
by `producer.go`) and atlas-notes' inbound copy (consumed by
`consumer.go`). The implementer's report correctly identifies this and
updated both; I traced `CreateNoteCommandProvider` (producer.go:13-27,
`GiftNote: giftNote` at line ~26) into the message body and confirmed the
consumer's `handleNoteCreate` (`kafka/consumer/note/consumer.go:51`) reads
`c.Body.GiftNote` off the matching-tagged struct — PASS.

`StatusEventCreatedBody` was deliberately left untouched
(`services/atlas-notes/.../kafka/message/note/kafka.go:59-60`, no `GiftNote`
field) — matches the brief's explicit instruction and the field is
server-only/write-path-only. PASS.

### 2. `buildFameAwardSaga` skip fires for gift notes, not ordinary notes, including in a mixed batch

`services/atlas-notes/atlas.com/notes/note/processor.go`:
- `Discard` (line ~288) loads the note fresh per iteration via
  `p.ByIdProvider(noteId)()` into `m`, then at line ~309 calls
  `p.buildFameAwardSaga(ch, characterId, m)` — the *loaded* model, not a
  passed-through flag, so `m.GiftNote()` reflects what's actually persisted.
- `buildFameAwardSaga` (line ~326) checks `senderId == 0`, then
  `senderId == recipientId`, then `m.GiftNote()` (line ~337) — three
  independent early returns, none of which is an `else`/`continue` that
  could accidentally skip a sibling loop iteration. The skip is local to
  the single note being evaluated.
- New test `TestDiscardAndEmit_GiftNoteSuppressesFameInMixedBatch`
  (`processor_fame_award_test.go:153-195` approx) creates one gift note and
  one ordinary note from the same sender, discards both in one
  `DiscardAndEmit` call, and asserts `len(fakeSaga.calls) == 1` with the
  surviving payload's `CharacterId == senderId`. Ran it directly:
  `go test ./note/... -run GiftNote -v` → `PASS`. This is a genuine
  regression test — reverting the `m.GiftNote()` check would make it see
  2 calls instead of 1 and fail. PASS.

Non-blocking finding: both notes in the mixed-batch test share the same
`senderId`. A test that additionally used *different* sender ids for the
gift vs. ordinary note (and asserted the surviving payload's `CharacterId`
is specifically the ordinary note's sender, not just "the" sender) would
also catch an inverted condition (e.g. `if !m.GiftNote()`) or a
note-identity mixup, which the current same-sender setup cannot
distinguish from a correct implementation by payload alone (both would
produce a payload with the same `CharacterId`). I read the actual
implementation directly and it is correct (three independent `if` early
returns, no swap), so this is not a live bug — it is a test-strength gap,
not a code defect. Not blocking.

### 3. Backward compatibility — old commands / persisted notes decode to `false`; `omitempty` harmless

- `omitempty` on a Go `bool` field omits the key from marshalled JSON only
  when the value is `false`; on unmarshal, an absent key leaves the Go
  zero value (`false`) in place. Traced through:
  - An in-flight `CreateNotePayload` created before this fix has no
    `giftNote` key; `unmarshal.go:737-742` unmarshals the whole struct via
    `json.Unmarshal(aux.Payload, &payload)` (not field-by-field), so the
    missing key simply leaves `payload.GiftNote == false`. PASS —
    confirmed by reading `unmarshal.go:730-742` directly, not inferred.
  - Same reasoning applies identically at the orchestrator→atlas-notes
    Kafka hop and at `CommandCreateBody` decode in `consumer.go`.
  - Already-persisted `notes` rows: `entity.go`'s `Migration` is
    `AutoMigrate`, so the new `GiftNote` column defaults to the Go zero
    value (`false`) for existing rows — matches report's "Not yet
    answered" §1, which explicitly and correctly flags this as an accepted
    gap for a not-yet-deployed branch, not a silent defect. PASS (as
    documented, not silently absorbed).
  - `createNote` (`administrator.go:10-21`) uses `tx.Create(&entity)`, a
    full-row insert — GORM writes the zero value for `GiftNote` on new
    rows created before this fix shipped (there are none, since the field
    didn't exist), and correctly writes `true`/`false` for rows created
    after. No masking here; the `Updates`-vs-`Create` GORM zero-value
    footgun only applies to `updateNote`, which is out of scope per the
    brief and re-verified below.

### 4. `note_send.go`'s ordinary player-note path is unchanged

`git diff 8ada25e79..d5281504e -- .../note_send.go` returns **no output** —
the file is untouched. `buildNoteSendSaga` keeps the zero-value `GiftNote:
false` implicitly (struct literal omits the field). Test
`note_send_test.go` was extended to assert `np.GiftNote == false` explicitly
(new assertion, not a pre-existing one) — PASS.

### 5. Tests assert the NEW contract at each seam, not just the endpoints

Checked each hop has a test that would fail without the fix, not merely
pass either way:

- `libs/atlas-saga/unmarshal_test.go:1278-1297` — round-trips
  `GiftNote: true` through marshal→unmarshal and asserts `!p.GiftNote` in
  the failure condition. PASS.
- `note_gift_forward_test.go` — asserts `np.GiftNote == true`; `note_send_test.go` — asserts `np.GiftNote == false`. Both new assertions. PASS.
- `saga-orchestrator/note/producer_test.go` — asserts `c.Body.GiftNote` round-trips through the provider with `giftNote=true`. PASS.
- `saga-orchestrator/saga/handler_test.go` — new "Gift note case" table entry, mock closure asserts `giftNote` forwards from payload to `CreateNote`. PASS.
- `atlas-notes/note/builder_test.go` — `TestBuilder_SetGiftNote` and `TestMakeEntityRoundTrip_GiftNote` (the latter covers both a gift note and a plain note through `MakeEntity`/`Make`, explicitly checking the plain note does not pick up a stray `true`). PASS.
- `atlas-notes/note/processor_fame_award_test.go` — the mixed-batch test discussed in §2. PASS (with the noted same-sender limitation).

All ran green: `go test ./note/... -run 'GiftNote|Discard|Builder' -v` in
atlas-notes (all PASS), `go test ./...` in saga-orchestrator (all `ok`),
`go build ./...` clean in both atlas-notes and saga-orchestrator.

### Known gap re-verified: `updateNote`

Read `administrator.go:24-38` directly: `updateNote` calls
`tx.Where("id = ?", note.Id()).Updates(&entity)`. GORM's `Updates` with a
struct argument does skip zero-valued fields, so a `GiftNote: false` passed
through an update would not clear a previously-`true` value, and (more to
the point for this bug) would also not *set* `true` from `false`. Confirmed
`Update`/`UpdateAndEmit` (`processor.go` interface, unchanged in this diff)
were deliberately NOT given a `giftNote` parameter, so this path can never
be asked to write `true` — the flagged gap is real but genuinely
unreachable today, matching both the bug file and the report. Correctly
scoped out, not silently dropped.

## Not evaluable

- Live re-test / actual gift-note discard against a running cluster —
  neither the bug file nor the report claims this was done, and it is
  outside the review surface (no live cluster access from this review
  either).
- Repo-wide `tools/verify.sh` — the report explicitly defers this to
  controller dispatch; I ran targeted `go build`/`go test` in the two
  Go modules with the deepest logic (atlas-notes, saga-orchestrator) as a
  spot check, not a substitute for the full gate.

## Findings

Non-blocking:
1. `services/atlas-notes/atlas.com/notes/note/processor_fame_award_test.go` (new `TestDiscardAndEmit_GiftNoteSuppressesFameInMixedBatch`) — both notes in the mixed batch share `senderId`, so the test cannot distinguish a correct `m.GiftNote()` skip from an inverted or note-identity-swapped one purely by asserting on the surviving payload's `CharacterId`. Verified the actual code is correct by direct read (three independent early returns in `buildFameAwardSaga`), so this is a test-strength observation, not a live defect. Suggest (not required): use distinct sender ids for the gift vs. ordinary note in a follow-up strengthening, no action needed now.

No blocking findings.

## Verdict

APPROVED_WITH_FINDINGS
